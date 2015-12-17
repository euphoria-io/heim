package backend

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"

	"github.com/gorilla/websocket"
	"github.com/juju/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MaxKeepAliveMisses      = 3
	MaxAuthFailures         = 5
	MaxConsecutiveThrottled = 10
)

var (
	KeepAlive     = 20 * time.Second
	FastKeepAlive = 2 * time.Second

	ErrUnresponsive = fmt.Errorf("connection unresponsive")
	ErrReplaced     = fmt.Errorf("connection replaced")
	ErrFlooding     = fmt.Errorf("connection flooding")

	sessionIDCounter uint64

	sessionCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "sessions",
		Subsystem: "backend",
		Help:      "Cumulative number of sessions served by this backend",
	}, []string{"room"})

	accountRegistrations = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "registrations",
		Subsystem: "account",
		Help:      "Counter of successful account registrations",
	})

	authAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "attempts",
		Subsystem: "auth",
		Help:      "Counter of authentication attempts",
	}, []string{"room"})

	authFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "failures",
		Subsystem: "auth",
		Help:      "Counter of authentication failures",
	}, []string{"room"})

	authTerminations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "terminations",
		Subsystem: "auth",
		Help:      "Counter of sessions ignored due to excessive auth failures",
	}, []string{"room"})
)

func init() {
	prometheus.MustRegister(sessionCount)
	prometheus.MustRegister(accountRegistrations)
	prometheus.MustRegister(authAttempts)
	prometheus.MustRegister(authFailures)
	prometheus.MustRegister(authTerminations)

	if err := binary.Read(rand.Reader, binary.BigEndian, &sessionIDCounter); err != nil {
		panic(fmt.Sprintf("random session id counter error: %s", err))
	}
}

type response struct {
	packet interface{}
	err    error
	cost   int64
}

type cmdState func(*proto.Packet) *response

type session struct {
	id         string
	ctx        scope.Context
	server     *Server
	conn       *websocket.Conn
	clientAddr string
	identity   *memIdentity
	serverID   string
	serverEra  string
	roomName   string
	room       proto.Room
	backend    proto.Backend
	kms        security.KMS
	heim       *proto.Heim

	state    cmdState
	client   *proto.Client
	agentKey *security.ManagedKey
	staffKMS security.KMS
	keyID    string
	onClose  func()

	incoming     chan *proto.Packet
	outgoing     chan *proto.Packet
	floodLimiter *ratelimit.Bucket

	authFailCount int

	m                   sync.Mutex
	joined              bool
	banned              bool
	maybeAbandoned      bool
	outstandingPings    int
	expectedPingReply   int64
	fastKeepAliveCancel func()
}

func newSession(
	ctx scope.Context, server *Server, conn *websocket.Conn, clientAddr string,
	roomName string, room proto.Room, client *proto.Client, agentKey *security.ManagedKey) *session {

	nextID := atomic.AddUint64(&sessionIDCounter, 1)
	sessionCount.WithLabelValues(roomName).Set(float64(nextID))
	sessionID := fmt.Sprintf("%x-%08x", client.Agent.IDString(), nextID)
	ctx = logging.LoggingContext(ctx, os.Stdout, fmt.Sprintf("[%s] ", sessionID))

	session := &session{
		id:         sessionID,
		ctx:        ctx,
		server:     server,
		conn:       conn,
		clientAddr: clientAddr,
		identity:   newMemIdentity(client.UserID(), server.ID, server.Era),
		client:     client,
		agentKey:   agentKey,
		serverID:   server.ID,
		serverEra:  server.Era,
		roomName:   roomName,
		room:       room,
		backend:    server.b,
		kms:        server.kms,
		heim:       server.heim,

		incoming:     make(chan *proto.Packet),
		outgoing:     make(chan *proto.Packet, 100),
		floodLimiter: ratelimit.NewBucketWithQuantum(time.Second, 50, 10),
	}

	return session
}

func (s *session) Close() {
	logger := logging.Logger(s.ctx)
	logger.Printf("closing session")
	s.ctx.Cancel()
}

