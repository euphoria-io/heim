package proto

import (
	"time"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/scope"
)

// A Backend provides Rooms and an implementation version.
type Backend interface {
	// BanAgent globally bans an agent. A zero value for until indicates a
	// permanent ban.
	BanAgent(ctx scope.Context, agentID string, until time.Time) error

	// UnbanAgent removes a global ban.
	UnbanAgent(ctx scope.Context, agentID string) error

	Close()

	// Gets a Room by name. If the Room doesn't already exist, it should
	// be created.
	GetRoom(name string) (Room, error)

	// Peers returns a snapshot of known peers in this backend's cluster.
	Peers() []cluster.PeerDesc

	// Version returns the implementation version string.
	Version() string
}
