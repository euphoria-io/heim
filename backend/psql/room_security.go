package psql

import (
	"database/sql"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"gopkg.in/gorp.v1"
)

type RoomCapability struct {
	Room         string
	CapabilityID string `db:"capability_id"`
	AccountID    string `db:"account_id"`
	Granted      time.Time
	Revoked      time.Time
}

type RoomCapabilityBinding struct {
	AccountID string `db:"account_id"`
	Capability
	RoomCapability
}

func (rcb *RoomCapabilityBinding) CapabilityID() string { return rcb.Capability.CapabilityID() }

type RoomManagerCapability RoomCapability

type RoomManagerCapabilityBinding struct {
	AccountID string `db:"account_id"`
	Capability
	RoomManagerCapability
}

func (rmcb *RoomManagerCapabilityBinding) CapabilityID() string { return rmcb.Capability.CapabilityID() }

type RoomManagerCapabilities struct {
	Room     *Room
	Executor gorp.SqlExecutor
}

func (rmc *RoomManagerCapabilities) Get(ctx scope.Context, cid string) (security.Capability, error) {
	rmcb := &RoomManagerCapabilityBinding{}
	err := rmc.Executor.SelectOne(
		rmcb,
		`SELECT r.room, r.capability_id, r.granted, r.revoked,`+
			` c.id, c.account_id, c.nonce, c.encrypted_private_data, c.public_data`+
			` FROM room_manager_capability r, capability c`+
			` WHERE r.room = $1 AND c.id = $2 AND r.capability_id = c.id AND r.revoked < r.granted`,
		rmc.Room.Name, cid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrCapabilityNotFound
		}
		return nil, err
	}
	rmcb.RoomManagerCapability.AccountID = rmcb.AccountID
	return rmcb, nil
}

func (rmc *RoomManagerCapabilities) Save(
	ctx scope.Context, account proto.Account, c security.Capability) error {

	capRow := &Capability{
		ID:                   c.CapabilityID(),
		NonceBytes:           c.Nonce(),
		EncryptedPrivateData: c.EncryptedPayload(),
		PublicData:           c.PublicPayload(),
	}
	rmCapRow := &RoomManagerCapability{
		Room:         rmc.Room.Name,
		CapabilityID: c.CapabilityID(),
		Granted:      time.Now(),
	}
	if account != nil {
		capRow.AccountID = account.ID().String()
		rmCapRow.AccountID = account.ID().String()
	}
	return rmc.Executor.Insert(capRow, rmCapRow)
}

func (rmc *RoomManagerCapabilities) Remove(ctx scope.Context, capabilityID string) error {
	resp, err := rmc.Executor.Exec("DELETE FROM capability WHERE id = $1", capabilityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return proto.ErrManagerNotFound
		}
		return err
	}
	n, err := resp.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return proto.ErrManagerNotFound
	}
	return nil
}

type RoomMessageCapabilities struct {
	Room     *Room
	Executor gorp.SqlExecutor
}

func (rmc *RoomMessageCapabilities) Get(ctx scope.Context, cid string) (security.Capability, error) {
	rcb := &RoomCapabilityBinding{}
	err := rmc.Executor.SelectOne(
		rcb,
		`SELECT r.room, r.capability_id, r.granted, r.revoked,`+
			` c.id, c.account_id, c.nonce, c.encrypted_private_data, c.public_data`+
			` FROM room_capability r, capability c`+
			` WHERE r.room = $1 AND c.id = $2 AND r.capability_id = c.id AND r.revoked < r.granted`,
		rmc.Room.Name, cid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrCapabilityNotFound
		}
		return nil, err
	}
	rcb.RoomCapability.AccountID = rcb.AccountID
	return rcb, nil
}

func (rmc *RoomMessageCapabilities) Save(
	ctx scope.Context, account proto.Account, c security.Capability) error {

	capRow := &Capability{
		ID:                   c.CapabilityID(),
		NonceBytes:           c.Nonce(),
		EncryptedPrivateData: c.EncryptedPayload(),
		PublicData:           c.PublicPayload(),
	}
	roomCapRow := &RoomCapability{
		Room:         rmc.Room.Name,
		CapabilityID: c.CapabilityID(),
		Granted:      time.Now(),
	}
	if account != nil {
		capRow.AccountID = account.ID().String()
		roomCapRow.AccountID = account.ID().String()
	}
	return rmc.Executor.Insert(capRow, roomCapRow)
}

func (rmc *RoomMessageCapabilities) Remove(ctx scope.Context, capabilityID string) error {
	resp, err := rmc.Executor.Exec("DELETE FROM capability WHERE id = $1", capabilityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return proto.ErrCapabilityNotFound
		}
		return err
	}
	n, err := resp.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return proto.ErrCapabilityNotFound
	}
	return nil
}

type RoomMessageKey struct {
	Room      string
	KeyID     string `db:"key_id"`
	Activated time.Time
	Expired   time.Time
	Comment   string
}

