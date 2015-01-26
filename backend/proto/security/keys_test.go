package security

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAES128(t *testing.T) {
	Convey("Key and block size", t, func() {
		So(AES128.BlockSize(), ShouldEqual, 16)
		So(AES128.KeySize(), ShouldEqual, 16)
	})

	Convey("String", t, func() {
		So(AES128.String(), ShouldEqual, "aes-128")
	})

	Convey("Block crypt round trip", t, func() {
		key := make([]byte, AES128.KeySize())
		iv := make([]byte, AES128.BlockSize())
		data := make([]byte, AES128.BlockSize()*3)
		saved := make([]byte, len(data))
		copy(saved, data)

		So(AES128.BlockCrypt(iv, key, data, true), ShouldBeNil)

		decrypted := make([]byte, len(data))
		copy(decrypted, data)
		So(AES128.BlockCrypt(iv, key, decrypted, false), ShouldBeNil)

		So(bytes.Equal(decrypted, saved), ShouldBeTrue)
	})

	Convey("BlockCrypt requires full blocks of data", t, func() {
		iv := make([]byte, AES128.BlockSize())
		key := make([]byte, AES128.KeySize())
		data := make([]byte, AES128.BlockSize()*2+1)

		So(AES128.BlockCrypt(iv, key, data, true), ShouldEqual, ErrInvalidKey)
		So(AES128.BlockCrypt(iv, key, data, false), ShouldEqual, ErrInvalidKey)
	})

	Convey("BlockMode requires iv of correct size", t, func() {
		iv := make([]byte, AES128.BlockSize()+1)
		key := make([]byte, AES128.KeySize())
		data := make([]byte, AES128.BlockSize()*2)

		So(AES128.BlockCrypt(iv, key, data, true), ShouldEqual, ErrInvalidKey)
		So(AES128.BlockCrypt(iv, key, data, false), ShouldEqual, ErrInvalidKey)
	})

	Convey("BlockCipher requires key of correct size", t, func() {
		iv := make([]byte, AES128.BlockSize())
		key := make([]byte, AES128.KeySize()+1)
		data := make([]byte, AES128.BlockSize()*2)

		So(AES128.BlockCrypt(iv, key, data, true), ShouldEqual, ErrInvalidKey)
		So(AES128.BlockCrypt(iv, key, data, false), ShouldEqual, ErrInvalidKey)
	})
}

func TestAES256(t *testing.T) {
	Convey("Key and block size", t, func() {
		So(AES256.BlockSize(), ShouldEqual, 16)
		So(AES256.KeySize(), ShouldEqual, 32)
	})

	Convey("String", t, func() {
		So(AES256.String(), ShouldEqual, "aes-256")
	})
}

func TestInvalidKeyType(t *testing.T) {
	invalid := KeyType(255)

	Convey("Expect panics", t, func() {
		So(func() { invalid.BlockSize() }, ShouldPanicWith,
			"no block size defined for key type 255")
		So(func() { invalid.KeySize() }, ShouldPanicWith,
			"no key size defined for key type 255")
	})

	Convey("String", t, func() {
		So(invalid.String(), ShouldEqual, "255")
	})
}
