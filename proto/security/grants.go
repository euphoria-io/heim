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
			IDString:         base64.URLEncoding.EncodeToString(id),
			Public:           publicPayload,
			EncryptedPrivate: encryptedPrivatePayload,
		},
	}
	return grant, nil
}

func GrantPublicKeyCapability(
	kms KMS, subjectKey, holderKey *ManagedKeyPair, publicData, privateData interface{}) (
	*PublicKeyCapability, error) {

	if subjectKey.KeyPairType != holderKey.KeyPairType {
		err := fmt.Errorf("key of type %s cannot grant to key of type %s",
			subjectKey.KeyPairType, holderKey.KeyPairType)
		return nil, err
	}

	if subjectKey.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	// Generate nonce for secure transmission of private data to holder.
	// It's imperative that this nonce is unique for this subject-holder pair.
	nonce, err := kms.GenerateNonce(subjectKey.NonceSize())
	if err != nil {
		return nil, err
	}

	// Generate a unique identifier from the subject's public key and the nonce.
	idBytes := make([]byte, len(subjectKey.PublicKey)+len(nonce))
	copy(idBytes, subjectKey.PublicKey)
	copy(idBytes[len(subjectKey.PublicKey):], nonce)

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
		privatePayload, nonce, holderKey.PublicKey, subjectKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Construct the capability and return.
	offer := &PublicKeyCapability{
		Capability: &capability{
			IDString:         base64.URLEncoding.EncodeToString(idBytes),
			Public:           publicPayload,
			EncryptedPrivate: encryptedPrivatePayload,
		},
	}
	return offer, nil
}

type capability struct {
	IDString         string
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

	idBytes, err := base64.URLEncoding.DecodeString(c.CapabilityID())
	if err != nil {
		return nil, err
	}

	if len(idBytes) != len(subjectKey.PublicKey)+subjectKey.NonceSize() {
		return nil, fmt.Errorf("invalid capability ID")
	}

	nonce := idBytes[len(subjectKey.PublicKey):]

	payload, err := holderKey.Open(
		c.EncryptedPayload(), nonce, subjectKey.PublicKey, holderKey.PrivateKey)
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
