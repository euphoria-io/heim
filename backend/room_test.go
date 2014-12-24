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
	name    string
	history []message
}

type message struct {
	cmdType PacketType
	payload interface{}
}

func newSession(id string) *session { return &session{id: id} }

func (s *session) ID() string          { return s.id }
func (s *session) Close()              {}
func (s *session) SetName(name string) { s.name = name }

func (s *session) Identity() Identity {
	id := newMemIdentity(s.id)
	if s.name != "" {
		id.name = s.name
	}
	return id
}

func (s *session) Send(ctx context.Context, cmdType PacketType, payload interface{}) error {
	s.Lock()
	s.history = append(s.history, message{cmdType, payload})
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
		So(room.identities["B"], ShouldResemble, userB.Identity())
		So(room.live["B"], ShouldResemble, []Session{userB})
	})

	Convey("Duplicate join", t, func() {
		So(room.Join(ctx, userA2), ShouldBeNil)
		So(room.live["A"], ShouldResemble, []Session{userA, userA2})
	})

	Convey("Deduplicate part", t, func() {
		So(room.Part(ctx, userA), ShouldBeNil)
		So(room.identities["A"], ShouldResemble, userA.Identity())
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
		So(room.broadcast(ctx, SendType, Message{Content: "1"}, userA, userB), ShouldBeNil)
		So(userA.history, ShouldResemble,
			[]message{
				{cmdType: JoinEventType, payload: PresenceEvent{ID: "B", Name: "B"}},
				{cmdType: JoinEventType, payload: PresenceEvent{ID: "C", Name: "C"}},
			})
		So(userB.history, ShouldResemble,
			[]message{{cmdType: JoinEventType, payload: PresenceEvent{ID: "C", Name: "C"}}})
		So(userC.history, ShouldResemble,
			[]message{{cmdType: SendEventType, payload: Message{Content: "1"}}})
	})

	Convey("No exclude", t, func() {
		So(room.broadcast(ctx, SendType, Message{Content: "2"}), ShouldBeNil)
		So(userA.history, ShouldResemble,
			[]message{
				{cmdType: JoinEventType, payload: PresenceEvent{ID: "B", Name: "B"}},
				{cmdType: JoinEventType, payload: PresenceEvent{ID: "C", Name: "C"}},
				{cmdType: SendEventType, payload: Message{Content: "2"}},
			})
		So(userB.history, ShouldResemble,
			[]message{
				{cmdType: JoinEventType, payload: PresenceEvent{ID: "C", Name: "C"}},
				{cmdType: SendEventType, payload: Message{Content: "2"}},
			})
		So(userC.history, ShouldResemble,
			[]message{
				{cmdType: SendEventType, payload: Message{Content: "1"}},
				{cmdType: SendEventType, payload: Message{Content: "2"}},
			})
	})
}
