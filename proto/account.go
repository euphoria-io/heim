package proto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/poly1305"

	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

const (
	MinPasswordLength = 6
	ClientKeyType     = security.AES128
)

type AccountManager interface {
	// GetAccount returns the account with the given ID.
	Get(ctx scope.Context, id snowflake.Snowflake) (Account, error)

	// RegisterAccount creates and returns a new, unverified account, along with
	// its (unencrypted) client key.
	Register(
		ctx scope.Context, kms security.KMS, namespace, id, password string,
		agentID string, agentKey *security.ManagedKey) (
		Account, *security.ManagedKey, error)

	// ResolveAccount returns any account registered under the given account identity.
	Resolve(ctx scope.Context, namespace, id string) (Account, error)

	// GrantStaff adds a StaffKMS capability to the identified account.
	GrantStaff(ctx scope.Context, accountID snowflake.Snowflake, kmsCred security.KMSCredential) error

	// RevokeStaff removes a StaffKMS capability from the identified account.
	RevokeStaff(ctx scope.Context, accountID snowflake.Snowflake) error

	// VerifyPersonalIdentity marks a personal identity as verified.
	VerifyPersonalIdentity(ctx scope.Context, namespace, id string) error

	// ChangeClientKey re-encrypts account keys with a new client key.
	// The correct former client key must also be given.
	ChangeClientKey(
		ctx scope.Context, accountID snowflake.Snowflake,
		oldClientKey, newClientKey *security.ManagedKey) error
}

type PersonalIdentity interface {
	Namespace() string
	ID() string
	Verified() bool
}

func ValidatePersonalIdentity(namespace, id string) (bool, string) {
	switch namespace {
	case "email":
		return true, ""
	default:
		return false, fmt.Sprintf("invalid namespace: %s", namespace)
	}
}

func ValidateAccountPassword(password string) (bool, string) {
	if len(password) < MinPasswordLength {
		return false, fmt.Sprintf("password must be at least %d characters long", MinPasswordLength)
	}
	return true, ""
}

type Account interface {
	ID() snowflake.Snowflake
	KeyFromPassword(password string) *security.ManagedKey
	KeyPair() security.ManagedKeyPair
	Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error)
	IsStaff() bool
	UnlockStaffKMS(clientKey *security.ManagedKey) (security.KMS, error)
	PersonalIdentities() []PersonalIdentity
	UserKey() security.ManagedKey
	SystemKey() security.ManagedKey
}

// NewAccountSecurity initializes the nonce and account secrets for a new account
// with the given password. Returns an encrypted key-encrypting-key, encrypted
// key-pair, nonce, and error.
func NewAccountSecurity(
	kms security.KMS, password string) (*AccountSecurity, *security.ManagedKey, error) {

	kpType := security.Curve25519

	// Use one KMS request to obtain all the randomness we need:
	//   - nonce
	//   - private key
	randomData, err := kms.GenerateNonce(kpType.NonceSize() + kpType.PrivateKeySize())
	if err != nil {
		return nil, nil, fmt.Errorf("rng error: %s", err)
	}
	randomReader := bytes.NewReader(randomData)

	// Generate nonce with random data. Use to populate IV.
	nonce := make([]byte, kpType.NonceSize())
	if _, err := io.ReadFull(randomReader, nonce); err != nil {
		return nil, nil, fmt.Errorf("rng error: %s", err)
	}
	iv := make([]byte, ClientKeyType.BlockSize())
	copy(iv, nonce)

	// Generate key-encrypting-key using KMS. This will be returned encrypted,
	// using the base64 encoding of the nonce as its context.
	nonceBase64 := base64.URLEncoding.EncodeToString(nonce)
	systemKey, err := kms.GenerateEncryptedKey(ClientKeyType, "nonce", nonceBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("key generation error: %s", err)
	}

	// Generate private key using randomReader.
	keyPair, err := kpType.Generate(randomReader)
	if err != nil {
		return nil, nil, fmt.Errorf("keypair generation error: %s", err)
	}

	// Decrypt key-encrypting-key so we can encrypt keypair, and so we can re-encrypt
	// it using the user's key.
	kek := systemKey.Clone()
	if err = kms.DecryptKey(&kek); err != nil {
		return nil, nil, fmt.Errorf("key decryption error: %s", err)
	}

	// Encrypt private key.
	keyPair.IV = iv
	if err = keyPair.Encrypt(&kek); err != nil {
		return nil, nil, fmt.Errorf("keypair encryption error: %s", err)
	}

	// Clone key-encrypting-key and encrypt with client key.
	clientKey := security.KeyFromPasscode([]byte(password), nonce, ClientKeyType)
	userKey := kek.Clone()
	userKey.IV = iv
	if err := userKey.Encrypt(clientKey); err != nil {
		return nil, nil, fmt.Errorf("key encryption error: %s", err)
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
		SystemKey: *systemKey,
		UserKey:   userKey,
		KeyPair:   *keyPair,
	}
	return sec, clientKey, nil
}

type AccountSecurity struct {
	Nonce     []byte
	MAC       []byte
	SystemKey security.ManagedKey
	UserKey   security.ManagedKey
	KeyPair   security.ManagedKeyPair
}

func (sec *AccountSecurity) unlock(clientKey *security.ManagedKey) (
	*security.ManagedKey, *security.ManagedKeyPair, error) {

	if clientKey.Encrypted() {
		return nil, nil, security.ErrKeyMustBeDecrypted
	}

	var (
		mac [16]byte
		key [32]byte
	)
	copy(mac[:], sec.MAC)
	copy(key[:], clientKey.Plaintext)
	if !poly1305.Verify(&mac, sec.Nonce, &key) {
		return nil, nil, ErrAccessDenied
	}

	kek := sec.UserKey.Clone()
	if err := kek.Decrypt(clientKey); err != nil {
		return nil, nil, err
	}

	kp := sec.KeyPair.Clone()
	if err := kp.Decrypt(&kek); err != nil {
		return nil, nil, err
	}

	return &kek, &kp, nil
}

func (sec *AccountSecurity) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	_, kp, err := sec.unlock(clientKey)
	return kp, err
}

func (sec *AccountSecurity) ChangeClientKey(oldKey, newKey *security.ManagedKey) error {
	if oldKey.Encrypted() || newKey.Encrypted() {
		return security.ErrKeyMustBeDecrypted
	}

	// Extract decrypted UserKey and verify correctness of oldKey.
	kek, _, err := sec.unlock(oldKey)
	if err != nil {
		return err
	}

	// Encrypt new UserKey.
	if err := kek.Encrypt(newKey); err != nil {
		return err
	}

	// Update MAC and encrypted UserKey.
	var (
		mac [16]byte
		key [32]byte
	)
	copy(key[:], newKey.Plaintext)
	poly1305.Sum(&mac, sec.Nonce, &key)
	sec.MAC = mac[:]
	sec.UserKey = *kek

	return nil
}

func (sec *AccountSecurity) ResetPassword(kms security.KMS, password string) (*AccountSecurity, error) {
	kek := sec.SystemKey.Clone()
	if err := kms.DecryptKey(&kek); err != nil {
		return nil, fmt.Errorf("key decryption error: %s", err)
	}
	kek.IV = make([]byte, ClientKeyType.BlockSize())
	copy(kek.IV, sec.Nonce)

	clientKey := security.KeyFromPasscode([]byte(password), sec.Nonce, sec.UserKey.KeyType)
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
		SystemKey: sec.SystemKey,
		UserKey:   kek,
		KeyPair:   sec.KeyPair,
	}
	return nsec, nil
}