func (s *session) ID() string               { return s.id }
func (s *session) AgentID() string          { return s.client.Agent.IDString() }
func (s *session) ServerID() string         { return s.serverID }
func (s *session) ServerEra() string        { return s.serverEra }
func (s *session) Identity() proto.Identity { return s.identity }
func (s *session) SetName(name string)      { s.identity.name = name }

func (s *session) View(level proto.PrivilegeLevel) *proto.SessionView {
	view := &proto.SessionView{
		IdentityView: s.identity.View(),
		SessionID:    s.id,
		IsStaff:      s.client.Account != nil && s.client.Account.IsStaff(),
		IsManager:    s.client.Authorization.ManagerKeyPair != nil,
	}

	switch level {
	case proto.Staff:
		view.ClientAddress = s.clientAddr
	}

	return view
}

func (s *session) writeMessage(messageType int, data []byte) error {
	if err := s.conn.SetWriteDeadline(time.Now().Add(MaxKeepAliveMisses * KeepAlive)); err != nil {
		return err
	}
	if err := s.conn.WriteMessage(messageType, data); err != nil {
		return err
	}
	if err := s.conn.SetWriteDeadline(time.Time{}); err != nil {
		return err
	}
	return nil
}

func (s *session) Send(ctx scope.Context, cmdType proto.PacketType, payload interface{}) error {
	// Special case: presence events have privileged info that may need to be stripped from them
	if pEvent, ok := payload.(*proto.PresenceEvent); ok {
		if s.privilegeLevel() != proto.Staff {
			pEvent.ClientAddress = ""
		}
	}

	var err error
	payload, err = proto.DecryptPayload(payload, &s.client.Authorization)
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	cmd := &proto.Packet{
		Type: cmdType,
		Data: encoded,
	}

	// Add to outgoing channel. If channel is full, defer to goroutine so as not to block
	// the caller (this may result in deliveries coming out of order).
	select {
	case <-ctx.Done():
		// Session is closed, return error.
		return ctx.Err()
	case s.outgoing <- cmd:
		// Packet delivered to queue.
	default:
		// Queue is full.
		logging.Logger(s.ctx).Printf("outgoing channel full, ordering cannot be guaranteed")
		go func() { s.outgoing <- cmd }()
	}

	return nil
}

