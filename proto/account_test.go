package proto

import (
	"fmt"
	"testing"

	"euphoria.io/heim/proto/security"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewAccountSecurity(t *testing.T) {
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	unlock := func(sec *AccountSecurity, password string) (*security.ManagedKeyPair, error) {
		return sec.Unlock(security.KeyFromPasscode([]byte(password), sec.Nonce, sec.UserKey.KeyType))
	}

	Convey("Encryption and decryption of generated keys", t, func() {
		sec, clientKey, err := NewAccountSecurity(kms, "hunter2")
		So(err, ShouldBeNil)
		So(sec.SystemKey.Encrypted(), ShouldBeTrue)
		So(sec.UserKey.Encrypted(), ShouldBeTrue)
		So(sec.KeyPair.Encrypted(), ShouldBeTrue)
		So(len(sec.Nonce), ShouldEqual, sec.KeyPair.NonceSize())
		So(clientKey.Encrypted(), ShouldBeFalse)

		kek := sec.SystemKey.Clone()
		So(kms.DecryptKey(&kek), ShouldBeNil)

		skp := sec.KeyPair.Clone()
		So(skp.Decrypt(&kek), ShouldBeNil)

		kp, err := unlock(sec, "")
		So(err, ShouldEqual, ErrAccessDenied)
		So(kp, ShouldBeNil)

		kp, err = unlock(sec, "hunter2")
		So(err, ShouldBeNil)
		So(kp.PrivateKey, ShouldResemble, skp.PrivateKey)
	})

	Convey("Password resets", t, func() {
		sec, _, err := NewAccountSecurity(kms, "hunter2")
		So(err, ShouldBeNil)

		nsec, err := sec.ResetPassword(kms, "hunter3")
		So(err, ShouldBeNil)

		skp, err := unlock(sec, "hunter2")
		So(err, ShouldBeNil)

		_, err = unlock(nsec, "hunter2")
		So(err, ShouldEqual, ErrAccessDenied)

		kp, err := unlock(nsec, "hunter3")
		So(err, ShouldBeNil)
		So(kp.PrivateKey, ShouldResemble, skp.PrivateKey)
	})
}

func TestValidatePersonalIdentity(t *testing.T) {
	Convey("Any email id is accepted", t, func() {
		ok, reason := ValidatePersonalIdentity("email", "test")
		So(ok, ShouldBeTrue)
		So(reason, ShouldEqual, "")
	})

	Convey("No other namespace is accepted", t, func() {
		ok, reason := ValidatePersonalIdentity("notemail", "test")
		So(ok, ShouldBeFalse)
		So(reason, ShouldEqual, "invalid namespace: notemail")
	})
}

func TestValidateAccountPassword(t *testing.T) {
	Convey("Password must be of sufficient length", t, func() {
		minPassword := make([]byte, MinPasswordLength)
		for i := 0; i < MinPasswordLength; i++ {
			minPassword[i] = '*'
		}

		ok, reason := ValidateAccountPassword(string(minPassword))
		So(ok, ShouldBeTrue)
		So(reason, ShouldEqual, "")

		ok, reason = ValidateAccountPassword(string(minPassword[:len(minPassword)-1]))
		So(ok, ShouldBeFalse)
		So(reason, ShouldEqual,
			fmt.Sprintf("password must be at least %d characters long", MinPasswordLength))
	})
}
