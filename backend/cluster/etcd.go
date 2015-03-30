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
	"euphoria.io/scope"

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

func EtcdCluster(ctx scope.Context, root, addr string, desc *PeerDesc) (Cluster, error) {
	fmt.Printf("connecting to %#v\n", addr)
	e := &etcdCluster{
		root:  strings.TrimRight(root, "/") + "/",
		c:     etcd.NewClient([]string{addr}),
		ch:    make(chan PeerEvent),
		stop:  make(chan bool),
		peers: map[string]PeerDesc{},
		ctx:   ctx,
	}
	idx, err := e.init()
	if err != nil {
		return nil, err
	}
	if desc != nil {
		idx, err = e.update(desc)
		if err != nil {
			return nil, err
		}
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
	ctx   scope.Context
}

func (e *etcdCluster) key(format string, args ...interface{}) string {
	return e.root + strings.TrimLeft(fmt.Sprintf(format, args...), "/")
}

func (e *etcdCluster) init() (uint64, error) {
	if !e.c.SyncCluster() {
		return 0, fmt.Errorf("cluster error: failed to sync with %s", e.c.GetCluster())
	}

	resp, err := e.c.Get(e.key("/peers"), false, false)
	if err != nil {
		if etcdErr, ok := err.(*etcd.EtcdError); ok && etcdErr.ErrorCode == 100 {
			return 0, nil
		}
		return 0, fmt.Errorf("cluster error: init: %s", err)
	}
	node := resp.Node
	if !node.Dir {
		return 0, fmt.Errorf("cluster error: init: expected directory")
	}

	latestIndex := uint64(0)
	for _, child := range node.Nodes {
		var desc PeerDesc
		if err := json.Unmarshal([]byte(child.Value), &desc); err != nil {
			return 0, fmt.Errorf("cluster error: init: bad node %s: %s\n", child.Key, err)
		}
		e.peers[desc.ID] = desc
		if child.ModifiedIndex > latestIndex {
			latestIndex = child.ModifiedIndex
		}
	}
	return latestIndex + 1, nil
}

type tree map[string]string

func (t tree) visit(n *etcd.Node, prefix string) {
	if len(n.Key) > len(prefix) {
		t[n.Key[len(prefix):]] = n.Value
	}
	for _, child := range n.Nodes {
		t.visit(child, prefix)
	}
}

func (e *etcdCluster) GetDir(key string) (map[string]string, error) {
	prefix := e.key("%s", key) + "/"
	resp, err := e.c.Get(prefix, false, false)
	if err != nil {
		if etcdErr, ok := err.(*etcd.EtcdError); ok && etcdErr.ErrorCode == 100 {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := tree{}
	result.visit(resp.Node, prefix)
	return map[string]string(result), nil
}

func (e *etcdCluster) GetValue(key string) (string, error) {
	resp, err := e.c.Get(e.key("%s", key), false, false)
	if err != nil {
		if etcdErr, ok := err.(*etcd.EtcdError); ok && etcdErr.ErrorCode == 100 {
			return "", ErrNotFound
		}
		return "", err
	}
	return resp.Node.Value, nil
}

func (e *etcdCluster) SetValue(key, value string) error {
	_, err := e.c.Set(e.key(key), value, 0)
	if err != nil {
		return fmt.Errorf("set on %s: %s", e.key(key), err)
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
	fmt.Printf("writing %s to %s\n", string(valueBytes), desc.ID)
	e.me = e.key("/peers/%s", desc.ID)
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
	go e.c.Watch(e.key("/peers"), waitIndex, true, recv, e.stop)

	for {
		resp := <-recv
		if resp == nil {
			// If this happens the etcd cluster is unhealthy. For now we'll
			// just shut down and hope some other backend instance is happy.
			fmt.Printf("cluster error: watch: nil response\n")
			e.ctx.Terminate(fmt.Errorf("cluster error: watch: nil response"))
			return
		}

		peerID := strings.TrimLeft(strings.TrimPrefix(resp.Node.Key, e.key("/peers")), "/")
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

func (e *etcdCluster) GetValueWithDefault(key string, setter func() (string, error)) (string, error) {
	for {
		resp, err := e.c.Get(e.key("%s", key), false, false)
		if err != nil {
			if etcdErr, ok := err.(*etcd.EtcdError); ok && etcdErr.ErrorCode == 100 {
				value, err := setter()
				if err != nil {
					return "", err
				}
				if _, err := e.c.Create(e.key("%s", key), value, 0); err != nil {
					// Lost the race, repeat.
					continue
				}
				return value, nil
			}
			return "", err
		}
		return resp.Node.Value, nil
	}
}
