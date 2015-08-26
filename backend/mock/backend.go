package mock

import (
	"sync"
	"time"

	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type TestBackend struct {
	sync.Mutex
	accountManager *accountManager
	accounts       map[snowflake.Snowflake]proto.Account
	accountIDs     map[string]*personalIdentity
	agents         map[string]*proto.Agent
	agentBans      map[string]time.Time
	ipBans         map[string]time.Time
	js             JobService
	resetReqs      map[snowflake.Snowflake]*proto.PasswordResetRequest
	rooms          map[string]proto.Room
	version        string
}

func (b *TestBackend) AccountManager() proto.AccountManager { return &accountManager{b: b} }
func (b *TestBackend) AgentTracker() proto.AgentTracker     { return &agentTracker{b} }
func (b *TestBackend) Jobs() proto.JobService               { return &b.js }

func (b *TestBackend) Close() {}

func (b *TestBackend) Version() string { return b.version }

func (b *TestBackend) GetRoom(ctx scope.Context, name string) (proto.Room, error) {
	b.Lock()
	defer b.Unlock()

	room, ok := b.rooms[name]
	if !ok {
		return nil, proto.ErrRoomNotFound
	}
	return room, nil
}

func (b *TestBackend) CreateRoom(
	ctx scope.Context, kms security.KMS, private bool, name string, managers ...proto.Account) (
	proto.Room, error) {

	b.Lock()
	defer b.Unlock()

	if b.rooms == nil {
		b.rooms = map[string]proto.Room{}
	}

	room, err := NewRoom(ctx, kms, private, name, b.version, managers...)
	if err != nil {
		return nil, err
	}

	b.rooms[name] = room
	return room, nil
}

func (b *TestBackend) Peers() []cluster.PeerDesc { return nil }

func (b *TestBackend) BanAgent(ctx scope.Context, agentID string, until time.Time) error {
	b.Lock()
	defer b.Unlock()

	if b.agentBans == nil {
		b.agentBans = map[string]time.Time{agentID: until}
	} else {
		b.agentBans[agentID] = until
	}
	return nil
}

func (b *TestBackend) UnbanAgent(ctx scope.Context, agentID string) error {
	b.Lock()
	defer b.Unlock()

	if _, ok := b.agentBans[agentID]; ok {
		delete(b.agentBans, agentID)
	}
	return nil
}

func (b *TestBackend) BanIP(ctx scope.Context, ip string, until time.Time) error {
	b.Lock()
	defer b.Unlock()

	if b.ipBans == nil {
		b.ipBans = map[string]time.Time{ip: until}
	} else {
		b.ipBans[ip] = until
	}
	return nil
}

func (b *TestBackend) UnbanIP(ctx scope.Context, ip string) error {
	b.Lock()
	defer b.Unlock()

	if _, ok := b.ipBans[ip]; ok {
		delete(b.ipBans, ip)
	}
	return nil
}
