package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
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

var agentIDCounter int

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

func newServerUnderTest(
	backend proto.Backend, app *Server, server *httptest.Server, kms security.MockKMS) *serverUnderTest {
	return &serverUnderTest{
		backend:     backend,
		app:         app,
		server:      server,
		kms:         kms,
		accounts:    map[string]proto.Account{},
		accountKeys: map[string]*security.ManagedKey{},
		rooms:       map[string]proto.Room{},
	}
}

type serverUnderTest struct {
	backend     proto.Backend
	app         *Server
	server      *httptest.Server
	kms         security.MockKMS
	once        sync.Once
	accounts    map[string]proto.Account
	accountKeys map[string]*security.ManagedKey
	rooms       map[string]proto.Room
}

func (s *serverUnderTest) Close() {
	s.server.CloseClientConnections()
	s.server.Close()
	s.backend.Close()
}

func (s *serverUnderTest) Connect(roomName string, cookies ...*http.Cookie) *testConn {
	if _, err := s.backend.GetRoom(scope.New(), roomName); err == proto.ErrRoomNotFound {
		_, err = s.backend.CreateRoom(scope.New(), s.app.kms, false, roomName)
		So(err, ShouldBeNil)
	}
	headers := http.Header{}
	for _, cookie := range cookies {
		headers.Add("Cookie", cookie.String())
	}
	url := strings.Replace(s.server.URL, "http:", "ws:", 1) + "/room/" + roomName + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		body, _ := ioutil.ReadAll(resp.Body)
		So(string(body), ShouldEqual, "")
	}
	So(err, ShouldBeNil)
	return &testConn{Conn: conn, cookies: resp.Cookies()}
}

func (s *serverUnderTest) Account(
	ctx scope.Context, kms security.KMS, namespace, id, password string) (
	proto.Account, *security.ManagedKey, error) {

	key := fmt.Sprintf("%s:%s", namespace, id)
	if account, ok := s.accounts[key]; ok {
		return account, s.accountKeys[key], nil
	}

	b := s.backend
	at := b.AgentTracker()
	agentKey := &security.ManagedKey{
		KeyType:   proto.AgentKeyType,
		Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
	}
	agent, err := proto.NewAgent([]byte(id), agentKey)
	if err != nil {
		return nil, nil, err
	}
	if err := at.Register(ctx, agent); err != nil {
		return nil, nil, err
	}

	account, clientKey, err := b.AccountManager().Register(
		ctx, kms, namespace, id, password, agent.IDString(), agentKey)
	if err != nil {
		return nil, nil, err
	}

	s.accounts[key] = account
	s.accountKeys[key] = clientKey
	return account, clientKey, nil
}

func (s *serverUnderTest) Room(
	ctx scope.Context, kms security.KMS, private bool, name string, managers ...proto.Account) (
	proto.Room, error) {

	if room, ok := s.rooms[name]; ok {
		return room, nil
	}

	room, err := s.backend.CreateRoom(ctx, kms, private, name, managers...)
	if err != nil {
		return nil, err
	}

	s.rooms[name] = room
	return room, nil
}

type testConn struct {
	*websocket.Conn
	cookies   []*http.Cookie
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

	fmt.Printf("%s received %s\n", tc.LocalAddr(), string(data))
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
			Message: fmt.Sprintf(
				"Expected: (%T) %s\nActual:   (%T) %s\nShould resemble!",
				expectedPayload, string(e), payload, string(a)),
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
	So(packetType, ShouldEqual, cmdType)
	err, ok := payload.(error)
	So(ok, ShouldBeTrue)
	So(err.Error(), ShouldEqual, errMsg)
}

