package psql

import (
	"encoding/json"
	"fmt"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/scope"
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

type roomPresence map[string]roomConn

func (rp roomPresence) load(sessionID string, userID proto.UserID, nick string) {
	rc, ok := rp[string(userID)]
	if !ok {
		rc = roomConn{
			sessions: map[string]proto.Session{},
			nicks:    map[string]string{},
		}
		rp[string(userID)] = rc
	}
	rc.nicks[sessionID] = nick
}

func (rp roomPresence) join(session proto.Session) {
	rp.load(session.ID(), session.Identity().ID(), session.Identity().Name())
	rp[string(session.Identity().ID())].sessions[session.ID()] = session
}

func (rp roomPresence) part(session proto.Session) { delete(rp, session.ID()) }

func (rp roomPresence) broadcast(ctx scope.Context, event *proto.Packet, exclude ...string) error {

	payload, err := event.Payload()
	if err != nil {
		return err
	}

	exc := make(map[string]struct{}, len(exclude))
	for _, x := range exclude {
		exc[x] = struct{}{}
	}

	for _, rc := range rp {
		for _, session := range rc.sessions {
			if _, ok := exc[session.ID()]; ok {
				continue
			}

			if err := session.Send(ctx, event.Type, payload); err != nil {
				// TODO: accumulate errors
				return fmt.Errorf("send message to %s: %s", session.ID(), err)
			}
		}
	}

	return nil
}

func (rp roomPresence) rename(nickEvent *proto.NickEvent) {
	rc, ok := rp[string(nickEvent.ID)]
	if !ok {
		rc = roomConn{
			sessions: map[string]proto.Session{},
			nicks:    map[string]string{},
		}
		rp[string(nickEvent.ID)] = rc
	}

	for _, session := range rc.sessions {
		if session.Identity().Name() == nickEvent.From {
			session.SetName(nickEvent.To)
			rc.nicks[session.ID()] = nickEvent.To
		}
	}
}
