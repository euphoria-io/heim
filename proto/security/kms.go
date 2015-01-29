package security

import (
	"crypto/rand"
	"errors"
	"io"
)

var (
	ErrNoMasterKey = errors.New("no master key")
)

type KMS interface {
	GenerateNonce(bytes int) ([]byte, error)

	GenerateEncryptedKey(KeyType) (*ManagedKey, error)
	DecryptKey(*ManagedKey) error
}

const mockCipher = AES256

type MockKMS interface {
	KMS

	SetMasterKey([]byte)
}

func LocalKMS() MockKMS                        { return LocalKMSWithRNG(rand.Reader) }
func LocalKMSWithRNG(random io.Reader) MockKMS { return &localKMS{random: random} }

type localKMS struct {
	random    io.Reader
	masterKey []byte
}

func (kms *localKMS) SetMasterKey(key []byte) { kms.masterKey = key }

func (kms *localKMS) GenerateNonce(bytes int) ([]byte, error) {
	nonce := make([]byte, bytes)
	_, err := io.ReadFull(kms.random, nonce)
	if err != nil {
		return nil, err
	}
	return nonce, nil
}

func (kms *localKMS) GenerateEncryptedKey(keyType KeyType) (*ManagedKey, error) {
	iv, err := kms.GenerateNonce(mockCipher.BlockSize())
	if err != nil {
		return nil, err
	}
	key, err := kms.GenerateNonce(keyType.KeySize())
	if err != nil {
		return nil, err
	}
	mkey := &ManagedKey{
		KeyType:   keyType,
		IV:        iv,
		Plaintext: key,
	}
	if err := kms.xorKey(mkey); err != nil {
		return nil, err
	}
	return mkey, nil
}

func (kms *localKMS) DecryptKey(mkey *ManagedKey) error {
	if !mkey.Encrypted() {
		return ErrInvalidKey
	}
	return kms.xorKey(mkey)
}

func (kms *localKMS) xorKey(mkey *ManagedKey) error {
	if kms.masterKey == nil {
		return ErrNoMasterKey
	}

	if len(mkey.IV) != mkey.BlockSize() {
		return ErrInvalidKey
	}

	var data []byte
	var encrypted bool
	if mkey.Encrypted() {
		if len(mkey.Ciphertext) != mkey.KeySize() {
			return ErrInvalidKey
		}
		encrypted = true
		data = mkey.Ciphertext
		mkey.Ciphertext = nil
		mkey.Plaintext = data
	} else {
		encrypted = false
		data = mkey.Plaintext
		mkey.Ciphertext = data
		mkey.Plaintext = nil
	}

	mockCipher.BlockCrypt(mkey.IV, kms.masterKey, data, encrypted)
	return nil
}
