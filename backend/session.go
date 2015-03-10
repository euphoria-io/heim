package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"euphoria.io/scope"

	"heim/proto"
	"heim/proto/snowflake"

	"github.com/gorilla/websocket"
)

const MaxKeepAliveMisses = 3

var (
	KeepAlive     = 20 * time.Second
	FastKeepAlive = 2 * time.Second

	ErrUnresponsive = fmt.Errorf("connection unresponsive")
	ErrReplaced     = fmt.Errorf("connection replaced")

	sessionIDCounter uint64
)

type cmdState func(*proto.Packet) (interface{}, error)

type session struct {
	ctx       scope.Context
	conn      *websocket.Conn
	identity  *memIdentity
	serverID  string
	serverEra string
	room      proto.Room

	state   cmdState
	auth    map[string]*proto.Authentication
	keyID   string
	onClose func()

	incoming chan *proto.Packet
	outgoing chan *proto.Packet

	m                   sync.Mutex
	maybeAbandoned      bool
	outstandingPings    int
	expectedPingReply   int64
	fastKeepAliveCancel func()
}

func newSession(
	ctx scope.Context, conn *websocket.Conn, serverID, serverEra string, room proto.Room,
	agentID []byte) *session {

	nextID := atomic.AddUint64(&sessionIDCounter, 1)
	sessionID := fmt.Sprintf("%x-%08x", agentID, nextID)
	ctx = LoggingContext(ctx, fmt.Sprintf("[%s] ", sessionID))

	session := &session{
		ctx:       ctx,
		conn:      conn,
		identity:  newMemIdentity(sessionID, serverID, serverEra),
		serverID:  serverID,
		serverEra: serverEra,
		room:      room,

		incoming: make(chan *proto.Packet),
		outgoing: make(chan *proto.Packet, 100),
	}

	return session
}

func (s *session) Close() {
	logger := Logger(s.ctx)
	logger.Printf("closing session")
	s.ctx.Cancel()
}

func (s *session) ID() string               { return s.identity.ID() }
func (s *session) ServerID() string         { return s.serverID }
func (s *session) ServerEra() string        { return s.serverEra }
func (s *session) Identity() proto.Identity { return s.identity }
func (s *session) SetName(name string)      { s.identity.name = name }

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

	// TODO: check room auth
	key, err := s.room.MasterKey(s.ctx)
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
			reply, err := s.state(cmd)
			if err != nil {
				logger.Printf("error: %v: %s", s.state, err)
				reply = err
			}
			if reply == nil {
				continue
			}

			// Write the response back over the socket.
			resp, err := proto.MakeResponse(cmd.ID, cmd.Type, reply)
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

func (s *session) handleAuth(cmd *proto.Packet) (interface{}, error) {
	payload, err := cmd.Payload()
	if err != nil {
		return nil, fmt.Errorf("payload: %s", err)
	}

	switch msg := payload.(type) {
	case *proto.AuthCommand:
		auth, err := proto.Authenticate(s.ctx, s.room, msg)
		if err != nil {
			return nil, err
		}
		if auth.FailureReason != "" {
			return &proto.AuthReply{Reason: auth.FailureReason}, nil
		}
		// TODO: support holding multiple keys
		s.auth = map[string]*proto.Authentication{auth.KeyID: auth}
		s.keyID = auth.KeyID
		s.state = s.handleCommand
		if err := s.join(); err != nil {
			return nil, err
		}
		return &proto.AuthReply{Success: true}, nil
	default:
		return nil, fmt.Errorf("access denied, please authenticate")
	}
}

func (s *session) handleCommand(cmd *proto.Packet) (interface{}, error) {
	payload, err := cmd.Payload()
	if err != nil {
		return nil, fmt.Errorf("payload: %s", err)
	}

	switch msg := payload.(type) {
	case *proto.SendCommand:
		return s.handleSendCommand(msg)
	case *proto.LogCommand:
		msgs, err := s.room.Latest(s.ctx, msg.N, msg.Before)
		if err != nil {
			return nil, err
		}
		return proto.DecryptPayload(proto.LogReply{Log: msgs, Before: msg.Before}, s.auth)
	case *proto.NickCommand:
		nick, err := proto.NormalizeNick(msg.Name)
		if err != nil {
			return nil, err
		}
		formerName := s.identity.Name()
		s.identity.name = nick
		event, err := s.room.RenameUser(s.ctx, s, formerName)
		if err != nil {
			return nil, err
		}
		return proto.NickReply(*event), nil
	case *proto.PingCommand:
		return &proto.PingReply{UnixTime: msg.UnixTime}, nil
	case *proto.PingReply:
		s.finishFastKeepalive()
		if msg.UnixTime == s.expectedPingReply {
			s.outstandingPings = 0
		} else if s.outstandingPings > 1 {
			s.outstandingPings--
		}
		return nil, nil
	case *proto.WhoCommand:
		listing, err := s.room.Listing(s.ctx)
		if err != nil {
			return nil, err
		}
		return &proto.WhoReply{Listing: listing}, nil
	default:
		return nil, fmt.Errorf("command type %T not implemented", payload)
	}
}

func (s *session) sendSnapshot() error {
	msgs, err := s.room.Latest(s.ctx, 100, 0)
	if err != nil {
		return err
	}

	for i, msg := range msgs {
		if msg.EncryptionKeyID != "" {
			dmsg, err := proto.DecryptMessage(msg, s.auth)
			if err != nil {
				continue
			}
			msgs[i] = dmsg
		}
	}

	listing, err := s.room.Listing(s.ctx)
	if err != nil {
		return err
	}

	snapshot := &proto.SnapshotEvent{
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
	if err := s.sendSnapshot(); err != nil {
		Logger(s.ctx).Printf("snapshot failed: %s", err)
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
	return nil
}

func (s *session) handleSendCommand(cmd *proto.SendCommand) (interface{}, error) {
	msgID, err := snowflake.New()
	if err != nil {
		return nil, err
	}

	// TODO: verify parent
	msg := proto.Message{
		ID:      msgID,
		Content: cmd.Content,
		Parent:  cmd.Parent,
		Sender:  s.identity.View(),
	}

	if s.keyID != "" {
		if err := proto.EncryptMessage(&msg, s.keyID, s.auth[s.keyID].Key); err != nil {
			return nil, err
		}
	}

	sent, err := s.room.Send(s.ctx, s, msg)
	if err != nil {
		return nil, err
	}

	return proto.DecryptPayload(proto.SendReply(sent), s.auth)
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
