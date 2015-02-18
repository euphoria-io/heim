package console

import (
	"bytes"
	"testing"

	"heim/backend/mock"
	"heim/proto/security"

	"golang.org/x/net/context"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLockRoom(t *testing.T) {
	ctx := context.Background()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	Convey("Usage and flags", t, func() {
		term := &bytes.Buffer{}
		runCommand(&Controller{}, "lock-room", term, []string{"-h"})
		So(term.String(), ShouldStartWith, "Usage of lock-room:")
	})

	Convey("Room not given", t, func() {
		term := &bytes.Buffer{}
		runCommand(&Controller{}, "lock-room", term, nil)
		So(term.String(), ShouldStartWith, "error: room name must be given\r\n")
	})

	// mock doesn't do not found
	SkipConvey("Room not found", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}
		term := &bytes.Buffer{}
		runCommand(ctrl, "lock-room", term, []string{"!!!!"})
		So(term.String(), ShouldStartWith, "error: room name must be given\r\n")
	})

	Convey("Locking room that is already locked", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}

		room, err := ctrl.backend.GetRoom("test")
		So(err, ShouldBeNil)
		orig, err := room.GenerateMasterKey(ctx, kms)
		So(err, ShouldBeNil)

		Convey("Requires --force", func() {
			term := &bytes.Buffer{}
			runCommand(ctrl, "lock-room", term, []string{"test"})
			So(term.String(), ShouldEqual,
				"error: room already locked; use --force to relock and invalidate all previous grants\r\n")
		})

		Convey("Proceeds with --force", func() {
			_ = orig
			term := &bytes.Buffer{}
			runCommand(ctrl, "lock-room", term, []string{"--force", "test"})
			rk, err := room.MasterKey(ctx)
			So(err, ShouldBeNil)
			So(rk, ShouldNotResemble, orig)
		})
	})
}
