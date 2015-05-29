package psql

import (
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
)

type RoomMessageKey struct {
	Room      string
	KeyID     string `db:"key_id"`
	Activated time.Time
	Expired   time.Time
	Comment   string
}

type RoomCapability struct {
	Room         string
	CapabilityID string `db:"capability_id"`
	Granted      time.Time
	Revoked      time.Time
}

type RoomMessageKeyBinding struct {
	MessageKey
	RoomMessageKey
}

func (rmkb *RoomMessageKeyBinding) KeyID() string        { return rmkb.RoomMessageKey.KeyID }
func (rmkb *RoomMessageKeyBinding) Timestamp() time.Time { return rmkb.RoomMessageKey.Activated }
func (rmkb *RoomMessageKeyBinding) Nonce() []byte        { return rmkb.MessageKey.Nonce }

func (rmkb *RoomMessageKeyBinding) ManagedKey() security.ManagedKey {
	dup := func(v []byte) []byte {
		w := make([]byte, len(v))
		copy(w, v)
		return w
	}

	mkey := security.ManagedKey{
		KeyType:      security.AES128,
		IV:           dup(rmkb.MessageKey.IV),
		Ciphertext:   dup(rmkb.MessageKey.EncryptedKey),
		ContextKey:   "room",
		ContextValue: rmkb.RoomMessageKey.Room,
	}
	return mkey
}

type RoomCapabilityBinding struct {
	Capability
	RoomCapability
}

func (rcb *RoomCapabilityBinding) CapabilityID() string { return rcb.Capability.CapabilityID() }

type RoomManagerKeyBinding struct {
	*Room
}

func (rmkb *RoomManagerKeyBinding) KeyPair() security.ManagedKeyPair {
	return security.ManagedKeyPair{
		KeyPairType:         security.Curve25519,
		IV:                  rmkb.Room.IV,
		EncryptedPrivateKey: rmkb.Room.EncryptedPrivateKey,
		PublicKey:           rmkb.Room.PublicKey,
	}
}

func (rmkb *RoomManagerKeyBinding) Unlock(
	managerKey *security.ManagedKey) (*security.ManagedKeyPair, error) {

	sec := &proto.RoomSecurity{
		MAC: rmkb.Room.MAC,
		KeyEncryptingKey: security.ManagedKey{
			KeyType:      security.AES256,
			Ciphertext:   rmkb.Room.EncryptedManagementKey,
			ContextKey:   "room",
			ContextValue: rmkb.Room.Name,
		},
		KeyPair: rmkb.KeyPair(),
	}
	return sec.Unlock(managerKey)
}

type RoomManager struct {
	Room         string
	AccountID    string `db:"account_id"`
	CapabilityID string `db:"capability_id"`
}
