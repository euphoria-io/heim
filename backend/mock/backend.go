package mock

import (
	"fmt"
	"sync"
	"time"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type TestBackend struct {
	sync.Mutex
	agentBans  map[string]time.Time
	ipBans     map[string]time.Time
	rooms      map[string]proto.Room
	accounts   map[string]proto.Account
	accountIDs map[string]string
	version    string
}

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
	ctx scope.Context, kms security.KMS, name string, managers ...proto.Account) (proto.Room, error) {

	b.Lock()
	defer b.Unlock()

	if b.rooms == nil {
		b.rooms = map[string]proto.Room{}
	}

	room, err := NewRoom(kms, name, b.version, managers...)
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

func (b *TestBackend) RegisterAccount(
	ctx scope.Context, kms security.KMS, namespace, id, password string) (proto.Account, error) {

	b.Lock()
	defer b.Unlock()

	key := fmt.Sprintf("%s:%s", namespace, id)
	if _, ok := b.accountIDs[key]; ok {
		return nil, proto.ErrPersonalIdentityInUse
	}

	account, err := NewAccount(kms, password)
	if err != nil {
		return nil, err
	}

	if b.accounts == nil {
		b.accounts = map[string]proto.Account{account.ID().String(): account}
	} else {
		b.accounts[account.ID().String()] = account
	}

	if b.accountIDs == nil {
		b.accountIDs = map[string]string{key: account.ID().String()}
	} else {
		b.accountIDs[key] = account.ID().String()
	}

	return account, nil
}

func (b *TestBackend) ResolveAccount(ctx scope.Context, namespace, id string) (proto.Account, error) {
	b.Lock()
	defer b.Unlock()

	key := fmt.Sprintf("%s:%s", namespace, id)
	accountID, ok := b.accountIDs[key]
	if !ok {
		return nil, proto.ErrAccountNotFound
	}
	return b.accounts[accountID], nil
}

func (b *TestBackend) GetAccount(ctx scope.Context, id snowflake.Snowflake) (proto.Account, error) {
	b.Lock()
	defer b.Unlock()

	account, ok := b.accounts[id.String()]
	if !ok {
		return nil, proto.ErrAccountNotFound
	}
	return account, nil
}
