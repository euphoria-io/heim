package mock

import (
	"sync"
	"time"

	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/jobs"
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
	agentBans      map[proto.UserID]time.Time
	et             EmailTracker
	ipBans         map[string]time.Time
	js             JobService
	otps           map[snowflake.Snowflake]*proto.OTP
	resetReqs      map[snowflake.Snowflake]*proto.PasswordResetRequest
	rooms          map[string]proto.ManagedRoom
	version        string
}

func (b *TestBackend) AccountManager() proto.AccountManager { return &accountManager{b: b} }
func (b *TestBackend) AgentTracker() proto.AgentTracker     { return &agentTracker{b} }
func (b *TestBackend) EmailTracker() proto.EmailTracker     { return &b.et }
func (b *TestBackend) Jobs() jobs.JobService                { return &b.js }

func (b *TestBackend) Close() {}

func (b *TestBackend) Version() string { return b.version }

func (b *TestBackend) GetRoom(ctx scope.Context, name string) (proto.ManagedRoom, error) {
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
	proto.ManagedRoom, error) {

	b.Lock()
	defer b.Unlock()

	if b.rooms == nil {
		b.rooms = map[string]proto.ManagedRoom{}
	}

	room, err := NewRoom(ctx, kms, private, name, b.version, managers...)
	if err != nil {
		return nil, err
	}

	b.rooms[name] = room
	return room, nil
}

func (b *TestBackend) Peers() []cluster.PeerDesc { return nil }

func (b *TestBackend) banAgent(ctx scope.Context, agentID proto.UserID, until time.Time) error {
	if b.agentBans == nil {
		b.agentBans = map[proto.UserID]time.Time{agentID: until}
	} else {
		b.agentBans[agentID] = until
	}
	return nil
}

func (b *TestBackend) unbanAgent(ctx scope.Context, agentID proto.UserID) error {
	if _, ok := b.agentBans[agentID]; ok {
		delete(b.agentBans, agentID)
	}
	return nil
}

func (b *TestBackend) banIP(ctx scope.Context, ip string, until time.Time) error {
	if b.ipBans == nil {
		b.ipBans = map[string]time.Time{ip: until}
	} else {
		b.ipBans[ip] = until
	}
	return nil
}

func (b *TestBackend) unbanIP(ctx scope.Context, ip string) error {
	if _, ok := b.ipBans[ip]; ok {
		delete(b.ipBans, ip)
	}
	return nil
}

func (b *TestBackend) Ban(ctx scope.Context, ban proto.Ban, until time.Time) error {
	b.Lock()
	defer b.Unlock()

	switch {
	case ban.IP != "":
		return b.banIP(ctx, ban.IP, until)
	case ban.ID != "":
		return b.banAgent(ctx, ban.ID, until)
	default:
		return nil
	}
}

func (b *TestBackend) Unban(ctx scope.Context, ban proto.Ban) error {
	b.Lock()
	defer b.Unlock()

	switch {
	case ban.IP != "":
		return b.unbanIP(ctx, ban.IP)
	case ban.ID != "":
		return b.unbanAgent(ctx, ban.ID)
	default:
		return nil
	}
}

func isExcluded(toCheck proto.Session, excluding []proto.Session) bool {
	for _, excSess := range excluding {
		if toCheck.ID() == excSess.ID() {
			return true
		}
	}
	return false
}

func (b *TestBackend) NotifyUser(ctx scope.Context, userID proto.UserID, packetType proto.PacketType, payload interface{}, excluding ...proto.Session) error {
	kind, id := userID.Parse()
	for _, room := range b.rooms {
		mRoom, _ := room.(*memRoom)
		for u, sessList := range mRoom.live {
			for _, sess := range sessList {
				if u == userID || (kind == "agent" && sess.AgentID() == id) {
					if !isExcluded(sess, excluding) {
						if err := sess.Send(ctx, packetType, payload); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}
