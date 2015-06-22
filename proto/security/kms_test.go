package security

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLocalKMS(t *testing.T) {
	Convey("GenerateNonce", t, func() {
		randomness := "randomness"

		Convey("Should return error from random source", func() {
			kms := LocalKMSWithRNG(bytes.NewReader([]byte(randomness)))
			nonce, err := kms.GenerateNonce(len(randomness) + 1)
			So(err, ShouldEqual, io.ErrUnexpectedEOF)
			So(nonce, ShouldBeNil)
		})

		Convey("Should return next n bytes of random source", func() {
			kms := LocalKMSWithRNG(bytes.NewReader([]byte(randomness)))
			nonce, err := kms.GenerateNonce(len(randomness) - 1)
			So(err, ShouldBeNil)
			So(string(nonce), ShouldEqual, randomness[:len(randomness)-1])
		})
	})

	Convey("Generated random keys", t, func() {
		Convey("Error should be checked when generating iv", func() {
			n := int64(AES128.BlockSize())
			kms := LocalKMSWithRNG(io.LimitReader(rand.Reader, n-1))
			encKey, err := kms.GenerateEncryptedKey(AES128, "room", "test")
			So(err, ShouldEqual, io.ErrUnexpectedEOF)
			So(encKey, ShouldBeNil)
		})

		Convey("Error should be checked when generating key", func() {
			n := int64(AES128.KeySize())
			kms := LocalKMSWithRNG(io.LimitReader(rand.Reader, n-1))
			kms.SetMasterKey(make([]byte, mockCipher.KeySize()))
			encKey, err := kms.GenerateEncryptedKey(AES128, "room", "test")
			So(err, ShouldEqual, io.ErrUnexpectedEOF)
			So(encKey, ShouldBeNil)
		})

		Convey("Should return ErrNoMasterKey if master key isn't configured", func() {
			kms := LocalKMS()
			encKey, err := kms.GenerateEncryptedKey(AES128, "room", "test")
			So(err, ShouldEqual, ErrNoMasterKey)
			So(encKey, ShouldBeNil)
		})

		Convey("Master key is required to be set", func() {
			kms := LocalKMS()
			mkey, err := kms.GenerateEncryptedKey(AES128, "room", "test")
			So(err, ShouldEqual, ErrNoMasterKey)
			So(mkey, ShouldBeNil)

			mkey = &ManagedKey{
				KeyType:    AES128,
				IV:         make([]byte, AES128.BlockSize()),
				Ciphertext: make([]byte, AES128.BlockSize()),
			}
			So(kms.DecryptKey(mkey), ShouldEqual, ErrNoMasterKey)
		})

		Convey("Encrypted key can be decrypted", func() {
			expectedKey := make([]byte, AES128.KeySize())

			// Generate encrypted key.
			kms := LocalKMSWithRNG(bytes.NewReader(expectedKey))
			kms.SetMasterKey(make([]byte, mockCipher.KeySize()))
			mkey, err := kms.GenerateEncryptedKey(AES128, "room", "test")
			So(err, ShouldBeNil)
			So(mkey, ShouldNotBeNil)
			So(mkey.Encrypted(), ShouldBeTrue)
			So(len(mkey.Ciphertext), ShouldEqual, AES128.KeySize()+sha256.Size)
			So(mkey.ContextKey, ShouldEqual, "room")
			So(mkey.ContextValue, ShouldEqual, "test")

			// Verify decryption.
			So(kms.DecryptKey(mkey), ShouldBeNil)
			So(mkey.Encrypted(), ShouldBeFalse)
			So(len(mkey.Plaintext), ShouldEqual, AES128.KeySize())
			So(bytes.Equal(mkey.Plaintext, expectedKey), ShouldBeTrue)
		})
	})

	Convey("Decrypted key cannot be decrypted", t, func() {
		mkey := &ManagedKey{
			KeyType:   AES128,
			IV:        make([]byte, AES128.BlockSize()),
			Plaintext: make([]byte, AES128.KeySize()),
		}
		So(LocalKMS().DecryptKey(mkey), ShouldEqual, ErrKeyMustBeEncrypted)
	})

	Convey("Encrypted key with bad IV cannot be decrypted", t, func() {
		kms := LocalKMS()
		kms.SetMasterKey(make([]byte, mockCipher.KeySize()))
		mkey, err := kms.GenerateEncryptedKey(AES128, "room", "test")
		So(err, ShouldBeNil)
		mkey.ContextValue = "test2"
		So(kms.DecryptKey(mkey), ShouldEqual, ErrInvalidKey)
	})

	Convey("Encrypted key with bad Ciphertext cannot be decrypted", t, func() {
		kms := LocalKMS()
		kms.SetMasterKey(make([]byte, mockCipher.KeySize()))
		mkey, err := kms.GenerateEncryptedKey(AES128, "room", "test")
		So(err, ShouldBeNil)
		mkey.Ciphertext = append(mkey.Ciphertext, mkey.Ciphertext...)
		So(kms.DecryptKey(mkey), ShouldEqual, ErrInvalidKey)
	})
}