func (s *session) serve() error {
	defer func() {
		s.finishFastKeepAlive()
		if s.onClose != nil {
			s.onClose()
		}
	}()

	logger := logging.Logger(s.ctx)
	logger.Printf("client connected")

	key, err := s.room.MessageKey(s.ctx)
	if err != nil {
		return err
	}

	accountHasAccess := false
	if key != nil {
		_, accountHasAccess = s.client.Authorization.MessageKeys[key.KeyID()]
	}

	if err := s.sendHello(key != nil, accountHasAccess); err != nil {
		return err
	}

	if err := s.sendPing(); err != nil {
		return err
	}

	// Verify agent age against site and room settings.
	allowed := true
	agentAge := time.Now().Sub(s.client.Agent.Created)
	if s.client.Account == nil && !s.client.Agent.Blessed && (agentAge < s.server.roomEntryMinAgentAge || agentAge < s.room.MinAgentAge()) {
		allowed = false
		s.sendBounce("room not open")
		s.state = s.ignoreState
	}

	// TODO: have user explicitly unlock staff KMS
	if s.client.Account != nil && s.client.Account.IsStaff() {
		kms, err := s.client.Account.UnlockStaffKMS(s.client.Authorization.ClientKey)
		if err != nil {
			logger.Printf("staff account %s unable to unlock staff capability: %s",
				s.client.Account.ID(), err)
		} else {
			s.staffKMS = kms
		}
	}

	// TODO: check room auth
	switch key {
	case nil:
		if allowed {
			if err := s.join(); err != nil {
				// TODO: send an error packet
				return err
			}
			s.state = s.joinedState
		}
	default:
		if _, ok := s.client.Authorization.MessageKeys[key.KeyID()]; ok {
			s.client.Authorization.CurrentMessageKeyID = key.KeyID()
			s.keyID = key.KeyID()
			if err := s.join(); err != nil {
				// TODO: send an error packet
				return err
			}
			s.state = s.joinedState
		} else {
			s.sendBounce("authentication required")
			s.state = s.unauthedState
		}
	}

	go s.readMessages()

	keepalive := time.NewTicker(KeepAlive)
	defer keepalive.Stop()

	consecutiveThrottled := 0

	for {
		select {
		case <-s.ctx.Done():
			// connection forced to close
			return s.ctx.Err()

		case <-keepalive.C:
			if s.outstandingPings > MaxKeepAliveMisses {
				logger.Printf("connection timed out")
				s.sendDisconnect("timed out")
				return ErrUnresponsive
			}

			if err := s.sendPing(); err != nil {
				return err
			}
		case cmd := <-s.incoming:
			reply := s.state(cmd)

			flooding := false
			shouldKickForFlooding := false
			if reply.cost > 0 {
				taken := s.floodLimiter.TakeAvailable(reply.cost)
				if taken < reply.cost {
					flooding = true
					if consecutiveThrottled++; consecutiveThrottled > MaxConsecutiveThrottled {
						shouldKickForFlooding = true
					}
					s.floodLimiter.Wait(reply.cost - taken)
				} else {
					consecutiveThrottled = 0
				}
			}

			if reply.err != nil {
				logger.Printf("error: %v: %s", s.state, reply.err)
				reply.packet = reply.err
			}

			if reply.packet == nil {
				if shouldKickForFlooding {
					return ErrFlooding
				}
				continue
			}

			// Write the response back over the socket.
			resp, err := proto.MakeResponse(cmd.ID, cmd.Type, reply.packet, flooding)
			if err != nil {
				logger.Printf("error: Response: %s", err)
				return err
			}

			data, err := resp.Encode()
			if err != nil {
				logger.Printf("error: Response encode: %s", err)
				return err
			}

			if err := s.writeMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return err
			}

			if shouldKickForFlooding {
				return ErrFlooding
			}

			// Some responses trigger bounces.
			switch msg := reply.packet.(type) {
			case *proto.LogoutReply:
				s.sendDisconnect("authentication changed")
			case *proto.RegisterAccountReply:
				if msg.Success {
					s.sendDisconnect("authentication changed")
				}
			}
		case cmd := <-s.outgoing:
			data, err := cmd.Encode()
			if err != nil {
				logger.Printf("error: push message encode: %s", err)
				return err
			}

			if err := s.writeMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return err
			}

			if cmd.Type == proto.DisconnectEventType {
				return nil
			}

		}
	}
	return nil
}

func (s *session) readMessages() {
	logger := logging.Logger(s.ctx)
	defer s.Close()

	for s.ctx.Err() == nil {
		messageType, data, err := s.conn.ReadMessage()
		if err != nil {
			if err == io.EOF {
				logger.Printf("client disconnected")
				return
			}
			logger.Printf("error: read message: %s", err)
			return
		}

		switch messageType {
		case websocket.TextMessage:
			cmd, err := proto.ParseRequest(data)
			if err != nil {
				logger.Printf("error: ParseRequest: %s", err)
				return
			}
			s.incoming <- cmd
		default:
			logger.Printf("error: unsupported message type: %v", messageType)
			return
		}
	}
}

func (s *session) sendSnapshot(msgs []proto.Message, listing proto.Listing) error {
	for i, msg := range msgs {
		if msg.EncryptionKeyID != "" {
			dmsg, err := proto.DecryptMessage(msg, s.client.Authorization.MessageKeys)
			if err != nil {
				continue
			}
			msgs[i] = dmsg
		}
	}

	snapshot := &proto.SnapshotEvent{
		Identity:  s.Identity().ID(),
		SessionID: s.ID(),
		Version:   s.room.Version(),
		Listing:   listing,
		Log:       msgs,
	}

	event, err := proto.MakeEvent(snapshot)
	if err != nil {
		return err
	}
	s.outgoing <- event
	return nil
}

func (s *session) sendBounce(reason string) error {
	bounce := &proto.BounceEvent{
		Reason: reason,
		// TODO: fill in AuthOptions
	}
	event, err := proto.MakeEvent(bounce)
	if err != nil {
		return err
	}
	s.outgoing <- event
	return nil
}

