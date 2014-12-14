package backend

type Identity interface {
	ID() string
	Name() string
	View() *IdentityView
}

type IdentityView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type memIdentity struct {
	id   string
	name string
}

func newMemIdentity(id string) *memIdentity {
	return &memIdentity{id: id, name: id}
}

func (s *memIdentity) ID() string   { return s.id }
func (s *memIdentity) Name() string { return s.name }

func (s *memIdentity) View() *IdentityView {
	return &IdentityView{ID: s.id, Name: s.name}
}
