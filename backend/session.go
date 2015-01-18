package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

const MaxKeepAliveMisses = 3

var (
	KeepAlive       = 20 * time.Second
	ErrUnresponsive = fmt.Errorf("connection unresponsive")
)

type Session interface {
	ID() string
	Identity() Identity
	SetName(name string)
	Send(context.Context, PacketType, interface{}) error
	Close()
}

type memSession struct {
	ctx      context.Context
	cancel   context.CancelFunc
	conn     *websocket.Conn
	identity *memIdentity
	room     Room

	incoming chan *Packet
	outgoing chan *Packet

	outstandingPings uint32
}

func newMemSession(ctx context.Context, conn *websocket.Conn, room Room) *memSession {
	id := conn.RemoteAddr().String()
	loggingCtx := LoggingContext(ctx, fmt.Sprintf("[%s] ", id))
	cancellableCtx, cancel := context.WithCancel(loggingCtx)

	session := &memSession{
		ctx:      cancellableCtx,
		cancel:   cancel,
		conn:     conn,
		identity: newMemIdentity(id),
		room:     room,

		incoming: make(chan *Packet),
		outgoing: make(chan *Packet, 100),
	}

	conn.SetPongHandler(session.handlePong)

	return session
}

func (s *memSession) Close() {
	logger := Logger(s.ctx)
	logger.Printf("closing session")
	s.cancel()
}

func (s *memSession) ID() string          { return s.conn.RemoteAddr().String() }
func (s *memSession) Identity() Identity  { return s.identity }
func (s *memSession) SetName(name string) { s.identity.name = name }

func (s *memSession) Send(ctx context.Context, cmdType PacketType, payload interface{}) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	cmd := &Packet{
		Type: cmdType,
		Data: encoded,
	}

	go func() {
		s.outgoing <- cmd
	}()

	return nil
}

func (s *memSession) handlePong(string) error {
	atomic.StoreUint32(&s.outstandingPings, 0)
	return nil
}

func (s *memSession) serve() error {
	go s.readMessages()

	logger := Logger(s.ctx)
	logger.Printf("client connected")

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

			reply, err := s.handleCommand(cmd)
			if err != nil {
				logger.Printf("error: handleCommand: %s", err)
				reply = ErrorReply{Error: err.Error()}
			}

			resp, err := MakeResponse(cmd.ID, cmd.Type, reply)
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

func (s *memSession) readMessages() {
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
			cmd, err := ParseRequest(data)
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

func (s *memSession) handleCommand(cmd *Packet) (interface{}, error) {
	payload, err := cmd.Payload()
	if err != nil {
		return nil, fmt.Errorf("payload: %s", err)
	}

	switch msg := payload.(type) {
	case *SendCommand:
		sent, err := s.room.Send(s.ctx, s, Message{Content: msg.Content, Parent: msg.Parent})
		if err != nil {
			return nil, err
		}
		return SendReply(sent), nil
	case *LogCommand:
		msgs, err := s.room.Latest(s.ctx, msg.N, msg.Before)
		if err != nil {
			return nil, err
		}
		return &LogReply{Log: msgs, Before: msg.Before}, nil
	case *NickCommand:
		formerName := s.identity.Name()
		s.identity.name = msg.Name
		event, err := s.room.RenameUser(s.ctx, s, formerName)
		if err != nil {
			return nil, err
		}
		return NickReply(*event), nil
	case *WhoCommand:
		listing, err := s.room.Listing(s.ctx)
		if err != nil {
			return nil, err
		}
		return &WhoReply{Listing: listing}, nil
	default:
		return nil, fmt.Errorf("command type %T not implemented", payload)
	}
}

func (s *memSession) sendSnapshot() error {
	msgs, err := s.room.Latest(s.ctx, 100, 0)
	if err != nil {
		return err
	}
	listing, err := s.room.Listing(s.ctx)
	if err != nil {
		return err
	}

	snapshot := &SnapshotEvent{
		Version: s.room.Version(),
		Listing: listing,
		Log:     msgs,
	}

	event, err := MakeEvent(snapshot)
	if err != nil {
		return err
	}
	s.outgoing <- event
	return nil
}
