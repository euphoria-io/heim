package proto

import (
	"crypto/rand"
	"time"

	"golang.org/x/crypto/poly1305"

	"encoding/base64"

	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

const (
	AgentIDSize  = 8
	AgentKeyType = security.AES128
)

type AgentTracker interface {
	// Register associates the given ID with the given unencrypted key. The key
	// is not stored, but will be required to access the agent.
	Register(ctx scope.Context, agent *Agent) error

	// Get returns details for the agent associated with the given ID.
	Get(ctx scope.Context, agentID string) (*Agent, error)

	// SetClientKey encrypts the given clientKey with accessKey and saves it
	// under the given agentID. Both keys must be unencrypted.
	SetClientKey(
		ctx scope.Context, agentID string, accessKey *security.ManagedKey,
		accountID snowflake.Snowflake, clientKey *security.ManagedKey) error

	// ClearClientKey logs the agent out.
	ClearClientKey(ctx scope.Context, agentID string) error
}

func NewAgent(agentID []byte, accessKey *security.ManagedKey) (*Agent, error) {
	if accessKey.Encrypted() {
		return nil, security.ErrKeyMustBeDecrypted
	}

	iv := make([]byte, accessKey.KeySize())
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	if agentID == nil {
		agentID = make([]byte, AgentIDSize)
		if _, err := rand.Read(agentID); err != nil {
			return nil, err
		}
	}

	var (
		mac [16]byte
		key [32]byte
	)
	copy(key[:], accessKey.Plaintext)
	poly1305.Sum(&mac, iv, &key)

	agent := &Agent{
		ID:      agentID,
		IV:      iv,
		MAC:     mac[:],
		Created: time.Now(),
	}
	return agent, nil
}

type Agent struct {
	ID                 []byte
	IV                 []byte
	MAC                []byte
	EncryptedClientKey *security.ManagedKey
	AccountID          string
	Created            time.Time
	Blessed            bool
	Bot                bool
}

func (a *Agent) IDString() string { return base64.URLEncoding.EncodeToString(a.ID) }

func (a *Agent) verify(accessKey *security.ManagedKey) bool {
	var (
		mac [16]byte
		key [32]byte
	)
	copy(mac[:], a.MAC)
	copy(key[:], accessKey.Plaintext)
	return poly1305.Verify(&mac, a.IV, &key)
}

func (a *Agent) Unlock(accessKey *security.ManagedKey) (*security.ManagedKey, error) {
	if a.EncryptedClientKey == nil {
		return nil, ErrClientKeyNotFound
	}

	if accessKey.Encrypted() {
		return nil, security.ErrKeyMustBeDecrypted
	}

	if !a.verify(accessKey) {
		return nil, ErrAccessDenied
	}

	clientKey := a.EncryptedClientKey.Clone()
	if err := clientKey.Decrypt(accessKey); err != nil {
		return nil, err
	}
	return &clientKey, nil
}

func (a *Agent) SetClientKey(accessKey, clientKey *security.ManagedKey) error {
	if accessKey.Encrypted() || clientKey.Encrypted() {
		return security.ErrKeyMustBeDecrypted
	}

	if !a.verify(accessKey) {
		return ErrAccessDenied
	}

	encryptedClientKey := clientKey.Clone()
	encryptedClientKey.IV = a.IV
	if err := encryptedClientKey.Encrypt(accessKey); err != nil {
		return err
	}

	a.EncryptedClientKey = &encryptedClientKey
	return nil
}
