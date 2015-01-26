package security

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrInvalidKey = errors.New("invalid key")
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

type ManagedKey struct {
	KeyType
	IV         []byte
	Plaintext  []byte
	Ciphertext []byte
}

func (k *ManagedKey) Encrypted() bool { return k.Ciphertext != nil }
