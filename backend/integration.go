package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"

	"heim/proto"
	"heim/proto/security"
	"heim/proto/snowflake"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"

	. "github.com/smartystreets/goconvey/convey"
)

// TODO: move time mocking to snowflake_test?
type testClock struct {
	secs            int64
	savedClock      func() time.Time
	savedSnowflaker snowflake.Snowflaker
	savedEpoch      time.Time
	savedSeqCounter uint64
}

func NewTestClock() io.Closer {
	tc := &testClock{
		savedClock:      snowflake.Clock,
		savedSnowflaker: snowflake.DefaultSnowflaker,
		savedEpoch:      snowflake.Epoch,
		savedSeqCounter: snowflake.SeqCounter,
	}
	snowflake.Clock = tc.clock
	snowflake.DefaultSnowflaker = tc
	snowflake.Epoch = time.Unix(0, 0)
	return tc
}

func (tc *testClock) Close() error {
	snowflake.Clock = tc.savedClock
	snowflake.DefaultSnowflaker = tc.savedSnowflaker
	snowflake.Epoch = tc.savedEpoch
	snowflake.SeqCounter = tc.savedSeqCounter
	return nil
}

func (tc *testClock) Next() (uint64, error) {
	sf := snowflake.NewFromTime(tc.clock())
	return uint64(sf), nil
}

func (tc *testClock) clock() time.Time {
	secs := atomic.AddInt64(&tc.secs, 1)
	return time.Unix(secs, 0)
}

type factoryTestSuite func(factory func() proto.Backend)
type testSuite func(*serverUnderTest)

type serverUnderTest struct {
	backend proto.Backend
	app     *Server
	server  *httptest.Server
}

func (s *serverUnderTest) Close() {
	s.server.CloseClientConnections()
	s.server.Close()
}

func (s *serverUnderTest) Connect(roomName string) *testConn {
	url := strings.Replace(s.server.URL, "http:", "ws:", 1) + "/room/" + roomName + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	So(err, ShouldBeNil)
	return &testConn{conn}
}

type testConn struct {
	*websocket.Conn
}

func (tc *testConn) send(id, cmdType, data string, args ...interface{}) {
	if len(args) > 0 {
		data = fmt.Sprintf(data, args...)
	}
	var msg string
	if data == "" {
		msg = fmt.Sprintf(`{"id":"%s","type":"%s"}`, id, cmdType)
	} else {
		msg = fmt.Sprintf(`{"id":"%s","type":"%s","data":%s}`, id, cmdType, data)
	}
	So(tc.Conn.WriteMessage(websocket.TextMessage, []byte(msg)), ShouldBeNil)
}

func (tc *testConn) readPacket() (proto.PacketType, interface{}) {
	msgType, data, err := tc.Conn.ReadMessage()
	So(err, ShouldBeNil)
	So(msgType, ShouldEqual, websocket.TextMessage)

	var packet proto.Packet
	So(json.Unmarshal(data, &packet), ShouldBeNil)

	if packet.Error != "" {
		return packet.Type, errors.New(packet.Error)
	}

	payload, err := packet.Payload()
	So(err, ShouldBeNil)
	return packet.Type, payload
}

func (tc *testConn) expect(id, cmdType, data string, args ...interface{}) {
	if len(args) > 0 {
		data = fmt.Sprintf(data, args...)
	}

	fmt.Printf("reading packet, expecting %s\n", cmdType)
	packetType, payload := tc.readPacket()
	fmt.Printf("%s received %v, %#v\n", tc.RemoteAddr(), packetType, payload)
	So(packetType, ShouldEqual, cmdType)

	var expected proto.Packet
	expectedString := fmt.Sprintf(`{"id":"%s","type":"%s","data":%s}`, id, cmdType, data)
	So(json.Unmarshal([]byte(expectedString), &expected), ShouldBeNil)
	expectedPayload, err := expected.Payload()
	So(err, ShouldBeNil)

	So(payload, ShouldResemble, expectedPayload)
}

