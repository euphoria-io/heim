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

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"github.com/gorilla/websocket"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/smartystreets/goconvey/convey/reporting"
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
	snowflake.SeqCounter = 0
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
	return &testConn{Conn: conn}
}

type testConn struct {
	*websocket.Conn
	sessionID string
	userID    string
}

func (tc *testConn) id() string { return tc.userID }

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

	fmt.Printf("packet: %s\n", string(data))
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

	if packetType == proto.SnapshotEventType {
		snapshot := payload.(*proto.SnapshotEvent)
		tc.sessionID = snapshot.SessionID
		tc.userID = snapshot.Identity
		snapshot.SessionID = "???"
		snapshot.Identity = "???"
	}

	result := ""

	if msg := ShouldResemble(payload, expectedPayload); msg != "" {
		e, _ := json.Marshal(expectedPayload)
		a, _ := json.Marshal(payload)
		view := reporting.FailureView{
			Message:  fmt.Sprintf("Expected: %s\nActual:   %s\nShould resemble!", string(e), string(a)),
			Expected: string(e),
			Actual:   string(a),
		}
		r, _ := json.Marshal(view)
		result = string(r)
	}

	So(nil, func(interface{}, ...interface{}) string { return result })
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

func (tc *testConn) expectPing() {
	fmt.Printf("reading packet, expecting ping-event\n")
	packetType, payload := tc.readPacket()
	fmt.Printf("%s received %v, %#v\n", tc.RemoteAddr(), packetType, payload)
	So(packetType, ShouldEqual, "ping-event")
}

func (tc *testConn) expectSnapshot(version string, listingParts []string, logParts []string) {
	tc.expect("", "snapshot-event",
		`{"identity":"???","session_id":"???","version":"%s","listing":[%s],"log":[%s]}`,
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
	agentIDCounter := 0

	runTest := func(test testSuite) {
		backend := factory()
		defer backend.Close()
		kms := security.LocalKMS()
		kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
		app, err := NewServer(scope.New(), backend, &cluster.TestCluster{}, kms, "test1", "era1", "")
		So(err, ShouldBeNil)
		app.AllowRoomCreation(true)
		app.agentIDGenerator = func() ([]byte, error) {
			agentIDCounter++
			return []byte(fmt.Sprintf("%d", agentIDCounter)), nil
		}
		server := httptest.NewServer(app)
		defer server.Close()
		defer server.CloseClientConnections()
		test(&serverUnderTest{backend, app, server})
	}

	runTestWithFactory := func(test factoryTestSuite) { test(factory) }

	runTest(testLurker)
	runTest(testBroadcast)
	runTest(testThreading)
	runTest(testAuthentication)

	runTestWithFactory(testPresence)
	runTest(testDeletion)
}

func testLurker(s *serverUnderTest) {
	Convey("Lurker", func() {
		conn1 := s.Connect("lurker")
		defer conn1.Close()

		conn1.expectPing()
		conn1.expectSnapshot(s.backend.Version(), nil, nil)
		id1 := conn1.id()

		conn2 := s.Connect("lurker")
		defer conn2.Close()

		conn2.expectPing()
		conn2.expectSnapshot(s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					conn1.sessionID, id1)},
			nil)
		id2 := conn2.id()

		conn2.send("1", "nick", `{"name":"speaker"}`)
		conn2.expect("1", "nick-reply",
			`{"session_id":"%s","id":"%s","from":"","to":"speaker"}`, conn2.sessionID, conn2.id())

		conn1.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`, conn2.sessionID, id2)
		conn1.expect("", "nick-event",
			`{"session_id":"%s","id":"%s","to":"speaker"}`, conn2.sessionID, conn2.id())
	})
}

func testBroadcast(s *serverUnderTest) {
	Convey("Broadcast", func() {
		tc := NewTestClock()
		defer tc.Close()

		conns := make([]*testConn, 3)

		ids := make(proto.Listing, len(conns))

		listingParts := []string{}

		for i := range conns {
			conn := s.Connect("broadcast")
			conns[i] = conn
			conn.send("1", "nick", `{"name":"user%d"}`, i)
			conn.send("2", "who", "")

			conn.expectPing()
			conn.expectSnapshot(s.backend.Version(), listingParts, nil)
			me := conn.id()
			ids[i] = proto.SessionView{
				SessionID:    conn.sessionID,
				IdentityView: &proto.IdentityView{ID: me, Name: fmt.Sprintf("user%d", i)},
			}
			listingParts = append(listingParts,
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"%s","server_id":"test1","server_era":"era1"}`,
					ids[i].SessionID, ids[i].ID, ids[i].Name))

			conn.expect("1", "nick-reply",
				`{"session_id":"%s","id":"%s","from":"","to":"%s"}`,
				ids[i].SessionID, ids[i].ID, ids[i].Name)
			conn.expect("2", "who-reply", `{"listing":[%s]}`, strings.Join(listingParts, ","))

			for _, c := range conns[:i] {
				c.expect("", "join-event",
					`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					ids[i].SessionID, ids[i].ID)
				c.expect("", "nick-event",
					`{"session_id":"%s","id":"%s","from":"","to":"%s"}`,
					ids[i].SessionID, ids[i].ID, ids[i].Name)
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
		server := `"server_id":"test1","server_era":"era1"`

		conns[1].send("2", "send", `{"content":"hi"}`)
		conns[0].expect("", "send-event",
			`{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"hi"}`,
			sf1, ids[1].SessionID, ids[1].ID, ids[1].Name, server)

		conns[2].send("2", "send", `{"content":"bye"}`)
		conns[0].expect("", "send-event",
			`{"id":"%s","time":2,"sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"bye"}`,
			sf2, ids[2].SessionID, ids[2].ID, ids[2].Name, server)

		conns[1].expect("2", "send-reply",
			`{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"hi"}`,
			sf1, ids[1].SessionID, ids[1].ID, ids[1].Name, server)
		conns[1].expect("", "send-event",
			`{"id":"%s","time":2,"sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"bye"}`,
			sf2, ids[2].SessionID, ids[2].ID, ids[2].Name, server)

		conns[2].expect("", "send-event",
			`{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"hi"}`,
			sf1, ids[1].SessionID, ids[1].ID, ids[1].Name, server)
		conns[2].expect("2", "send-reply",
			`{"id":"%s","time":2,"sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"bye"}`,
			sf2, ids[2].SessionID, ids[2].ID, ids[2].Name, server)
	})
}

