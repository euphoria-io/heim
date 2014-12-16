package backend

import (
	"sync"
)

type Backend interface {
	GetRoom(name string) (Room, error)
}

type TestBackend struct {
	sync.Mutex
	rooms map[string]Room
}

func (b *TestBackend) GetRoom(name string) (Room, error) {
	b.Lock()
	defer b.Unlock()

	if room, ok := b.rooms[name]; ok {
		return room, nil
	}

	if b.rooms == nil {
		b.rooms = map[string]Room{}
	}

	room := newMemRoom(name)
	b.rooms[name] = room
	return room, nil
}
