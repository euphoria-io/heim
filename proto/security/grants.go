package security

import (
	"encoding/base64"
	"encoding/json"
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

type CapabilitySubject interface {
	Nonce(size int) []byte
	PublicData() interface{}
	PrivateData(kms KMS) (interface{}, error)
}

type CapabilityHolder interface {
	NonceSize() int
	Sign(nonce []byte) ([]byte, error)
	Seal(iv, data []byte) ([]byte, error)
	Open(iv, data []byte) ([]byte, error)
}

// NewCapability creates a capability that is granted to holder on subject.
// The public and private values will be encoded to JSON. The encoding of
// the private value will be sealed such that only the holder can access it.
func NewCapability(kms KMS, holder CapabilityHolder, subject CapabilitySubject) (Capability, error) {
	nonce := subject.Nonce(holder.NonceSize())
	id, err := holder.Sign(nonce)
	if err != nil {
		return nil, err
	}

	publicData, err := json.Marshal(subject.PublicData())
	if err != nil {
		return nil, err
	}

	data, err := subject.PrivateData(kms)
	if err != nil {
		return nil, err
	}
	privateData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	encryptedPrivateData, err := holder.Seal(id, privateData)
	if err != nil {
		return nil, err
	}

	grant := &capability{
		IDString:         base64.URLEncoding.EncodeToString(id),
		Public:           publicData,
		EncryptedPrivate: encryptedPrivateData,
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

func PasscodeCapabilityHolder(passcode, salt []byte) CapabilityHolder {
	key := KeyFromPasscode(passcode, salt, AES128.KeySize())
	return key.CapabilityHolder()
}

func GetCapabilityID(holder CapabilityHolder, subject CapabilitySubject) (string, error) {
	nonce := subject.Nonce(holder.NonceSize())
	id, err := holder.Sign(nonce)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(id), nil
}
