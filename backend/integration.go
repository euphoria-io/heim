package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"github.com/gorilla/websocket"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/smartystreets/goconvey/convey/reporting"
)

var agentIDCounter int

type factoryTestSuite func(factory proto.BackendFactory)
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
		if resp != nil {
			body, _ := ioutil.ReadAll(resp.Body)
			So(string(body), ShouldEqual, "")
		}
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

func (s *serverUnderTest) RoomAndManager(
	ctx scope.Context, kms security.KMS, private bool, roomName, namespace, id, password string) (
	proto.Room, proto.Account, *security.ManagedKey, error) {

	manager, key, err := s.Account(ctx, kms, namespace, id, password)
	if err != nil {
		return nil, nil, nil, err
	}

	room, err := s.Room(ctx, kms, private, roomName, manager)
	if err != nil {
		return nil, nil, nil, err
	}

	return room, manager, key, err
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
	fmt.Printf("sent %s\n", msg)
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

func (tc *testConn) expect(id, cmdType, data string, args ...interface{}) map[string]interface{} {
	if len(args) > 0 {
		data = fmt.Sprintf(data, args...)
	}

	// Construct expected map
	var expected map[string]interface{}
	So(json.Unmarshal([]byte(data), &expected), ShouldBeNil)

	// Read packet
	msgType, packetData, err := tc.Conn.ReadMessage()
	So(err, ShouldBeNil)
	So(msgType, ShouldEqual, websocket.TextMessage)

	fmt.Printf("%s received %s\n", tc.LocalAddr(), string(packetData))
	var packet proto.Packet
	So(json.Unmarshal(packetData, &packet), ShouldBeNil)
	So(packet.Error, ShouldEqual, "")

	// Construct actual map
	var actual map[string]interface{}
	So(json.Unmarshal([]byte(packet.Data), &actual), ShouldBeNil)

	// Compare.
	var result string
	captures := map[string]interface{}{}
	if msg := matchPayload(captures, "", actual, expected); msg != "" {
		view := reporting.FailureView{
			Message: fmt.Sprintf(
				"Expected: %s\nActual:   %s\nReason:   (%s) %s",
				data, string(packet.Data), packet.Type, msg),
			Expected: data,
			Actual:   string(packet.Data),
		}
		r, _ := json.Marshal(view)
		result = string(r)
	}
	So(nil, func(interface{}, ...interface{}) string { return result })

	// Side-effect on snapshot-event: capture some session state.
	if packet.Type == proto.SnapshotEventType {
		payload, err := packet.Payload()
		So(err, ShouldBeNil)
		snapshot := payload.(*proto.SnapshotEvent)
		tc.sessionID = snapshot.SessionID
		tc.userID = string(snapshot.Identity)
	}

	return captures
}

func matchField(captures map[string]interface{}, name string, actual, expected interface{}) string {
	if evStr, ok := expected.(string); ok && evStr == "*" {
		captures[name] = actual
		return ""
	}
	if subExp, ok := expected.(map[string]interface{}); ok {
		subAct, ok := actual.(map[string]interface{})
		if !ok {
			return fmt.Sprintf("%s: expected object, got %T", name, actual)
		}
		return matchPayload(captures, name+".", subAct, subExp)
	}
	if listExp, ok := expected.([]interface{}); ok {
		listAct, ok := actual.([]interface{})
		if !ok {
			return fmt.Sprintf("%s: expected list, got %T", name, actual)
		}
		for i, _ := range listExp {
			msg := matchField(captures, fmt.Sprintf("%s[%d]", name, i), listAct[i], listExp[i])
			if msg != "" {
				return msg
			}
		}
		return ""
	}
	if msg := ShouldEqual(actual, expected); msg != "" {
		return fmt.Sprintf("%s: %s", name, msg)
	}
	return ""
}

func matchPayload(
	captures map[string]interface{}, prefix string, actual, expected map[string]interface{}) string {

	// Verify that each entry in expected has the correct value in actual.
	for name, expectedValue := range expected {
		actualValue, ok := actual[name]
		if !ok {
			return fmt.Sprintf("expected field %s=%#v", name, expectedValue)
		}
		if msg := matchField(captures, prefix+name, actualValue, expectedValue); msg != "" {
			return msg
		}
	}

	// Verify that each entry in actual was covered by expected.
	for name, actualValue := range actual {
		if _, ok := expected[name]; !ok && actualValue != nil {
			return fmt.Sprintf("unexpected field %s%s=%#v", prefix, name, actualValue)
		}
	}

	return ""
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

func (tc *testConn) expectPing() *proto.PingEvent {
	fmt.Printf("reading packet, expecting ping-event\n")
	packetType, payload := tc.readPacket()
	So(packetType, ShouldEqual, "ping-event")
	return payload.(*proto.PingEvent)
}

func (tc *testConn) expectSnapshot(version string, listingParts []string, logParts []string) {
	tc.expect("", "snapshot-event",
		`{"identity":"*","session_id":"*","version":"%s","listing":[%s],"log":[%s]}`,
		version, strings.Join(listingParts, ","), strings.Join(logParts, ","))
}

func (tc *testConn) Close() {
	tc.Conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "normal closure"))
}

