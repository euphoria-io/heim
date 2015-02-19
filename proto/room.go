package proto

import (
	"time"

	"heim/proto/security"

	"golang.org/x/net/context"
)

// A Listing is a sortable list of Identitys present in a Room.
// TODO: these should be Sessions
type Listing []IdentityView

func (l Listing) Len() int           { return len(l) }
func (l Listing) Less(i, j int) bool { return l[i].ID < l[j].ID }
func (l Listing) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// A Room is a nexus of communication. Users connect to a Room via
// Session and interact.
type Room interface {
	Log

	// Join inserts a Session into the Room's global presence.
	Join(context.Context, Session) error

	// Part removes a Session from the Room's global presence.
	Part(context.Context, Session) error

	// Send broadcasts a Message from a Session to the Room.
	Send(context.Context, Session, Message) (Message, error)

	// Listing returns the current global list of connected sessions to this
	// Room.
	Listing(context.Context) (Listing, error)

	// RenameUser updates the nickname of a Session in this Room.
	RenameUser(ctx context.Context, session Session, formerName string) (*NickEvent, error)

	// Version returns the version of the server hosting this Room.
	Version() string

	// GenerateMasterKey generates and stores a new key and nonce
	// for the room. This invalidates all grants made with the
	// previous key.
	GenerateMasterKey(ctx context.Context, kms security.KMS) (RoomKey, error)

	// MasterKey returns the room's current key, or nil if the room is unlocked.
	MasterKey(ctx context.Context) (RoomKey, error)

	// SaveCapability saves the given capability.
	SaveCapability(ctx context.Context, capability security.Capability) error

	// GetCapability retrieves the capability under the given ID, or
	// returns nil if it doesn't exist.
	GetCapability(ctx context.Context, id string) (security.Capability, error)
}

type RoomKey interface {
	// Timestamp returns when the key was generated.
	Timestamp() time.Time

	// Nonce returns the current 128-bit nonce for the room.
	Nonce() []byte

	// ManagedKey returns the current encrypted ManagedKey for the room.
	ManagedKey() security.ManagedKey
}
