package proto

import (
	"time"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

// A Backend provides Rooms and an implementation version.
type Backend interface {
	// BanAgent globally bans an agent. A zero value for until indicates a
	// permanent ban.
	BanAgent(ctx scope.Context, agentID string, until time.Time) error

	// UnbanAgent removes a global ban.
	UnbanAgent(ctx scope.Context, agentID string) error

	// BanIP globally bans an IP. A zero value for until indicates a
	// permanent ban.
	BanIP(ctx scope.Context, ip string, until time.Time) error

	// UnbanIP removes a global ban.
	UnbanIP(ctx scope.Context, ip string) error

	Close()

	// Gets a Room by name. If the Room doesn't already exist and create is
	// true, a new room will be created and returned.
	GetRoom(name string, create bool) (Room, error)

	// Peers returns a snapshot of known peers in this backend's cluster.
	Peers() []cluster.PeerDesc

	// Version returns the implementation version string.
	Version() string

	// GetAccount returns the account with the given ID.
	GetAccount(ctx scope.Context, id snowflake.Snowflake) (Account, error)

	// RegisterAccount creates and returns a new, unverified account.
	RegisterAccount(ctx scope.Context, kms security.KMS, namespace, id, password string) (Account, error)

	// ResolveAccount returns any account registered under the given account identity.
	ResolveAccount(ctx scope.Context, namespace, id string) (Account, error)
}