func (tc *testConn) expectPing() {
	fmt.Printf("reading packet, expecting ping-event\n")
	packetType, _ := tc.readPacket()
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

func IntegrationTest(t *testing.T, factory func() proto.Backend) {
	runTest := func(name string, test testSuite) {
		backend := factory()
		defer backend.Close()

		kms := security.LocalKMS()
		kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
		app, err := NewServer(scope.New(), backend, &cluster.TestCluster{}, kms, "test1", "era1", "")
		if err != nil {
			t.Fatal(err)
		}

		app.AllowRoomCreation(true)
		app.agentIDGenerator = func() ([]byte, error) {
			agentIDCounter++
			return []byte(fmt.Sprintf("%d", agentIDCounter)), nil
		}

		server := httptest.NewServer(app)
		defer server.Close()
		defer server.CloseClientConnections()

		s := newServerUnderTest(backend, app, server, kms)
		Convey(name, t, func() { test(s) })
	}

	runTestWithFactory := func(name string, test factoryTestSuite) {
		Convey(name, t, func() { test(factory) })
	}
	_ = runTestWithFactory

	// Internal API tests
	runTest("Accounts low-level API", testAccountsLowLevel)
	runTest("Managers low-level API", testManagersLowLevel)
	runTest("Staff low-level API", testStaffLowLevel)

	// Websocket tests
	runTest("Lurker", testLurker)
	runTest("Broadcast", testBroadcast)
	runTest("Threading", testThreading)
	runTest("Authentication", testAuthentication)
	runTestWithFactory("Presence", testPresence)
	runTest("Deletion", testDeletion)
	runTest("Account login", testAccountLogin)
	runTest("Account registration", testAccountRegistration)
	runTest("Room creation", testRoomCreation)
	runTest("Room grants", testRoomGrants)
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

		// Connect a lurker first. We'll receive events through this connection
		// first, to control timing.
		lurker := s.Connect("broadcast")
		defer lurker.Close()

		lurker.expectPing()
		lurker.expectSnapshot(s.backend.Version(), nil, nil)
		listingParts := []string{
			fmt.Sprintf(
				`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
				lurker.sessionID, lurker.id()),
		}

		for i := range conns {
			conn := s.Connect("broadcast")
			conns[i] = conn

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

			conn.send("1", "nick", `{"name":"user%d"}`, i)
			conn.send("2", "who", "")

			conn.expect("1", "nick-reply",
				`{"session_id":"%s","id":"%s","from":"","to":"%s"}`,
				ids[i].SessionID, ids[i].ID, ids[i].Name)
			conn.expect("2", "who-reply", `{"listing":[%s]}`, strings.Join(listingParts, ","))

			lurker.expect("", "join-event",
				`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
				ids[i].SessionID, ids[i].ID)
			lurker.expect("", "nick-event",
				`{"session_id":"%s","id":"%s","from":"","to":"%s"}`,
				ids[i].SessionID, ids[i].ID, ids[i].Name)

			for j, c := range conns[:i] {
				fmt.Printf("\n>>> id %s expecting events for new conn %s\n\n", ids[j].ID, ids[i].ID)
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
	app.agentIDGenerator = func() ([]byte, error) {
		agentIDCounter++
		return []byte(fmt.Sprintf("%d", agentIDCounter)), nil
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer server.CloseClientConnections()
	s := newServerUnderTest(backend, app, server, kms)

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
	ctx := scope.New()
	kms := security.LocalKMS()
	kms.SetMasterKey(make([]byte, security.AES256.KeySize()))

	logan, loganKey, err := s.Account(ctx, kms, "email", "logan-private", "loganpass")
	So(err, ShouldBeNil)

	room, err := s.Room(ctx, kms, true, "private", logan)
	So(err, ShouldBeNil)

	s.once.Do(func() {
		rkey, err := room.MessageKey(ctx)
		So(err, ShouldBeNil)
		So(rkey.GrantToPasscode(ctx, logan, loganKey, "hunter2"), ShouldBeNil)
	})

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

	Convey("Access granted to passcode", func() {
		conn := s.Connect("private")
		defer conn.Close()
		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)

		conn.send("1", "auth", `{"type":"passcode","passcode":"hunter2"}`)
		conn.expect("1", "auth-reply", `{"success":true}`)
	})

	Convey("Access granted to account", func() {
		// Authenticate in new session.
		conn := s.Connect("private")
		defer conn.Close()
		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		conn.send("1", "login",
			`{"namespace":"email","id":"logan-private","password":"loganpass"}`)
		conn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Reconnect with authentication.
		conn = s.Connect("private", conn.cookies...)
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
	})
}

func testDeletion(s *serverUnderTest) {
	Convey("Deletion", func() {
		tc := NewTestClock()
		defer tc.Close()

		b := s.backend
		ctx := scope.New()
		kms := s.app.kms
		at := b.AgentTracker()
		agentKey := &security.ManagedKey{
			KeyType:   proto.AgentKeyType,
			Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
		}

		// Create manager account and room.
		nonce := fmt.Sprintf("deletion-%s", time.Now())
		loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, loganAgent), ShouldBeNil)
		logan, _, err := b.AccountManager().Register(
			ctx, kms, "email", "logan"+nonce, "loganpass", loganAgent.IDString(), agentKey)
		So(err, ShouldBeNil)

		_, err = b.CreateRoom(ctx, kms, false, "deletion", logan)
		So(err, ShouldBeNil)

		// Connect to stage room to log in.
		conn := s.Connect("deletionstage")
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "login",
			`{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		conn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Connect to deletion room as manager.
		conn = s.Connect("deletion", conn.cookies...)
		defer conn.Close()

		sfs := snowflakes(3)
		sf := sfs[1]

		server := `"name":"speaker","server_id":"test1","server_era":"era1"`

		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "nick", `{"name":"speaker"}`)
		conn.expect("1", "nick-reply",
			`{"session_id":"%s","id":"%s","from":"","to":"speaker"}`, conn.sessionID, conn.id())

		conn.send("1", "send", `{"content":"@#$!"}`)
		conn.expect("1", "send-reply",
			`{"id":"%s","time":2,"sender":{"session_id":"%s","id":"%s",%s},"content":"@#$!"}`,
			sf, conn.sessionID, conn.userID, server)

		conn.send("3", "log", `{"n":10}`)
		conn.expect("3", "log-reply",
			`{"log":[{"id":"%s","time":2,"sender":{"session_id":"%s","id":"%s",%s},"content":"@#$!"}]}`,
			sf, conn.sessionID, conn.userID, server)

		// Delete message.
		conn.send("4", "edit-message", `{"id":"%s","delete":true,"announce":true}`, sf)
		conn.expect("4", "edit-message-reply", `{"edit_id":"%s","deleted":true}`, sfs[2])

		conn2 := s.Connect("deletion")
		defer conn2.Close()

		conn2.expectPing()
		conn2.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s",%s}`, conn.sessionID, conn.userID, server)},
			nil)
	})
}