func IntegrationTest(t *testing.T, factory proto.BackendFactory) {
	save := security.TestMode
	defer func() { security.TestMode = save }()
	security.TestMode = true

	runTest := func(name string, test testSuite) {
		heim := &proto.Heim{
			Cluster: &cluster.TestCluster{},
			Context: scope.New(),
			Emailer: &emails.TestEmailer{},
			KMS:     security.LocalKMS(),
		}
		heim.KMS.(security.MockKMS).SetMasterKey(make([]byte, security.AES256.KeySize()))

		backend, err := factory(heim)
		if err != nil {
			t.Fatal(err)
		}
		heim.Backend = backend
		defer heim.Backend.Close()

		app, err := NewServer(heim, "test1", "era1", "")
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

		s := newServerUnderTest(backend, app, server, heim.KMS.(security.MockKMS))
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
	runTest("Account change password", testAccountChangePassword)
	runTest("Account reset password", testAccountResetPassword)
	runTest("Room creation", testRoomCreation)
	runTest("Room grants", testRoomGrants)
	runTest("Room not found", testRoomNotFound)
	runTest("KeepAlive", testKeepAlive)
	runTest("Bans", testBans)
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
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					conn1.sessionID, id1)},
			nil)
		id2 := conn2.id()

		conn2.send("1", "nick", `{"name":"speaker"}`)
		conn2.expect("1", "nick-reply",
			`{"session_id":"%s","id":"%s","from":"","to":"speaker"}`, conn2.sessionID, conn2.id())

		conn1.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			conn2.sessionID, id2)
		conn1.expect("", "nick-event",
			`{"session_id":"%s","id":"%s","from":"","to":"speaker"}`, conn2.sessionID, conn2.id())
	})
}

func testBroadcast(s *serverUnderTest) {
	Convey("Broadcast", func() {
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
				SessionID: conn.sessionID,
				IdentityView: &proto.IdentityView{
					ID:   proto.UserID(me),
					Name: fmt.Sprintf("user%d", i),
				},
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
				`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
				ids[i].SessionID, ids[i].ID)
			lurker.expect("", "nick-event",
				`{"session_id":"%s","id":"%s","from":"","to":"%s"}`,
				ids[i].SessionID, ids[i].ID, ids[i].Name)

			for j, c := range conns[:i] {
				fmt.Printf("\n>>> id %s expecting events for new conn %s\n\n", ids[j].ID, ids[i].ID)
				c.expect("", "join-event",
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
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

		server := `"server_id":"test1","server_era":"era1"`

		conns[1].send("2", "send", `{"content":"hi"}`)
		conns[0].expect("", "send-event",
			`{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"hi"}`,
			ids[1].SessionID, ids[1].ID, ids[1].Name, server)

		conns[2].send("2", "send", `{"content":"bye"}`)
		conns[0].expect("", "send-event",
			`{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"bye"}`,
			ids[2].SessionID, ids[2].ID, ids[2].Name, server)

		conns[1].expect("2", "send-reply",
			`{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"hi"}`,
			ids[1].SessionID, ids[1].ID, ids[1].Name, server)
		conns[1].expect("", "send-event",
			`{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"bye"}`,
			ids[2].SessionID, ids[2].ID, ids[2].Name, server)

		conns[2].expect("", "send-event",
			`{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"hi"}`,
			ids[1].SessionID, ids[1].ID, ids[1].Name, server)
		conns[2].expect("2", "send-reply",
			`{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s","name":"%s",%s},"content":"bye"}`,
			ids[2].SessionID, ids[2].ID, ids[2].Name, server)
	})
}

