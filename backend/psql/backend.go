package psql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
)

var ErrPsqlConnectionLost = errors.New("postgres connection lost")

var schema = []struct {
	Name       string
	Table      interface{}
	PrimaryKey []string
}{
	// Rooms.
	{"room_master_key", RoomMessageKey{}, []string{"Room", "KeyID"}},
	{"room_capability", RoomCapability{}, []string{"Room", "CapabilityID"}},
	{"room_manager_capability", RoomManagerCapability{}, []string{"Room", "CapabilityID"}},
	{"room", Room{}, []string{"Name"}},

	// Presence.
	{"presence", Presence{}, []string{"Room", "Topic", "ServerID", "ServerEra", "SessionID"}},

	// Bans.
	{"banned_agent", BannedAgent{}, []string{"AgentID", "Room"}},
	{"banned_ip", BannedIP{}, []string{"IP", "Room"}},

	// Messages.
	{"message", Message{}, []string{"Room", "ID"}},
	{"message_edit_log", MessageEditLog{}, []string{"EditID"}},

	// Sessions.
	{"session_log", SessionLog{}, []string{"SessionID"}},

	// Accounts.
	{"agent", Agent{}, []string{"ID"}},
	{"personal_identity", PersonalIdentity{}, []string{"Namespace", "ID"}},
	{"account", Account{}, []string{"ID"}},

	// Keys and capabilities.
	{"master_key", MessageKey{}, []string{"ID"}},
	{"capability", Capability{}, []string{"ID"}},
}

type Backend struct {
	sync.Mutex
	*sql.DB
	*gorp.DbMap

	dsn       string
	cancel    func()
	cluster   cluster.Cluster
	desc      *cluster.PeerDesc
	version   string
	peers     map[string]string
	listeners map[string]ListenerMap
	ctx       scope.Context
	logger    *log.Logger
}