func testAccountsLowLevel(s *serverUnderTest) {
	b := s.backend
	kms := s.app.kms

	ctx := scope.New()
	at := b.AgentTracker()
	agentKey := &security.ManagedKey{
		KeyType:   proto.AgentKeyType,
		Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
	}
	nonce := fmt.Sprintf("%s", time.Now())

	loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
	So(err, ShouldBeNil)
	So(at.Register(ctx, loganAgent), ShouldBeNil)

	maxAgent, err := proto.NewAgent([]byte("max"+nonce), agentKey)
	So(err, ShouldBeNil)
	So(at.Register(ctx, maxAgent), ShouldBeNil)

	Convey("Account registration", func() {
		account, key, err := b.AccountManager().Register(
			ctx, kms, "email", "logan@euphoria.io", "hunter2", loganAgent.IDString(), agentKey)
		So(err, ShouldBeNil)
		So(account, ShouldNotBeNil)
		So(key, ShouldNotBeNil)
		So(key.Encrypted(), ShouldBeFalse)

		kp, err := account.Unlock(account.KeyFromPassword(""))
		So(err, ShouldEqual, proto.ErrAccessDenied)
		So(kp, ShouldBeNil)

		kp, err = account.Unlock(account.KeyFromPassword("hunter2"))
		So(err, ShouldBeNil)
		So(kp, ShouldNotBeNil)

		kp, err = account.Unlock(key)
		So(err, ShouldBeNil)
		So(kp, ShouldNotBeNil)

		dup, _, err := b.AccountManager().Register(
			ctx, kms, "email", "logan@euphoria.io", "hunter2", loganAgent.IDString(), agentKey)
		So(err, ShouldEqual, proto.ErrPersonalIdentityInUse)
		So(dup, ShouldBeNil)
	})

	Convey("Account lookup", func() {
		var badID snowflake.Snowflake
		badID.FromString("nosuchaccount")
		account, err := b.AccountManager().Get(ctx, badID)
		So(err, ShouldEqual, proto.ErrAccountNotFound)
		So(account, ShouldBeNil)

		account, err = b.AccountManager().Resolve(ctx, "email", "max@euphoria.io")
		So(err, ShouldEqual, proto.ErrAccountNotFound)
		So(account, ShouldBeNil)

		_, _, err = b.AccountManager().Register(
			ctx, kms, "email", "max@euphoria.io", "hunter2", maxAgent.IDString(), agentKey)
		So(err, ShouldBeNil)

		account, err = b.AccountManager().Resolve(ctx, "email", "max@euphoria.io")
		So(err, ShouldBeNil)

		kp, err := account.Unlock(account.KeyFromPassword(""))
		So(err, ShouldEqual, proto.ErrAccessDenied)
		So(kp, ShouldBeNil)

		kp, err = account.Unlock(account.KeyFromPassword("hunter2"))
		So(err, ShouldBeNil)
		So(kp, ShouldNotBeNil)

		dup, err := b.AccountManager().Get(ctx, account.ID())
		So(err, ShouldBeNil)
		So(dup, ShouldNotBeNil)
		So(dup.KeyPair().PublicKey, ShouldResemble, kp.PublicKey)
	})
}