func testThreading(s *serverUnderTest) {
	Convey("Send with parent", func() {
		ctx := scope.New()
		kms := s.app.kms

		owner, ownerKey, err := s.Account(ctx, kms, "email", "threading-owner", "passcode")
		So(err, ShouldBeNil)
		room, err := s.Room(ctx, kms, true, "threading", owner)
		So(err, ShouldBeNil)
		rkey, err := room.MessageKey(ctx)
		So(rkey.GrantToPasscode(ctx, owner, ownerKey, "hunter2"), ShouldBeNil)

		conn := s.Connect("threading")
		defer conn.Close()

		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		conn.send("1", "auth", `{"type":"passcode","passcode":"hunter2"}`)
		conn.expect("1", "auth-reply", `{"success":true}`)
		conn.expectSnapshot(s.backend.Version(), nil, nil)

		id := &proto.SessionView{
			SessionID:    conn.sessionID,
			IdentityView: &proto.IdentityView{ID: proto.UserID(conn.id()), Name: conn.id()},
		}
		server := `"name":"test","server_id":"test1","server_era":"era1"`
		sendReplyCommon := fmt.Sprintf(
			`"id":"*","time":"*","sender":{"session_id":"%s","id":"%s",%s},"encryption_key_id":"*"`,
			id.SessionID, id.ID, server)

		conn.send("2", "nick", `{"name":"test"}`)
		conn.expect("2", "nick-reply",
			`{"session_id":"%s","id":"%s","from":"","to":"test"}`, conn.sessionID, conn.id())

		conn.send("3", "send", `{"content":"root"}`)
		capture := conn.expect("3", "send-reply", `{%s,"content":"root"}`, sendReplyCommon)

		conn.send("4", "send", `{"parent":"%s","content":"ch1"}`, capture["id"])
		conn.expect("4", "send-reply", `{%s,"parent":"%s","content":"ch1"}`,
			sendReplyCommon, capture["id"])

		conn.send("5", "log", `{"n":10}`)
		conn.expect("5", "log-reply",
			`{"log":[`+
				`{"id":"%s","time":"*","sender":{"session_id":"%s","id":"%s",%s},"content":"root","encryption_key_id":"*"},`+
				`{"id":"*","parent":"%s","time":"*","sender":{"session_id":"%s","id":"%s",%s},"content":"ch1","encryption_key_id":"*"}]}`,
			capture["id"], id.SessionID, id.ID, server, capture["id"], id.SessionID, id.ID, server)
	})
}

func testPresence(factory proto.BackendFactory) {
	heim := &proto.Heim{
		Cluster: &cluster.TestCluster{},
		Context: scope.New(),
		KMS:     security.LocalKMS(),
	}
	heim.KMS.(security.MockKMS).SetMasterKey(make([]byte, security.AES256.KeySize()))

	backend, err := factory(heim)
	So(err, ShouldBeNil)
	heim.Backend = backend
	defer heim.Backend.Close()

	app, err := NewServer(heim, "test1", "era1", "")
	So(err, ShouldBeNil)
	app.AllowRoomCreation(true)
	app.agentIDGenerator = func() ([]byte, error) {
		agentIDCounter++
		return []byte(fmt.Sprintf("%d", agentIDCounter)), nil
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer server.CloseClientConnections()
	s := newServerUnderTest(backend, app, server, heim.KMS.(security.MockKMS))

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
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					self.sessionID, selfID),
			}, nil)
		otherID := other.id()

		self.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			other.sessionID, otherID)
		self.send("1", "who", "")
		server := `"name":"","server_id":"test1","server_era":"era1"`
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
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					other.sessionID, otherID)},
			nil)
		selfID := self.id()

		other.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			self.sessionID, selfID)
		self.send("1", "who", "")
		server := `"name":"","server_id":"test1","server_era":"era1"`
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

	Convey("Ignore after excessive failures", func() {
		conn := s.Connect("private")
		defer conn.Close()
		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		for i := 0; i < MaxAuthFailures; i++ {
			conn.send(fmt.Sprintf("%d", i+1), "auth", `{"type":"passcode","passcode":"dunno"}`)
			conn.expect(fmt.Sprintf("%d", i+1), "auth-reply",
				`{"success":false,"reason":"passcode incorrect"}`)
		}
		conn.send(fmt.Sprintf("%d", MaxAuthFailures+1), "auth", `{"type":"passcode","passcode":"dunno"}`)
		conn.send(fmt.Sprintf("%d", MaxAuthFailures+2), "ping", `{}`)
		conn.expect(fmt.Sprintf("%d", MaxAuthFailures+2), "ping-reply", `{}`)
	})
}

