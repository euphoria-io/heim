package security

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"golang.org/x/crypto/nacl/box"
)

var (
	ErrInvalidPublicKey       = errors.New("invalid public key")
	ErrInvalidPrivateKey      = errors.New("invalid private key")
	ErrInvalidNonce           = errors.New("invalid nonce")
	ErrMessageIntegrityFailed = errors.New("message integrity failed")
)

type KeyPairType byte

const (
	Curve25519 KeyPairType = iota
)

func (t KeyPairType) PrivateKeySize() int {
	switch t {
	case Curve25519:
		return 32
	default:
		panic(fmt.Sprintf("no private key size defined for key type %s", t))
	}
}

func (t KeyPairType) PublicKeySize() int {
	switch t {
	case Curve25519:
		return 32
	default:
		panic(fmt.Sprintf("no public key size defined for key type %s", t))
	}
}

func (t KeyPairType) NonceSize() int {
	switch t {
	case Curve25519:
		return 24
	default:
		return 0
	}
}

func (t KeyPairType) String() string {
	switch t {
	case Curve25519:
		return "curve25519"
	default:
		return strconv.Itoa(int(t))
	}
}

func (t KeyPairType) checkNonceAndKeys(nonce, publicKey, privateKey []byte) error {
	if len(publicKey) != t.PublicKeySize() {
		return ErrInvalidPublicKey
	}

	if len(privateKey) != t.PrivateKeySize() {
		return ErrInvalidPrivateKey
	}

	if len(nonce) != t.NonceSize() {
		return ErrInvalidNonce
	}

	return nil
}

func (t KeyPairType) Seal(message, nonce, peersPublicKey, privateKey []byte) ([]byte, error) {
	if err := t.checkNonceAndKeys(nonce, peersPublicKey, privateKey); err != nil {
		return nil, err
	}

	switch t {
	case Curve25519:
		var (
			pubKey, privKey [32]byte
			n               [24]byte
		)
		copy(n[:], nonce)
		copy(pubKey[:], peersPublicKey)
		copy(privKey[:], privateKey)
		out := make([]byte, 0, len(message)+box.Overhead)
		out = box.Seal(out, message, &n, &pubKey, &privKey)
		return out, nil
	default:
		return nil, ErrInvalidKey
	}
}

func (t KeyPairType) Open(message, nonce, peersPublicKey, privateKey []byte) ([]byte, error) {
	if err := t.checkNonceAndKeys(nonce, peersPublicKey, privateKey); err != nil {
		return nil, err
	}

	switch t {
	case Curve25519:
		var (
			ok              bool
			pubKey, privKey [32]byte
			n               [24]byte
		)
		copy(n[:], nonce)
		copy(pubKey[:], peersPublicKey)
		copy(privKey[:], privateKey)
		out := make([]byte, 0, len(message)-box.Overhead)
		out, ok = box.Open(out, message, &n, &pubKey, &privKey)
		if !ok {
			return nil, ErrMessageIntegrityFailed
		}
		return out, nil
	default:
		return nil, ErrInvalidKey
	}
}

func (t KeyPairType) Generate(randomReader io.Reader) (*ManagedKeyPair, error) {
	switch t {
	case Curve25519:
		publicKey, privateKey, err := box.GenerateKey(randomReader)
		if err != nil {
			return nil, err
		}
		key := &ManagedKeyPair{
			KeyPairType: t,
			PrivateKey:  privateKey[:],
			PublicKey:   publicKey[:],
		}
		return key, nil
	default:
		return nil, ErrInvalidKey
	}
}

type ManagedKeyPair struct {
	KeyPairType
	IV                  []byte
	PrivateKey          []byte
	EncryptedPrivateKey []byte
	PublicKey           []byte
}

func (k *ManagedKeyPair) Clone() ManagedKeyPair {
	dup := func(v []byte) []byte {
		if v == nil {
			return nil
		}
		w := make([]byte, len(v))
		copy(w, v)
		return w
	}
	return ManagedKeyPair{
		KeyPairType:         k.KeyPairType,
		IV:                  dup(k.IV),
		PrivateKey:          dup(k.PrivateKey),
		EncryptedPrivateKey: dup(k.EncryptedPrivateKey),
		PublicKey:           dup(k.PublicKey),
	}
}

func (k *ManagedKeyPair) Encrypted() bool { return k.EncryptedPrivateKey != nil }

func (k *ManagedKeyPair) Encrypt(keyKey *ManagedKey) error {
	if keyKey.Encrypted() || k.Encrypted() {
		return ErrKeyMustBeDecrypted
	}

	if k.IV == nil {
		return ErrIVRequired
	}

	buf := k.PrivateKey
	if k.PrivateKeySize()%keyKey.BlockSize() != 0 {
		buf = keyKey.Pad(buf)
	}

	if err := keyKey.BlockCrypt(k.IV, keyKey.Plaintext, buf, true); err != nil {
		return err
	}

	k.EncryptedPrivateKey = buf
	k.PrivateKey = nil
	return nil
}

func (k *ManagedKeyPair) Decrypt(keyKey *ManagedKey) error {
	if keyKey.Encrypted() {
		return ErrKeyMustBeDecrypted
	}

	if !k.Encrypted() {
		return ErrKeyMustBeEncrypted
	}

	if k.IV == nil {
		return ErrIVRequired
	}

	buf := k.EncryptedPrivateKey

	if err := keyKey.BlockCrypt(k.IV, keyKey.Plaintext, buf, false); err != nil {
		return fmt.Errorf("key-encrypting-key decrypt error: %s", err)
	}

	if k.PrivateKeySize()%keyKey.BlockSize() != 0 {
		buf = keyKey.Unpad(buf)
	}
	k.PrivateKey = buf
	k.EncryptedPrivateKey = nil
	return nil
}
