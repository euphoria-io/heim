package security

import (
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/nacl/box"

	. "github.com/smartystreets/goconvey/convey"
)

func TestKeyPairType(t *testing.T) {
	invalidKeyPairType := KeyPairType(255)

	Convey("PrivateKeySize and PublicKeySize", t, func() {
		So(Curve25519.PrivateKeySize(), ShouldEqual, 32)
		So(Curve25519.PublicKeySize(), ShouldEqual, 32)

		So(func() { invalidKeyPairType.PrivateKeySize() }, ShouldPanicWith,
			"no private key size defined for key type 255")
		So(func() { invalidKeyPairType.PublicKeySize() }, ShouldPanicWith,
			"no public key size defined for key type 255")
	})

	Convey("NonceSize", t, func() {
		So(Curve25519.NonceSize(), ShouldEqual, 24)
		So(invalidKeyPairType.NonceSize(), ShouldEqual, 0)
	})

	Convey("String", t, func() {
		So(Curve25519.String(), ShouldEqual, "curve25519")
		So(invalidKeyPairType.String(), ShouldEqual, "255")
	})

	Convey("checkNonceAndKeys", t, func() {
		buf := make([]byte, 32)
		So(Curve25519.checkNonceAndKeys(buf[:24], buf[:32], buf[:32]), ShouldBeNil)
		So(Curve25519.checkNonceAndKeys(buf[:23], buf[:32], buf[:32]), ShouldEqual, ErrInvalidNonce)
		So(Curve25519.checkNonceAndKeys(buf[:24], buf[:31], buf[:32]), ShouldEqual, ErrInvalidPublicKey)
		So(Curve25519.checkNonceAndKeys(buf[:24], buf[:32], buf[:31]), ShouldEqual, ErrInvalidPrivateKey)
	})

	Convey("Seal and Open", t, func() {
		Convey("Curve25519", func() {
			Convey("Round trip", func() {
				publicKey1, privateKey1, err := box.GenerateKey(rand.Reader)
				So(err, ShouldBeNil)
				publicKey2, privateKey2, err := box.GenerateKey(rand.Reader)
				So(err, ShouldBeNil)
				nonce := make([]byte, Curve25519.NonceSize())
				message := []byte("content")

				encrypted, err := Curve25519.Seal(message, nonce, publicKey2[:], privateKey1[:])
				So(err, ShouldBeNil)

				decrypted, err := Curve25519.Open(encrypted, nonce, publicKey1[:], privateKey2[:])
				So(err, ShouldBeNil)
				So(string(decrypted), ShouldEqual, string(message))

				nonce[0] = 1
				_, err = Curve25519.Open(encrypted, nonce, publicKey1[:], privateKey2[:])
				So(err, ShouldEqual, ErrMessageIntegrityFailed)
			})

			Convey("Input errors", func() {
				buf := make([]byte, 32)
				_, err := Curve25519.Seal(buf, buf[:23], buf[:32], buf[:32])
				So(err, ShouldEqual, ErrInvalidNonce)
				_, err = Curve25519.Open(buf, buf[:23], buf[:32], buf[:32])
				So(err, ShouldEqual, ErrInvalidNonce)
			})
		})
	})
}

func TestManagedKeyPair(t *testing.T) {
	Convey("Clone", t, func() {
		x := ManagedKeyPair{}
		y := x.Clone()
		So(x, ShouldResemble, y)
		So(y.IV, ShouldBeNil)

		x.IV = []byte("iv")
		x.PrivateKey = []byte("private key")
		x.EncryptedPrivateKey = []byte("encrypted private key")
		x.PublicKey = []byte("public key")
		y = x.Clone()
		So(x, ShouldResemble, y)
		So(x.IV, ShouldNotEqual, y.IV)
		So(x.PrivateKey, ShouldNotEqual, y.PrivateKey)
		So(x.EncryptedPrivateKey, ShouldNotEqual, y.EncryptedPrivateKey)
		So(x.PublicKey, ShouldNotEqual, y.PublicKey)
	})

	Convey("Crypto", t, func() {
		k := &ManagedKeyPair{PrivateKey: []byte("private key with len of 32 bytes")}
		So(k.Encrypted(), ShouldBeFalse)

		keyKey := &ManagedKey{
			KeyType:   AES128,
			Plaintext: make([]byte, AES128.KeySize()),
		}
		So(k.Decrypt(keyKey), ShouldEqual, ErrKeyMustBeEncrypted)
		So(k.Encrypt(keyKey), ShouldEqual, ErrIVRequired)

		k.IV = make([]byte, AES128.BlockSize())
		So(k.Encrypt(keyKey), ShouldBeNil)
		So(k.Encrypted(), ShouldBeTrue)
		So(k.Encrypt(keyKey), ShouldEqual, ErrKeyMustBeDecrypted)

		k.IV = nil
		So(k.Decrypt(keyKey), ShouldEqual, ErrIVRequired)

		k.IV = make([]byte, AES128.BlockSize())
		So(k.Decrypt(keyKey), ShouldBeNil)
		So(k.Encrypted(), ShouldBeFalse)
		So(string(k.PrivateKey), ShouldEqual, "private key with len of 32 bytes")
	})
}
