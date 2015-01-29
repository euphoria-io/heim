package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"heim/backend/proto"
	"heim/backend/proto/security"

	"golang.org/x/net/context"
)

// NewCapability creates a generic capability for a given grant. It
// requires a decrypted client key (TODO: asymmetric key support),
// a random nonce associated with the subject of the grant, and public
// and private payloads.
//
// Note: nonce *must* be truly random.
func NewCapability(
	kms security.KMS, clientKey *security.ManagedKey, nonce []byte, public, private interface{}) (
	*Capability, error) {

	if len(nonce) != clientKey.BlockSize() {
		return nil, fmt.Errorf("nonce must be %d bytes", clientKey.BlockSize())
	}

	if clientKey.Encrypted() {
		return nil, fmt.Errorf("client key must be decrypted")
	}

	publicData, err := json.Marshal(public)
	if err != nil {
		return nil, err
	}

	privateData, err := json.Marshal(private)
	if err != nil {
		return nil, err
	}

	// Generate capability ID by encrypting nonce with client key. We use
	// the nonce itself as the IV.
	id := make([]byte, len(nonce))
	copy(id, nonce)
	if err := clientKey.KeyType.BlockCrypt(nonce, clientKey.Plaintext, id, true); err != nil {
		return nil, err
	}

	// Use the ID as the IV for encrypting the private payload.
	privateData = clientKey.KeyType.Pad(privateData)
	if err := clientKey.KeyType.BlockCrypt(id, clientKey.Plaintext, privateData, true); err != nil {
		return nil, err
	}

	grant := &Capability{
		IDString:         base64.URLEncoding.EncodeToString(id),
		Nonce:            nonce,
		Public:           publicData,
		EncryptedPrivate: privateData,
	}
	return grant, nil
}

type Capability struct {
	IDString         string
	Nonce            []byte
	Public           []byte
	EncryptedPrivate []byte
}

func (c *Capability) ID() string            { return c.IDString }
func (c *Capability) PublicPayload() []byte { return c.Public }

func (c *Capability) EncryptedPayload() []byte {
	dup := make([]byte, len(c.EncryptedPrivate))
	copy(dup, c.EncryptedPrivate)
	return dup
}

func GrantCapabilityOnRoom(
	ctx context.Context, kms security.KMS, roomKey proto.RoomKey, clientKey *security.ManagedKey) (
	*Capability, error) {

	// Decrypt room key.
	roomManagedKey := roomKey.ManagedKey()
	if err := kms.DecryptKey(&roomManagedKey); err != nil {
		return nil, err
	}

	// TODO: make private data a struct
	return NewCapability(kms, clientKey, roomKey.Nonce(), nil, roomManagedKey.Plaintext)
}
