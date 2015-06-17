package security

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Capability is a generic handle on a cryptographic grant of access.
type Capability interface {
	// CapabilityID() returns the globally unique identifier of the
	// capability. It should be a string derived from a secret shared
	// with the recipient.
	CapabilityID() string

	// Nonce() returns the random nonce associated with the capability.
	// This is an optional feature (used for public-key based grants).
	// If there is no nonce, nil is returned.
	Nonce() []byte

	// PublicPayload returns the publicly exposed data associated
	// with the capability.
	PublicPayload() []byte

	// EncryptedPayload returns the encrypted payload associated with
	// this capability. Apply your shared secret to the value that
	// Challenge() returns and pass it to Verify() in order to gain
	// access to the plaintext of the payload.
	EncryptedPayload() []byte
}

func GrantSharedSecretCapability(key *ManagedKey, nonce []byte, publicData, privateData interface{}) (
	*SharedSecretCapability, error) {

	if key.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	// The ID for a shared secret capability is the nonce encrypted by the
	// shared secret.
	id := make([]byte, len(nonce))
	copy(id, nonce)
	if err := key.BlockCrypt(nonce, key.Plaintext, id, true); err != nil {
		return nil, fmt.Errorf("id generation error: %s", err)
	}

	publicPayload, err := json.Marshal(publicData)
	if err != nil {
		return nil, err
	}

	privatePayload, err := json.Marshal(privateData)
	if err != nil {
		return nil, err
	}

	encryptedPrivatePayload := key.Pad(privatePayload)
	if err := key.BlockCrypt(id, key.Plaintext, encryptedPrivatePayload, true); err != nil {
		return nil, fmt.Errorf("payload encryption error: %s", err)
	}

	grant := &SharedSecretCapability{
		Capability: &capability{
			idString:         base64.URLEncoding.EncodeToString(id),
			public:           publicPayload,
			encryptedPrivate: encryptedPrivatePayload,
		},
	}
	return grant, nil
}

func GrantPublicKeyCapability(
	kms KMS, nonce []byte, subjectKey, holderKey *ManagedKeyPair,
	publicData, privateData interface{}) (
	*PublicKeyCapability, error) {

	if subjectKey.KeyPairType != holderKey.KeyPairType {
		err := fmt.Errorf("key of type %s cannot grant to key of type %s",
			subjectKey.KeyPairType, holderKey.KeyPairType)
		return nil, err
	}

	if subjectKey.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	// Extend nonce for secure transmission of private data to holder.
	// It's imperative that this nonce is unique for this subject-holder pair.
	pkNonce := make([]byte, subjectKey.NonceSize())
	n := copy(pkNonce, nonce)
	if n < len(pkNonce) {
		fill, err := kms.GenerateNonce(len(pkNonce) - n)
		if err != nil {
			return nil, err
		}
		copy(pkNonce[n:], fill)
	}

	// Encode the payloads as JSON.
	publicPayload, err := json.Marshal(publicData)
	if err != nil {
		return nil, err
	}
	privatePayload, err := json.Marshal(privateData)
	if err != nil {
		return nil, err
	}

	// Encrypt the private payload JSON.
	encryptedPrivatePayload, err := subjectKey.Seal(
		privatePayload, pkNonce, holderKey.PublicKey, subjectKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Construct the capability and return.
	offer := &PublicKeyCapability{
		Capability: &capability{
			idString:         PublicKeyCapabilityID(subjectKey, holderKey, nonce),
			public:           publicPayload,
			encryptedPrivate: encryptedPrivatePayload,
			nonce:            pkNonce,
		},
	}
	return offer, nil
}

type capability struct {
	idString         string
	nonce            []byte
	public           []byte
	encryptedPrivate []byte
}

func (c *capability) CapabilityID() string  { return c.idString }
func (c *capability) Nonce() []byte         { return c.nonce }
func (c *capability) PublicPayload() []byte { return c.public }

func (c *capability) EncryptedPayload() []byte {
	dup := make([]byte, len(c.encryptedPrivate))
	copy(dup, c.encryptedPrivate)
	return dup
}

type SharedSecretCapability struct {
	Capability
}

func (c *SharedSecretCapability) DecryptPayload(key *ManagedKey) ([]byte, error) {
	if key.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	nonce, err := base64.URLEncoding.DecodeString(c.CapabilityID())
	if err != nil {
		return nil, err
	}

	payload := make([]byte, len(c.EncryptedPayload()))
	copy(payload, c.EncryptedPayload())
	if err := key.BlockCrypt(nonce, key.Plaintext, payload, false); err != nil {
		return nil, err
	}

	return key.Unpad(payload), nil
}

type PublicKeyCapability struct {
	Capability
}

func (c *PublicKeyCapability) DecryptPayload(subjectKey, holderKey *ManagedKeyPair) ([]byte, error) {
	if holderKey.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	payload, err := holderKey.Open(
		c.EncryptedPayload(), c.Nonce(), subjectKey.PublicKey, holderKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func SharedSecretCapabilityID(key *ManagedKey, nonce []byte) (string, error) {
	if key.Encrypted() {
		return "", ErrKeyMustBeDecrypted
	}

	id := make([]byte, len(nonce))
	copy(id, nonce)
	if err := key.BlockCrypt(nonce, key.Plaintext, id, true); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(id), nil
}

func PublicKeyCapabilityID(subjectKey, holderKey *ManagedKeyPair, nonce []byte) string {
	idBytes := make([]byte, len(subjectKey.PublicKey)+len(holderKey.PublicKey)+len(nonce))
	n := copy(idBytes, subjectKey.PublicKey)
	n += copy(idBytes[n:], holderKey.PublicKey)
	copy(idBytes[n:], nonce)
	return base64.URLEncoding.EncodeToString(idBytes)
}
