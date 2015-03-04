package cluster

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	etcdAddrs = flag.String("etcd-peers", "", "comma-separated addresses of etcd peers")
	etcdPath  = flag.String("etcd", "", "etcd path for cluster coordination")

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

func EtcdClusterFromFlags(desc *PeerDesc) (Cluster, error) {
	return EtcdCluster(*etcdPath, strings.Split(*etcdAddrs, ","), desc)
}

func EtcdCluster(root string, peers []string, desc *PeerDesc) (Cluster, error) {
	e := &etcdCluster{
		root:  strings.TrimRight(root, "/") + "/",
		c:     etcd.NewClient(peers),
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
	resp, err := e.c.Get(e.root, false, false)
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
	e.me = e.key("/%s", desc.ID)
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

func (e *etcdCluster) Watch() chan PeerEvent { return e.ch }

func (e *etcdCluster) watch(waitIndex uint64) {
	defer close(e.ch)

	for {
		resp, err := e.c.Watch(e.root, waitIndex, true, nil, e.stop)
		if err != nil {
			if err == etcd.ErrWatchStoppedByUser {
				break
			}
			fmt.Printf("cluster error: watch: %s\n", err)
			peerWatchErrors.Inc()
			break
		}
		if resp == nil {
			fmt.Printf("cluster error: watch: nil response\n")
			peerWatchErrors.Inc()
			break
		}

		waitIndex = resp.Node.ModifiedIndex + 1

		peerID := strings.TrimLeft(strings.TrimPrefix(resp.Node.Key, e.root), "/")
		switch resp.Action {
		case "set":
			var desc PeerDesc
			if err := json.Unmarshal([]byte(resp.Node.Value), &desc); err != nil {
				fmt.Printf("cluster error: set: %s\n", err)
				peerWatchErrors.Inc()
				continue
			}
			e.m.Lock()
			_, updated := e.peers[desc.ID]
			e.peers[desc.ID] = desc
			e.m.Unlock()
			if updated {
				e.ch <- &PeerAliveEvent{desc}
			} else {
				e.ch <- &PeerJoinedEvent{desc}
			}
			peerEvents.Inc()
		case "expire", "delete":
			e.m.Lock()
			delete(e.peers, peerID)
			e.m.Unlock()
			e.ch <- &PeerLostEvent{PeerDesc{ID: peerID}}
			peerEvents.Inc()
		default:
			//fmt.Printf("ignoring watch event: %v\n", resp)
		}

		peerLiveCount.Set(float64(len(e.peers)))
	}
}
