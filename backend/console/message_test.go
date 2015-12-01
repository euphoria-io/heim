package console

import (
	"testing"
	"time"

	"euphoria.io/heim/backend/mock"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDeleteMessage(t *testing.T) {
	ctx := scope.New()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
	session := mock.TestSession("test", "T1")

	sendMessage := func(room proto.Room) (proto.Message, error) {
		msg := proto.Message{
			Sender: &proto.SessionView{
				SessionID:    "test",
				IdentityView: &proto.IdentityView{ID: "test"},
			},
			Content: "test",
		}

		key, err := room.MessageKey(ctx)
		if err != nil {
			return proto.Message{}, err
		}

		if key != nil {
			mkey := key.ManagedKey()
			if err := kms.DecryptKey(&mkey); err != nil {
				return proto.Message{}, err
			}
			if err := proto.EncryptMessage(&msg, key.KeyID(), &mkey); err != nil {
				return proto.Message{}, err
			}
		}

		return room.Send(ctx, session, msg)
	}

	Convey("Delete message in public room", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}
		term := &testTerm{}

		public, err := ctrl.backend.CreateRoom(ctx, kms, false, "public")
		So(err, ShouldBeNil)
		sent, err := sendMessage(public)
		So(err, ShouldBeNil)

		runCommand(ctx, ctrl, "delete-message", term, []string{"public:" + sent.ID.String()})

		deleted, err := public.GetMessage(ctx, sent.ID)
		So(err, ShouldBeNil)
		So(time.Time(deleted.Deleted).IsZero(), ShouldBeFalse)
	})

	Convey("Delete message in private room", t, func() {
		ctrl := &Controller{
			backend: &mock.TestBackend{},
			kms:     kms,
		}
		term := &testTerm{}

		private, err := ctrl.backend.CreateRoom(ctx, kms, true, "private")
		So(err, ShouldBeNil)
		runCommand(ctx, ctrl, "lock-room", term, []string{"private"})

		sent, err := sendMessage(private)
		So(err, ShouldBeNil)

		runCommand(ctx, ctrl, "delete-message", term, []string{"private:" + sent.ID.String()})

		deleted, err := private.GetMessage(ctx, sent.ID)
		So(err, ShouldBeNil)
		So(time.Time(deleted.Deleted).IsZero(), ShouldBeFalse)
	})
}
