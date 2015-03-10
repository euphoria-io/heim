package mock

import (
	"sync"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto"
)

type TestBackend struct {
	sync.Mutex
	rooms   map[string]proto.Room
	version string
}

func (b *TestBackend) Close() {}

func (b *TestBackend) Version() string { return b.version }

func (b *TestBackend) GetRoom(name string) (proto.Room, error) {
	b.Lock()
	defer b.Unlock()

	if room, ok := b.rooms[name]; ok {
		return room, nil
	}

	if b.rooms == nil {
		b.rooms = map[string]proto.Room{}
	}

	room := newMemRoom(name, b.version)
	b.rooms[name] = room
	return room, nil
}

func (b *TestBackend) Peers() []cluster.PeerDesc { return nil }
