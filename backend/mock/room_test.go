package mock

import (
	"testing"

	"euphoria.io/scope"

	"heim/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRoomPresence(t *testing.T) {
	userA := newSession("A")
	userA2 := newSession("A")
	userB := newSession("B")

	ctx := scope.New()
	room := newMemRoom("test", "testver")

	Convey("First join", t, func() {
		So(room.Join(ctx, userA), ShouldBeNil)
		So(room.identities, ShouldResemble,
			map[string]proto.Identity{"A": userA.Identity()})
		So(room.live, ShouldResemble,
			map[string][]proto.Session{"A": []proto.Session{userA}})
	})

	Convey("Second join", t, func() {
		So(room.Join(ctx, userB), ShouldBeNil)
		So(room.identities["B"], ShouldResemble, userB.Identity())
		So(room.live["B"], ShouldResemble, []proto.Session{userB})
	})

	Convey("Duplicate join", t, func() {
		So(room.Join(ctx, userA2), ShouldBeNil)
		So(room.live["A"], ShouldResemble, []proto.Session{userA, userA2})
	})

	Convey("Deduplicate part", t, func() {
		So(room.Part(ctx, userA), ShouldBeNil)
		So(room.identities["A"], ShouldResemble, userA.Identity())
		So(room.live["A"], ShouldResemble, []proto.Session{userA2})
	})

	Convey("More parts", t, func() {
		So(room.Part(ctx, userA2), ShouldBeNil)
		So(room.identities["A"], ShouldBeNil)
		So(room.live["A"], ShouldBeNil)
		So(room.Part(ctx, userB), ShouldBeNil)
		So(room.identities["B"], ShouldBeNil)
		So(room.live["B"], ShouldBeNil)
	})
}

func TestRoomBroadcast(t *testing.T) {
	userA := newSession("A")
	userB := newSession("B")
	userC := newSession("C")

	ctx := scope.New()
	room := newMemRoom("test", "testver")

	Convey("Setup", t, func() {
		So(room.Join(ctx, userA), ShouldBeNil)
		So(room.Join(ctx, userB), ShouldBeNil)
		So(room.Join(ctx, userC), ShouldBeNil)
	})

	Convey("Multiple exclude", t, func() {
		So(room.broadcast(ctx, proto.SendType, proto.Message{Content: "1"}, userA, userB),
			ShouldBeNil)
		So(userA.history, ShouldResemble,
			[]message{
				{
					cmdType: proto.JoinEventType,
					payload: proto.PresenceEvent{ID: "B"},
				},
				{
					cmdType: proto.JoinEventType,
					payload: proto.PresenceEvent{ID: "C"},
				},
			})
		So(userB.history, ShouldResemble,
			[]message{
				{
					cmdType: proto.JoinEventType,
					payload: proto.PresenceEvent{ID: "C"},
				},
			})
		So(userC.history, ShouldResemble,
			[]message{{cmdType: proto.SendEventType, payload: proto.Message{Content: "1"}}})
	})

	Convey("No exclude", t, func() {
		So(room.broadcast(ctx, proto.SendType, proto.Message{Content: "2"}), ShouldBeNil)
		So(userA.history, ShouldResemble,
			[]message{
				{
					cmdType: proto.JoinEventType,
					payload: proto.PresenceEvent{ID: "B"},
				},
				{
					cmdType: proto.JoinEventType,
					payload: proto.PresenceEvent{ID: "C"},
				},
				{
					cmdType: proto.SendEventType,
					payload: proto.Message{Content: "2"},
				},
			})
		So(userB.history, ShouldResemble,
			[]message{
				{
					cmdType: proto.JoinEventType,
					payload: proto.PresenceEvent{ID: "C"},
				},
				{cmdType: proto.SendEventType, payload: proto.Message{Content: "2"}},
			})
		So(userC.history, ShouldResemble,
			[]message{
				{cmdType: proto.SendEventType, payload: proto.Message{Content: "1"}},
				{cmdType: proto.SendEventType, payload: proto.Message{Content: "2"}},
			})
	})
}