type RoomMessageKeyBinding struct {
	*proto.GrantManager
	MessageKey
	RoomMessageKey
}

func NewRoomMessageKeyBinding(
	rb *RoomBinding, keyID snowflake.Snowflake, msgKey *security.ManagedKey,
	nonce []byte) *RoomMessageKeyBinding {

	rmkb := &RoomMessageKeyBinding{
		GrantManager: &proto.GrantManager{
			Capabilities: &RoomMessageCapabilities{
				Room:     rb.Room,
				Executor: rb.Backend.DbMap,
			},
			Managers: NewRoomManagerKeyBinding(rb),
			KeyEncryptingKey: &security.ManagedKey{
				Ciphertext:   rb.Room.EncryptedManagementKey,
				ContextKey:   "room",
				ContextValue: rb.Room.Name,
			},
			SubjectKeyPair: &security.ManagedKeyPair{
				KeyPairType:         security.Curve25519,
				IV:                  rb.Room.IV,
				EncryptedPrivateKey: rb.Room.EncryptedPrivateKey,
				PublicKey:           rb.Room.PublicKey,
			},
			PayloadKey:   msgKey,
			SubjectNonce: nonce,
		},
		MessageKey: MessageKey{
			ID:           keyID.String(),
			EncryptedKey: msgKey.Ciphertext,
			IV:           msgKey.IV,
			Nonce:        nonce,
		},
		RoomMessageKey: RoomMessageKey{
			Room:      rb.Room.Name,
			KeyID:     keyID.String(),
			Activated: time.Now(),
		},
	}
	return rmkb
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
		KeyType:      proto.RoomMessageKeyType,
		IV:           dup(rmkb.MessageKey.IV),
		Ciphertext:   dup(rmkb.MessageKey.EncryptedKey),
		ContextKey:   "room",
		ContextValue: rmkb.RoomMessageKey.Room,
	}
	return mkey
}

type RoomManagerKeyBinding struct {
	*Room
	*RoomManagerCapabilities
	*proto.GrantManager
}

func NewRoomManagerKeyBinding(rb *RoomBinding) *RoomManagerKeyBinding {
	rmkb := &RoomManagerKeyBinding{
		Room: rb.Room,
		RoomManagerCapabilities: &RoomManagerCapabilities{
			Room:     rb.Room,
			Executor: rb.Backend.DbMap,
		},
		GrantManager: &proto.GrantManager{
			KeyEncryptingKey: &security.ManagedKey{
				KeyType:      proto.RoomManagerKeyType,
				Ciphertext:   rb.Room.EncryptedManagementKey,
				ContextKey:   "room",
				ContextValue: rb.Room.Name,
			},
			SubjectKeyPair: &security.ManagedKeyPair{
				KeyPairType:         security.Curve25519,
				IV:                  rb.Room.IV,
				EncryptedPrivateKey: rb.Room.EncryptedPrivateKey,
				PublicKey:           rb.Room.PublicKey,
			},
			SubjectNonce: rb.Room.Nonce,
		},
	}
	rmkb.GrantManager.Capabilities = rmkb.RoomManagerCapabilities
	rmkb.GrantManager.Managers = rmkb
	return rmkb
}

func (rmkb *RoomManagerKeyBinding) SetExecutor(executor gorp.SqlExecutor) {
	rmkb.RoomManagerCapabilities.Executor = executor
}

func (rmkb *RoomManagerKeyBinding) Nonce() []byte { return rmkb.GrantManager.SubjectNonce }

func (rmkb *RoomManagerKeyBinding) KeyPair() security.ManagedKeyPair {
	return rmkb.GrantManager.SubjectKeyPair.Clone()
}

func (rmkb *RoomManagerKeyBinding) Unlock(
	managerKey *security.ManagedKey) (*security.ManagedKeyPair, error) {

	sec := &proto.RoomSecurity{
		MAC: rmkb.Room.MAC,
		KeyEncryptingKey: security.ManagedKey{
			KeyType:      proto.RoomManagerKeyType,
			Ciphertext:   rmkb.Room.EncryptedManagementKey,
			ContextKey:   "room",
			ContextValue: rmkb.Room.Name,
		},
		KeyPair: rmkb.KeyPair(),
	}
	return sec.Unlock(managerKey)
}

func (rmkb *RoomManagerKeyBinding) StaffUnlock(kms security.KMS) (*security.ManagedKeyPair, error) {
	kek := security.ManagedKey{
		KeyType:      proto.RoomManagerKeyType,
		Ciphertext:   rmkb.Room.EncryptedManagementKey,
		ContextKey:   "room",
		ContextValue: rmkb.Room.Name,
	}
	if err := kms.DecryptKey(&kek); err != nil {
		return nil, err
	}
	kp := rmkb.KeyPair()
	if err := kp.Decrypt(&kek); err != nil {
		return nil, err
	}
	return &kp, nil
}
