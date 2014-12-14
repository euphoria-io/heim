package backend

import (
	"fmt"

	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

type Session interface {
	Identity() Identity
	Send(context.Context, Message) error
}

type memSession struct {
	ctx      context.Context
	conn     *websocket.Conn
	identity Identity
	room     Room

	incoming chan *Command
	outgoing chan *Command
}

func newMemSession(ctx context.Context, conn *websocket.Conn, room Room) *memSession {
	session := &memSession{
		ctx:      ctx,
		conn:     conn,
		identity: rawIdentity(conn.LocalAddr().String()),
		room:     room,

		incoming: make(chan *Command),
		outgoing: make(chan *Command, 100),
	}
	return session
}

func (s *memSession) Identity() Identity { return s.identity }

func (s *memSession) Send(ctx context.Context, msg Message) error {
	encoded, err := msg.Encode()
	if err != nil {
		return err
	}

	cmd := &Command{
		Type: SendType,
		Data: encoded,
	}

	go func() { s.outgoing <- cmd }()

	return nil
}

func (s *memSession) serve() {
	go s.readMessages()

	for {
		select {
		case cmd := <-s.incoming:
			reply, err := s.handleCommand(cmd)
			if err != nil {
				// TODO: log error?
				reply = err
			}

			resp, err := Response(cmd.ID, cmd.Type, reply)
			if err != nil {
				// TODO: log error, disconnect?
				return
			}

			data, err := resp.Encode()
			if err != nil {
				// TODO: log error, disconnect?
				return
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				// TODO: log error, disconnect?
				return
			}
		case cmd := <-s.outgoing:
			data, err := cmd.Encode()
			if err != nil {
				// TODO: log error, disconnect?
				return
			}

			if err := s.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				// TODO: log error, disconnect?
				return
			}
		}
	}
}

func (s *memSession) readMessages() {
	// TODO: termination condition?
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			// TODO: log error, disconnect?
			return
		}

		// TODO: check messageType

		cmd, err := ParseRequest(data)
		if err != nil {
			// TODO: log error, disconnect?
			return
		}

		s.incoming <- cmd
	}
}

func (s *memSession) handleCommand(cmd *Command) (interface{}, error) {
	payload, err := cmd.Payload()
	if err != nil {
		return nil, err
	}

	switch msg := payload.(type) {
	case *SendCommand:
		return s.room.Send(s.ctx, s, Message{Content: msg.Content})
	case *LogCommand:
		return s.room.Latest(s.ctx, msg.N)
	default:
		return nil, fmt.Errorf("command type %T not implemented", payload)
	}
}
