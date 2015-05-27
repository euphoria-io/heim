package proto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/poly1305"

	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
)

type AccountIdentity interface {
	Namespace() string
	ID() string
}

type Account interface {
	ID() snowflake.Snowflake
	KeyEncryptingKey() security.ManagedKey
	KeyPair() security.ManagedKeyPair
	Nonce() []byte
	Verified() bool
}

// NewAccountSecurity initializes the nonce and account secrets for a new account
// with the given password. Returns an encrypted key-encrypting-key, encrypted
// key-pair, nonce, and error.
func NewAccountSecurity(kms security.KMS, password string) (*AccountSecurity, error) {
	kType := security.AES256
	kpType := security.Curve25519

	// Use one KMS request to obtain all the randomness we need:
	//   - nonce
	//   - private key
	randomData, err := kms.GenerateNonce(kpType.NonceSize() + kpType.PrivateKeySize())
	if err != nil {
		return nil, fmt.Errorf("rng error: %s", err)
	}
	randomReader := bytes.NewReader(randomData)

	// Generate nonce with random data. Use to populate IV.
	nonce := make([]byte, kpType.NonceSize())
	if _, err := io.ReadFull(randomReader, nonce); err != nil {
		return nil, fmt.Errorf("rng error: %s", err)
	}
	iv := make([]byte, kType.BlockSize())
	copy(iv, nonce)

	// Generate key-encrypting-key using KMS. This will be returned encrypted,
	// using the base64 encoding of the nonce as its context.
	nonceBase64 := base64.URLEncoding.EncodeToString(nonce)
	systemKek, err := kms.GenerateEncryptedKey(kType, "nonce", nonceBase64)
	if err != nil {
		return nil, fmt.Errorf("key generation error: %s", err)
	}

	// Generate private key using randomReader.
	keyPair, err := kpType.Generate(randomReader)
	if err != nil {
		return nil, fmt.Errorf("keypair generation error: %s", err)
	}

	// Decrypt key-encrypting-key so we can encrypt keypair, and so we can re-encrypt
	// it using the user's key.
	kek := systemKek.Clone()
	if err = kms.DecryptKey(&kek); err != nil {
		return nil, fmt.Errorf("key decryption error: %s", err)
	}

	// Encrypt private key.
	keyPair.IV = iv
	if err = keyPair.Encrypt(&kek); err != nil {
		return nil, fmt.Errorf("keypair encryption error: %s", err)
	}

	// Clone key-encrypting-key and encrypt with client key.
	clientKey := security.KeyFromPasscode([]byte(password), nonce, kType)
	userKek := kek.Clone()
	if err := userKek.Encrypt(clientKey); err != nil {
		return nil, fmt.Errorf("key encryption error: %s", err)
	}

	// Generate message authentication code, for verifying passwords.
	var (
		mac [16]byte
		key [32]byte
	)
	copy(key[:], clientKey.Plaintext)
	poly1305.Sum(&mac, nonce, &key)

	sec := &AccountSecurity{
		Nonce:     nonce,
		MAC:       mac[:],
		SystemKek: *systemKek,
		UserKek:   userKek,
		KeyPair:   *keyPair,
	}
	return sec, nil
}

type AccountSecurity struct {
	Nonce     []byte
	MAC       []byte
	SystemKek security.ManagedKey
	UserKek   security.ManagedKey
	KeyPair   security.ManagedKeyPair
}

func (sec *AccountSecurity) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	if clientKey.Encrypted() {
		return nil, security.ErrKeyMustBeDecrypted
	}

	var (
		mac [16]byte
		key [32]byte
	)
	copy(mac[:], sec.MAC)
	copy(key[:], clientKey.Plaintext)
	if !poly1305.Verify(&mac, sec.Nonce, &key) {
		return nil, ErrAccessDenied
	}

	kek := sec.UserKek.Clone()
	if err := kek.Decrypt(clientKey); err != nil {
		return nil, err
	}

	kp := sec.KeyPair.Clone()
	if err := kp.Decrypt(&kek); err != nil {
		return nil, err
	}

	return &kp, nil
}

func (sec *AccountSecurity) ResetPassword(kms security.KMS, password string) (*AccountSecurity, error) {
	kek := sec.SystemKek.Clone()
	if err := kms.DecryptKey(&kek); err != nil {
		return nil, fmt.Errorf("key decryption error: %s", err)
	}

	clientKey := security.KeyFromPasscode([]byte(password), sec.Nonce, sec.UserKek.KeyType)
	if err := kek.Encrypt(clientKey); err != nil {
		return nil, fmt.Errorf("key encryption error: %s", err)
	}

	var (
		mac [16]byte
		key [32]byte
	)
	copy(key[:], clientKey.Plaintext)
	poly1305.Sum(&mac, sec.Nonce, &key)

	nsec := &AccountSecurity{
		Nonce:     sec.Nonce,
		MAC:       mac[:],
		SystemKek: sec.SystemKek,
		UserKek:   kek,
		KeyPair:   sec.KeyPair,
	}
	return nsec, nil
}
