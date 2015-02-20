package mock

import (
	"sort"
	"sync"
	"time"

	"heim/backend"
	"heim/proto"
	"heim/proto/security"
	"heim/proto/snowflake"

	"golang.org/x/net/context"
)

type memRoom struct {
	sync.Mutex

	name         string
	version      string
	log          *memLog
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
		capabilities: map[string]security.Capability{},
	}
}

func (r *memRoom) Version() string { return r.version }

func (r *memRoom) Latest(ctx context.Context, n int, before snowflake.Snowflake) (
	[]proto.Message, error) {

	return r.log.Latest(ctx, n, before)
}

func (r *memRoom) Join(ctx context.Context, session proto.Session) error {
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

	if _, ok := r.identities[id]; !ok {
		r.identities[id] = ident
	}

	r.live[id] = append(r.live[id], session)
	return r.broadcast(ctx, proto.JoinType,
		proto.PresenceEvent(*session.Identity().View()), session)
}

func (r *memRoom) Part(ctx context.Context, session proto.Session) error {
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
		proto.PresenceEvent(*session.Identity().View()), session)
}

func (r *memRoom) Send(ctx context.Context, session proto.Session, message proto.Message) (
	proto.Message, error) {

	r.Lock()
	defer r.Unlock()

	msg := proto.Message{
		ID:       message.ID,
		UnixTime: message.ID.Time().Unix(),
		Parent:   message.Parent,
		Sender:   message.Sender,
		Content:  message.Content,
	}
	r.log.post(&msg)
	return msg, r.broadcast(ctx, proto.SendType, msg, session)
}

func (r *memRoom) broadcast(
	ctx context.Context, cmdType proto.PacketType, payload interface{},
	excluding ...proto.Session) error {

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

func (r *memRoom) Listing(ctx context.Context) (proto.Listing, error) {
	listing := proto.Listing{}
	for _, sessions := range r.live {
		for _, session := range sessions {
			listing = append(listing, *session.Identity().View())
		}
	}
	sort.Sort(listing)
	return listing, nil
}

func (r *memRoom) RenameUser(
	ctx context.Context, session proto.Session, formerName string) (*proto.NickEvent, error) {
	backend.Logger(ctx).Printf(
		"renaming %s from %s to %s\n", session.ID(), formerName, session.Identity().Name())
	payload := &proto.NickEvent{
		ID:   session.Identity().ID(),
		From: formerName,
		To:   session.Identity().Name(),
	}
	return payload, r.broadcast(ctx, proto.NickType, payload, session)
}

func (r *memRoom) MasterKey(ctx context.Context) (proto.RoomKey, error) {
	if r.key == nil {
		return nil, nil
	}
	return r.key, nil
}

func (r *memRoom) GenerateMasterKey(ctx context.Context, kms security.KMS) (proto.RoomKey, error) {
	nonce, err := kms.GenerateNonce(security.AES128.KeySize())
	if err != nil {
		return nil, err
	}

	mkey, err := kms.GenerateEncryptedKey(security.AES128)
	if err != nil {
		return nil, err
	}

	r.key = &roomKey{
		timestamp: time.Now(),
		nonce:     nonce,
		key:       *mkey,
	}
	return r.key, nil
}

func (r *memRoom) SaveCapability(ctx context.Context, capability security.Capability) error {
	r.Lock()
	r.capabilities[capability.CapabilityID()] = capability
	r.Unlock()
	return nil
}

func (r *memRoom) GetCapability(ctx context.Context, id string) (security.Capability, error) {
	return r.capabilities[id], nil
}

type roomKey struct {
	timestamp time.Time
	nonce     []byte
	key       security.ManagedKey
}

func (k *roomKey) Timestamp() time.Time            { return k.timestamp }
func (k *roomKey) Nonce() []byte                   { return k.nonce }
func (k *roomKey) ManagedKey() security.ManagedKey { return k.key.Clone() }
