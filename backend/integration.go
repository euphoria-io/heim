package backend

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/websocket"

	. "github.com/smartystreets/goconvey/convey"
)

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

func readPacket(conn *websocket.Conn) (CommandType, interface{}) {
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
	if msg := ShouldBeTrue(ok); msg != "" {
		return msg
	}
	if msg := ShouldEqual(len(expected), 2); msg != "" {
		return msg
	}
	expectedType, ok := expected[0].(string)
	if msg := ShouldBeTrue(ok); msg != "" {
		return msg
	}
	expectedPayload := expected[1]

	packetType, payload := readPacket(conn)
	if msg := ShouldEqual(packetType, expectedType); msg != "" {
		return msg
	}

	return ShouldResemble(payload, expectedPayload)
}

func IntegrationTest(factory func() Backend) func() {
	runTest := func(test func(*serverUnderTest)) {
		backend := factory()
		app := NewServer(backend, "")
		server := httptest.NewServer(app)
		defer server.Close()
		test(&serverUnderTest{backend, app, server})
	}

	return func() {
		runTest(testLurker)
	}
}

func testLurker(s *serverUnderTest) {
	Convey("Lurker", func() {
		conn1 := s.Connect("test")
		defer conn1.Close()

		conn2 := s.Connect("test")
		defer conn2.Close()

		err := conn2.WriteMessage(
			websocket.TextMessage,
			[]byte(`{"id":"1","type":"nick","data":{"name":"speaker"}}`))
		So(err, ShouldBeNil)

		So(conn1, shouldReceive,
			NickType, &NickCommand{From: conn2.LocalAddr().String(), Name: "speaker"})
	})
}

func testBroadcast(s *serverUnderTest) {
	Convey("Broadcast", func() {
		saveClock := clock
		clock = func() int64 { return 0 }
		defer func() { clock = saveClock }()

		conns := make([]*websocket.Conn, 3)
		for i := 0; i < 3; i++ {
			conns[i] = s.Connect("test")
		}
		defer func() {
			for _, conn := range conns {
				conn.Close()
			}
		}()

		ids := make([]*IdentityView, len(conns))

		for i, conn := range conns {
			ids[i] = &IdentityView{ID: conn.LocalAddr().String(), Name: fmt.Sprintf("test%d", i)}
			msg := fmt.Sprintf(`{"id":"1","type":"nick","data":{"name":"user%d"}}`, i)
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
			So(err, ShouldBeNil)
		}

		err := conns[1].WriteMessage(
			websocket.TextMessage,
			[]byte(`{"id":"2","type":"send","data":{"content":"hi"}}`))
		So(err, ShouldBeNil)

		err = conns[2].WriteMessage(
			websocket.TextMessage,
			[]byte(`{"id":"2","type":"send","data":{"content":"bye"}}`))
		So(err, ShouldBeNil)

		So(conns[0], shouldReceive,
			NickType, &NickCommand{From: conns[1].LocalAddr().String(), Name: "test1"})
		So(conns[0], shouldReceive,
			NickType, &NickCommand{From: conns[2].LocalAddr().String(), Name: "test2"})

		So(conns[0], shouldReceive,
			SendType, &Message{UnixTime: 0, Sender: ids[1], Content: "hi"})
		So(conns[0], shouldReceive,
			SendType, &Message{UnixTime: 0, Sender: ids[2], Content: "bye"})

		So(conns[1], shouldReceive,
			SendType, &Message{UnixTime: 0, Sender: ids[2], Content: "bye"})

		So(conns[2], shouldReceive,
			SendType, &Message{UnixTime: 0, Sender: ids[1], Content: "hi"})
	})
}