func testThreading(s *serverUnderTest) {
	Convey("Send with parent", func() {
		tc := NewTestClock()
		defer tc.Close()

		conn := s.Connect("threading")
		defer conn.Close()

		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)

		id := &proto.SessionView{
			SessionID:    conn.sessionID,
			IdentityView: &proto.IdentityView{ID: conn.id(), Name: conn.id()},
		}
		sfs := snowflakes(2)
		sf1 := sfs[0]
		sf2 := sfs[1]
		server := `"name":"test","server_id":"test1","server_era":"era1"`

		conn.send("1", "nick", `{"name":"test"}`)
		conn.expect("1", "nick-reply",
			`{"session_id":"%s","id":"%s","from":"","to":"test"}`, conn.sessionID, conn.id())

		conn.send("1", "send", `{"content":"root"}`)
		conn.expect("1", "send-reply",
			`{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s",%s},"content":"root"}`,
			sf1, id.SessionID, id.ID, server)

		conn.send("2", "send", `{"parent":"%s","content":"ch1"}`, sf1)
		conn.expect("2", "send-reply",
			`{"id":"%s","parent":"%s","time":2,"sender":{"session_id":"%s","id":"%s",%s},"content":"ch1"}`,
			sf2, sf1, id.SessionID, id.ID, server)

		conn.send("3", "log", `{"n":10}`)
		conn.expect("3", "log-reply",
			`{"log":[`+
				`{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s",%s},"content":"root"},`+
				`{"id":"%s","parent":"%s","time":2,"sender":{"session_id":"%s","id":"%s",%s},"content":"ch1"}]}`,
			sf1, id.SessionID, id.ID, server, sf2, sf1, id.SessionID, id.ID, server)
	})
}

