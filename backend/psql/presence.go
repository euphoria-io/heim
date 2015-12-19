package psql

import (
	"encoding/json"
	"time"

	"euphoria.io/heim/proto"
)

type Presence struct {
	Room      string
	Topic     string
	ServerID  string `db:"server_id"`
	ServerEra string `db:"server_era"`
	SessionID string `db:"session_id"`
	Updated   time.Time
	KeyID     string `db:"key_id"`
	Fact      []byte
}

func (p *Presence) SetFact(fact *proto.Presence) error {
	data, err := json.Marshal(fact)
	if err != nil {
		return err
	}
	p.Fact = data
	return nil
}

func (p *Presence) SessionView(level proto.PrivilegeLevel) (proto.SessionView, error) {
	var fact proto.Presence
	if err := json.Unmarshal(p.Fact, &fact); err != nil {
		return proto.SessionView{}, err
	}
	switch level {
	case proto.Staff:
	case proto.Host:
		fact.RealClientAddress = ""
	case proto.General:
		fact.ClientAddress = ""
		fact.RealClientAddress = ""
	}
	return fact.SessionView, nil
}

type roomConn struct {
	sessions map[string]proto.Session
	nicks    map[string]string
}

type VirtualAddress struct {
	Room    string
	Virtual string
	Real    string
	Created time.Time
}
