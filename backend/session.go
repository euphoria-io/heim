package backend

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/snowflake"
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
	prometheus.MustRegister(authAttempts)
	prometheus.MustRegister(authFailures)
	prometheus.MustRegister(authTerminations)
}

type response struct {
	packet interface{}
	err    error
	cost   int64
}

type cmdState func(*proto.Packet) *response

type session struct {
	id        string
	ctx       scope.Context
	conn      *websocket.Conn
	identity  *memIdentity
	serverID  string
	serverEra string
	roomName  string
	room      proto.Room

	state   cmdState
	auth    map[string]*proto.Authentication
	keyID   string
	onClose func()

	incoming     chan *proto.Packet
	outgoing     chan *proto.Packet
	floodLimiter *ratelimit.Bucket

	authFailCount int

	m                   sync.Mutex
	banned              bool
	maybeAbandoned      bool
	outstandingPings    int
	expectedPingReply   int64
	fastKeepAliveCancel func()
}

func newSession(
	ctx scope.Context, conn *websocket.Conn, serverID, serverEra, roomName string, room proto.Room,
	agentID []byte) *session {

	nextID := atomic.AddUint64(&sessionIDCounter, 1)
	sessionCount.WithLabelValues(roomName).Set(float64(nextID))
	sessionID := fmt.Sprintf("%x-%08x", agentID, nextID)
	ctx = LoggingContext(ctx, fmt.Sprintf("[%s] ", sessionID))

	session := &session{
		id:        sessionID,
		ctx:       ctx,
		conn:      conn,
		identity:  newMemIdentity(fmt.Sprintf("agent:%08x", agentID), serverID, serverEra),
		serverID:  serverID,
		serverEra: serverEra,
		roomName:  roomName,
		room:      room,

		incoming:     make(chan *proto.Packet),
		outgoing:     make(chan *proto.Packet, 100),
		floodLimiter: ratelimit.NewBucketWithQuantum(time.Second, 50, 10),
	}

	return session
}

func (s *session) Close() {
	logger := Logger(s.ctx)
	logger.Printf("closing session")
	s.ctx.Cancel()
}

func (s *session) ID() string               { return s.id }
func (s *session) ServerID() string         { return s.serverID }
func (s *session) ServerEra() string        { return s.serverEra }
func (s *session) Identity() proto.Identity { return s.identity }
func (s *session) SetName(name string)      { s.identity.name = name }

func (s *session) View() *proto.SessionView {
	return &proto.SessionView{
		IdentityView: s.identity.View(),
		SessionID:    s.id,
	}
}

func (s *session) Send(ctx scope.Context, cmdType proto.PacketType, payload interface{}) error {
	var err error
	payload, err = proto.DecryptPayload(payload, s.auth)
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

	go func() {
		s.outgoing <- cmd
	}()

	return nil
}

func (s *session) serve() error {
	defer func() {
		s.finishFastKeepalive()
		if s.onClose != nil {
			s.onClose()
		}
	}()

	logger := Logger(s.ctx)
	logger.Printf("client connected")

	if err := s.sendPing(); err != nil {
		return err
	}

	// TODO: check room auth
	key, err := s.room.MessageKey(s.ctx)
	if err != nil {
		return err
	}
	switch key {
	case nil:
		if err := s.join(); err != nil {
			// TODO: send an error packet
			return err
		}
		s.state = s.handleCommand
	default:
		s.sendBounce()
		s.state = s.handleAuth
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

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return err
			}

			if shouldKickForFlooding {
				return ErrFlooding
			}
		case cmd := <-s.outgoing:
			data, err := cmd.Encode()
			if err != nil {
				logger.Printf("error: push message encode: %s", err)
				return err
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return err
			}
		}
	}
	return nil
}

