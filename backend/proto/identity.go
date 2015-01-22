package proto

// An Identity maps to a global persona. It may exist only in the context
// of a single Room. An Identity may be anonymous.
type Identity interface {
	ID() string
	Name() string
	View() *IdentityView
}

type IdentityView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