func (s *session) sendDisconnect(reason string) error {
	event, err := proto.MakeEvent(&proto.DisconnectEvent{Reason: reason})
	if err != nil {
		return err
	}
	s.outgoing <- event
	return nil
}

func (s *session) privilegeLevel() proto.PrivilegeLevel {
	switch {
	case s.client.Account != nil && s.client.Account.IsStaff():
		return proto.Staff
	case s.client.Authorization.ManagerKeyPair != nil:
		return proto.Host
	default:
		return proto.General
	}
}

func (s *session) join() error {
	msgs, err := s.room.Latest(s.ctx, 100, 0)
	if err != nil {
		return err
	}

	listing, err := s.room.Listing(s.ctx, s.privilegeLevel())
	if err != nil {
		return err
	}

	if err := s.room.Join(s.ctx, s); err != nil {
		logging.Logger(s.ctx).Printf("join failed: %s", err)
		return err
	}

	s.onClose = func() {
		// Use a fork of the server's root context, because the session's context
		// might be closed.
		ctx := s.server.rootCtx.Fork()
		if err := s.room.Part(ctx, s); err != nil {
			logging.Logger(ctx).Printf("room part failed: %s", err)
			return
		}
	}

	if err := s.sendSnapshot(msgs, listing); err != nil {
		logging.Logger(s.ctx).Printf("snapshot failed: %s", err)
		return err
	}

	s.joined = true
	return nil
}

func (s *session) sendHello(roomIsPrivate, accountHasAccess bool) error {
	logger := logging.Logger(s.ctx)
	event := &proto.HelloEvent{
		SessionView:      s.View(s.privilegeLevel()),
		AccountHasAccess: accountHasAccess,
		RoomIsPrivate:    roomIsPrivate,
		Version:          s.room.Version(),
	}
	if s.client.Account != nil {
		event.AccountView = &proto.PersonalAccountView{
			AccountView: *s.client.Account.View(s.roomName),
		}
		event.AccountView.Email, event.AccountEmailVerified = s.client.Account.Email()
	}
	event.ID = event.SessionView.ID
	cmd, err := proto.MakeEvent(event)
	if err != nil {
		logger.Printf("error: hello event: %s", err)
		return err
	}
	data, err := cmd.Encode()
	if err != nil {
		logger.Printf("error: hello event encode: %s", err)
		return err
	}

	if err := s.writeMessage(websocket.TextMessage, data); err != nil {
		logger.Printf("error: write hello event: %s", err)
		return err
	}

	return nil
}

func (s *session) sendPing() error {
	logger := logging.Logger(s.ctx)
	now := time.Now()
	cmd, err := proto.MakeEvent(&proto.PingEvent{
		UnixTime:     proto.Time(now),
		NextUnixTime: proto.Time(now.Add(3 * KeepAlive / 2)),
	})
	if err != nil {
		logger.Printf("error: ping event: %s", err)
		return err
	}
	data, err := cmd.Encode()
	if err != nil {
		logger.Printf("error: ping event encode: %s", err)
		return err
	}

	if err := s.writeMessage(websocket.TextMessage, data); err != nil {
		logger.Printf("error: write ping event: %s", err)
		return err
	}

	s.expectedPingReply = now.Unix()
	s.outstandingPings++
	return nil
}

func (s *session) CheckAbandoned() error {
	s.m.Lock()
	defer s.m.Unlock()

	logger := logging.Logger(s.ctx)

	if s.maybeAbandoned {
		// already in fast-keepalive state
		return nil
	}
	s.maybeAbandoned = true

	child := s.ctx.Fork()
	s.fastKeepAliveCancel = child.Cancel

	go func() {
		logger.Printf("starting fast-keepalive timer")
		timer := time.After(FastKeepAlive)
		select {
		case <-child.Done():
			logger.Printf("aliased session still alive")
		case <-timer:
			logger.Printf("connection replaced")
			s.ctx.Terminate(ErrReplaced)
		}
	}()

	return s.sendPing()
}

func (s *session) finishFastKeepAlive() {
	s.m.Lock()
	defer s.m.Unlock()

	if s.maybeAbandoned {
		s.maybeAbandoned = false
		s.fastKeepAliveCancel()
		s.fastKeepAliveCancel = nil
	}
}
