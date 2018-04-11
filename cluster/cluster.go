package cluster // import "euphoria.io/heim/cluster"

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"euphoria.io/heim/proto/security"
)

var (
	TTL = 30 * time.Second

	ErrNotFound = fmt.Errorf("not found")
)

type Cluster interface {
	GetDir(key string) (map[string]string, error)
	GetValue(key string) (string, error)
	SetValue(key, value string) error
	GetValueWithDefault(key string, setter func() (string, error)) (string, error)

	GetSecret(kms security.KMS, name string, bytes int) ([]byte, error)

	Update(desc *PeerDesc) error
	Part()
	Peers() []PeerDesc
	Watch() <-chan PeerEvent
}

type PeerEvent interface {
	Peer() *PeerDesc
}

type PeerDesc struct {
	ID      string `json:"id"`
	Era     string `json:"era"`
	Version string `json:"version"`
}

func (p *PeerDesc) Peer() *PeerDesc { return p }

type PeerJoinedEvent struct {
	PeerDesc
}

type PeerAliveEvent struct {
	PeerDesc
}

type PeerLostEvent struct {
	PeerDesc
}

type PeerList []PeerDesc

func (ps PeerList) Len() int           { return len(ps) }
func (ps PeerList) Less(i, j int) bool { return ps[i].ID < ps[j].ID }
func (ps PeerList) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }

type TestCluster struct {
	sync.Mutex
	data    map[string]string
	peers   map[string]PeerDesc
	secrets map[string][]byte
	c       chan PeerEvent
	myID    string
}

func (tc *TestCluster) GetDir(key string) (map[string]string, error) {
	tc.Lock()
	defer tc.Unlock()
	key = strings.TrimRight(key, "/") + "/"
	result := map[string]string{}
	for k, v := range tc.data {
		if strings.HasPrefix(k, key) {
			result[k[len(key):]] = v
		}
	}
	return result, nil
}

func (tc *TestCluster) GetValue(key string) (string, error) {
	tc.Lock()
	defer tc.Unlock()
	data, ok := tc.data[key]
	if !ok {
		return "", ErrNotFound
	}
	return data, nil
}

func (tc *TestCluster) SetValue(key, value string) error {
	tc.Lock()
	tc.data[key] = value
	tc.Unlock()
	return nil
}

func (tc *TestCluster) GetValueWithDefault(key string, setter func() (string, error)) (string, error) {
	tc.Lock()
	defer tc.Unlock()
	if val, ok := tc.data[key]; ok {
		return val, nil
	}
	val, err := setter()
	if err != nil {
		return "", err
	}
	if tc.data == nil {
		tc.data = map[string]string{}
	}
	tc.data[key] = val
	return val, nil
}

func (tc *TestCluster) GetSecret(kms security.KMS, name string, bytes int) ([]byte, error) {
	tc.Lock()
	defer tc.Unlock()

	if secret, ok := tc.secrets[name]; ok {
		if len(secret) != bytes {
			return nil, fmt.Errorf("secret inconsistent: expected %d bytes, got %d", bytes, len(secret))
		}
		return secret, nil
	}

	secret, err := kms.GenerateNonce(bytes)
	if err != nil {
		return nil, err
	}

	if tc.secrets == nil {
		tc.secrets = map[string][]byte{name: secret}
	} else {
		tc.secrets[name] = secret
	}
	return secret, nil
}

func (tc *TestCluster) update(desc *PeerDesc) PeerEvent {
	tc.Lock()
	defer tc.Unlock()

	if tc.myID == "" {
		tc.myID = desc.ID
	}

	if tc.peers == nil {
		tc.peers = map[string]PeerDesc{}
	}

	if tc.c == nil {
		tc.peers[desc.ID] = *desc
		return nil
	}

	_, ok := tc.peers[desc.ID]
	tc.peers[desc.ID] = *desc
	if ok {
		return &PeerAliveEvent{*desc}
	} else {
		return &PeerJoinedEvent{*desc}
	}
}

func (tc *TestCluster) Update(desc *PeerDesc) error {
	if event := tc.update(desc); event != nil {
		tc.c <- event
	}
	return nil
}

func (tc *TestCluster) part() PeerEvent {
	tc.Lock()
	defer tc.Unlock()
	desc, ok := tc.peers[tc.myID]
	delete(tc.peers, tc.myID)
	if ok {
		return &PeerLostEvent{desc}
	}
	return nil
}

func (tc *TestCluster) Part() {
	if event := tc.part(); event != nil {
		tc.c <- event
	}
}

func (tc *TestCluster) Peers() []PeerDesc {
	tc.Lock()
	defer tc.Unlock()
	peers := []PeerDesc{}
	for _, peer := range tc.peers {
		peers = append(peers, peer)
	}
	return peers
}

func (tc *TestCluster) Watch() <-chan PeerEvent {
	tc.Lock()
	defer tc.Unlock()
	if tc.c == nil {
		tc.c = make(chan PeerEvent)
	}
	return tc.c
}
