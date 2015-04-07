package mock

import (
	"fmt"
	"sync"
	"time"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/scope"
)

type TestBackend struct {
	sync.Mutex
	agentBans map[string]time.Time
	ipBans    map[string]time.Time
	rooms     map[string]proto.Room
	version   string
}

func (b *TestBackend) Close() {}

func (b *TestBackend) Version() string { return b.version }

func (b *TestBackend) GetRoom(name string, create bool) (proto.Room, error) {
	b.Lock()
	defer b.Unlock()

	if room, ok := b.rooms[name]; ok {
		return room, nil
	}

	if !create {
		return nil, fmt.Errorf("no such room")
	}

	if b.rooms == nil {
		b.rooms = map[string]proto.Room{}
	}

	room := newMemRoom(name, b.version)
	b.rooms[name] = room
	return room, nil
}

func (b *TestBackend) Peers() []cluster.PeerDesc { return nil }

func (b *TestBackend) BanAgent(ctx scope.Context, agentID string, until time.Time) error {
	b.Lock()
	defer b.Unlock()

	if b.agentBans == nil {
		b.agentBans = map[string]time.Time{agentID: until}
	} else {
		b.agentBans[agentID] = until
	}
	return nil
}

func (b *TestBackend) UnbanAgent(ctx scope.Context, agentID string) error {
	b.Lock()
	defer b.Unlock()

	if _, ok := b.agentBans[agentID]; ok {
		delete(b.agentBans, agentID)
	}
	return nil
}

func (b *TestBackend) BanIP(ctx scope.Context, ip string, until time.Time) error {
	b.Lock()
	defer b.Unlock()

	if b.ipBans == nil {
		b.ipBans = map[string]time.Time{ip: until}
	} else {
		b.ipBans[ip] = until
	}
	return nil
}

func (b *TestBackend) UnbanIP(ctx scope.Context, ip string) error {
	b.Lock()
	defer b.Unlock()

	if _, ok := b.ipBans[ip]; ok {
		delete(b.ipBans, ip)
	}
	return nil
}
