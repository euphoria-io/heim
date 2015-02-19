package kms

import (
	"flag"
	"testing"

	"heim/proto/security"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	region = flag.String("region", "us-west-2", "Region of KMS to test against")
	keyID  = flag.String("keyid", "", "Master key to test against (must be defined in given region)")
)

func TestKMS(t *testing.T) {
	if *keyID == "" {
		t.Skip()
	}

	kms, err := New(*region, *keyID)
	if err != nil {
		t.Skipf("unable to instantiate aws kms: %s", err)
	}

	Convey("GenerateNonce", t, func() {
		nonce, err := kms.GenerateNonce(20)
		So(err, ShouldBeNil)
		So(len(nonce), ShouldEqual, 20)
	})

	Convey("GenerateEncryptedKey and Decrypt", t, func() {
		Convey("AES-256", func() {
			key, err := kms.GenerateEncryptedKey(security.AES256)
			So(err, ShouldBeNil)
			So(key.Encrypted(), ShouldBeTrue)

			So(kms.DecryptKey(key), ShouldBeNil)
			So(key.Encrypted(), ShouldBeFalse)
			So(len(key.Plaintext), ShouldEqual, security.AES256.KeySize())
		})

		Convey("AES-128", func() {
			key, err := kms.GenerateEncryptedKey(security.AES128)
			So(err, ShouldBeNil)
			So(key.Encrypted(), ShouldBeTrue)

			So(kms.DecryptKey(key), ShouldBeNil)
			So(key.Encrypted(), ShouldBeFalse)
			So(len(key.Plaintext), ShouldEqual, security.AES128.KeySize())
		})

		Convey("Invalid key type", func() {
			key, err := kms.GenerateEncryptedKey(security.KeyType(255))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "aws kms: key type 255 not supported")
			So(key, ShouldBeNil)
		})

		Convey("Key already decrypted", func() {
			err := kms.DecryptKey(&security.ManagedKey{Plaintext: []byte{0}})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "aws kms: key is already decrypted")
		})
	})
}
