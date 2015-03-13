package cluster

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"euphoria.io/heim/proto/security"

	"github.com/coreos/go-etcd/etcd"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	selfAnnouncements = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "self_announcements",
		Subsystem: "peer",
		Help:      "Count of self-announcements to the cluster by this backend.",
	})

	peerEvents = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "events",
		Subsystem: "peer",
		Help:      "Count of cluster peer events observed by this backend.",
	})

	peerLiveCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "live_count",
		Subsystem: "peer",
		Help:      "Count of peers currently live (including self).",
	})

	peerWatchErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "watch_errors",
		Subsystem: "peer",
		Help:      "Count of errors encountered while watching for peer events.",
	})
)

func init() {
	prometheus.MustRegister(selfAnnouncements)
	prometheus.MustRegister(peerEvents)
	prometheus.MustRegister(peerLiveCount)
	prometheus.MustRegister(peerWatchErrors)
}

func EtcdCluster(root, addr string, desc *PeerDesc) (Cluster, error) {
	fmt.Printf("connecting to %#v\n", addr)
	e := &etcdCluster{
		root:  strings.TrimRight(root, "/") + "/",
		c:     etcd.NewClient([]string{addr}),
		ch:    make(chan PeerEvent),
		stop:  make(chan bool),
		peers: map[string]PeerDesc{},
	}
	if err := e.init(desc); err != nil {
		return nil, err
	}
	idx, err := e.update(desc)
	if err != nil {
		return nil, err
	}
	go e.watch(idx)
	return e, nil
}

type etcdCluster struct {
	m     sync.RWMutex
	c     *etcd.Client
	root  string
	me    string
	ch    chan PeerEvent
	stop  chan bool
	peers map[string]PeerDesc
}

func (e *etcdCluster) key(format string, args ...interface{}) string {
	return e.root + strings.TrimLeft(fmt.Sprintf(format, args...), "/")
}

func (e *etcdCluster) init(desc *PeerDesc) error {
	if !e.c.SyncCluster() {
		return fmt.Errorf("cluster error: failed to sync with %s", e.c.GetCluster())
	}

	resp, err := e.c.Get(e.key("/heim"), false, false)
	if err != nil {
		if etcdErr, ok := err.(*etcd.EtcdError); ok && etcdErr.ErrorCode == 100 {
			return nil
		}
		return fmt.Errorf("cluster error: init: %s", err)
	}
	node := resp.Node
	if !node.Dir {
		return fmt.Errorf("cluster error: init: expected directory")
	}
	for _, child := range node.Nodes {
		var desc PeerDesc
		if err := json.Unmarshal([]byte(child.Value), &desc); err != nil {
			return fmt.Errorf("cluster error: init: bad node %s: %s\n", child.Key, err)
		}
		e.peers[desc.ID] = desc
	}
	return nil
}

func (e *etcdCluster) Peers() []PeerDesc {
	e.m.RLock()
	defer e.m.RUnlock()
	peers := make(PeerList, 0, len(e.peers))
	for _, desc := range e.peers {
		peers = append(peers, desc)
	}
	sort.Sort(peers)
	return peers
}

func (e *etcdCluster) Update(desc *PeerDesc) error {
	if _, err := e.update(desc); err != nil {
		return err
	}
	return nil
}

func (e *etcdCluster) update(desc *PeerDesc) (uint64, error) {
	valueBytes, err := json.Marshal(desc)
	if err != nil {
		return 0, err
	}
	e.me = e.key("/heim/%s", desc.ID)
	resp, err := e.c.Set(e.me, string(valueBytes), uint64(TTL/time.Second))
	if err != nil {
		return 0, fmt.Errorf("set on %s: %s", e.me, err)
	}
	selfAnnouncements.Inc()
	e.m.Lock()
	e.peers[desc.ID] = *desc
	e.m.Unlock()
	return resp.Node.ModifiedIndex + 1, nil
}

func (e *etcdCluster) Part() {
	close(e.stop)
	e.c.Delete(e.me, false)
}

func (e *etcdCluster) Watch() <-chan PeerEvent { return e.ch }

func (e *etcdCluster) watch(waitIndex uint64) {
	defer close(e.ch)

	recv := make(chan *etcd.Response)
	go e.c.Watch(e.key("/heim"), waitIndex, true, recv, e.stop)

	for {
		resp := <-recv
		if resp == nil {
			fmt.Printf("cluster error: watch: nil response\n")
			peerWatchErrors.Inc()
			break
		}

		peerID := strings.TrimLeft(strings.TrimPrefix(resp.Node.Key, e.key("/heim")), "/")
		switch resp.Action {
		case "set":
			var desc PeerDesc
			if err := json.Unmarshal([]byte(resp.Node.Value), &desc); err != nil {
				fmt.Printf("cluster error: set: %s\n", err)
				peerWatchErrors.Inc()
				continue
			}
			e.m.Lock()
			prev, updated := e.peers[desc.ID]
			e.peers[desc.ID] = desc
			e.m.Unlock()
			if updated {
				if prev.Era != desc.Era {
					fmt.Printf("peer watch: update %s\n", desc.ID)
				}
				e.ch <- &PeerAliveEvent{desc}
			} else {
				fmt.Printf("peer watch: set %s\n", desc.ID)
				e.ch <- &PeerJoinedEvent{desc}
			}
			peerEvents.Inc()
		case "expire", "delete":
			fmt.Printf("peer watch: %s %s\n", resp.Action, peerID)
			e.m.Lock()
			delete(e.peers, peerID)
			e.m.Unlock()
			e.ch <- &PeerLostEvent{PeerDesc{ID: peerID}}
			peerEvents.Inc()
		default:
			fmt.Printf("peer watch: ignoring watch event: %v\n", resp)
		}

		peerLiveCount.Set(float64(len(e.peers)))
	}
}

func (e *etcdCluster) GetSecret(kms security.KMS, name string, bytes int) ([]byte, error) {
	resp, err := e.c.Get(e.key("/secrets/%s", name), false, false)
	if err != nil {
		if etcdErr, ok := err.(*etcd.EtcdError); ok && etcdErr.ErrorCode == 100 {
			return e.setSecret(kms, name, bytes)
		}
		return nil, err
	}

	secret, err := hex.DecodeString(resp.Node.Value)
	if err != nil {
		return nil, err
	}

	if len(secret) != bytes {
		return nil, fmt.Errorf("secret inconsistent: expected %d bytes, got %d", bytes, len(secret))
	}

	return secret, nil
}

func (e *etcdCluster) setSecret(kms security.KMS, name string, bytes int) ([]byte, error) {
	// Generate our own key.
	secret, err := kms.GenerateNonce(bytes)
	if err != nil {
		return nil, err
	}

	// Try to stake our claim on this secret.
	if _, err := e.c.Create(e.key("/secrets/%s", name), hex.EncodeToString(secret), 0); err != nil {
		if etcdErr, ok := err.(*etcd.EtcdError); ok && etcdErr.ErrorCode == 105 {
			// Lost the race, try to use GetSecret again.
			return e.GetSecret(kms, name, bytes)
		}
		return nil, err
	}

	return secret, nil
}