func testDeletion(s *serverUnderTest) {
	Convey("Deletion", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms

		// Create manager account and room.
		nonce := fmt.Sprintf("deletion-%s", time.Now())
		logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
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

		server := `"name":"speaker","server_id":"test1","server_era":"era1"`

		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "nick", `{"name":"speaker"}`)
		conn.expect("1", "nick-reply",
			`{"session_id":"%s","id":"%s","from":"","to":"speaker"}`, conn.sessionID, conn.id())

		conn.send("1", "send", `{"content":"@#$!"}`)
		capture := conn.expect("1", "send-reply",
			`{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s",%s},"content":"@#$!"}`,
			conn.sessionID, conn.userID, server)

		conn.send("3", "log", `{"n":10}`)
		conn.expect("3", "log-reply",
			`{"log":[{"id":"*","time":"*","sender":{"session_id":"%s","id":"%s",%s},"content":"@#$!"}]}`,
			conn.sessionID, conn.userID, server)

		// Delete message.
		conn.send("4", "edit-message", `{"id":"%s","delete":true,"announce":true}`, capture["id"])
		conn.expect("4", "edit-message-reply", `{"edit_id":"*","deleted":true}`)

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

		// Create test account.
		nonce := fmt.Sprintf("%s", time.Now())
		logan, loganKey, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
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

	// Create test accounts.
	nonce := fmt.Sprintf("%s", time.Now())

	alice, aliceKey, err := s.Account(ctx, kms, "email", "alice"+nonce, "alicepass")
	So(err, ShouldBeNil)

	bob, bobKey, err := s.Account(ctx, kms, "email", "bob"+nonce, "bobpass")
	So(err, ShouldBeNil)

	carol, carolKey, err := s.Account(ctx, kms, "email", "carol"+nonce, "carolpass")
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

func testAccountChangePassword(s *serverUnderTest) {
	ctx := scope.New()
	kms := s.app.kms
	nonce := fmt.Sprintf("%s", time.Now())
	logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "oldpass")
	So(err, ShouldBeNil)

	Convey("Change password", func() {
		inbox := s.app.heim.Emailer.(emails.MockEmailer).Inbox("logan" + nonce)

		conn := s.Connect("changepass1a")
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "change-password", `{}`)
		conn.expectError("1", "change-password-reply", "not logged in")
		conn.send("2", "login",
			`{"namespace":"email","id":"logan%s","password":"oldpass"}`, nonce)
		conn.expect("2", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		conn = s.Connect("changepass1b", conn.cookies...)
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "change-password", `{"old_password":"wrongpass","new_password":"newpass"}`)
		conn.expectError("1", "change-password-reply", "access denied")
		conn.send("2", "change-password", `{"old_password":"oldpass","new_password":"newpass"}`)
		conn.expect("2", "change-password-reply", `{}`)
		conn.Close()

		conn = s.Connect("changepass1c")
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("2", "login",
			`{"namespace":"email","id":"logan%s","password":"newpass"}`, nonce)
		conn.expect("2", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Password change email should have been sent.
		msg := <-inbox
		So(msg.Template, ShouldEqual, proto.PasswordChangedEmail)
		_, ok := msg.Data.(*proto.PasswordChangedEmailParams)
		So(ok, ShouldBeTrue)
	})
}

