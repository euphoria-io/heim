package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	. "github.com/smartystreets/goconvey/convey"
)

// TODO: move time mocking to snowflake_test?
type testClock struct {
	secs                  int64
	savedClock            func() time.Time
	savedSnowflaker       Snowflaker
	savedEpoch            time.Time
	savedFromTimeSequence uint64
}

func NewTestClock() io.Closer {
	tc := &testClock{
		savedClock:            Clock,
		savedSnowflaker:       DefaultSnowflaker,
		savedEpoch:            Epoch,
		savedFromTimeSequence: fromTimeSequence,
	}
	Clock = tc.clock
	DefaultSnowflaker = tc
	Epoch = time.Unix(0, 0)
	return tc
}

func (tc *testClock) Close() error {
	Clock = tc.savedClock
	DefaultSnowflaker = tc.savedSnowflaker
	Epoch = tc.savedEpoch
	fromTimeSequence = tc.savedFromTimeSequence
	return nil
}

func (tc *testClock) Next() (uint64, error) {
	sf := NewSnowflakeFromTime(tc.clock())
	return uint64(sf), nil
}

func (tc *testClock) clock() time.Time {
	secs := atomic.AddInt64(&tc.secs, 1)
	return time.Unix(secs, 0)
}

type testSuite func(testing.TB, *serverUnderTest)

type serverUnderTest struct {
	backend Backend
	app     *Server
	server  *httptest.Server
}

func (s *serverUnderTest) Close() {
	s.server.CloseClientConnections()
	s.server.Close()
}

func (s *serverUnderTest) Connect(roomName string) *websocket.Conn {
	url := strings.Replace(s.server.URL, "http:", "ws:", 1) + "/room/" + roomName + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	So(err, ShouldBeNil)
	return conn
}

func closeConn(conn *websocket.Conn) {
	conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "normal closure"))
}

func readPacket(conn *websocket.Conn) (PacketType, interface{}) {
	msgType, data, err := conn.ReadMessage()
	So(err, ShouldBeNil)
	So(msgType, ShouldEqual, websocket.TextMessage)

	var packet Packet
	So(json.Unmarshal(data, &packet), ShouldBeNil)
	payload, err := packet.Payload()
	So(err, ShouldBeNil)

	return packet.Type, payload
}

func shouldReceive(actual interface{}, expected ...interface{}) string {
	conn, ok := actual.(*websocket.Conn)
	if !ok {
		return fmt.Sprintf("shouldReceive expects a *websocket.Conn on the left, got %T", actual)
	}
	if len(expected) != 2 {
		return "shouldReceive expects string, payload on right"
	}
	expectedType, ok := expected[0].(PacketType)
	if !ok {
		return fmt.Sprintf(
			"shouldReceive expects string, payload on right, got %T, %T", expected...)
	}
	expectedPayload := expected[1]

	fmt.Printf("%s should receive %v, %#v\n", conn.RemoteAddr(), expectedType, expectedPayload)

	packetType, payload := readPacket(conn)
	fmt.Printf("%s received %v, %#v\n", conn.RemoteAddr(), packetType, payload)

	if packetType != expectedType {
		return fmt.Sprintf("Expected: %s -> %#v\nActual:   %s -> %#v\n",
			expectedType, expectedPayload, packetType, payload)
	}

	return ShouldResemble(payload, expectedPayload)
}

func shouldSend(actual interface{}, expected ...interface{}) string {
	conn, ok := actual.(*websocket.Conn)
	if !ok {
		return fmt.Sprintf("shouldSend expects a *websocket.Conn on the left, got %T", actual)
	}
	if len(expected) == 0 {
		return "shouldSend expects format string and parameters on right"
	}
	format, ok := expected[0].(string)
	if !ok {
		return fmt.Sprintf("shouldSend expects format string on right, got %T", expected)
	}
	msg := fmt.Sprintf(format, expected[1:]...)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		return fmt.Sprintf("send error: %s\n   message: %s", err, msg)
	}
	return ""
}

func snowflakes(n int) []Snowflake {
	fc := NewTestClock()
	defer fc.Close()

	snowflakes := make([]Snowflake, n)
	for i := range snowflakes {
		var err error
		snowflakes[i], err = NewSnowflake()
		So(err, ShouldBeNil)
	}
	return snowflakes
}

func IntegrationTest(t testing.TB, factory func() Backend) {
	runTest := func(test testSuite) {
		backend := factory()
		app := NewServer(backend, "")
		server := httptest.NewServer(app)
		defer server.Close()
		test(t, &serverUnderTest{backend, app, server})
	}

	runTest(testLurker)
	runTest(testBroadcast)
	runTest(testThreading)
}