func (tc *testConn) expectError(id, cmdType, errFormat string, errArgs ...interface{}) {
	errMsg := errFormat
	if len(errArgs) > 0 {
		errMsg = fmt.Sprintf(errFormat, errArgs...)
	}

	fmt.Printf("reading packet, expecting %s error\n", cmdType)
	packetType, payload := tc.readPacket()
	fmt.Printf("%s received %v, %#v\n", tc.RemoteAddr(), packetType, payload)
	So(packetType, ShouldEqual, cmdType)
	err, ok := payload.(error)
	So(ok, ShouldBeTrue)
	So(err.Error(), ShouldEqual, errMsg)
}

func (tc *testConn) expectSnapshot(version string, listingParts []string, logParts []string) {
	tc.expect("", "snapshot-event", `{"version":"%s","listing":[%s],"log":[%s]}`,
		version, strings.Join(listingParts, ","), strings.Join(logParts, ","))
}

func (tc *testConn) Close() {
	tc.Conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "normal closure"))
}

func snowflakes(n int) []snowflake.Snowflake {
	fc := NewTestClock()
	defer fc.Close()

	snowflakes := make([]snowflake.Snowflake, n)
	for i := range snowflakes {
		var err error
		snowflakes[i], err = snowflake.New()
		So(err, ShouldBeNil)
	}
	return snowflakes
}

func IntegrationTest(factory func() proto.Backend) {
	runTest := func(test testSuite) {
		backend := factory()
		kms := security.LocalKMS()
		kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
		app := NewServer(backend, kms, "test1", "")
		server := httptest.NewServer(app)
		defer server.Close()
		test(&serverUnderTest{backend, app, server})
	}

	runTestWithFactory := func(test factoryTestSuite) { test(factory) }

	runTest(testLurker)
	runTest(testBroadcast)
	runTest(testThreading)
	runTest(testAuthentication)

	runTestWithFactory(testPresence)
}

var skipConvey = SkipConvey

func testLurker(s *serverUnderTest) {
	skipConvey("Lurker", func() {
		conn1 := s.Connect("lurker")
		defer conn1.Close()
		id1 := conn1.LocalAddr().String()

		conn1.expectSnapshot(s.backend.Version(), nil, nil)

		conn2 := s.Connect("lurker")
		defer conn2.Close()
		id2 := conn2.LocalAddr().String()

		conn2.expectSnapshot(s.backend.Version(),
			[]string{fmt.Sprintf(`{"id":"%s","name":"guest"}`, id1)},
			nil)

		conn2.send("1", "nick", `{"name":"speaker"}`)
		conn2.expect("1", "nick-reply", `{"id":"%s","from":"guest","to":"speaker"}`, id2)

		conn1.expect("", "join-event", `{"id":"%s","name":"guest"}`, id2)
		conn1.expect("", "nick-event", `{"id":"%s","from":"guest","to":"speaker"}`, id2)
	})
}