func NewBackend(
	ctx scope.Context, dsn string, c cluster.Cluster, serverDesc *cluster.PeerDesc) (*Backend, error) {

	var version string

	if serverDesc == nil {
		version = "dev"
	} else {
		version = serverDesc.Version
	}

	parsedDSN, err := url.Parse(dsn)
	if err == nil {
		if parsedDSN.User != nil {
			parsedDSN.User = url.UserPassword(parsedDSN.User.Username(), "xxxxxx")
		}
		log.Printf("psql backend %s on %s", version, parsedDSN.String())
	} else {
		return nil, fmt.Errorf("url.Parse: %s", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %s", err)
	}

	b := &Backend{
		DB:        db,
		dsn:       dsn,
		desc:      serverDesc,
		version:   version,
		cluster:   c,
		peers:     map[string]string{},
		listeners: map[string]ListenerMap{},
		ctx:       ctx,
	}
	b.logger = log.New(os.Stdout, fmt.Sprintf("[backend %p] ", b), log.LstdFlags)

	if serverDesc != nil {
		b.peers[serverDesc.ID] = serverDesc.Era
		for _, desc := range c.Peers() {
			b.peers[desc.ID] = desc.Era
		}
	}

	if err := b.start(); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *Backend) debug(format string, args ...interface{}) { b.logger.Printf(format, args...) }

func (b *Backend) start() error {
	b.DbMap = &gorp.DbMap{Db: b.DB, Dialect: gorp.PostgresDialect{}}
	// TODO: make debug configurable
	b.DbMap.TraceOn("[gorp]", log.New(os.Stdout, "", log.LstdFlags))

	for _, item := range schema {
		b.DbMap.AddTableWithName(item.Table, item.Name).SetKeys(false, item.PrimaryKey...)
	}

	if b.desc != nil {
		if _, err := b.DbMap.Exec("DELETE FROM presence WHERE server_id = $1", b.desc.ID); err != nil {
			return fmt.Errorf("presence reset error: %s", err)
		}
	}

	b.cancel = b.ctx.Cancel
	b.ctx.WaitGroup().Add(1)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go b.background(wg)
	wg.Wait()
	return nil
}

func (b *Backend) Version() string { return b.version }

func (b *Backend) Close() {
	b.cancel()
	b.cluster.Part()
	b.ctx.WaitGroup().Wait()
	b.DbMap.Db.Close()
}

func (b *Backend) background(wg *sync.WaitGroup) {
	ctx := b.ctx.Fork()
	logger := b.logger

	defer ctx.WaitGroup().Done()

	listener := pq.NewListener(b.dsn, 200*time.Millisecond, 5*time.Second, nil)
	if err := listener.Listen("broadcast"); err != nil {
		// TODO: manage this more nicely
		panic("pq listen: " + err.Error())
	}
	logger.Printf("pq listener started")

	peerWatcher := b.cluster.Watch()
	keepalive := time.NewTicker(3 * cluster.TTL / 4)
	defer keepalive.Stop()

	// Signal to constructor that we're ready to handle client connections.
	wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepalive.C:
			if b.desc != nil {
				if err := b.cluster.Update(b.desc); err != nil {
					logger.Printf("cluster: keepalive error: %s", err)
				}
			}
			// Ping to make sure the database connection is still live.
			if err := listener.Ping(); err != nil {
				logger.Printf("pq ping: %s\n", err)
				b.ctx.Terminate(fmt.Errorf("pq ping: %s", err))
				return
			}
		case event := <-peerWatcher:
			b.Lock()
			switch e := event.(type) {
			case *cluster.PeerJoinedEvent:
				logger.Printf("cluster: peer %s joining with era %s", e.ID, e.Era)
				b.peers[e.ID] = e.Era
			case *cluster.PeerAliveEvent:
				if prevEra := b.peers[e.ID]; prevEra != e.Era {
					b.invalidatePeer(ctx, e.ID, prevEra)
					logger.Printf("cluster: peer %s changing era from %s to %s", e.ID, prevEra, e.Era)
				}
				b.peers[e.ID] = e.Era
			case *cluster.PeerLostEvent:
				logger.Printf("cluster: peer %s departing", e.ID)
				if era, ok := b.peers[e.ID]; ok {
					b.invalidatePeer(ctx, e.ID, era)
					delete(b.peers, e.ID)
				}
			}
			b.Unlock()
		case notice := <-listener.Notify:
			if notice == nil {
				logger.Printf("pq listen: received nil notification")
				// A nil notice indicates a loss of connection. We could
				// re-snapshot for all connected clients, but for now it's
				// easier to just shut down and force everyone to reconnect.
				b.ctx.Terminate(ErrPsqlConnectionLost)
				return
			}

			var msg BroadcastMessage

			if err := json.Unmarshal([]byte(notice.Extra), &msg); err != nil {
				logger.Printf("error: pq listen: invalid broadcast: %s", err)
				logger.Printf("         payload: %#v", notice.Extra)
				continue
			}

			// Check for global ban, which is a special-case broadcast.
			if msg.Room == "" && msg.Event.Type == proto.BounceEventType {
				for _, lm := range b.listeners {
					if err := lm.Broadcast(ctx, msg.Event, msg.Exclude...); err != nil {
						logger.Printf("error: pq listen: bounce broadcast error on %s: %s", msg.Room, err)
					}
				}
				continue
			}

			if lm, ok := b.listeners[msg.Room]; ok {
				logger.Printf("broadcasting %s to %s", msg.Event.Type, msg.Room)
				if err := lm.Broadcast(ctx, msg.Event, msg.Exclude...); err != nil {
					logger.Printf("error: pq listen: broadcast error on %s: %s", msg.Room, err)
				}
				continue
			}

			logger.Printf("pq listen: dropping notification %s to %s because no listeners",
				msg.Event.Type, msg.Room)

			// TODO: if room name is empty, broadcast globally
		}
	}
}

func (b *Backend) GetRoom(ctx scope.Context, name string) (proto.Room, error) {
	obj, err := b.DbMap.Get(Room{}, name)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, proto.ErrRoomNotFound
	}
	return obj.(*Room).Bind(b), nil
}