func testLurker(t testing.TB, s *serverUnderTest) {
	Convey("Lurker", t, func() {
		conn1 := s.Connect("lurker")
		defer closeConn(conn1)
		id1 := conn1.LocalAddr().String()

		So(conn1, shouldReceive, SnapshotEventType,
			&SnapshotEvent{
				Version: s.backend.Version(),
				Listing: Listing{},
				Log:     []Message{},
			})

		conn2 := s.Connect("lurker")
		defer closeConn(conn2)
		id2 := conn2.LocalAddr().String()

		So(conn2, shouldReceive, SnapshotEventType,
			&SnapshotEvent{
				Version: s.backend.Version(),
				Listing: Listing{IdentityView{ID: id1, Name: id1}},
				Log:     []Message{},
			})

		So(conn2, shouldSend, `{"id":"1","type":"nick","data":{"name":"speaker"}}`)
		So(conn2, shouldReceive, NickReplyType, &NickReply{ID: id2, From: id2, To: "speaker"})

		So(conn1, shouldReceive, JoinEventType, &PresenceEvent{ID: id2, Name: id2})
		So(conn1, shouldReceive, NickEventType, &NickEvent{ID: id2, From: id2, To: "speaker"})
	})
}

func testBroadcast(t testing.TB, s *serverUnderTest) {
	Convey("Broadcast", t, func() {
		tc := NewTestClock()
		defer tc.Close()

		conns := make([]*websocket.Conn, 3)

		ids := make(Listing, len(conns))

		for i := range conns {
			conn := s.Connect("broadcast")
			conns[i] = conn
			me := conn.LocalAddr().String()
			ids[i] = IdentityView{ID: me, Name: fmt.Sprintf("user%d", i)}
			So(conn, shouldSend, `{"id":"1","type":"nick","data":{"name":"user%d"}}`, i)
			So(conn, shouldSend, `{"id":"2","type":"who"}`)

			So(conn, shouldReceive, SnapshotEventType,
				&SnapshotEvent{
					Version: s.backend.Version(),
					Listing: ids[:i],
					Log:     []Message{},
				})

			So(conn, shouldReceive, NickReplyType,
				&NickReply{ID: ids[i].ID, From: ids[i].ID, To: fmt.Sprintf("user%d", i)})
			So(conn, shouldReceive, WhoReplyType, &WhoReply{Listing: ids[:(i + 1)]})

			for _, c := range conns[:i] {
				So(c, shouldReceive, JoinEventType, &PresenceEvent{ID: ids[i].ID, Name: ids[i].ID})
				So(c, shouldReceive, NickEventType,
					&NickEvent{ID: ids[i].ID, From: ids[i].ID, To: fmt.Sprintf("user%d", i)})
			}
		}

		defer func() {
			for _, conn := range conns {
				defer closeConn(conn)
			}
		}()

		sfs := snowflakes(2)
		sf1 := sfs[0]
		sf2 := sfs[1]

		So(conns[1], shouldSend, `{"id":"2","type":"send","data":{"content":"hi"}}`)

		So(conns[0], shouldReceive, SendEventType,
			&SendEvent{ID: sf1, UnixTime: 1, Sender: &ids[1], Content: "hi"})

		So(conns[2], shouldSend, `{"id":"2","type":"send","data":{"content":"bye"}}`)

		So(conns[0], shouldReceive, SendEventType,
			&SendEvent{ID: sf2, UnixTime: 2, Sender: &ids[2], Content: "bye"})

		So(conns[1], shouldReceive, SendReplyType,
			&SendReply{ID: sf1, UnixTime: 1, Sender: &ids[1], Content: "hi"})
		So(conns[1], shouldReceive, SendEventType,
			&SendEvent{ID: sf2, UnixTime: 2, Sender: &ids[2], Content: "bye"})

		So(conns[2], shouldReceive, SendEventType,
			&SendEvent{ID: sf1, UnixTime: 1, Sender: &ids[1], Content: "hi"})
		So(conns[2], shouldReceive, SendReplyType,
			&SendReply{ID: sf2, UnixTime: 2, Sender: &ids[2], Content: "bye"})
	})
}

func testThreading(t testing.TB, s *serverUnderTest) {
	Convey("Send with parent", t, func() {
		tc := NewTestClock()
		defer tc.Close()

		conn := s.Connect("threading")
		defer closeConn(conn)

		id := &IdentityView{ID: conn.LocalAddr().String(), Name: "user"}
		id.Name = id.ID
		sfs := snowflakes(2)
		sf1 := sfs[0]
		sf2 := sfs[1]

		So(conn, shouldReceive, SnapshotEventType,
			&SnapshotEvent{
				Version: s.backend.Version(),
				Listing: Listing{},
				Log:     []Message{},
			})

		So(conn, shouldSend, `{"id":"1","type":"send","data":{"content":"root"}}`)
		So(conn, shouldReceive, SendReplyType,
			&SendReply{ID: sf1, UnixTime: 1, Sender: id, Content: "root"})

		So(conn, shouldSend,
			`{"id":"2","type":"send","data":{"parent":"%s","content":"child1"}}`, sf1)
		So(conn, shouldReceive, SendReplyType,
			&SendReply{ID: sf2, Parent: sf1, UnixTime: 2, Sender: id, Content: "child1"})

		So(conn, shouldSend, `{"id":"3","type":"log","data":{"n":10}}`)
		So(conn, shouldReceive, LogReplyType,
			&LogReply{
				Log: []Message{
					{
						ID:       sf1,
						UnixTime: 1,
						Sender:   id,
						Content:  "root",
					},
					{
						ID:       sf2,
						Parent:   sf1,
						UnixTime: 2,
						Sender:   id,
						Content:  "child1",
					},
				},
			})
	})
}
