package proto

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNormalizeNick(t *testing.T) {
	pass := func(name string) string {
		name, err := NormalizeNick(name)
		So(err, ShouldBeNil)
		return name
	}

	reject := func(name string) error {
		name, err := NormalizeNick(name)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "invalid nick")
		So(name, ShouldEqual, "")
		return err
	}

	Convey("Spaces are stripped", t, func() {
		So(pass("test"), ShouldEqual, "test")
		So(pass(" test"), ShouldEqual, "test")
		So(pass("\r  test"), ShouldEqual, "test")
		So(pass("test "), ShouldEqual, "test")
		So(pass("test\v  "), ShouldEqual, "test")
		So(pass("  test  "), ShouldEqual, "test")
	})

	Convey("Non-spaces are required", t, func() {
		reject("")
		reject(" ")
		reject(" \t\n\v\f ")
	})

	Convey("Internal spaces are collapsed", t, func() {
		So(pass("test test"), ShouldEqual, "test test")
		So(pass("test \r \n \v test"), ShouldEqual, "test test")
		So(pass(" test\ntest test "), ShouldEqual, "test test test")
	})

	Convey("UTF-8 is handled", t, func() {
		input := `
        ᕦ( ͡° ͜ʖ ͡°)ᕤ    


                   ─=≡Σᕕ( ͡° ͜ʖ ͡°)ᕗ
            ` + "\t\r\n:)"
		expected := "ᕦ( ͡° ͜ʖ ͡°)ᕤ ─=≡Σᕕ( ͡° ͜ʖ ͡°)ᕗ :)"
		So(pass(input), ShouldEqual, expected)
	})

	Convey("Max length is enforced", t, func() {
		name := make([]byte, MaxNickLength)
		for i := 0; i < MaxNickLength; i++ {
			name[i] = 'a'
		}
		name[1] = ' '
		name[4] = ' '
		name[5] = ' '
		expected := "a aa " + string(name[6:])
		So(pass(string(name)), ShouldEqual, expected)
		So(pass(string(name)+"a"), ShouldEqual, expected+"a")
		reject(string(name) + "aa")
	})
}
