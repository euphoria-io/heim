package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/crypto/pbkdf2"
)

const keyDerivationIterations = 4096

var (
	ErrInvalidKey         = errors.New("invalid key")
	ErrKeyMustBeDecrypted = errors.New("key must be decrypted")
	ErrKeyMustBeEncrypted = errors.New("key must be encrypted")
	ErrIVRequired         = errors.New("IV is required")

	TestMode bool
)

type KeyType byte

const (
	AES128 KeyType = iota
	AES256
)

func (t KeyType) KeySize() int {
	switch t {
	case AES128:
		return 16
	case AES256:
		return 32
	default:
		panic(fmt.Sprintf("no key size defined for key type %s", t))
	}
}

func (t KeyType) BlockSize() int {
	switch t {
	case AES128, AES256:
		return 16
	default:
		panic(fmt.Sprintf("no block size defined for key type %s", t))
	}
}

func (t KeyType) String() string {
	switch t {
	case AES128:
		return "aes-128"
	case AES256:
		return "aes-256"
	default:
		return strconv.Itoa(int(t))
	}
}

func (t KeyType) BlockCrypt(iv, key, data []byte, encrypt bool) error {
	if len(data)%t.BlockSize() != 0 {
		return ErrInvalidKey
	}
	blockMode, err := t.BlockMode(iv, key, encrypt)
	if err != nil {
		return err
	}
	blockMode.CryptBlocks(data, data)
	return nil
}

func (t KeyType) BlockMode(iv, key []byte, encrypt bool) (cipher.BlockMode, error) {
	if len(iv) != t.BlockSize() {
		return nil, ErrInvalidKey
	}
	blockCipher, err := t.BlockCipher(key)
	if err != nil {
		return nil, err
	}
	if encrypt {
		return cipher.NewCBCEncrypter(blockCipher, iv), nil
	} else {
		return cipher.NewCBCDecrypter(blockCipher, iv), nil
	}
}

func (t KeyType) BlockCipher(key []byte) (cipher.Block, error) {
	if len(key) != t.KeySize() {
		return nil, ErrInvalidKey
	}
	switch t {
	case AES128, AES256:
		return aes.NewCipher(key)
	default:
		return nil, ErrInvalidKey
	}
}

func (t KeyType) Pad(data []byte) []byte {
	padding := t.BlockSize() - len(data)%t.BlockSize()
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func (t KeyType) Unpad(data []byte) []byte {
	unpadding := int(data[len(data)-1])
	return data[:len(data)-unpadding]
}

type ManagedKey struct {
	KeyType
	IV           []byte
	Plaintext    []byte
	Ciphertext   []byte
	ContextKey   string
	ContextValue string
}

func (k *ManagedKey) Clone() ManagedKey {
	dup := func(v []byte) []byte {
		if v == nil {
			return nil
		}
		w := make([]byte, len(v))
		copy(w, v)
		return w
	}
	return ManagedKey{
		KeyType:      k.KeyType,
		IV:           dup(k.IV),
		Plaintext:    dup(k.Plaintext),
		Ciphertext:   dup(k.Ciphertext),
		ContextKey:   k.ContextKey,
		ContextValue: k.ContextValue,
	}
}

func (k *ManagedKey) Encrypted() bool { return k.Ciphertext != nil }

func (k *ManagedKey) Encrypt(keyKey *ManagedKey) error {
	if keyKey.Encrypted() || k.Encrypted() {
		return ErrKeyMustBeDecrypted
	}

	if k.IV == nil {
		return ErrIVRequired
	}

	if err := keyKey.BlockCrypt(k.IV, keyKey.Plaintext, k.Plaintext, true); err != nil {
		return err
	}

	k.Ciphertext = k.Plaintext
	k.Plaintext = nil
	return nil
}

func (k *ManagedKey) Decrypt(keyKey *ManagedKey) error {
	if keyKey.Encrypted() {
		return ErrKeyMustBeDecrypted
	}

	if !k.Encrypted() {
		return ErrKeyMustBeEncrypted
	}

	if k.IV == nil {
		return ErrIVRequired
	}

	if err := keyKey.BlockCrypt(k.IV, keyKey.Plaintext, k.Ciphertext, false); err != nil {
		return err
	}

	k.Plaintext = k.Ciphertext
	k.Ciphertext = nil
	return nil
}

func KeyFromPasscode(passcode, salt []byte, keyType KeyType) *ManagedKey {
	iterations := keyDerivationIterations
	if TestMode {
		iterations = 1
	}
	return &ManagedKey{
		KeyType:   keyType,
		Plaintext: pbkdf2.Key(passcode, salt, iterations, keyType.KeySize(), sha256.New),
	}
}