func testStaffLowLevel(s *serverUnderTest) {
	Convey("Setting and checking staff capability", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms
		at := b.AgentTracker()
		agentKey := &security.ManagedKey{
			KeyType:   proto.AgentKeyType,
			Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
		}

		// Create test account.
		nonce := fmt.Sprintf("%s", time.Now())
		loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, loganAgent), ShouldBeNil)
		logan, loganKey, err := b.AccountManager().Register(
			ctx, kms, "email", "logan"+nonce, "loganpass", loganAgent.IDString(), agentKey)
		So(err, ShouldBeNil)
		So(logan.IsStaff(), ShouldBeFalse)

		// Enable staff
		So(b.AccountManager().GrantStaff(ctx, logan.ID(), s.kms.KMSCredential()), ShouldBeNil)
		logan, err = b.AccountManager().Get(ctx, logan.ID())
		So(err, ShouldBeNil)
		So(logan.IsStaff(), ShouldBeTrue)

		// Unlock staff KMS and verify compatibility.
		staffKMS, err := logan.UnlockStaffKMS(loganKey)
		So(err, ShouldBeNil)
		testKey, err := kms.GenerateEncryptedKey(security.AES128, "test", "test")
		So(err, ShouldBeNil)
		clonedKey := testKey.Clone()
		So(kms.DecryptKey(testKey), ShouldBeNil)
		So(staffKMS.DecryptKey(&clonedKey), ShouldBeNil)
		So(&clonedKey, ShouldResemble, testKey)

		// Revoke staff
		So(b.AccountManager().RevokeStaff(ctx, logan.ID()), ShouldBeNil)
		logan, err = b.AccountManager().Get(ctx, logan.ID())
		So(err, ShouldBeNil)
		So(logan.IsStaff(), ShouldBeFalse)

		// Account not found error
		So(b.AccountManager().GrantStaff(ctx, 0, s.kms.KMSCredential()),
			ShouldEqual, proto.ErrAccountNotFound)
	})
}

