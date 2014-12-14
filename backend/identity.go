package backend

type Identity interface {
	ID() string
	Name() string
}

type rawIdentity string

func (s rawIdentity) ID() string   { return string(s) }
func (s rawIdentity) Name() string { return string(s) }
