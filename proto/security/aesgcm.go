package security

import (
	"crypto/cipher"
	"fmt"
)

func newGCM(key *ManagedKey) (cipher.AEAD, error) {
	if key.Encrypted() {
		return nil, fmt.Errorf("key must be decrypted")
	}

	block, err := key.BlockCipher(key.Plaintext)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func fixNonce(gcm cipher.AEAD, nonce []byte) []byte {
	realNonce := make([]byte, gcm.NonceSize())
	copy(realNonce, nonce)
	return realNonce
}

func EncryptGCM(
	key *ManagedKey, nonce, plaintext, data []byte) (digest []byte, ciphertext []byte, err error) {

	gcm, err := newGCM(key)
	if err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, fixNonce(gcm, nonce), plaintext, data)

	// tag is the last gcm.Overhead() bytes of returned ciphertext
	split := len(ciphertext) - gcm.Overhead()
	return ciphertext[split:], ciphertext[:split], nil
}

func DecryptGCM(key *ManagedKey, nonce, digest, ciphertext, data []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, fixNonce(gcm, nonce), append(ciphertext, digest...), data)
}
