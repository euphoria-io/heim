package psql

import (
	"time"

	"heim/proto/security"
)

type RoomMasterKey struct {
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

type RoomMasterKeyBinding struct {
	MasterKey
	RoomMasterKey
}

func (rmkb *RoomMasterKeyBinding) KeyID() string        { return rmkb.RoomMasterKey.KeyID }
func (rmkb *RoomMasterKeyBinding) Timestamp() time.Time { return rmkb.RoomMasterKey.Activated }
func (rmkb *RoomMasterKeyBinding) Nonce() []byte        { return rmkb.MasterKey.Nonce }

func (rmkb *RoomMasterKeyBinding) ManagedKey() security.ManagedKey {
	dup := func(v []byte) []byte {
		w := make([]byte, len(v))
		copy(w, v)
		return w
	}

	mkey := security.ManagedKey{
		KeyType:      security.AES128,
		IV:           dup(rmkb.MasterKey.IV),
		Ciphertext:   dup(rmkb.MasterKey.EncryptedKey),
		ContextKey:   "room",
		ContextValue: rmkb.RoomMasterKey.Room,
	}
	return mkey
}

type RoomCapabilityBinding struct {
	Capability
	RoomCapability
}

func (rcb *RoomCapabilityBinding) CapabilityID() string { return rcb.Capability.CapabilityID() }