func testPresence(factory func() proto.Backend) {
	backend := factory()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
	app, err := NewServer(scope.New(), backend, &cluster.TestCluster{}, kms, "test1", "era1", "")
	So(err, ShouldBeNil)
	app.AllowRoomCreation(true)
	agentIDCounter := 0
	app.agentIDGenerator = func() ([]byte, error) {
		agentIDCounter++
		return []byte(fmt.Sprintf("%d", agentIDCounter)), nil
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer server.CloseClientConnections()
	s := &serverUnderTest{backend, app, server}

	Convey("Other party joins then parts", func() {
		self := s.Connect("presence")
		defer self.Close()
		self.expectPing()
		self.expectSnapshot(s.backend.Version(), nil, nil)
		selfID := self.id()

		other := s.Connect("presence")
		other.expectPing()
		other.expectSnapshot(s.backend.Version(),
			[]string{
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					self.sessionID, selfID),
			}, nil)
		otherID := other.id()

		self.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			other.sessionID, otherID)
		self.send("1", "who", "")
		server := `"server_id":"test1","server_era":"era1"`
		self.expect("1", "who-reply",
			`{"listing":[{"session_id":"%s","id":"%s",%s},{"session_id":"%s","id":"%s",%s}]}`,
			self.sessionID, selfID, server, other.sessionID, otherID, server)

		other.Close()
		self.expect("", "part-event",
			`{"session_id":"%s","id":"%s",%s}`, other.sessionID, otherID, server)

		self.send("2", "who", "")
		self.expect("2", "who-reply",
			`{"listing":[{"session_id":"%s","id":"%s",%s}]}`, self.sessionID, selfID, server)
	})

	Convey("Join after other party, other party parts", func() {
		other := s.Connect("presence2")
		other.expectPing()
		other.expectSnapshot(s.backend.Version(), nil, nil)
		otherID := other.id()

		self := s.Connect("presence2")
		defer self.Close()
		self.expectPing()
		self.expectSnapshot(s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					other.sessionID, otherID)},
			nil)
		selfID := self.id()

		other.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			self.sessionID, selfID)
		self.send("1", "who", "")
		server := `"server_id":"test1","server_era":"era1"`
		self.expect("1", "who-reply",
			`{"listing":[{"session_id":"%s","id":"%s",%s},{"session_id":"%s","id":"%s",%s}]}`,
			other.sessionID, otherID, server, self.sessionID, selfID, server)

		other.Close()
		self.expect("", "part-event",
			`{"session_id":"%s","id":"%s",%s}`, other.sessionID, otherID, server)

		self.send("2", "who", "")
		self.expect("2", "who-reply",
			`{"listing":[{"session_id":"%s","id":"%s",%s}]}`, self.sessionID, selfID, server)
	})

	/*
		// Only run the following against distributed backends.
		if _, ok := backend.(*TestBackend); ok {
			return
		}

		backend2 := factory()
		kms := security.LocalKMS()
		app2, err := NewServer(scope.New(), backend2, &cluster.TestCluster{}, kms, "test2", "", "")
		So(err, ShouldBeNil)
		app2.AllowRoomCreation(true)
		server2 := httptest.NewServer(app2)
		defer server2.Close()
		s2 := &serverUnderTest{backend2, app2, server2}

		SkipConvey("Learns presence on startup", func() {
			self1 := s.Connect("presence3")
			defer self1.Close()
			self1.expectSnapshot(s.backend.Version(), nil, nil)
			id1 := self1.id()

			self2 := s2.Connect("presence3")
			defer self2.Close()
			self2.expectSnapshot(s.backend.Version(),
				[]string{fmt.Sprintf(`{"id":"%s"}`, id1)}, nil)
			fmt.Printf("ok!\n")
			//id2 := self2.id()
		})

		// TODO:
		SkipSkipConvey("Loses presence on shutdown", func() {
		})
	*/

}

func testAuthentication(s *serverUnderTest) {
	room, err := s.backend.GetRoom("private", true)
	So(err, ShouldBeNil)

	ctx := scope.New()
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
		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)

		conn.send("1", "ping", "{}")
		conn.expect("1", "ping-reply", `{}`)

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
		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)

		conn.send("1", "auth", `{"type":"passcode","passcode":"hunter2"}`)
		conn.expect("1", "auth-reply", `{"success":true}`)
	})
}

func testDeletion(s *serverUnderTest) {
	Convey("Deletion", func() {
		tc := NewTestClock()
		defer tc.Close()

		conn := s.Connect("deletion")
		defer conn.Close()

		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)

		id := &proto.SessionView{
			SessionID:    conn.sessionID,
			IdentityView: &proto.IdentityView{ID: conn.id(), Name: conn.id()},
		}

		sfs := snowflakes(2)
		sf := sfs[0]

		server := `"name":"speaker","server_id":"test1","server_era":"era1"`

		conn.send("1", "nick", `{"name":"speaker"}`)
		conn.expect("1", "nick-reply",
			`{"session_id":"%s","id":"%s","from":"","to":"speaker"}`, conn.sessionID, conn.id())

		conn.send("1", "send", `{"content":"@#$!"}`)
		conn.expect("1", "send-reply",
			`{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s",%s},"content":"@#$!"}`,
			sf, id.SessionID, id.ID, server)

		conn.send("3", "log", `{"n":10}`)
		conn.expect("3", "log-reply",
			`{"log":[{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s",%s},"content":"@#$!"}]}`,
			sf, id.SessionID, id.ID, server)

		room, err := s.backend.GetRoom("deletion", false)
		So(err, ShouldBeNil)

		cmd := proto.EditMessageCommand{
			ID:       sf,
			Delete:   true,
			Announce: true,
		}
		So(room.EditMessage(scope.New(), nil, cmd), ShouldBeNil)

		conn.expect("", "edit-message-event",
			`{"id":"%s","time":1,"sender":{"session_id":"%s","id":"%s",%s},"deleted":3,"edited":3,`+
				`"content":"@#$!","edit_id":"%s"}`,
			sf, id.SessionID, id.ID, server, sfs[1])

		conn2 := s.Connect("deletion")
		defer conn2.Close()

		conn2.expectPing()
		conn2.expectSnapshot(
			s.backend.Version(),
			[]string{fmt.Sprintf(`{"session_id":"%s","id":"%s",%s}`, conn.sessionID, id.ID, server)},
			nil)
	})
}
