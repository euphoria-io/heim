package security

import (
	"encoding/base64"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGCM(t *testing.T) {
	kms := LocalKMS()
	bits, err := kms.GenerateNonce(AES128.BlockSize())
	if err != nil {
		t.Fatal(err)
	}
	key := &ManagedKey{KeyType: AES128, Plaintext: bits}
	nonce := []byte("1111111111111")

	message := "This is a test of AES 128-bit Galois Counter Mode encryption."

	Convey("Encrypt and decrypt with AES-GCM", t, func() {
		digest, ciphertext, err := EncryptGCM(key, nonce, []byte(message), nil)
		So(err, ShouldBeNil)
		Printf("encrypted: %s/%s",
			base64.URLEncoding.EncodeToString(digest),
			base64.URLEncoding.EncodeToString(ciphertext))

		plaintext, err := DecryptGCM(key, nonce, digest, ciphertext, nil)
		So(err, ShouldBeNil)
		So(string(plaintext), ShouldEqual, message)
	})
}
