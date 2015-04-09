package mock

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type memRoom struct {
	sync.Mutex

	name         string
	version      string
	log          *memLog
	agentBans    map[string]time.Time
	ipBans       map[string]time.Time
	identities   map[string]proto.Identity
	live         map[string][]proto.Session
	capabilities map[string]security.Capability

	key *roomKey
}

func newMemRoom(name, version string) *memRoom {
	return &memRoom{
		name:         name,
		version:      version,
		log:          newMemLog(),
		agentBans:    map[string]time.Time{},
		ipBans:       map[string]time.Time{},
		capabilities: map[string]security.Capability{},
	}
}

func (r *memRoom) Version() string { return r.version }

func (r *memRoom) GetMessage(ctx scope.Context, id snowflake.Snowflake) (*proto.Message, error) {
	return r.log.GetMessage(ctx, id)
}

func (r *memRoom) Latest(ctx scope.Context, n int, before snowflake.Snowflake) ([]proto.Message, error) {
	return r.log.Latest(ctx, n, before)
}

func (r *memRoom) Join(ctx scope.Context, session proto.Session) error {
	client := &proto.Client{}
	if !client.FromContext(ctx) {
		return fmt.Errorf("client data not found in scope")
	}

	r.Lock()
	defer r.Unlock()

	if r.identities == nil {
		r.identities = map[string]proto.Identity{}
	}
	if r.live == nil {
		r.live = map[string][]proto.Session{}
	}

	ident := session.Identity()
	id := ident.ID()

	if banned, ok := r.agentBans[client.AgentID]; ok && banned.After(time.Now()) {
		return proto.ErrAccessDenied
	}

	if banned, ok := r.ipBans[client.IP]; ok && banned.After(time.Now()) {
		return proto.ErrAccessDenied
	}

	if _, ok := r.identities[id]; !ok {
		r.identities[id] = ident
	}

	r.live[id] = append(r.live[id], session)
	return r.broadcast(ctx, proto.JoinType,
		proto.PresenceEvent(*session.View()), session)
}

func (r *memRoom) Part(ctx scope.Context, session proto.Session) error {
	r.Lock()
	defer r.Unlock()

	ident := session.Identity()
	id := ident.ID()
	live := r.live[id]
	for i, s := range live {
		if s == session {
			copy(live[i:], live[i+1:])
			r.live[id] = live[:len(live)-1]
		}
	}
	if len(r.live[id]) == 0 {
		delete(r.live, id)
		delete(r.identities, id)
	}
	return r.broadcast(ctx, proto.PartType,
		proto.PresenceEvent(*session.View()), session)
}

func (r *memRoom) Send(ctx scope.Context, session proto.Session, message proto.Message) (
	proto.Message, error) {

	r.Lock()
	defer r.Unlock()

	msg := proto.Message{
		ID:       message.ID,
		UnixTime: proto.Time(message.ID.Time()),
		Parent:   message.Parent,
		Sender:   message.Sender,
		Content:  message.Content,
	}
	r.log.post(&msg)
	return msg, r.broadcast(ctx, proto.SendType, msg, session)
}

func (r *memRoom) EditMessage(
	ctx scope.Context, session proto.Session, edit proto.EditMessageCommand) error {

	msg, err := r.log.edit(edit)
	if err != nil {
		return err
	}

	if edit.Announce {
		editID, err := snowflake.New()
		if err != nil {
			return err
		}
		event := &proto.EditMessageEvent{
			EditID:  editID,
			Message: *msg,
		}
		return r.broadcast(ctx, proto.EditMessageType, event, session)
	}

	return nil
}

func (r *memRoom) broadcast(
	ctx scope.Context, cmdType proto.PacketType, payload interface{}, excluding ...proto.Session) error {

	excMap := make(map[string]struct{}, len(excluding))
	for _, x := range excluding {
		excMap[x.ID()] = struct{}{}
	}

	for _, sessions := range r.live {
		for _, session := range sessions {
			if _, ok := excMap[session.ID()]; ok {
				continue
			}
			if err := session.Send(ctx, cmdType.Event(), payload); err != nil {
				// TODO: accumulate errors
				return err
			}
		}
	}
	return nil
}

func (r *memRoom) Listing(ctx scope.Context) (proto.Listing, error) {
	listing := proto.Listing{}
	for _, sessions := range r.live {
		for _, session := range sessions {
			listing = append(listing, *session.View())
		}
	}
	sort.Sort(listing)
	return listing, nil
}

func (r *memRoom) RenameUser(
	ctx scope.Context, session proto.Session, formerName string) (*proto.NickEvent, error) {

	backend.Logger(ctx).Printf(
		"renaming %s from %s to %s\n", session.ID(), formerName, session.Identity().Name())
	payload := &proto.NickEvent{
		ID:   session.ID(),
		From: formerName,
		To:   session.Identity().Name(),
	}
	return payload, r.broadcast(ctx, proto.NickType, payload, session)
}

func (r *memRoom) MasterKey(ctx scope.Context) (proto.RoomKey, error) {
	if r.key == nil {
		return nil, nil
	}
	return r.key, nil
}

func (r *memRoom) GenerateMasterKey(ctx scope.Context, kms security.KMS) (proto.RoomKey, error) {
	nonce, err := kms.GenerateNonce(security.AES128.KeySize())
	if err != nil {
		return nil, err
	}

	mkey, err := kms.GenerateEncryptedKey(security.AES128, "room", r.name)
	if err != nil {
		return nil, err
	}

	r.key = &roomKey{
		timestamp: time.Now(),
		nonce:     nonce,
		key:       *mkey,
	}
	r.key.id = fmt.Sprintf("%s", r.key.timestamp)
	return r.key, nil
}

func (r *memRoom) SaveCapability(ctx scope.Context, capability security.Capability) error {
	r.Lock()
	r.capabilities[capability.CapabilityID()] = capability
	r.Unlock()
	return nil
}

func (r *memRoom) GetCapability(ctx scope.Context, id string) (security.Capability, error) {
	return r.capabilities[id], nil
}

func (r *memRoom) BanAgent(ctx scope.Context, agentID string, until time.Time) error {
	r.Lock()
	r.agentBans[agentID] = until
	r.Unlock()
	return nil
}

func (r *memRoom) UnbanAgent(ctx scope.Context, agentID string) error {
	r.Lock()
	if _, ok := r.agentBans[agentID]; ok {
		delete(r.agentBans, agentID)
	}
	r.Unlock()
	return nil
}

func (r *memRoom) BanIP(ctx scope.Context, ip string, until time.Time) error {
	r.Lock()
	r.ipBans[ip] = until
	r.Unlock()
	return nil
}

func (r *memRoom) UnbanIP(ctx scope.Context, ip string) error {
	r.Lock()
	if _, ok := r.ipBans[ip]; ok {
		delete(r.ipBans, ip)
	}
	r.Unlock()
	return nil
}

type roomKey struct {
	id        string
	timestamp time.Time
	nonce     []byte
	key       security.ManagedKey
}

func (k *roomKey) KeyID() string                   { return k.id }
func (k *roomKey) Timestamp() time.Time            { return k.timestamp }
func (k *roomKey) Nonce() []byte                   { return k.nonce }
func (k *roomKey) ManagedKey() security.ManagedKey { return k.key.Clone() }
