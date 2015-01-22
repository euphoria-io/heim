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
	"heim/backend/proto"

	"github.com/coopernurse/gorp"
	"github.com/lib/pq"
	"golang.org/x/net/context"
)

var schema = map[interface{}][]string{
	Room{}:     []string{"Name"},
	Message{}:  []string{"Room", "ID"},
	Presence{}: []string{"Room", "SessionID"},
}

type AfterCreateTabler interface {
	AfterCreateTable(*sql.DB) error
}

type Backend struct {
	sync.Mutex
	*sql.DB
	*gorp.DbMap

	dsn       string
	version   string
	cancel    context.CancelFunc
	presence  map[string]roomPresence
	liveNicks map[string]string
}

func NewBackend(dsn, version string) (*Backend, error) {
	log.Printf("persistence backend %s on %s", version, dsn)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %s", err)
	}

	b := &Backend{DB: db, dsn: dsn, version: version}
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

func (b *Backend) Version() string { return b.version }
func (b *Backend) Close()          { b.cancel() }

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

			var msg BroadcastMessage

			if err := json.Unmarshal([]byte(notice.Extra), &msg); err != nil {
				logger.Printf("error: pq listen: invalid broadcast: %s", err)
				logger.Printf("         payload: %#v", notice.Extra)
				continue
			}

			if msg.Event.Type == proto.NickEventType {
				payload, err := msg.Event.Payload()
				nickEvent, ok := payload.(*proto.NickEvent)
				if err != nil || !ok {
					logger.Printf("error: pq listen: invalid nick event: %s", err)
					logger.Printf("         payload: %#v", notice.Extra)
				} else {
					b.Lock()
					if b.presence == nil {
						b.presence = map[string]roomPresence{}
					}
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

func (b *Backend) GetRoom(name string) (proto.Room, error) {
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
	ctx context.Context, room *Room, session proto.Session, msg proto.Message,
	exclude ...proto.Session) (proto.Message, error) {

	stored, err := NewMessage(room, session.Identity().View(), msg.Parent, msg.Content)
	if err != nil {
		return proto.Message{}, err
	}

	if err := b.DbMap.Insert(stored); err != nil {
		return proto.Message{}, err
	}

	result := stored.ToBackend()
	event := proto.SendEvent(result)
	return result, b.broadcast(ctx, room, session, proto.SendEventType, &event, exclude...)
}

func (b *Backend) broadcast(
	ctx context.Context, room *Room, session proto.Session, packetType proto.PacketType,
	payload interface{}, exclude ...proto.Session) error {

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	packet := &proto.Packet{Type: packetType, Data: json.RawMessage(encodedPayload)}
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

func (b *Backend) join(ctx context.Context, room *Room, session proto.Session) error {
	b.Lock()
	defer b.Unlock()

	logger := backend.Logger(ctx)

	if b.presence == nil {
		b.presence = map[string]roomPresence{}
	}

	rp := b.presence[room.Name]
	if rp == nil {
		rp = roomPresence{}
		b.presence[room.Name] = rp
		rows, err := b.DbMap.Select(Presence{}, "SELECT * FROM presence WHERE room = $1", room.Name)
		if err != nil {
			logger.Printf("error loading presence for %s: %s", room.Name, err)
		} else {
			for _, row := range rows {
				p := row.(*Presence)
				rp.load(p.SessionID, p.UserID, p.Name)
			}
		}
		_ = rows
	}

	rp.join(session)

	if err := b.DbMap.Insert(&Presence{
		Room:         room.Name,
		SessionID:    session.ID(),
		UserID:       session.Identity().ID(),
		ServerID:     session.ServerID(),
		Name:         session.Identity().Name(),
		Connected:    true,
		LastActivity: time.Now(),
	}); err != nil {
		logger.Printf("failed to persist join: %s", err)
	}

	return b.broadcast(ctx, room, session,
		proto.JoinEventType, proto.PresenceEvent(*session.Identity().View()), session)
}

func (b *Backend) part(ctx context.Context, room *Room, session proto.Session) error {
	b.Lock()
	defer b.Unlock()

	if rp, ok := b.presence[room.Name]; ok {
		rp.part(session)
	}

	_, err := b.DbMap.Exec(
		"UPDATE presence SET connected = false WHERE room = $1 AND session_id = $2",
		room.Name, session.ID())
	if err != nil {
		backend.Logger(ctx).Printf("failed to persist departure: %s", err)
	}

	return b.broadcast(ctx, room, session,
		proto.PartEventType, proto.PresenceEvent(*session.Identity().View()), session)
}

func (b *Backend) listing(ctx context.Context, room *Room) (proto.Listing, error) {
	result := proto.Listing{}
	for _, rc := range b.presence[room.Name] {
		for _, session := range rc.sessions {
			result = append(result, *session.Identity().View())
		}
	}
	sort.Sort(result)
	return result, nil
}

func (b *Backend) latest(ctx context.Context, room *Room, n int, before proto.Snowflake) (
	[]proto.Message, error) {

	if n <= 0 {
		return nil, nil
	}
	// TODO: define constant
	if n > 1000 {
		n = 1000
	}

	var query string
	args := []interface{}{room.Name, n}
	if before.IsZero() {
		query = "SELECT * FROM message WHERE room = $1 ORDER BY id DESC LIMIT $2"
	} else {
		query = "SELECT * FROM message WHERE room = $1 AND id < $3 ORDER BY id DESC LIMIT $2"
		args = append(args, before.String())
	}

	msgs, err := b.DbMap.Select(Message{}, query, args...)
	if err != nil {
		return nil, err
	}

	results := make([]proto.Message, len(msgs))
	for i, row := range msgs {
		msg := row.(*Message)
		results[len(msgs)-i-1] = msg.ToBackend()
	}

	return results, nil
}

type BroadcastMessage struct {
	Room    string
	Exclude []string
	Event   *proto.Packet
}
