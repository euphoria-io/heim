package mock

import (
	"sync"

	"euphoria.io/scope"

	"heim/backend"
	"heim/proto"
)

type session struct {
	sync.Mutex
	id      string
	name    string
	history []message
}

type message struct {
	cmdType proto.PacketType
	payload interface{}
}

func TestSession(id string) proto.Session { return newSession(id) }

func newSession(id string) *session { return &session{id: id} }

func (s *session) ServerID() string      { return "test" }
func (s *session) ID() string            { return s.id }
func (s *session) Close()                {}
func (s *session) CheckAbandoned() error { return nil }
func (s *session) SetName(name string)   { s.name = name }

func (s *session) Identity() proto.Identity { return backend.NewIdentity(s.id, s.name) }

func (s *session) Send(ctx scope.Context, cmdType proto.PacketType, payload interface{}) error {
	s.Lock()
	s.history = append(s.history, message{cmdType, payload})
	s.Unlock()
	return nil
}

func (s *session) clear() {
	s.Lock()
	s.history = nil
	s.Unlock()
}