func testManagersLowLevel(s *serverUnderTest) {
	b := s.backend
	ctx := scope.New()
	kms := s.app.kms
	at := b.AgentTracker()
	agentKey := &security.ManagedKey{
		KeyType:   proto.AgentKeyType,
		Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
	}

	// Create test accounts.
	nonce := fmt.Sprintf("%s", time.Now())

	aliceAgent, err := proto.NewAgent([]byte("alice"+nonce), agentKey)
	So(err, ShouldBeNil)
	So(at.Register(ctx, aliceAgent), ShouldBeNil)
	alice, aliceKey, err := b.AccountManager().Register(
		ctx, kms, "email", "alice"+nonce, "alicepass", aliceAgent.IDString(), agentKey)
	So(err, ShouldBeNil)

	bobAgent, err := proto.NewAgent([]byte("bob"+nonce), agentKey)
	So(err, ShouldBeNil)
	So(at.Register(ctx, bobAgent), ShouldBeNil)
	bob, bobKey, err := b.AccountManager().Register(
		ctx, kms, "email", "bob"+nonce, "bobpass", bobAgent.IDString(), agentKey)
	So(err, ShouldBeNil)

	carolAgent, err := proto.NewAgent([]byte("carol"+nonce), agentKey)
	So(err, ShouldBeNil)
	So(at.Register(ctx, carolAgent), ShouldBeNil)
	carol, carolKey, err := b.AccountManager().Register(
		ctx, kms, "email", "carol"+nonce, "carolpass", carolAgent.IDString(), agentKey)
	So(err, ShouldBeNil)

	names := map[string]string{
		alice.ID().String(): "alice",
		bob.ID().String():   "bob",
		carol.ID().String(): "carol",
	}

	// Create room owned by alice and bob.
	room, err := b.CreateRoom(ctx, kms, false, "management"+nonce, alice, bob)
	So(err, ShouldBeNil)

	shouldComprise := func(actual interface{}, expected ...interface{}) string {
		expectedNames := make([]string, len(expected))
		for i, v := range expected {
			expectedNames[i] = v.(string)
		}
		managers := actual.([]proto.Account)
		actualNames := make([]string, len(managers))
		for i, manager := range managers {
			actualNames[i] = names[manager.ID().String()]
		}
		sort.Strings(actualNames)
		sort.Strings(expectedNames)
		return ShouldResemble(actualNames, expectedNames)
	}

	Convey("GetManagers should return initial managers from room creation", func() {
		managers, err := room.Managers(ctx)
		So(err, ShouldBeNil)
		So(managers, shouldComprise, "alice", "bob")
	})

	Convey("Non-manager should be unable to add or remove manager", func() {
		So(room.AddManager(ctx, kms, carol, carolKey, carol), ShouldEqual, proto.ErrAccessDenied)
		So(room.RemoveManager(ctx, carol, carolKey, carol), ShouldEqual, proto.ErrAccessDenied)
		So(room.RemoveManager(ctx, carol, carolKey, alice), ShouldEqual, proto.ErrAccessDenied)
	})

	Convey("Manager should be able to add new manager", func() {
		So(room.AddManager(ctx, kms, alice, aliceKey, carol), ShouldBeNil)
		managers, err := room.Managers(ctx)
		So(err, ShouldBeNil)
		So(managers, shouldComprise, "alice", "bob", "carol")
	})

	Convey("New manager should be able to remove other manager", func() {
		So(room.AddManager(ctx, kms, bob, bobKey, carol), ShouldBeNil)
		So(room.RemoveManager(ctx, carol, carolKey, bob), ShouldBeNil)
		managers, err := room.Managers(ctx)
		So(err, ShouldBeNil)
		So(managers, shouldComprise, "alice", "carol")
	})

	Convey("Manager should be able to remove self", func() {
		So(room.RemoveManager(ctx, alice, aliceKey, alice), ShouldBeNil)
		managers, err := room.Managers(ctx)
		So(err, ShouldBeNil)
		So(managers, shouldComprise, "bob")

		So(room.AddManager(ctx, kms, alice, aliceKey, alice), ShouldEqual, proto.ErrAccessDenied)
		So(room.RemoveManager(ctx, alice, aliceKey, bob), ShouldEqual, proto.ErrAccessDenied)
	})

	Convey("Redundant manager addition should be a no-op", func() {
		So(room.AddManager(ctx, kms, alice, aliceKey, bob), ShouldBeNil)
		managers, err := room.Managers(ctx)
		So(err, ShouldBeNil)
		So(managers, shouldComprise, "alice", "bob")
		So(room.AddManager(ctx, kms, carol, carolKey, bob), ShouldEqual, proto.ErrAccessDenied)
	})

	Convey("Redundant manager removal should be an error", func() {
		So(room.RemoveManager(ctx, alice, aliceKey, bob), ShouldBeNil)
		So(room.RemoveManager(ctx, alice, aliceKey, bob), ShouldEqual, proto.ErrManagerNotFound)
	})
}

