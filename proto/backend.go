package proto

import (
	"euphoria.io/heim/backend/cluster"
)

// A Backend provides Rooms and an implementation version.
type Backend interface {
	Close()

	// Gets a Room by name. If the Room doesn't already exist, it should
	// be created.
	GetRoom(name string) (Room, error)

	// Peers returns a snapshot of known peers in this backend's cluster.
	Peers() []cluster.PeerDesc

	// Version returns the implementation version string.
	Version() string
}
