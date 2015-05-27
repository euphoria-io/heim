package proto

import (
	"testing"

	"euphoria.io/heim/proto/security"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewAccountSecurity(t *testing.T) {
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	Convey("Check encryption of generated keys", t, func() {
		sec, err := NewAccountSecurity(kms, "hunter2")
		So(err, ShouldBeNil)
		So(sec.SystemKek.Encrypted(), ShouldBeTrue)
		So(sec.UserKek.Encrypted(), ShouldBeTrue)
		So(sec.KeyPair.Encrypted(), ShouldBeTrue)
		So(len(sec.Nonce), ShouldEqual, sec.KeyPair.NonceSize())

		kek := sec.SystemKek.Clone()
		So(kms.DecryptKey(&kek), ShouldBeNil)

		skp := sec.KeyPair.Clone()
		So(skp.Decrypt(&kek), ShouldBeNil)

		kp, err := sec.Unlock(security.KeyFromPasscode([]byte(""), sec.Nonce, kek.KeyType))
		So(err, ShouldEqual, ErrAccessDenied)
		So(kp, ShouldBeNil)

		kp, err = sec.Unlock(security.KeyFromPasscode([]byte("hunter2"), sec.Nonce, kek.KeyType))
		So(err, ShouldBeNil)
		So(kp.PrivateKey, ShouldResemble, skp.PrivateKey)
	})
}