func (s *session) readMessages() {
	logger := Logger(s.ctx)
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

func (s *session) ignore(cmd *proto.Packet) *response {
	switch cmd.Type {
	case proto.PingType, proto.PingReplyType:
		return s.handleCommand(cmd)
	default:
		return &response{}
	}
}

func (s *session) handleAuth(cmd *proto.Packet) *response {
	payload, err := cmd.Payload()
	if err != nil {
		return &response{err: fmt.Errorf("payload: %s", err)}
	}

	switch msg := payload.(type) {
	case *proto.AuthCommand:
		if s.authFailCount > 0 {
			buf := []byte{0}
			if _, err := rand.Read(buf); err != nil {
				return &response{err: err}
			}
			jitter := 4 * time.Duration(int(buf[0])-128) * time.Millisecond
			time.Sleep(2*time.Second + jitter)
		}

		authAttempts.WithLabelValues(s.roomName).Inc()
		auth, err := proto.Authenticate(s.ctx, s.room, msg)
		if err != nil {
			return &response{err: err}
		}
		if auth.FailureReason != "" {
			authFailures.WithLabelValues(s.roomName).Inc()
			s.authFailCount++
			if s.authFailCount >= MaxAuthFailures {
				Logger(s.ctx).Printf(
					"max authentication failures on room %s by %s", s.roomName, s.Identity().ID())
				authTerminations.WithLabelValues(s.roomName).Inc()
				s.state = s.ignore
			}
			return &response{packet: &proto.AuthReply{Reason: auth.FailureReason}}
		}

		// TODO: support holding multiple keys
		s.auth = map[string]*proto.Authentication{auth.KeyID: auth}
		s.keyID = auth.KeyID
		s.state = s.handleCommand
		if err := s.join(); err != nil {
			s.keyID = ""
			s.state = s.handleAuth
			return &response{err: err}
		}
		return &response{packet: &proto.AuthReply{Success: true}}
	case *proto.PingCommand, *proto.PingReply:
		return s.handleCommand(cmd)
	default:
		return &response{err: fmt.Errorf("access denied, please authenticate")}
	}
}

func (s *session) handleCommand(cmd *proto.Packet) *response {
	payload, err := cmd.Payload()
	if err != nil {
		return &response{err: fmt.Errorf("payload: %s", err)}
	}

	switch msg := payload.(type) {
	case *proto.SendCommand:
		return s.handleSendCommand(msg)
	case *proto.LogCommand:
		msgs, err := s.room.Latest(s.ctx, msg.N, msg.Before)
		if err != nil {
			return &response{err: err}
		}
		packet, err := proto.DecryptPayload(proto.LogReply{Log: msgs, Before: msg.Before}, s.auth)
		return &response{
			packet: packet,
			err:    err,
			cost:   1,
		}
	case *proto.NickCommand:
		nick, err := proto.NormalizeNick(msg.Name)
		if err != nil {
			return &response{err: err}
		}
		formerName := s.identity.Name()
		s.identity.name = nick
		event, err := s.room.RenameUser(s.ctx, s, formerName)
		if err != nil {
			return &response{err: err}
		}
		return &response{
			packet: proto.NickReply(*event),
			cost:   1,
		}
	case *proto.PingCommand:
		return &response{packet: &proto.PingReply{UnixTime: msg.UnixTime}}
	case *proto.PingReply:
		s.finishFastKeepalive()
		if msg.UnixTime == s.expectedPingReply {
			s.outstandingPings = 0
		} else if s.outstandingPings > 1 {
			s.outstandingPings--
		}
		return &response{}
	case *proto.WhoCommand:
		listing, err := s.room.Listing(s.ctx)
		if err != nil {
			return &response{err: err}
		}
		return &response{packet: &proto.WhoReply{Listing: listing}}
	default:
		return &response{err: fmt.Errorf("command type %T not implemented", payload)}
	}
}

func (s *session) sendSnapshot(msgs []proto.Message, listing proto.Listing) error {
	for i, msg := range msgs {
		if msg.EncryptionKeyID != "" {
			dmsg, err := proto.DecryptMessage(msg, s.auth)
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

func (s *session) sendBounce() error {
	bounce := &proto.BounceEvent{
		Reason: "authentication required",
		// TODO: fill in AuthOptions
	}
	event, err := proto.MakeEvent(bounce)
	if err != nil {
		return err
	}
	s.outgoing <- event
	return nil
}

func (s *session) join() error {
	msgs, err := s.room.Latest(s.ctx, 100, 0)
	if err != nil {
		return err
	}

	listing, err := s.room.Listing(s.ctx)
	if err != nil {
		return err
	}

	if err := s.room.Join(s.ctx, s); err != nil {
		Logger(s.ctx).Printf("join failed: %s", err)
		return err
	}

	s.onClose = func() {
		if err := s.room.Part(s.ctx, s); err != nil {
			// TODO: error handling
			return
		}
	}

	if err := s.sendSnapshot(msgs, listing); err != nil {
		Logger(s.ctx).Printf("snapshot failed: %s", err)
		return err
	}

	return nil
}

func (s *session) handleSendCommand(cmd *proto.SendCommand) *response {
	if s.Identity().Name() == "" {
		return &response{err: fmt.Errorf("you must choose a name before you may begin chatting")}
	}

	msgID, err := snowflake.New()
	if err != nil {
		return &response{err: err}
	}

	isValidParent, err := s.room.IsValidParent(cmd.Parent)
	if err != nil {
		return &response{err: err}
	}
	if !isValidParent {
		return &response{err: proto.ErrInvalidParent}
	}
	msg := proto.Message{
		ID:      msgID,
		Content: cmd.Content,
		Parent:  cmd.Parent,
		Sender:  s.View(),
	}

	if s.keyID != "" {
		if err := proto.EncryptMessage(&msg, s.keyID, s.auth[s.keyID].Key); err != nil {
			return &response{err: err}
		}
	}

	sent, err := s.room.Send(s.ctx, s, msg)
	if err != nil {
		return &response{err: err}
	}

	packet, err := proto.DecryptPayload(proto.SendReply(sent), s.auth)
	return &response{
		packet: packet,
		err:    err,
		cost:   10,
	}
}

func (s *session) sendPing() error {
	logger := Logger(s.ctx)
	now := time.Now()
	cmd, err := proto.MakeEvent(&proto.PingEvent{
		UnixTime:     now.Unix(),
		NextUnixTime: now.Add(3 * KeepAlive / 2).Unix(),
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

	if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
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

	logger := Logger(s.ctx)

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

func (s *session) finishFastKeepalive() {
	s.m.Lock()
	defer s.m.Unlock()

	if s.maybeAbandoned {
		s.maybeAbandoned = false
		s.fastKeepAliveCancel()
		s.fastKeepAliveCancel = nil
	}
}
