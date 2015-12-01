package mock

import (
	"sync"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/scope"
)

type session struct {
	sync.Mutex
	id      string
	agentID string
	name    string
	history []message
}

type message struct {
	cmdType proto.PacketType
	payload interface{}
}

func TestSession(id, agentID string) proto.Session { return newSession(id, agentID) }

func newSession(id, agentID string) *session { return &session{id: id, agentID: agentID} }

func (s *session) ServerID() string         { return "test" }
func (s *session) ID() string               { return s.id }
func (s *session) AgentID() string          { return s.agentID }
func (s *session) Close()                   {}
func (s *session) CheckAbandoned() error    { return nil }
func (s *session) SetName(name string)      { s.name = name }
func (s *session) Identity() proto.Identity { return backend.NewIdentity(s.id, s.name) }

func (s *session) View() *proto.SessionView {
	return &proto.SessionView{
		IdentityView: s.Identity().View(),
		SessionID:    s.id,
	}
}

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