func (b *Backend) CreateRoom(
	ctx scope.Context, kms security.KMS, private bool, name string, managers ...proto.Account) (
	proto.Room, error) {

	sec, err := proto.NewRoomSecurity(kms, name)
	if err != nil {
		return nil, err
	}

	backend.Logger(ctx).Printf("creating room: %s", name)
	room := &Room{
		Name:  name,
		IV:    sec.KeyPair.IV,
		MAC:   sec.MAC,
		Nonce: sec.Nonce,
		EncryptedManagementKey: sec.KeyEncryptingKey.Ciphertext,
		EncryptedPrivateKey:    sec.KeyPair.EncryptedPrivateKey,
		PublicKey:              sec.KeyPair.PublicKey,
	}

	var (
		rmkb   *RoomMessageKeyBinding
		msgKey security.ManagedKey
	)
	if private {
		rmkb, err = room.generateMessageKey(b, kms)
		if err != nil {
			return nil, err
		}

		msgKey = rmkb.ManagedKey()
		if err := kms.DecryptKey(&msgKey); err != nil {
			return nil, err
		}
	}

	// Generate manager capabilities.
	managerKey := sec.KeyEncryptingKey.Clone()
	if err := kms.DecryptKey(&managerKey); err != nil {
		return nil, fmt.Errorf("manager key decrypt error: %s", err)
	}
	roomKeyPair, err := sec.Unlock(&managerKey)
	if err != nil {
		return nil, fmt.Errorf("room security unlock error: %s", err)
	}
	managerCaps := make([]*security.PublicKeyCapability, len(managers))
	for i, manager := range managers {
		kp := manager.KeyPair()
		c, err := security.GrantPublicKeyCapability(
			kms, sec.Nonce, roomKeyPair, &kp, nil, managerKey.Plaintext)
		if err != nil {
			return nil, fmt.Errorf("manager grant error: %s", err)
		}
		managerCaps[i] = c
	}

	accessCaps := []*security.PublicKeyCapability{}
	if private {
		accessCaps = make([]*security.PublicKeyCapability, len(managers))
		for i, manager := range managers {
			kp := manager.KeyPair()
			c, err := security.GrantPublicKeyCapability(
				kms, rmkb.Nonce(), roomKeyPair, &kp, nil, msgKey.Plaintext)
			if err != nil {
				return nil, fmt.Errorf("access grant error: %s", err)
			}
			accessCaps[i] = c
		}
	}

	// Insert data.
	t, err := b.DbMap.Begin()
	if err != nil {
		return nil, err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	if err := t.Insert(room); err != nil {
		backend.Logger(ctx).Printf("room creation error on %s: %s", name, err)
		rollback()
		return nil, err
	}

	if rmkb != nil {
		if err := t.Insert(&rmkb.MessageKey, &rmkb.RoomMessageKey); err != nil {
			backend.Logger(ctx).Printf("room creation error on %s (message key): %s", name, err)
			rollback()
			return nil, err
		}
	}

	managerCapTable := RoomManagerCapabilities{
		Room:     room,
		Executor: t,
	}
	for i, capability := range managerCaps {
		if err := managerCapTable.Save(ctx, managers[i], capability); err != nil {
			backend.Logger(ctx).Printf(
				"room creation error on %s (manager %s): %s", name, managers[i].ID().String(), err)
			rollback()
			return nil, err
		}
	}

	messageCapTable := RoomMessageCapabilities{
		Room:     room,
		Executor: t,
	}
	for i, capability := range accessCaps {
		if err := messageCapTable.Save(ctx, managers[i], capability); err != nil {
			backend.Logger(ctx).Printf(
				"room creation error on %s (access capability): %s", name, err)
			rollback()
			return nil, err
		}
	}

	if err := t.Commit(); err != nil {
		backend.Logger(ctx).Printf("room creation error on %s (commit): %s", name, err)
		return nil, err
	}

	return room.Bind(b), nil
}

func (b *Backend) BanIP(ctx scope.Context, ip string, until time.Time) error {
	ban := &BannedIP{
		IP:      ip,
		Created: time.Now(),
		Expires: gorp.NullTime{
			Time:  until,
			Valid: !until.IsZero(),
		},
	}

	if err := b.DbMap.Insert(ban); err != nil {
		return err
	}

	bounceEvent := &proto.BounceEvent{Reason: "banned", IP: ip}
	return b.broadcast(ctx, nil, proto.BounceEventType, bounceEvent)
}

func (b *Backend) UnbanIP(ctx scope.Context, ip string) error {
	_, err := b.DbMap.Exec("DELETE FROM banned_ip WHERE ip = $1 AND room IS NULL", ip)
	return err
}

func (b *Backend) sendMessageToRoom(
	ctx scope.Context, room *Room, msg proto.Message, exclude ...proto.Session) (proto.Message, error) {

	stored, err := NewMessage(room, msg.Sender, msg.ID, msg.Parent, msg.EncryptionKeyID, msg.Content)
	if err != nil {
		return proto.Message{}, err
	}

	if err := b.DbMap.Insert(stored); err != nil {
		return proto.Message{}, err
	}

	result := stored.ToBackend()
	event := proto.SendEvent(result)
	return result, b.broadcast(ctx, room, proto.SendEventType, &event, exclude...)
}

func (b *Backend) broadcast(
	ctx scope.Context, room *Room, packetType proto.PacketType, payload interface{},
	exclude ...proto.Session) error {

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	packet := &proto.Packet{Type: packetType, Data: json.RawMessage(encodedPayload)}
	broadcastMsg := BroadcastMessage{
		Event:   packet,
		Exclude: make([]string, 0, len(exclude)),
	}
	if room != nil {
		broadcastMsg.Room = room.Name
	}
	for _, s := range exclude {
		if s != nil {
			broadcastMsg.Exclude = append(broadcastMsg.Exclude, s.ID())
		}
	}

	encoded, err := json.Marshal(broadcastMsg)
	if err != nil {
		return err
	}

	escaped := strings.Replace(string(encoded), "'", "''", -1)
	_, err = b.DbMap.Exec(fmt.Sprintf("NOTIFY broadcast, '%s'", escaped))
	return err
}

func (b *Backend) join(ctx scope.Context, room *Room, session proto.Session) error {
	client := &proto.Client{}
	if !client.FromContext(ctx) {
		return fmt.Errorf("client data not found in scope")
	}

	// Check for agent ID bans.
	agentBans, err := b.DbMap.Select(
		BannedAgent{},
		"SELECT agent_id, room, created, expires, room_reason, agent_reason, private_reason"+
			" FROM banned_agent"+
			" WHERE agent_id = $1 AND (room IS NULL OR room = $2)"+
			" AND (expires IS NULL OR expires > NOW())",
		client.Agent.IDString(), room.Name)
	if err != nil {
		return err
	}
	if len(agentBans) > 0 {
		backend.Logger(ctx).Printf("access denied to %s: %#v", client.Agent.IDString(), agentBans)
		return proto.ErrAccessDenied
	}

	// Check for IP bans.
	ipBans, err := b.DbMap.Select(
		BannedIP{},
		"SELECT ip, room, created, expires, reason FROM banned_ip"+
			" WHERE ip = $1 AND (room IS NULL OR room = $2)"+
			" AND (expires IS NULL OR expires > NOW())",
		client.IP, room.Name)
	if err != nil {
		return err
	}
	if len(ipBans) > 0 {
		backend.Logger(ctx).Printf("access denied to %s: %#v", client.IP, ipBans)
		return proto.ErrAccessDenied
	}

	// Write to session log.
	entry := &SessionLog{
		SessionID: session.ID(),
		IP:        client.IP,
		Room:      room.Name,
		UserAgent: client.UserAgent,
		Connected: client.Connected,
	}
	t, err := b.DbMap.Begin()
	if err != nil {
		return err
	}
	if _, err := t.Delete(entry); err != nil {
		if rerr := t.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return err
	}
	if err := t.Insert(entry); err != nil {
		if rerr := t.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return err
	}
	if err := t.Commit(); err != nil {
		return err
	}

	b.Lock()
	defer b.Unlock()

	// Add session to listeners.
	lm, ok := b.listeners[room.Name]
	if !ok {
		b.debug("registering listener for %s", room.Name)
		lm = ListenerMap{}
		b.listeners[room.Name] = lm
	}
	lm[session.ID()] = Listener{Session: session, Client: client}

	// Broadcast a presence event.
	// TODO: make this an explicit action via the Room protocol, to support encryption

	presence := &Presence{
		Room:      room.Name,
		ServerID:  b.desc.ID,
		ServerEra: b.desc.Era,
		SessionID: session.ID(),
		Updated:   time.Now(),
	}
	err = presence.SetFact(&proto.Presence{
		SessionView:    *session.View(),
		LastInteracted: presence.Updated,
	})
	if err != nil {
		return fmt.Errorf("presence marshal error: %s", err)
	}
	if err := b.DbMap.Insert(presence); err != nil {
		return fmt.Errorf("presence insert error: %s", err)
	}
	b.debug("joining session: %#v", session)
	b.debug(" -> %#v", session.View())
	return b.broadcast(ctx, room,
		proto.JoinEventType, proto.PresenceEvent(*session.View()), session)
}

func (b *Backend) part(ctx scope.Context, room *Room, session proto.Session) error {
	b.Lock()
	defer b.Unlock()

	if lm, ok := b.listeners[room.Name]; ok {
		delete(lm, session.ID())
	}

	_, err := b.DbMap.Exec(
		"DELETE FROM presence"+
			" WHERE room = $1 AND server_id = $2 AND server_era = $3 AND session_id = $4",
		room.Name, b.desc.ID, b.desc.Era, session.ID())
	if err != nil {
		backend.Logger(ctx).Printf("failed to persist departure: %s", err)
	}

	// Broadcast a presence event.
	// TODO: make this an explicit action via the Room protocol, to support encryption
	return b.broadcast(ctx, room,
		proto.PartEventType, proto.PresenceEvent(*session.View()), session)
}

func (b *Backend) listing(ctx scope.Context, room *Room) (proto.Listing, error) {
	// TODO: return presence in an envelope, to support encryption
	// TODO: cache for performance

	rows, err := b.DbMap.Select(
		Presence{},
		"SELECT room, topic, server_id, server_era, session_id, updated, key_id, fact"+
			" FROM presence WHERE room = $1",
		room.Name)
	if err != nil {
		return nil, fmt.Errorf("presence listing error: %s", err)
	}

	result := proto.Listing{}
	for _, row := range rows {
		p := row.(*Presence)
		if b.peers[p.ServerID] == p.ServerEra {
			if view, err := p.SessionView(); err == nil {
				result = append(result, view)
			} else {
				b.debug("ignoring presence row because error: %s", err)
			}
		} else {
			b.debug("ignoring presence row because era doesn't match (%s != %s)",
				p.ServerEra, b.peers[p.ServerID])
		}
	}

	sort.Sort(result)
	return result, nil
}

func (b *Backend) latest(ctx scope.Context, room *Room, n int, before snowflake.Snowflake) (
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

	// Get the time before which messages will be expired
	nDays, err := b.DbMap.SelectInt("SELECT retention_days FROM room WHERE name = $1", room.Name)
	if err != nil {
		return nil, err
	}
	if nDays == 0 {
		if before.IsZero() {
			query = ("SELECT room, id, previous_edit_id, parent, posted, edited, deleted," +
				" session_id, sender_id, sender_name, server_id, server_era, content, encryption_key_id" +
				" FROM message WHERE room = $1 AND deleted IS NULL ORDER BY id DESC LIMIT $2")
		} else {
			query = ("SELECT room, id, previous_edit_id, parent, posted, edited, deleted," +
				" session_id, sender_id, sender_name, server_id, server_era, content, encryption_key_id" +
				" FROM message WHERE room = $1 AND id < $3 AND deleted IS NULL ORDER BY id DESC LIMIT $2")
			args = append(args, before.String())
		}
	} else {
		threshold := time.Now().Add(time.Duration(-nDays) * 24 * time.Hour)
		if before.IsZero() {
			query = ("SELECT room, id, previous_edit_id, parent, posted, edited, deleted," +
				" session_id, sender_id, sender_name, server_id, server_era, content, encryption_key_id" +
				" FROM message WHERE room = $1 AND posted > $3 AND deleted IS NULL ORDER BY id DESC LIMIT $2")
		} else {
			query = ("SELECT room, id, previous_edit_id, parent, posted, edited, deleted," +
				" session_id, sender_id, sender_name, server_id, server_era, content, encryption_key_id" +
				" FROM message WHERE room = $1 AND id < $3 AND deleted IS NULL AND posted > $4 ORDER BY id DESC LIMIT $2")
			args = append(args, before.String())
		}
		args = append(args, threshold)
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

// invalidatePeer must be called with lock held
func (b *Backend) invalidatePeer(ctx scope.Context, id, era string) {
	logger := backend.Logger(ctx)
	packet, err := proto.MakeEvent(&proto.NetworkEvent{
		Type:      "partition",
		ServerID:  id,
		ServerEra: era,
	})
	if err != nil {
		logger.Printf("cluster: make network event error: %s", err)
		return
	}
	for _, lm := range b.listeners {
		if err := lm.Broadcast(ctx, packet); err != nil {
			logger.Printf("cluster: network event error: %s", err)
		}
	}
}

func (b *Backend) Peers() []cluster.PeerDesc { return b.cluster.Peers() }

func (b *Backend) AccountManager() proto.AccountManager { return &AccountManagerBinding{b} }
func (b *Backend) AgentTracker() proto.AgentTracker     { return &AgentTrackerBinding{b} }

type BroadcastMessage struct {
	Room    string
	Exclude []string
	Event   *proto.Packet
}
