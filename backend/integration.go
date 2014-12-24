package backend

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	. "github.com/smartystreets/goconvey/convey"
)

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
}

func testLurker(t testing.TB, s *serverUnderTest) {
	Convey("Lurker", t, func() {
		conn1 := s.Connect("test")
		defer closeConn(conn1)

		conn2 := s.Connect("test")
		defer closeConn(conn2)

		err := conn2.WriteMessage(
			websocket.TextMessage,
			[]byte(`{"id":"1","type":"nick","data":{"name":"speaker"}}`))
		So(err, ShouldBeNil)

		id := conn2.LocalAddr().String()
		So(conn2, shouldReceive, NickReplyType, &NickReply{ID: id, From: id, To: "speaker"})
		So(conn1, shouldReceive, JoinEventType, &PresenceEvent{ID: id, Name: id})
		So(conn1, shouldReceive, NickEventType, &NickEvent{ID: id, From: id, To: "speaker"})
	})
}

func testBroadcast(t testing.TB, s *serverUnderTest) {
	Convey("Broadcast", t, func() {
		saveClock := Clock
		timer := int64(0)
		Clock = func() time.Time {
			defer func() { timer++ }()
			return time.Unix(timer, 0)
		}
		defer func() { Clock = saveClock }()

		conns := make([]*websocket.Conn, 3)

		ids := make(Listing, len(conns))

		for i := range conns {
			conn := s.Connect("test")
			conns[i] = conn
			ids[i] = IdentityView{ID: conn.LocalAddr().String(), Name: fmt.Sprintf("user%d", i)}
			msg := fmt.Sprintf(`{"id":"1","type":"nick","data":{"name":"user%d"}}`, i)
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			So(err, ShouldBeNil)

			msg = fmt.Sprintf(`{"id":"2","type":"who"}`)
			err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
			So(err, ShouldBeNil)

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

		err := conns[1].WriteMessage(
			websocket.TextMessage,
			[]byte(`{"id":"2","type":"send","data":{"content":"hi"}}`))
		So(err, ShouldBeNil)

		So(conns[0], shouldReceive, SendEventType,
			&SendEvent{UnixTime: 0, Sender: &ids[1], Content: "hi"})

		err = conns[2].WriteMessage(
			websocket.TextMessage,
			[]byte(`{"id":"2","type":"send","data":{"content":"bye"}}`))
		So(err, ShouldBeNil)

		So(conns[0], shouldReceive, SendEventType,
			&SendEvent{UnixTime: 1, Sender: &ids[2], Content: "bye"})

		So(conns[1], shouldReceive, SendReplyType,
			&SendReply{UnixTime: 0, Sender: &ids[1], Content: "hi"})
		So(conns[1], shouldReceive, SendEventType,
			&SendEvent{UnixTime: 1, Sender: &ids[2], Content: "bye"})

		So(conns[2], shouldReceive, SendEventType,
			&SendEvent{UnixTime: 0, Sender: &ids[1], Content: "hi"})
		So(conns[2], shouldReceive, SendReplyType,
			&SendReply{UnixTime: 1, Sender: &ids[2], Content: "bye"})
	})
}
