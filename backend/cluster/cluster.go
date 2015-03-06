package cluster

import (
	"heim/proto/security"
	"time"
)

var TTL = 30 * time.Second

type Cluster interface {
	GetSecret(kms security.KMS, name string, bytes int) ([]byte, error)
	Update(desc *PeerDesc) error
	Part()
	Peers() []PeerDesc
	Watch() chan PeerEvent
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