func testBroadcast(s *serverUnderTest) {
	skipConvey("Broadcast", func() {
		tc := NewTestClock()
		defer tc.Close()

		conns := make([]*testConn, 3)

		ids := make(proto.Listing, len(conns))

		listingParts := []string{}

		for i := range conns {
			conn := s.Connect("broadcast")
			conns[i] = conn
			me := conn.LocalAddr().String()
			ids[i] = proto.IdentityView{ID: me, Name: fmt.Sprintf("user%d", i)}
			conn.send("1", "nick", `{"name":"user%d"}`, i)
			conn.send("2", "who", "")

			conn.expectSnapshot(s.backend.Version(), listingParts, nil)
			listingParts = append(listingParts,
				fmt.Sprintf(`{"id":"%s","name":"%s"}`, ids[i].ID, ids[i].Name))

			conn.expect("1", "nick-reply",
				`{"id":"%s","from":"guest","to":"%s"}`, ids[i].ID, ids[i].Name)
			conn.expect("2", "who-reply", `{"listing":[%s]}`, strings.Join(listingParts, ","))

			for _, c := range conns[:i] {
				c.expect("", "join-event", `{"id":"%s","name":"guest"}`, ids[i].ID)
				c.expect("", "nick-event",
					`{"id":"%s","from":"guest","to":"%s"}`, ids[i].ID, ids[i].Name)
			}
		}

		defer func() {
			for _, conn := range conns {
				defer conn.Close()
			}
		}()

		sfs := snowflakes(2)
		sf1 := sfs[0]
		sf2 := sfs[1]

		conns[1].send("2", "send", `{"content":"hi"}`)
		conns[0].expect("", "send-event",
			`{"id":"%s","time":1,"sender":{"id":"%s","name":"%s"},"content":"hi"}`,
			sf1, ids[1].ID, ids[1].Name)

		conns[2].send("2", "send", `{"content":"bye"}`)
		conns[0].expect("", "send-event",
			`{"id":"%s","time":2,"sender":{"id":"%s","name":"%s"},"content":"bye"}`,
			sf2, ids[2].ID, ids[2].Name)

		conns[1].expect("2", "send-reply",
			`{"id":"%s","time":1,"sender":{"id":"%s","name":"%s"},"content":"hi"}`,
			sf1, ids[1].ID, ids[1].Name)
		conns[1].expect("", "send-event",
			`{"id":"%s","time":2,"sender":{"id":"%s","name":"%s"},"content":"bye"}`,
			sf2, ids[2].ID, ids[2].Name)

		conns[2].expect("", "send-event",
			`{"id":"%s","time":1,"sender":{"id":"%s","name":"%s"},"content":"hi"}`,
			sf1, ids[1].ID, ids[1].Name)
		conns[2].expect("2", "send-reply",
			`{"id":"%s","time":2,"sender":{"id":"%s","name":"%s"},"content":"bye"}`,
			sf2, ids[2].ID, ids[2].Name)
	})
}

func testThreading(s *serverUnderTest) {
	skipConvey("Send with parent", func() {
		tc := NewTestClock()
		defer tc.Close()

		conn := s.Connect("threading")
		defer conn.Close()

		id := &proto.IdentityView{ID: conn.LocalAddr().String(), Name: "user"}
		id.Name = id.ID
		sfs := snowflakes(2)
		sf1 := sfs[0]
		sf2 := sfs[1]

		conn.expectSnapshot(s.backend.Version(), nil, nil)

		conn.send("1", "send", `{"content":"root"}`)
		conn.expect("1", "send-reply",
			`{"id":"%s","time":1,"sender":{"id":"%s","name":"guest"},"content":"root"}`,
			sf1, id.ID)

		conn.send("2", "send", `{"parent":"%s","content":"ch1"}`, sf1)
		conn.expect("2", "send-reply",
			`{"id":"%s","parent":"%s","time":2,"sender":{"id":"%s","name":"guest"},"content":"ch1"}`,
			sf2, sf1, id.ID)

		conn.send("3", "log", `{"n":10}`)
		conn.expect("3", "log-reply",
			`{"log":[`+
				`{"id":"%s","time":1,"sender":{"id":"%s","name":"guest"},"content":"root"},`+
				`{"id":"%s","parent":"%s","time":2,"sender":{"id":"%s","name":"guest"},"content":"ch1"}]}`,
			sf1, id.ID, sf2, sf1, id.ID)
	})
}