func testAccountResetPassword(s *serverUnderTest) {
	ctx := scope.New()
	kms := s.app.kms
	nonce := fmt.Sprintf("%s", time.Now())
	logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "oldpass")
	So(err, ShouldBeNil)

	Convey("Reset password", func() {
		inbox := s.app.heim.Emailer.(emails.MockEmailer).Inbox("logan" + nonce)

		// Issue password reset requests.
		conn := s.Connect("resetpass1")
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "reset-password", `{"namespace":"email","id":"logan%s"}`, nonce)
		conn.expect("1", "reset-password-reply", `{}`)

		// Receive confirmation code in email.
		msg := <-inbox
		So(msg.Template, ShouldEqual, proto.PasswordResetEmail)
		p, ok := msg.Data.(*proto.PasswordResetEmailParams)
		So(ok, ShouldBeTrue)
		firstConfirmation := p.Confirmation

		// Issue a second password reset request and grab confirmation code from email.
		conn.send("2", "reset-password", `{"namespace":"email","id":"logan%s"}`, nonce)
		conn.expect("2", "reset-password-reply", `{}`)
		conn.Close()
		msg = <-inbox
		So(msg.Template, ShouldEqual, proto.PasswordResetEmail)
		p, ok = msg.Data.(*proto.PasswordResetEmailParams)
		So(ok, ShouldBeTrue)

		// Apply new password with confirmation code.
		resp, err := http.PostForm(s.server.URL+"/prefs/reset-password", url.Values{
			"confirmation": []string{p.Confirmation},
			"password":     []string{"newpass"},
		})
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, 200)

		// Log in with new password.
		conn = s.Connect("resetpass2")
		conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("1", "login",
			`{"namespace":"email","id":"logan%s","password":"newpass"}`, nonce)
		conn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Attempt to use other confirmation code should fail.
		resp, err = http.PostForm(s.server.URL+"/prefs/reset-password", url.Values{
			"confirmation": []string{firstConfirmation},
			"password":     []string{"newpass2"},
		})
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, http.StatusBadRequest)
	})
}

