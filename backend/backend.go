package backend

import (
	"sync"
)

type Backend interface {
	GetRoom(name string) (Room, error)
	Version() string
}

type TestBackend struct {
	sync.Mutex
	rooms   map[string]Room
	version string
}

func (b *TestBackend) Version() string { return b.version }

func (b *TestBackend) GetRoom(name string) (Room, error) {
	b.Lock()
	defer b.Unlock()

	if room, ok := b.rooms[name]; ok {
		return room, nil
	}

	if b.rooms == nil {
		b.rooms = map[string]Room{}
	}

	room := newMemRoom(name, b.version)
	b.rooms[name] = room
	return room, nil
}
