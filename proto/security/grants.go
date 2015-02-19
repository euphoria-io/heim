package security

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/net/context"
)

const keyDerivationIterations = 4096

// Capability is a generic handle on a cryptographic grant of access.
type Capability interface {
	// CapabilityID() returns the globally unique identifier of the
	// capability. It should be a string derived from a secret shared
	// with the recipient.
	CapabilityID() string

	// PublicPayload returns the publicly exposed data associated
	// with the capability.
	PublicPayload() []byte

	// EncryptedPayload returns the encrypted payload associated with
	// this capability. Apply your shared secret to the value that
	// Challenge() returns and pass it to Verify() in order to gain
	// access to the plaintext of the payload.
	EncryptedPayload() []byte
}

// NewCapability creates a generic capability for a given grant. It
// requires a decrypted client key (TODO: asymmetric key support),
// a random nonce associated with the subject of the grant, and public
// and private payloads.
//
// The nonce *must* be truly random, and must be the same size as
// the clientKey's BlockSize.
func NewCapability(kms KMS, clientKey *ManagedKey, nonce []byte, public, private interface{}) (
	Capability, error) {

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

	grant := &capability{
		IDString:         base64.URLEncoding.EncodeToString(id),
		Nonce:            nonce,
		Public:           publicData,
		EncryptedPrivate: privateData,
	}
	return grant, nil
}

type capability struct {
	IDString         string
	Nonce            []byte
	Public           []byte
	EncryptedPrivate []byte
}

func (c *capability) CapabilityID() string  { return c.IDString }
func (c *capability) PublicPayload() []byte { return c.Public }

func (c *capability) EncryptedPayload() []byte {
	dup := make([]byte, len(c.EncryptedPrivate))
	copy(dup, c.EncryptedPrivate)
	return dup
}

func GrantCapabilityOnSubject(
	ctx context.Context, kms KMS, nonce []byte, encryptedSubjectKey, clientKey *ManagedKey) (
	Capability, error) {

	// Decrypt subject key.
	subjectKey := encryptedSubjectKey.Clone()
	if err := kms.DecryptKey(&subjectKey); err != nil {
		return nil, err
	}

	// TODO: make private data a struct
	return NewCapability(kms, clientKey, nonce, nil, subjectKey.Plaintext)
}

func GrantCapabilityOnSubjectWithPasscode(
	ctx context.Context, kms KMS, nonce []byte, encryptedSubjectKey *ManagedKey, passcode []byte) (
	Capability, error) {

	// Use nonce as salt.
	clientKey := &ManagedKey{
		Plaintext: pbkdf2.Key(passcode, nonce, keyDerivationIterations, AES256.KeySize(), sha256.New),
	}
	return GrantCapabilityOnSubject(ctx, kms, nonce, encryptedSubjectKey, clientKey)
}