func testAccountLogin(s *serverUnderTest) {
	b := s.backend
	at := b.AgentTracker()
	ctx := scope.New()
	kms := s.app.kms
	nonce := fmt.Sprintf("%s", time.Now())
	agentKey := &security.ManagedKey{
		KeyType:   proto.AgentKeyType,
		Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
	}
	loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
	So(err, ShouldBeNil)
	So(at.Register(ctx, loganAgent), ShouldBeNil)
	logan, _, err := b.AccountManager().Register(
		ctx, kms, "email", "logan"+nonce, "loganpass", loganAgent.IDString(), agentKey)
	So(err, ShouldBeNil)

	Convey("Agent logs into account", func() {
		tc := NewTestClock()
		defer tc.Close()

		// Add observer for timing control.
		observer := s.Connect("login")
		defer observer.Close()

		observer.expectPing()
		observer.expectSnapshot(s.backend.Version(), nil, nil)

		// Connect as test user.
		conn := s.Connect("login")
		conn.expectPing()
		conn.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldStartWith, "agent:")

		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Log in.
		conn.send("1", "login",
			`{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		conn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Wait for part.
		observer.expect("", "part-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Verify logged in on reconnect.
		conn = s.Connect("login", conn.cookies...)
		conn.expectPing()
		conn.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldEqual, fmt.Sprintf("account:%s", logan.ID()))

		// Wait for join.
		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Log out first party and wait for part.
		conn.send("1", "logout", "")
		conn.expect("1", "logout-reply", "{}")
		conn.expect("", "disconnect-event", `{"reason": "authentication changed"}`)
		conn.Close()
		observer.expect("", "part-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Observer should fail to log in with incorrect identity or password.
		observer.send("1", "login", `{"namespace":"email","id":"wrongid","password":"wrongpass"}`)
		observer.expect("1", "login-reply", `{"success":false,"reason":"account not found"}`)
		observer.send("2", "login", `{"namespace":"email","id":"logan%s","password":"wrongpass"}`, nonce)
		observer.expect("2", "login-reply", `{"success":false,"reason":"access denied"}`)
	})
}

func testAccountRegistration(s *serverUnderTest) {
	Convey("Agent upgrades to account", func() {
		tc := NewTestClock()
		defer tc.Close()

		// Skip ahead in snowflakes to avoid account_id collision.
		for i := 0; i < 1000; i++ {
			snowflake.New()
		}

		// Add observer for timing control.
		observer := s.Connect("registration")
		defer observer.Close()

		observer.expectPing()
		observer.expectSnapshot(s.backend.Version(), nil, nil)

		// Connect as test user.
		conn := s.Connect("registration")
		defer conn.Close()

		conn.expectPing()
		conn.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldStartWith, "agent:")

		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Upgrade to account.
		sfs := snowflakes(1001)[1000:]
		conn.send("1", "register-account",
			`{"namespace":"email","id":"registration@euphoria.io","password":"hunter2"}`)
		conn.expect("1", "register-account-reply", `{"success":true,"account_id":"%s"}`, sfs[0])
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Wait for part.
		observer.expect("", "part-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Verify logged in on reconnect.
		conn = s.Connect("registration", conn.cookies...)
		conn.expectPing()
		conn.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldEqual, fmt.Sprintf("account:%s", sfs[0]))

		// Wait for join.
		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Observer should fail to register the same personal identity.
		observer.send("1", "register-account",
			`{"namespace":"email","id":"registration@euphoria.io","password":"hunter2"}`)
		fmt.Printf("SECOND REGISTRATION\n")
		observer.expect("1", "register-account-reply",
			`{"success":false,"reason":"personal identity already in use"}`)
	})

	Convey("Min agent age prevents account registration", func() {
		s.app.NewAccountMinAgentAge(time.Hour)
		conn := s.Connect("registration2")
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "register-account",
			`{"namespace":"email","id":"newaccount@euphoria.io","password":"hunter2"}`)
		conn.expect("1", "register-account-reply",
			`{"success":false,"reason":"not familiar yet, try again later"}`)
	})
}

func testRoomCreation(s *serverUnderTest) {
	Convey("Unlock staff capability and create room", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms
		at := b.AgentTracker()
		agentKey := &security.ManagedKey{
			KeyType:   proto.AgentKeyType,
			Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
		}

		// Create staff account.
		nonce := fmt.Sprintf("%s", time.Now())
		loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, loganAgent), ShouldBeNil)
		logan, _, err := b.AccountManager().Register(
			ctx, kms, "email", "logan"+nonce, "loganpass", loganAgent.IDString(), agentKey)
		So(err, ShouldBeNil)
		So(b.AccountManager().GrantStaff(ctx, logan.ID(), s.kms.KMSCredential()), ShouldBeNil)

		// Connect and log into staff account in a throwaway room.
		conn := s.Connect("createroomstage")
		defer conn.Close()

		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "login",
			`{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		conn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Reconnect, and fail to create room because staff capability is locked.
		conn = s.Connect("createroom", conn.cookies...)
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		/*
			conn.send("1", "staff-create-room", `{"name":"create-room-new","managers":["%s"],"private":true}`,
				logan.ID())
			conn.expect("1", "staff-create-room-reply",
				`{"success":false,"failure_reason":"must unlock staff capability first"}`)

			// Unlock staff capability and try again.
			conn.send("2", "unlock-staff-capability", `{"password":"loganpass"}`)
			conn.expect("2", "unlock-staff-capability-reply", `{"success":true}`)
		*/
		conn.send("3", "staff-create-room", `{"name":"create-room-new","managers":["%s"],"private":true}`,
			logan.ID())
		conn.expect("3", "staff-create-room-reply", `{"success":true}`)

		// Verify room.
		room, err := s.backend.GetRoom(ctx, "create-room-new")
		So(err, ShouldBeNil)
		managers, err := room.Managers(ctx)
		So(len(managers), ShouldEqual, 1)
		So(managers[0].ID(), ShouldEqual, logan.ID())
		mkey, err := room.MessageKey(ctx)
		So(err, ShouldBeNil)
		So(mkey, ShouldNotBeNil)
	})
}

func testRoomGrants(s *serverUnderTest) {
	Convey("Grant access to account", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms
		at := b.AgentTracker()
		agentKey := &security.ManagedKey{
			KeyType:   proto.AgentKeyType,
			Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
		}

		// Create manager account and room.
		nonce := fmt.Sprintf("%s", time.Now())
		loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, loganAgent), ShouldBeNil)
		logan, _, err := b.AccountManager().Register(
			ctx, kms, "email", "logan"+nonce, "loganpass", loganAgent.IDString(), agentKey)
		So(err, ShouldBeNil)

		_, err = b.CreateRoom(ctx, kms, true, "grants", logan)
		So(err, ShouldBeNil)

		// Create access account (without access yet).
		maxAgent, err := proto.NewAgent([]byte("max"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, maxAgent), ShouldBeNil)
		max, _, err := b.AccountManager().Register(
			ctx, kms, "email", "max"+nonce, "maxpass", maxAgent.IDString(), agentKey)

		// Connect and log into manager account in a throwaway room.
		loganConn := s.Connect("grantsstage")
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)
		loganConn.send("1", "login", `{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		loganConn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		loganConn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		loganConn.Close()

		// Reconnect manager to private room.
		loganConn = s.Connect("grants", loganConn.cookies...)
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)

		// Fail to connect with access account.
		maxConn := s.Connect("grants")
		maxConn.expectPing()
		maxConn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		maxConn.send("1", "login", `{"namespace":"email","id":"max%s","password":"maxpass"}`, nonce)
		maxConn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, max.ID())
		maxConn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		maxConn.Close()
		maxConn = s.Connect("grants", maxConn.cookies...)
		maxConn.expectPing()
		maxConn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		maxConn.Close()

		// Grant access.
		loganConn.send("1", "grant-access", `{"account_id":"%s"}`, max.ID())
		loganConn.expect("1", "grant-access-reply", `{}`)

		// Connect with access account.
		maxConn = s.Connect("grants", maxConn.cookies...)
		maxConn.expectPing()
		maxConn.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
					loganConn.sessionID, loganConn.userID)},
			nil)

		// Synchronize and revoke access.
		maxConn.Close()
		loganConn.expect("", "join-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			maxConn.sessionID, maxConn.id())
		loganConn.expect("", "part-event",
			`{"session_id":"%s","id":"%s","server_id":"test1","server_era":"era1"}`,
			maxConn.sessionID, maxConn.id())
		loganConn.send("2", "revoke-access", `{"account_id":"%s"}`, max.ID())
		loganConn.expect("2", "revoke-access-reply", `{}`)
		loganConn.Close()

		maxConn = s.Connect("grants", maxConn.cookies...)
		maxConn.expectPing()
		maxConn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		maxConn.Close()
	})

	Convey("Grant manager by staff", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms
		at := b.AgentTracker()
		agentKey := &security.ManagedKey{
			KeyType:   proto.AgentKeyType,
			Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
		}

		// Create staff account and room.
		_, err := b.CreateRoom(ctx, kms, true, "staffmanagergrants")
		So(err, ShouldBeNil)

		nonce := fmt.Sprintf("+%s", time.Now())
		loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, loganAgent), ShouldBeNil)
		logan, _, err := b.AccountManager().Register(
			ctx, kms, "email", "logan"+nonce, "loganpass", loganAgent.IDString(), agentKey)
		So(err, ShouldBeNil)
		So(b.AccountManager().GrantStaff(ctx, logan.ID(), s.kms.KMSCredential()), ShouldBeNil)

		// Connect to room, log in, reconnect, and grant management to self.
		loganConn := s.Connect("staffmanagergrants")
		loganConn.expectPing()
		loganConn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		loganConn.send("1", "login", `{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		loganConn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		loganConn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)

		loganConn = s.Connect("staffmanagergrants", loganConn.cookies...)
		loganConn.expectPing()
		loganConn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		loganConn.send("1", "unlock-staff-capability", `{"password":"loganpass"}`)
		loganConn.expect("1", "unlock-staff-capability-reply", `{"success":true}`)
		loganConn.send("2", "staff-grant-manager", `{"account_id":"%s"}`, logan.ID())
		loganConn.expect("2", "staff-grant-manager-reply", `{}`)
		loganConn.Close()

		// Reconnect to verify.
		loganConn = s.Connect("staffmanagergrants", loganConn.cookies...)
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)

		// Revoke self as manager.
		loganConn.send("1", "unlock-staff-capability", `{"password":"loganpass"}`)
		loganConn.expect("1", "unlock-staff-capability-reply", `{"success":true}`)
		loganConn.send("2", "staff-revoke-manager", `{"account_id":"%s"}`, logan.ID())
		loganConn.expect("2", "staff-revoke-manager-reply", `{}`)
	})

	Convey("Grant manager to account", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms
		at := b.AgentTracker()
		agentKey := &security.ManagedKey{
			KeyType:   proto.AgentKeyType,
			Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
		}

		// Create manager account and room.
		nonce := fmt.Sprintf("+%s", time.Now())
		loganAgent, err := proto.NewAgent([]byte("logan"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, loganAgent), ShouldBeNil)
		logan, _, err := b.AccountManager().Register(
			ctx, kms, "email", "logan"+nonce, "loganpass", loganAgent.IDString(), agentKey)
		So(err, ShouldBeNil)

		room, err := b.CreateRoom(ctx, kms, true, "managergrants", logan)
		So(err, ShouldBeNil)

		// Create access account (without access yet).
		maxAgent, err := proto.NewAgent([]byte("max"+nonce), agentKey)
		So(err, ShouldBeNil)
		So(at.Register(ctx, maxAgent), ShouldBeNil)
		max, _, err := b.AccountManager().Register(
			ctx, kms, "email", "max"+nonce, "maxpass", maxAgent.IDString(), agentKey)

		// Connect and log into manager account in a throwaway room.
		loganConn := s.Connect("managergrantsstage")
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)
		loganConn.send("1", "login", `{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		loganConn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		loganConn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		loganConn.Close()

		// Reconnect manager to private room.
		loganConn = s.Connect("managergrants", loganConn.cookies...)
		defer loganConn.Close()
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)
		loganConn.send("1", "grant-manager", `{"account_id":"%s"}`, max.ID())
		loganConn.expect("1", "grant-manager-reply", `{}`)

		// Revoke self as manager.
		loganConn.send("2", "revoke-manager", `{"account_id":"%s"}`, logan.ID())
		loganConn.expect("2", "revoke-manager-reply", `{}`)

		managers, err := room.Managers(ctx)
		So(err, ShouldBeNil)
		So(len(managers), ShouldEqual, 1)
		So(managers[0].ID(), ShouldEqual, max.ID())
	})
}