func testPresence(factory func() proto.Backend) {
	backend := factory()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
	app := NewServer(backend, kms, "test1", "")
	server := httptest.NewServer(app)
	defer server.Close()
	s := &serverUnderTest{backend, app, server}

	skipConvey("Other party joins then parts", func() {
		self := s.Connect("presence")
		defer self.Close()
		self.expectSnapshot(s.backend.Version(), nil, nil)
		selfID := self.LocalAddr().String()

		other := s.Connect("presence")
		other.expectSnapshot(s.backend.Version(),
			[]string{fmt.Sprintf(`{"id":"%s","name":"guest"}`, selfID)}, nil)
		otherID := other.LocalAddr().String()

		self.expect("", "join-event", `{"id":"%s","name":"guest"}`, otherID)
		self.send("1", "who", "")
		self.expect("1", "who-reply",
			`{"listing":[{"id":"%s","name":"guest"},{"id":"%s","name":"guest"}]}`, selfID, otherID)

		other.Close()
		self.expect("", "part-event", `{"id":"%s","name":"guest"}`, otherID)

		self.send("2", "who", "")
		self.expect("2", "who-reply", `{"listing":[{"id":"%s","name":"guest"}]}`, selfID)
	})

	skipConvey("Join after other party, other party parts", func() {
		other := s.Connect("presence2")
		otherID := other.LocalAddr().String()
		other.expectSnapshot(s.backend.Version(), nil, nil)

		self := s.Connect("presence2")
		defer self.Close()
		selfID := self.LocalAddr().String()
		self.expectSnapshot(s.backend.Version(),
			[]string{fmt.Sprintf(`{"id":"%s","name":"guest"}`, otherID)},
			nil)

		other.expect("", "join-event", `{"id":"%s","name":"guest"}`, selfID)
		self.send("1", "who", "")
		self.expect("1", "who-reply",
			`{"listing":[{"id":"%s","name":"guest"},{"id":"%s","name":"guest"}]}`, otherID, selfID)

		other.Close()
		self.expect("", "part-event", `{"id":"%s","name":"guest"}`, otherID)

		self.send("2", "who", "")
		self.expect("2", "who-reply",
			`{"listing":[{"id":"%s","name":"guest"}]}`, selfID)
	})

	/*
		// Only run the following against distributed backends.
		if _, ok := backend.(*TestBackend); ok {
			return
		}

		backend2 := factory()
		app2 := NewServer(backend2, "tes2", "")
		server2 := httptest.NewServer(app2)
		defer server2.Close()
		s2 := &serverUnderTest{backend2, app2, server2}

		Convey("Learns presence on startup", func() {
			self1 := s.Connect("presence3")
			defer self1.Close()
			self1.expectSnapshot(s.backend.Version(), nil, nil)
			id1 := self1.LocalAddr().String()

			self2 := s2.Connect("presence3")
			defer self2.Close()
			self2.expectSnapshot(s.backend.Version(),
				[]string{fmt.Sprintf(`{"id":"%s","name":"guest"}`, id1)}, nil)
			fmt.Printf("ok!\n")
			//id2 := self2.LocalAddr().String()
		})

		// TODO:
		SkipConvey("Loses presence on shutdown", func() {
		})
	*/

}

func testAuthentication(s *serverUnderTest) {
	room, err := s.backend.GetRoom("private")
	So(err, ShouldBeNil)

	ctx := context.Background()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
	rkey, err := room.GenerateMasterKey(ctx, kms)
	So(err, ShouldBeNil)

	mkey := rkey.ManagedKey()
	capability, err := security.GrantCapabilityOnSubjectWithPasscode(
		ctx, kms, rkey.Nonce(), &mkey, []byte("hunter2"))
	So(err, ShouldBeNil)
	So(room.SaveCapability(ctx, capability), ShouldBeNil)

	Convey("Access denied", func() {
		conn := s.Connect("private")
		defer conn.Close()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)

		conn.send("1", "who", "")
		conn.expectError("1", "who-reply", "access denied, please authenticate")

		conn.send("1", "auth", `{"type":"passcode","passcode":"dunno"}`)
		conn.expect("1", "auth-reply", `{"success":false,"reason":"passcode incorrect"}`)

		conn.send("1", "who", "")
		conn.expectError("1", "who-reply", "access denied, please authenticate")
	})

	Convey("Access granted", func() {
		conn := s.Connect("private")
		defer conn.Close()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)

		conn.send("1", "auth", `{"type":"passcode","passcode":"hunter2"}`)
		conn.expect("1", "auth-reply", `{"success":true}`)
	})
}
