package backend

import (
	"sync"
	"testing"

	"golang.org/x/net/context"

	. "github.com/smartystreets/goconvey/convey"
)

type session struct {
	sync.Mutex
	id      string
	history []Message
}

func newSession(id string) *session { return &session{id: id} }

func (s *session) Identity() Identity { return rawIdentity(s.id) }

func (s *session) Send(ctx context.Context, msg Message) error {
	s.Lock()
	s.history = append(s.history, msg)
	s.Unlock()
	return nil
}

func (s *session) clear() {
	s.Lock()
	s.history = nil
	s.Unlock()
}

func TestRoomPresence(t *testing.T) {
	userA := newSession("A")
	userA2 := newSession("A")
	userB := newSession("B")

	ctx := context.Background()
	room := newMemRoom("test")

	Convey("First join", t, func() {
		So(room.Join(ctx, userA), ShouldBeNil)
		So(room.identities, ShouldResemble,
			map[string]Identity{"A": userA.Identity()})
		So(room.live, ShouldResemble,
			map[string][]Session{"A": []Session{userA}})
	})

	Convey("Second join", t, func() {
		So(room.Join(ctx, userB), ShouldBeNil)
		So(room.identities["B"], ShouldEqual, userB.Identity())
		So(room.live["B"], ShouldResemble, []Session{userB})
	})

	Convey("Duplicate join", t, func() {
		So(room.Join(ctx, userA2), ShouldBeNil)
		So(room.live["A"], ShouldResemble, []Session{userA, userA2})
	})

	Convey("Deduplicate part", t, func() {
		So(room.Part(ctx, userA), ShouldBeNil)
		So(room.identities["A"], ShouldEqual, userA.Identity())
		So(room.live["A"], ShouldResemble, []Session{userA2})
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

	ctx := context.Background()
	room := newMemRoom("test")

	Convey("Setup", t, func() {
		So(room.Join(ctx, userA), ShouldBeNil)
		So(room.Join(ctx, userB), ShouldBeNil)
		So(room.Join(ctx, userC), ShouldBeNil)
	})

	Convey("Multiple exclude", t, func() {
		So(room.broadcast(ctx, &Message{Content: "1"}, userA, userB), ShouldBeNil)
		So(userA.history, ShouldBeNil)
		So(userB.history, ShouldBeNil)
		So(userC.history, ShouldResemble, []Message{{Content: "1"}})
	})

	Convey("No exclude", t, func() {
		So(room.broadcast(ctx, &Message{Content: "2"}), ShouldBeNil)
		So(userA.history, ShouldResemble, []Message{{Content: "2"}})
		So(userB.history, ShouldResemble, []Message{{Content: "2"}})
		So(userC.history, ShouldResemble, []Message{{Content: "1"}, {Content: "2"}})
	})
}