func testAccountLogin(s *serverUnderTest) {
	ctx := scope.New()
	kms := s.app.kms
	nonce := fmt.Sprintf("%s", time.Now())
	logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
	So(err, ShouldBeNil)

	Convey("Agent logs into account", func() {
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
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldStartWith, "agent:")

		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Log in.
		conn.send("1", "login",
			`{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		conn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Wait for part.
		observer.expect("", "part-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Verify logged in on reconnect.
		conn = s.Connect("login", conn.cookies...)
		conn.expectPing()
		conn.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldEqual, fmt.Sprintf("account:%s", logan.ID()))

		// Wait for join.
		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Log out first party and wait for part.
		conn.send("1", "logout", "")
		conn.expect("1", "logout-reply", "{}")
		conn.expect("", "disconnect-event", `{"reason": "authentication changed"}`)
		conn.Close()
		observer.expect("", "part-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
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
		inbox := s.app.heim.Emailer.(emails.MockEmailer).Inbox("registration@euphoria.io")

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
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldStartWith, "agent:")

		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Upgrade to account.
		conn.send("1", "register-account",
			`{"namespace":"email","id":"registration@euphoria.io","password":"hunter2"}`)
		capture := conn.expect("1", "register-account-reply", `{"success":true,"account_id":"*"}`)
		accountID := capture["account_id"]
		conn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		conn.Close()

		// Wait for part.
		observer.expect("", "part-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Verify logged in on reconnect.
		conn = s.Connect("registration", conn.cookies...)
		conn.expectPing()
		conn.expectSnapshot(
			s.backend.Version(),
			[]string{
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					observer.sessionID, observer.userID)},
			nil)
		So(conn.userID, ShouldEqual, fmt.Sprintf("account:%s", accountID))

		// Wait for join.
		observer.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			conn.sessionID, conn.userID)

		// Observer should fail to register the same personal identity.
		observer.send("1", "register-account",
			`{"namespace":"email","id":"registration@euphoria.io","password":"hunter2"}`)
		observer.expect("1", "register-account-reply",
			`{"success":false,"reason":"personal identity already in use"}`)

		// Registration email should have been sent.
		msg := <-inbox
		So(msg.Template, ShouldEqual, proto.WelcomeEmail)
		params, ok := msg.Data.(*proto.WelcomeEmailParams)
		So(ok, ShouldBeTrue)

		// The verification token should be valid.
		url := fmt.Sprintf("%s/prefs/verify?email=registration@euphoria.io&token=%s",
			s.server.URL, params.VerificationToken)
		resp, err := http.Get(url)
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, 200)

		// Personal identity should now be verified.
		ctx := scope.New()
		account, err := s.backend.AccountManager().Resolve(ctx, "email", "registration@euphoria.io")
		So(err, ShouldBeNil)
		verified := false
		for _, pid := range account.PersonalIdentities() {
			if pid.Verified() {
				verified = true
				break
			}
		}
		So(verified, ShouldBeTrue)
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

		// Create staff account.
		nonce := fmt.Sprintf("%s", time.Now())
		logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
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
		/* TODO: require unlock-staff-capability
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
	Convey("Grant access to passcode", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms

		// Create manager account and room.
		nonce := fmt.Sprintf("%s", time.Now())
		logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
		So(err, ShouldBeNil)
		_, err = b.CreateRoom(ctx, kms, true, "passcodegrants", logan)
		So(err, ShouldBeNil)

		// Connect and log into manager account in a throwaway room.
		loganConn := s.Connect("passcodegrantsstage")
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)
		loganConn.send("1", "login", `{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		loganConn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		loganConn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		loganConn.Close()

		// Reconnect manager to private room and grant access to passcode.
		loganConn = s.Connect("passcodegrants", loganConn.cookies...)
		defer loganConn.Close()
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)
		loganConn.send("1", "grant-access", `{"passcode":"hunter2"}`)
		loganConn.expect("1", "grant-access-reply", `{}`)

		// Authenticate with passcode.
		conn := s.Connect("passcodegrants")
		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		conn.send("1", "auth", `{"type":"passcode","passcode":"hunter2"}`)
		conn.expect("1", "auth-reply", `{"success":true}`)
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.Close()

		// Revoke passcode access.
		loganConn.expect(
			"", "join-event", `{"id":"*", "name":"", "server_id":"*","server_era":"*","session_id":"*"}`)
		loganConn.expect(
			"", "part-event", `{"id":"*", "name":"", "server_id":"*","server_era":"*","session_id":"*"}`)
		loganConn.send("2", "revoke-access", `{"passcode":"hunter2"}`)
		loganConn.expect("2", "revoke-access-reply", `{}`)
		conn = s.Connect("passcodegrants")
		defer conn.Close()
		conn.expectPing()
		conn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		conn.send("1", "auth", `{"type":"passcode","passcode":"hunter2"}`)
		conn.expect("1", "auth-reply", `{"success":false,"reason":"passcode incorrect"}`)
	})

	Convey("Grant access to account", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms

		// Create manager account and room.
		nonce := fmt.Sprintf("%s", time.Now())
		logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
		So(err, ShouldBeNil)
		_, err = b.CreateRoom(ctx, kms, true, "grants", logan)
		So(err, ShouldBeNil)

		// Create access account (without access yet).
		max, _, err := s.Account(ctx, kms, "email", "max"+nonce, "maxpass")
		So(err, ShouldBeNil)

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
				fmt.Sprintf(
					`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
					loganConn.sessionID, loganConn.userID)},
			nil)

		// Synchronize and revoke access.
		maxConn.Close()
		loganConn.expect("", "join-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			maxConn.sessionID, maxConn.id())
		loganConn.expect("", "part-event",
			`{"session_id":"%s","id":"%s","name":"","server_id":"test1","server_era":"era1"}`,
			maxConn.sessionID, maxConn.id())
		loganConn.send("2", "revoke-access", `{"account_id":"%s"}`, max.ID())
		loganConn.expect("2", "revoke-access-reply", `{}`)
		loganConn.Close()

		maxConn = s.Connect("grants", maxConn.cookies...)
		maxConn.expectPing()
		maxConn.expect("", "bounce-event", `{"reason":"authentication required"}`)
		maxConn.Close()
	})

	Convey("Grant manager and revoke access by staff", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms

		// Create staff account and room.
		_, err := b.CreateRoom(ctx, kms, false, "staffmanagergrants")
		So(err, ShouldBeNil)

		nonce := fmt.Sprintf("+%s", time.Now())
		logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
		So(err, ShouldBeNil)
		So(b.AccountManager().GrantStaff(ctx, logan.ID(), s.kms.KMSCredential()), ShouldBeNil)

		// Connect to room and make it private.

		// Connect to room, log in, reconnect, lock room, and grant management to self.
		loganConn := s.Connect("staffmanagergrants")
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)
		loganConn.send("1", "login", `{"namespace":"email","id":"logan%s","password":"loganpass"}`, nonce)
		loganConn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, logan.ID())
		loganConn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		loganConn.Close()

		loganConn = s.Connect("staffmanagergrants", loganConn.cookies...)
		loganConn.expectPing()
		loganConn.expectSnapshot(s.backend.Version(), nil, nil)
		loganConn.send("1", "staff-lock-room", `{}`)
		loganConn.expect("1", "staff-lock-room-reply", `{}`)
		loganConn.Close()

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

		// Revoke access to self.
		loganConn.send("3", "staff-revoke-access", `{"account_id":"%s"}`, logan.ID())
		loganConn.expect("3", "staff-revoke-access-reply", `{}`)
		loganConn.Close()

		loganConn = s.Connect("staffmanagergrants", loganConn.cookies...)
		loganConn.expectPing()
		loganConn.expect("", "bounce-event", `{"reason":"authentication required"}`)
	})

	Convey("Grant manager to account", func() {
		b := s.backend
		ctx := scope.New()
		kms := s.app.kms

		// Create manager account and room.
		nonce := fmt.Sprintf("+%s", time.Now())
		logan, _, err := s.Account(ctx, kms, "email", "logan"+nonce, "loganpass")
		So(err, ShouldBeNil)
		room, err := b.CreateRoom(ctx, kms, true, "managergrants", logan)
		So(err, ShouldBeNil)

		// Create access account (without access yet).
		max, _, err := s.Account(ctx, kms, "email", "max"+nonce, "maxpass")
		So(err, ShouldBeNil)

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

func testRoomNotFound(s *serverUnderTest) {
	s.app.AllowRoomCreation(false)
	url := strings.Replace(s.server.URL, "http:", "ws:", 1) + "/room/roomnotfound/ws"
	_, resp, err := websocket.DefaultDialer.Dial(url, nil)
	So(err, ShouldNotBeNil)
	So(resp, ShouldNotBeNil)
	So(resp.StatusCode, ShouldEqual, http.StatusNotFound)
}

func testKeepAlive(s *serverUnderTest) {
	Convey("Ping event, reply, and timeout", func() {
		save := KeepAlive
		defer func() { KeepAlive = save }()
		KeepAlive = 10 * time.Millisecond

		conn := s.Connect("ping")
		event := conn.expectPing()
		conn.expectSnapshot(s.backend.Version(), nil, nil)
		conn.send("", "ping-reply", `{"time":%d}`, time.Time(event.UnixTime).Unix())
		time.Sleep(KeepAlive * MaxKeepAliveMisses)
		for i := 0; i < MaxKeepAliveMisses+1; i++ {
			conn.expectPing()
		}
		_, _, err := conn.Conn.ReadMessage()
		So(err, ShouldNotBeNil)
	})
}

func testBans(s *serverUnderTest) {
	Convey("Ban by agent", func() {
		ctx := scope.New()
		kms := s.app.kms

		// Create manager and log in (via staging room).
		nonce := fmt.Sprintf("%s", time.Now())
		_, manager, _, err := s.RoomAndManager(ctx, kms, false, "bans", "email", nonce, "password")
		So(err, ShouldBeNil)

		mconn := s.Connect("bansstage")
		mconn.expectPing()
		mconn.expectSnapshot(s.backend.Version(), nil, nil)
		mconn.send("1", "login", `{"namespace":"email","id":"%s","password":"password"}`, nonce)
		mconn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, manager.ID())
		mconn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		mconn.Close()

		// Connect manager to managed room and wait for victim.
		mconn = s.Connect("bans", mconn.cookies...)
		mconn.expectPing()
		mconn.expectSnapshot(s.backend.Version(), nil, nil)

		// Connect victim.
		vconn := s.Connect("bans")
		vconn.expectPing()
		vconn.expectSnapshot(s.backend.Version(), nil, nil)

		// Wait for manager to see join, acquire agentID.
		capture := mconn.expect("", "join-event",
			`{"session_id":"*","id":"*","name":"","server_id":"test1","server_era":"era1"}`)
		agentID := capture["id"]
		So(agentID, ShouldNotBeNil)

		// Ban agent.
		mconn.send("1", "ban", `{"id":"%s"}`, agentID)
		mconn.expect("1", "ban-reply", `{}`)

		vconn.expect("", "disconnect-event", `{"reason":"banned"}`)
		vconn.Close()

		mconn.expect("", "part-event",
			`{"session_id":"*","id":"%s","name":"","server_id":"test1","server_era":"era1"}`, agentID)

		// Agent should be unable to reconnect.
		vconn = s.Connect("bans", vconn.cookies...)
		vconn.expectPing()
		_, _, err = vconn.Conn.ReadMessage()
		So(err, ShouldNotBeNil)
		vconn.Close()

		// Unban agent, who should be able to reconnect.
		mconn.send("2", "unban", `{"id":"%s"}`, agentID)
		mconn.expect("2", "unban-reply", `{}`)
		mconn.Close()

		vconn = s.Connect("bans", vconn.cookies...)
		vconn.expectPing()
		vconn.expectSnapshot(s.backend.Version(), nil, nil)
	})

	Convey("Ban by account", func() {
		ctx := scope.New()
		kms := s.app.kms

		// Create manager and log in (via staging room).
		nonce := fmt.Sprintf("%s", time.Now())
		_, manager, _, err := s.RoomAndManager(ctx, kms, false, "acctbans", "email", nonce, "password")
		So(err, ShouldBeNil)

		mconn := s.Connect("acctbansstage1")
		mconn.expectPing()
		mconn.expectSnapshot(s.backend.Version(), nil, nil)
		mconn.send("1", "login", `{"namespace":"email","id":"%s","password":"password"}`, nonce)
		mconn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, manager.ID())
		mconn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		mconn.Close()

		// Connect manager to managed room and wait for victim.
		mconn = s.Connect("acctbans", mconn.cookies...)
		mconn.expectPing()
		mconn.expectSnapshot(s.backend.Version(), nil, nil)

		// Create victim account and log in (via staging room).
		victim, _, err := s.Account(ctx, kms, "email", "victim"+nonce, "password")
		So(err, ShouldBeNil)

		vconn := s.Connect("acctbansstage2")
		vconn.expectPing()
		vconn.expectSnapshot(s.backend.Version(), nil, nil)
		vconn.send("1", "login", `{"namespace":"email","id":"victim%s","password":"password"}`, nonce)
		vconn.expect("1", "login-reply", `{"success":true,"account_id":"%s"}`, victim.ID())
		vconn.expect("", "disconnect-event", `{"reason":"authentication changed"}`)
		vconn.Close()

		// Connect victim.
		vconn = s.Connect("acctbans", vconn.cookies...)
		vconn.expectPing()
		vconn.expectSnapshot(s.backend.Version(), nil, nil)

		// Wait for manager to see join, acquire agentID.
		mconn.expect("", "join-event",
			`{"session_id":"*","id":"account:%s","name":"","server_id":"test1","server_era":"era1"}`,
			victim.ID())

		// Ban account.
		mconn.send("1", "ban", `{"id":"account:%s"}`, victim.ID())
		mconn.expect("1", "ban-reply", `{}`)

		vconn.expect("", "disconnect-event", `{"reason":"banned"}`)
		vconn.Close()

		mconn.expect("", "part-event",
			`{"session_id":"*","id":"account:%s","name":"","server_id":"test1","server_era":"era1"}`,
			victim.ID())

		// Account should be unable to reconnect.
		vconn = s.Connect("acctbans", vconn.cookies...)
		vconn.expectPing()
		_, _, err = vconn.Conn.ReadMessage()
		So(err, ShouldNotBeNil)
		vconn.Close()

		// Unban account, who should be able to reconnect.
		mconn.send("2", "unban", `{"id":"account:%s"}`, victim.ID())
		mconn.expect("2", "unban-reply", `{}`)
		mconn.Close()

		vconn = s.Connect("acctbans", vconn.cookies...)
		vconn.expectPing()
		vconn.expectSnapshot(s.backend.Version(), nil, nil)
	})
}
