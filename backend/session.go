package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"heim/proto"
	"heim/proto/security"
	"heim/proto/snowflake"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

const MaxKeepAliveMisses = 3

var (
	KeepAlive       = 20 * time.Second
	ErrUnresponsive = fmt.Errorf("connection unresponsive")
)

type cmdState func(*proto.Packet) (interface{}, error)

type session struct {
	ctx      context.Context
	cancel   context.CancelFunc
	conn     *websocket.Conn
	identity *memIdentity
	serverID string
	room     proto.Room

	state      cmdState
	roomKey    *security.ManagedKey
	capability security.Capability
	onClose    func()

	incoming chan *proto.Packet
	outgoing chan *proto.Packet

	outstandingPings uint32
}

func newSession(ctx context.Context, conn *websocket.Conn, serverID string, room proto.Room) *session {
	id := conn.RemoteAddr().String()
	loggingCtx := LoggingContext(ctx, fmt.Sprintf("[%s] ", id))
	cancellableCtx, cancel := context.WithCancel(loggingCtx)

	session := &session{
		ctx:      cancellableCtx,
		cancel:   cancel,
		conn:     conn,
		identity: newMemIdentity(id),
		serverID: serverID,
		room:     room,

		incoming: make(chan *proto.Packet),
		outgoing: make(chan *proto.Packet, 100),
	}

	conn.SetPongHandler(session.handlePong)

	return session
}

func (s *session) Close() {
	logger := Logger(s.ctx)
	logger.Printf("closing session")
	s.cancel()
}

func (s *session) ID() string               { return s.conn.RemoteAddr().String() }
func (s *session) ServerID() string         { return s.serverID }
func (s *session) Identity() proto.Identity { return s.identity }
func (s *session) SetName(name string)      { s.identity.name = name }

func (s *session) Send(
	ctx context.Context, cmdType proto.PacketType, payload interface{}) error {

	if s.capability != nil {
		var err error
		payload, err = decryptPayload(payload, s.roomKey, s.capability)
		if err != nil {
			return err
		}
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

func (s *session) handlePong(string) error {
	atomic.StoreUint32(&s.outstandingPings, 0)
	return nil
}

func (s *session) serve() error {
	defer func() {
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

	keepalive := time.NewTimer(KeepAlive)
	defer keepalive.Stop()

	for {
		select {
		case <-s.ctx.Done():
			// connection forced to close
			return s.ctx.Err()

		case <-keepalive.C:
			// keepalive expired
			if pings := atomic.AddUint32(&s.outstandingPings, 1); pings > MaxKeepAliveMisses {
				logger.Printf("connection timed out")
				return ErrUnresponsive
			}

			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return err
			}

		case cmd := <-s.incoming:
			keepalive.Stop()

			reply, err := s.state(cmd)
			if err != nil {
				logger.Printf("error: %v: %s", s.state, err)
				reply = err
			}

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

			keepalive.Reset(KeepAlive)

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
		reply, roomKey, capability, err := Authenticate(s.ctx, s.room, msg)
		if err != nil {
			return nil, err
		}
		if reply.Success {
			s.roomKey = roomKey
			s.capability = capability
			s.state = s.handleCommand
			if err := s.join(); err != nil {
				return nil, err
			}
		}
		return reply, nil
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
		return &proto.LogReply{Log: msgs, Before: msg.Before}, nil
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

	for i := range msgs {
		if s.capability != nil {
			if err := decryptMessage(&msgs[i], s.roomKey, s.capability); err != nil {
				return err
			}
		}
	}

	listing, err := s.room.Listing(s.ctx)
	if err != nil {
		return err
	}

	snapshot := &proto.SnapshotEvent{
		Version: s.room.Version(),
		Listing: listing,
		Log:     msgs,
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

	if s.capability != nil {
		if err := encryptMessage(&msg, s.roomKey, s.capability); err != nil {
			return nil, err
		}
	}

	sent, err := s.room.Send(s.ctx, s, msg)
	if err != nil {
		return nil, err
	}
	return proto.SendReply(sent), nil
}
