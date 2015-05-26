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

type SharedSecretCapability interface {
	Capability
	DecryptPayload(*ManagedKey) ([]byte, error)
}

type PublicKeyCapability interface {
	Capability
	DecryptPayload(subjectKey, holderKey *ManagedKeyPair) ([]byte, error)
}

func GrantSharedSecretCapability(key *ManagedKey, nonce []byte, publicData, privateData interface{}) (
	SharedSecretCapability, error) {

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

	grant := &sharedSecretCapability{
		capability: capability{
			IDString:         base64.URLEncoding.EncodeToString(id),
			Public:           publicPayload,
			EncryptedPrivate: encryptedPrivatePayload,
		},
	}
	return grant, nil
}

func OfferPublicKeyCapability(
	subjectKey, holderKey *ManagedKeyPair, nonce []byte, publicData, privateData interface{}) (
	PublicKeyCapability, error) {

	if subjectKey.KeyPairType != holderKey.KeyPairType {
		err := fmt.Errorf("key of type %s cannot grant to key of type %s",
			subjectKey.KeyPairType, holderKey.KeyPairType)
		return nil, err
	}

	if subjectKey.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	fullNonce := make([]byte, subjectKey.NonceSize())
	copy(fullNonce, nonce)

	seal := func(message []byte) ([]byte, error) {
		return subjectKey.Seal(message, fullNonce, holderKey.PublicKey, subjectKey.PrivateKey)
	}

	id, err := seal(nonce)
	if err != nil {
		return nil, err
	}

	publicPayload, err := json.Marshal(publicData)
	if err != nil {
		return nil, err
	}

	privatePayload, err := json.Marshal(privateData)
	if err != nil {
		return nil, err
	}

	encryptedPrivatePayload, err := seal(privatePayload)
	if err != nil {
		return nil, err
	}

	offer := &publicKeyCapability{
		capability: capability{
			IDString:         base64.URLEncoding.EncodeToString(id),
			Public:           publicPayload,
			EncryptedPrivate: encryptedPrivatePayload,
		},
	}
	return offer, nil
}

func AcceptPublicKeyCapability(
	subjectKey, holderKey *ManagedKeyPair, nonce []byte, offer Capability) (
	PublicKeyCapability, error) {

	if holderKey.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	// Verify that holder can decrypt the offer.
	fullNonce := make([]byte, subjectKey.NonceSize())
	copy(fullNonce, nonce)
	_, err := holderKey.Open(
		offer.EncryptedPayload(), fullNonce, subjectKey.PublicKey, holderKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Return clone with new id.
	id, err := subjectKey.Seal(nonce, fullNonce, subjectKey.PublicKey, holderKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	grant := &publicKeyCapability{
		capability: capability{
			IDString:         base64.URLEncoding.EncodeToString(id),
			Public:           offer.PublicPayload(),
			EncryptedPrivate: offer.EncryptedPayload(),
		},
	}
	return grant, nil
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

type sharedSecretCapability struct {
	capability
}

func (c *sharedSecretCapability) DecryptPayload(key *ManagedKey) ([]byte, error) {
	if key.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	nonce, err := base64.URLEncoding.DecodeString(c.IDString)
	if err != nil {
		return nil, err
	}

	payload := make([]byte, len(c.EncryptedPrivate))
	copy(payload, c.EncryptedPrivate)
	if err := key.BlockCrypt(nonce, key.Plaintext, payload, false); err != nil {
		return nil, err
	}

	return key.Unpad(payload), nil
}

type publicKeyCapability struct {
	capability
}

func (c *publicKeyCapability) DecryptPayload(subjectKey, holderKey *ManagedKeyPair) ([]byte, error) {
	if holderKey.Encrypted() {
		return nil, ErrKeyMustBeDecrypted
	}

	nonce, err := base64.URLEncoding.DecodeString(c.IDString)
	if err != nil {
		return nil, err
	}

	fullNonce := make([]byte, subjectKey.NonceSize())
	copy(fullNonce, nonce)

	payload, err := holderKey.Open(
		c.EncryptedPrivate, fullNonce, subjectKey.PublicKey, holderKey.PrivateKey)
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
