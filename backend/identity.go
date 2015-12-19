package backend

import "euphoria.io/heim/proto"

type memIdentity struct {
	id        string
	name      string
	serverID  string
	serverEra string
}

func newMemIdentity(id, serverID, serverEra string) *memIdentity {
	return &memIdentity{
		id:        id,
		serverID:  serverID,
		serverEra: serverEra,
	}
}

func (s *memIdentity) ID() proto.UserID  { return proto.UserID(s.id) }
func (s *memIdentity) Name() string      { return s.name }
func (s *memIdentity) ServerID() string  { return s.serverID }
func (s *memIdentity) ServerEra() string { return s.serverEra }

func (s *memIdentity) View() proto.IdentityView {
	return proto.IdentityView{
		ID:        proto.UserID(s.id),
		Name:      s.name,
		ServerID:  s.serverID,
		ServerEra: s.serverEra,
	}
}

func NewIdentity(id, name string) proto.Identity {
	return &memIdentity{id: id, name: name}
}
