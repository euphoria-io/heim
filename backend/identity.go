package backend

import (
	"heim/backend/proto"
)

type memIdentity struct {
	id   string
	name string
}

func newMemIdentity(id string) *memIdentity {
	return &memIdentity{id: id, name: "guest"}
}

func (s *memIdentity) ID() string   { return s.id }
func (s *memIdentity) Name() string { return s.name }

func (s *memIdentity) View() *proto.IdentityView {
	return &proto.IdentityView{ID: s.id, Name: s.name}
}
