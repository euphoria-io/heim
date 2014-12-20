package backend

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
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
	return session
}

func (s *memSession) ID() string          { return s.conn.RemoteAddr().String() }
func (s *memSession) Close()              { s.cancel() }
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

func (s *memSession) serve() {
	go s.readMessages()

	logger := Logger(s.ctx)
	logger.Printf("client connected")

	for {
		select {
		case <-s.ctx.Done():
			return
		case cmd := <-s.incoming:
			logger.Printf("received command: id=%s, type=%s", cmd.ID, cmd.Type)

			reply, err := s.handleCommand(cmd)
			if err != nil {
				logger.Printf("error: handleCommand: %s", err)
				reply = ErrorReply{Error: err.Error()}
			}

			resp, err := Response(cmd.ID, cmd.Type, reply)
			if err != nil {
				logger.Printf("error: Response: %s", err)
				return
			}

			data, err := resp.Encode()
			if err != nil {
				logger.Printf("error: Response encode: %s", err)
				return
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return
			}
		case cmd := <-s.outgoing:
			data, err := cmd.Encode()
			if err != nil {
				logger.Printf("error: push message encode: %s", err)
				return
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logger.Printf("error: write message: %s", err)
				return
			}
		}
	}
}

func (s *memSession) readMessages() {
	logger := Logger(s.ctx)
	defer s.Close()

	for s.ctx.Err() == nil {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			if err == io.EOF {
				logger.Printf("client disconnected")
				return
			}
			logger.Printf("error: read message: %s", err)
			return
		}

		// TODO: check messageType

		cmd, err := ParseRequest(data)
		if err != nil {
			logger.Printf("error: ParseRequest: %s", err)
			return
		}

		s.incoming <- cmd
	}
}

func (s *memSession) handleCommand(cmd *Packet) (interface{}, error) {
	payload, err := cmd.Payload()
	if err != nil {
		return nil, fmt.Errorf("payload: %s", err)
	}

	switch msg := payload.(type) {
	case *SendCommand:
		sent, err := s.room.Send(s.ctx, s, Message{Content: msg.Content})
		if err != nil {
			return nil, err
		}
		return SendReply(sent), nil
	case *LogCommand:
		msgs, err := s.room.Latest(s.ctx, msg.N)
		if err != nil {
			return nil, err
		}
		return LogReply{Log: msgs}, nil
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
		return WhoReply{Listing: listing}, nil
	default:
		return nil, fmt.Errorf("command type %T not implemented", payload)
	}
}
