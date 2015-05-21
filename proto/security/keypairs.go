package security

import (
	"errors"
	"fmt"
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
