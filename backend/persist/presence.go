package persist

import (
	"database/sql"
	"fmt"
	"time"

	"heim/proto"

	"golang.org/x/net/context"
)

type Presence struct {
	Room         string
	UserID       string `db:"user_id"`
	SessionID    string `db:"session_id"`
	ServerID     string `db:"server_id"`
	Name         string
	Connected    bool
	LastActivity time.Time `db:"last_activity"`
}

func (Presence) AfterCreateTable(db *sql.DB) error {
	if _, err := db.Exec("CREATE INDEX presence_room ON presence(room)"); err != nil {
		return err
	}
	if _, err := db.Exec("CREATE INDEX presence_user_id ON presence(user_id)"); err != nil {
		return err
	}
	return nil
}

type roomConn struct {
	sessions map[string]proto.Session
	nicks    map[string]string
}

type roomPresence map[string]roomConn

func (rp roomPresence) load(sessionID, userID, nick string) {
	rc, ok := rp[userID]
	if !ok {
		rc = roomConn{
			sessions: map[string]proto.Session{},
			nicks:    map[string]string{},
		}
		rp[userID] = rc
	}
	rc.nicks[sessionID] = nick
}

func (rp roomPresence) join(session proto.Session) {
	rp.load(session.ID(), session.Identity().ID(), session.Identity().Name())
	rp[session.Identity().ID()].sessions[session.ID()] = session
}

func (rp roomPresence) part(session proto.Session) { delete(rp, session.ID()) }

func (rp roomPresence) broadcast(
	ctx context.Context, event *proto.Packet, exclude ...string) error {

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
	rc, ok := rp[nickEvent.ID]
	if !ok {
		rc = roomConn{
			sessions: map[string]proto.Session{},
			nicks:    map[string]string{},
		}
		rp[nickEvent.ID] = rc
	}

	for _, session := range rc.sessions {
		if session.Identity().Name() == nickEvent.From {
			session.SetName(nickEvent.To)
			rc.nicks[session.ID()] = nickEvent.To
		}
	}
}
