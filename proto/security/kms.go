package security

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
)

var (
	ErrNoMasterKey = errors.New("no master key")
)

type KMS interface {
	GenerateNonce(bytes int) ([]byte, error)

	GenerateEncryptedKey(keyType KeyType, ctxKey, ctxVal string) (*ManagedKey, error)
	DecryptKey(*ManagedKey) error
}

type KMSCredential interface {
	json.Marshaler
	json.Unmarshaler
	KMS() KMS
}

const mockCipher = AES256

type MockKMS interface {
	KMS

	KMSCredential() KMSCredential
	SetMasterKey([]byte)
}

func LocalKMS() MockKMS                        { return LocalKMSWithRNG(rand.Reader) }
func LocalKMSWithRNG(random io.Reader) MockKMS { return &localKMS{random: random} }

type localKMS struct {
	random    io.Reader
	masterKey []byte
}

func (kms *localKMS) KMS() KMS                     { return kms }
func (kms *localKMS) KMSCredential() KMSCredential { return kms }
func (kms *localKMS) SetMasterKey(key []byte)      { kms.masterKey = key }

func (kms *localKMS) GenerateNonce(bytes int) ([]byte, error) {
	nonce := make([]byte, bytes)
	_, err := io.ReadFull(kms.random, nonce)
	if err != nil {
		return nil, err
	}
	return nonce, nil
}

func (kms *localKMS) GenerateEncryptedKey(keyType KeyType, ctxKey, ctxVal string) (*ManagedKey, error) {
	key, err := kms.GenerateNonce(keyType.KeySize())
	if err != nil {
		return nil, err
	}

	mkey := &ManagedKey{
		KeyType:      keyType,
		Plaintext:    key,
		ContextKey:   ctxKey,
		ContextValue: ctxVal,
	}
	if err := kms.xorKey(mkey); err != nil {
		return nil, err
	}

	return mkey, nil
}

func (kms *localKMS) DecryptKey(mkey *ManagedKey) error {
	if !mkey.Encrypted() {
		return ErrKeyMustBeEncrypted
	}
	return kms.xorKey(mkey)
}

func (kms *localKMS) xorKey(mkey *ManagedKey) error {
	if kms.masterKey == nil {
		return ErrNoMasterKey
	}

	// Generate IV from md5 hash of context.
	hash := md5.New()
	hash.Write([]byte(mkey.ContextKey))
	hash.Write([]byte(mkey.ContextValue))
	iv := hash.Sum(nil)

	if mkey.Encrypted() {
		if len(mkey.Ciphertext) != mkey.KeySize()+sha256.Size {
			return ErrInvalidKey
		}
		macsum := mkey.Ciphertext[:sha256.Size]
		data := mkey.Ciphertext[sha256.Size:]
		mockCipher.BlockCrypt(iv, kms.masterKey, data, true)
		mac := hmac.New(sha256.New, data)
		mac.Write([]byte(mkey.ContextKey))
		mac.Write([]byte(mkey.ContextValue))
		if !hmac.Equal(macsum, mac.Sum(nil)) {
			return ErrInvalidKey
		}
		mkey.Ciphertext = nil
		mkey.Plaintext = data
	} else {
		// Generate mac for context.
		mac := hmac.New(sha256.New, mkey.Plaintext)
		mac.Write([]byte(mkey.ContextKey))
		mac.Write([]byte(mkey.ContextValue))
		macsum := mac.Sum(nil)
		data := mkey.Plaintext
		mockCipher.BlockCrypt(iv, kms.masterKey, data, false)
		mkey.Plaintext = nil
		mkey.Ciphertext = append(macsum, data...)
	}

	return nil
}

func (kms *localKMS) MarshalJSON() ([]byte, error)    { return json.Marshal(kms.masterKey) }
func (kms *localKMS) UnmarshalJSON(data []byte) error { return json.Unmarshal(data, &kms.masterKey) }
