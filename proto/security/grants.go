package security

// Capability is a generic handle on a cryptographic grant of access.
type Capability interface {
	// ID() returns the globally unique identifier of the capability.
	// It should be a string derived from a secret shared with the
	// recipient.
	ID() string

	// PublicPayload returns the publicly exposed data associated
	// with the capability.
	PublicPayload() []byte

	// EncryptedPayload returns the encrypted payload associated with
	// this capability. Apply your shared secret to the value that
	// Challenge() returns and pass it to Verify() in order to gain
	// access to the plaintext of the payload.
	EncryptedPayload() []byte
}
