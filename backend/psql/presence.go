package psql

import (
	"encoding/json"
	"fmt"
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
	fmt.Printf("presence fact: %#v\n", fact)
	data, err := json.Marshal(fact)
	if err != nil {
		return err
	}
	p.Fact = data
	return nil
}

func (p *Presence) SessionView() (proto.SessionView, error) {
	var fact proto.Presence
	if err := json.Unmarshal(p.Fact, &fact); err != nil {
		return proto.SessionView{}, err
	}
	return fact.SessionView, nil
}

type roomConn struct {
	sessions map[string]proto.Session
	nicks    map[string]string
}
