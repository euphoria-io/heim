package proto

import (
	"time"

	"euphoria.io/scope"

	"heim/proto/security"
)

// A Listing is a sortable list of Identitys present in a Room.
// TODO: these should be Sessions
type Listing []IdentityView

func (l Listing) Len() int      { return len(l) }
func (l Listing) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

func (l Listing) Less(i, j int) bool {
	if l[i].Name == l[j].Name {
		return l[i].ID < l[j].ID
	}
	return l[i].Name < l[j].Name
}

// A Room is a nexus of communication. Users connect to a Room via
// Session and interact.
type Room interface {
	Log

	// Join inserts a Session into the Room's global presence.
	Join(scope.Context, Session) error

	// Part removes a Session from the Room's global presence.
	Part(scope.Context, Session) error

	// Send broadcasts a Message from a Session to the Room.
	Send(scope.Context, Session, Message) (Message, error)

	// Listing returns the current global list of connected sessions to this
	// Room.
	Listing(scope.Context) (Listing, error)

	// RenameUser updates the nickname of a Session in this Room.
	RenameUser(ctx scope.Context, session Session, formerName string) (*NickEvent, error)

	// Version returns the version of the server hosting this Room.
	Version() string

	// GenerateMasterKey generates and stores a new key and nonce
	// for the room. This invalidates all grants made with the
	// previous key.
	GenerateMasterKey(ctx scope.Context, kms security.KMS) (RoomKey, error)

	// MasterKey returns the room's current key, or nil if the room is unlocked.
	MasterKey(ctx scope.Context) (RoomKey, error)

	// SaveCapability saves the given capability.
	SaveCapability(ctx scope.Context, capability security.Capability) error

	// GetCapability retrieves the capability under the given ID, or
	// returns nil if it doesn't exist.
	GetCapability(ctx scope.Context, id string) (security.Capability, error)
}

type RoomKey interface {
	// ID returns a unique identifier for the key.
	KeyID() string

	// Timestamp returns when the key was generated.
	Timestamp() time.Time

	// Nonce returns the current 128-bit nonce for the room.
	Nonce() []byte

	// ManagedKey returns the current encrypted ManagedKey for the room.
	ManagedKey() security.ManagedKey
}
