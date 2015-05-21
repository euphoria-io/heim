package proto

import (
	"fmt"
	"time"

	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

// A Listing is a sortable list of Identitys present in a Room.
// TODO: these should be Sessions
type Listing []SessionView

func (l Listing) Len() int      { return len(l) }
func (l Listing) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

func (l Listing) Less(i, j int) bool {
	if l[i].Name == l[j].Name {
		if l[i].ID == l[j].ID {
			return l[i].SessionID < l[j].SessionID
		}
		return l[i].ID < l[j].ID
	}
	return l[i].Name < l[j].Name
}

// A Room is a nexus of communication. Users connect to a Room via
// Session and interact.
type Room interface {
	Log

	// BanAgent bans an agent from the room. A zero value for until
	// indicates a permanent ban.
	BanAgent(ctc scope.Context, agentID string, until time.Time) error

	// UnbanAgent removes an agent ban from the room.
	UnbanAgent(ctc scope.Context, agentID string) error

	// BanIP bans an IP from the room. A zero value for until indicates
	// a permanent ban.
	BanIP(ctc scope.Context, ip string, until time.Time) error

	// UnbanIP removes an IP ban from the room.
	UnbanIP(ctc scope.Context, ip string) error

	// Join inserts a Session into the Room's global presence.
	Join(scope.Context, Session) error

	// Part removes a Session from the Room's global presence.
	Part(scope.Context, Session) error

	// Send broadcasts a Message from a Session to the Room.
	Send(scope.Context, Session, Message) (Message, error)

	// Edit modifies or deletes a message.
	EditMessage(scope.Context, Session, EditMessageCommand) error

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

	// IsValidParent checks whether the message with the given ID is able to be replied to.
	IsValidParent(id snowflake.Snowflake) (bool, error)
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

func RoomCapabilitySubject(ctx scope.Context, room Room) (security.CapabilitySubject, error) {
	key, err := room.MasterKey(ctx)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, fmt.Errorf("room is unlocked")
	}
	return &roomCapabilitySubject{RoomKey: key}, nil
}

type roomCapabilitySubject struct {
	RoomKey
}

func (s *roomCapabilitySubject) Nonce(size int) []byte {
	nonce := make([]byte, size)
	data := s.RoomKey.Nonce()
	fmt.Printf("data: %#v\n", data)
	//copy(nonce, s.RoomKey.Nonce())
	copy(nonce, data)
	return nonce
}

func (s *roomCapabilitySubject) PublicData() interface{} { return nil }

func (s *roomCapabilitySubject) PrivateData(kms security.KMS) (interface{}, error) {
	key := s.ManagedKey()
	if err := kms.DecryptKey(&key); err != nil {
		return nil, err
	}
	return key.Plaintext, nil
}
