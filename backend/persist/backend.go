package persist

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"heim/backend"

	"github.com/coopernurse/gorp"
	"github.com/lib/pq"
	"golang.org/x/net/context"
)

var schema = map[interface{}][]string{
	Room{}:    []string{"Name"},
	Message{}: []string{"Room", "ID"},
}

type AfterCreateTabler interface {
	AfterCreateTable(*sql.DB) error
}

type Backend struct {
	sync.Mutex
	*sql.DB
	*gorp.DbMap

	dsn       string
	cancel    context.CancelFunc
	presence  map[string]roomPresence
	liveNicks map[string]string
}

func NewBackend(dsn string) (*Backend, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %s", err)
	}

	b := &Backend{DB: db, dsn: dsn}
	b.start()
	return b, nil
}

func (b *Backend) start() {
	b.DbMap = &gorp.DbMap{Db: b.DB, Dialect: gorp.PostgresDialect{}}
	// TODO: make debug configurable
	b.DbMap.TraceOn("[gorp]", log.New(os.Stdout, "", log.LstdFlags))

	for t, keys := range schema {
		b.DbMap.AddTable(t).SetKeys(false, keys...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.background(ctx)
}

func (b *Backend) UpgradeDB() error {
	// TODO: inspect existing schema and adapt; for now, assume empty DB
	return b.createSchema()
}

func (b *Backend) createSchema() error {
	if err := b.DbMap.CreateTables(); err != nil {
		return err
	}

	for t, _ := range schema {
		if after, ok := t.(AfterCreateTabler); ok {
			fmt.Printf("after create tabler on %T\n", t)
			if err := after.AfterCreateTable(b.DB); err != nil {
				tableName := "???"
				if table, err := b.DbMap.TableFor(reflect.TypeOf(t), false); err == nil {
					tableName = table.TableName
				}
				return fmt.Errorf("%s post-creation: %s", tableName, err)
			}
		}
	}

	return nil
}

func (b *Backend) Close() { b.cancel() }

func (b *Backend) background(ctx context.Context) {
	logger := backend.Logger(ctx)

	listener := pq.NewListener(b.dsn, 200*time.Millisecond, 5*time.Second, nil)
	if err := listener.Listen("broadcast"); err != nil {
		// TODO: manage this more nicely
		panic("pq listen: " + err.Error())
	}

	for {
		select {
		// TODO: add keep-alive timeout to trigger ping
		case <-ctx.Done():
			return
		case notice := <-listener.Notify:
			if notice == nil {
				// TODO: notify clients of potential hiccup
				continue
			}

			logger.Printf("notify: %s\n", notice.Extra)
			var msg BroadcastMessage

			if err := json.Unmarshal([]byte(notice.Extra), &msg); err != nil {
				logger.Printf("error: pq listen: invalid broadcast: %s", err)
				logger.Printf("         payload: %#v", notice.Extra)
				continue
			}

			if msg.Event.Type == backend.NickEventType {
				payload, err := msg.Event.Payload()
				nickEvent, ok := payload.(*backend.NickEvent)
				if err != nil || !ok {
					logger.Printf("error: pq listen: invalid nick event: %s", err)
					logger.Printf("         payload: %#v", notice.Extra)
				} else {
					b.Lock()
					rp := b.presence[msg.Room]
					if rp == nil {
						rp = roomPresence{}
						b.presence[msg.Room] = rp
					}
					rp.rename(nickEvent)
					b.Unlock()
				}
			}

			if rp, ok := b.presence[msg.Room]; ok {
				if err := rp.broadcast(ctx, msg.Event, msg.Exclude...); err != nil {
					logger.Printf("error: pq listen: broadcast error on %s: %s", msg.Room, err)
				}
				continue
			}

			// TODO: if room name is empty, broadcast globally
		}
	}
}

func (b *Backend) GetRoom(name string) (backend.Room, error) {
	obj, err := b.DbMap.Get(Room{}, name)
	if err != nil {
		return nil, err
	}

	var room *Room
	if obj == nil {
		room = &Room{
			Name: name,
		}
		if err := b.DbMap.Insert(room); err != nil {
			return nil, err
		}
	} else {
		room = obj.(*Room)
	}

	return room.Bind(b), nil
}

func (b *Backend) sendMessageToRoom(
	ctx context.Context, room *Room, session backend.Session, msg backend.Message,
	exclude ...backend.Session) (backend.Message, error) {

	logger := backend.Logger(ctx)
	logger.Printf("inserting message")

	stored, err := NewMessage(room, session.Identity().View(), msg.Parent, msg.Content)
	if err != nil {
		return backend.Message{}, err
	}

	if err := b.DbMap.Insert(stored); err != nil {
		return backend.Message{}, err
	}

	result := stored.ToBackend()
	event := backend.SendEvent(result)
	return result, b.broadcast(ctx, room, session, backend.SendEventType, &event, exclude...)
}

func (b *Backend) broadcast(
	ctx context.Context, room *Room, session backend.Session,
	packetType backend.PacketType, payload interface{}, exclude ...backend.Session) error {

	logger := backend.Logger(ctx)
	logger.Printf("broadcast [%s:%s] %v %#v (ex. %#v)",
		room.Name, session.ID(), packetType, payload, exclude)

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	packet := &backend.Packet{Type: packetType, Data: json.RawMessage(encodedPayload)}
	broadcastMsg := BroadcastMessage{
		Room:    room.Name,
		Event:   packet,
		Exclude: make([]string, len(exclude)),
	}
	for i, s := range exclude {
		broadcastMsg.Exclude[i] = s.ID()
	}

	encoded, err := json.Marshal(broadcastMsg)
	if err != nil {
		return err
	}

	escaped := strings.Replace(string(encoded), "'", "''", -1)
	_, err = b.DbMap.Exec(fmt.Sprintf("NOTIFY broadcast, '%s'", escaped))
	return err
}

func (b *Backend) join(ctx context.Context, room *Room, session backend.Session) error {
	b.Lock()
	defer b.Unlock()

	if b.presence == nil {
		b.presence = map[string]roomPresence{}
	}

	rp := b.presence[room.Name]
	if rp == nil {
		rp = roomPresence{}
		b.presence[room.Name] = rp
	}

	rp.join(session)
	return b.broadcast(ctx, room, session,
		backend.JoinEventType, backend.PresenceEvent(*session.Identity().View()), session)
}

func (b *Backend) part(ctx context.Context, room *Room, session backend.Session) error {
	b.Lock()
	defer b.Unlock()

	if rp, ok := b.presence[room.Name]; ok {
		rp.part(session)
	}
	return b.broadcast(ctx, room, session,
		backend.PartEventType, backend.PresenceEvent(*session.Identity().View()), session)
}

func (b *Backend) listing(ctx context.Context, room *Room) (backend.Listing, error) {
	result := backend.Listing{}
	for _, rc := range b.presence[room.Name] {
		for _, session := range rc.sessions {
			result = append(result, *session.Identity().View())
		}
	}
	sort.Sort(result)
	return result, nil
}

func (b *Backend) latest(ctx context.Context, room *Room, n int) ([]backend.Message, error) {
	if n <= 0 {
		return nil, nil
	}
	// TODO: define constant
	if n > 1000 {
		n = 1000
	}

	msgs, err := b.DbMap.Select(
		Message{}, "SELECT * FROM message WHERE room = $1 ORDER BY posted DESC LIMIT $2",
		room.Name, n)
	if err != nil {
		return nil, err
	}

	results := make([]backend.Message, len(msgs))
	for i, row := range msgs {
		msg := row.(*Message)
		results[len(msgs)-i-1] = backend.Message{
			UnixTime: msg.Posted.Unix(),
			Sender: &backend.IdentityView{
				ID:   msg.SenderID,
				Name: msg.SenderName,
			},
			Content: msg.Content,
		}
	}

	return results, nil
}

type BroadcastMessage struct {
	Room    string
	Exclude []string
	Event   *backend.Packet
}

type roomConn struct {
	sessions map[string]backend.Session
	nicks    map[string]string
}

type roomPresence map[string]roomConn

func (rp roomPresence) join(session backend.Session) {
	rc, ok := rp[session.Identity().ID()]
	if !ok {
		rc = roomConn{
			sessions: map[string]backend.Session{},
			nicks:    map[string]string{},
		}
		rp[session.Identity().ID()] = rc
	}

	rc.sessions[session.ID()] = session
	rc.nicks[session.ID()] = session.Identity().Name()
}

func (rp roomPresence) part(session backend.Session) { delete(rp, session.ID()) }

func (rp roomPresence) broadcast(
	ctx context.Context, event *backend.Packet, exclude ...string) error {

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

func (rp roomPresence) rename(nickEvent *backend.NickEvent) {
	rc, ok := rp[nickEvent.ID]
	if !ok {
		rc = roomConn{
			sessions: map[string]backend.Session{},
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
