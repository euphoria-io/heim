package mock

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"encoding/json"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type memRoom struct {
	m sync.Mutex

	name            string
	version         string
	log             *memLog
	agentBans       map[string]time.Time
	ipBans          map[string]time.Time
	identities      map[string]proto.Identity
	managers        map[string]string
	managerAccounts map[string]proto.Account
	live            map[string][]proto.Session
	capabilities    map[string]security.Capability

	sec        *proto.RoomSecurity
	messageKey *roomMessageKey
	managerKey *roomManagerKey
}

func NewRoom(kms security.KMS, name, version string, managers ...proto.Account) (proto.Room, error) {
	sec, err := proto.NewRoomSecurity(kms, name)
	if err != nil {
		return nil, err
	}

	managerKey := sec.KeyEncryptingKey.Clone()
	if err := kms.DecryptKey(&managerKey); err != nil {
		return nil, err
	}
	roomKeyPair, err := sec.Unlock(&managerKey)
	if err != nil {
		return nil, err
	}

	room := &memRoom{
		name:            name,
		version:         version,
		log:             newMemLog(),
		agentBans:       map[string]time.Time{},
		ipBans:          map[string]time.Time{},
		managers:        map[string]string{},
		managerAccounts: map[string]proto.Account{},
		capabilities:    map[string]security.Capability{},
		sec:             sec,
	}

	for _, manager := range managers {
		kp := manager.KeyPair()
		c, err := security.GrantPublicKeyCapability(kms, roomKeyPair, &kp, nil, managerKey.Plaintext)
		if err != nil {
			return nil, err
		}
		room.capabilities[c.CapabilityID()] = c
		room.managers[manager.ID().String()] = c.CapabilityID()
		room.managerAccounts[manager.ID().String()] = manager
	}

	return room, nil
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

	r.m.Lock()
	defer r.m.Unlock()

	if r.identities == nil {
		r.identities = map[string]proto.Identity{}
	}
	if r.live == nil {
		r.live = map[string][]proto.Session{}
	}

	ident := session.Identity()
	id := ident.ID()

	if banned, ok := r.agentBans[client.Agent.IDString()]; ok && banned.After(time.Now()) {
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
	r.m.Lock()
	defer r.m.Unlock()

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

	r.m.Lock()
	defer r.m.Unlock()

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

	editID, err := snowflake.New()
	if err != nil {
		return err
	}

	msg, err := r.log.edit(edit)
	if err != nil {
		return err
	}

	if edit.Announce {
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
		if x != nil {
			excMap[x.ID()] = struct{}{}
		}
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
		SessionID: session.ID(),
		ID:        session.Identity().ID(),
		From:      formerName,
		To:        session.Identity().Name(),
	}
	return payload, r.broadcast(ctx, proto.NickType, payload, session)
}

func (r *memRoom) MessageKey(ctx scope.Context) (proto.RoomMessageKey, error) {
	if r.messageKey == nil {
		return nil, nil
	}
	return r.messageKey, nil
}

func (r *memRoom) ManagerKey(ctx scope.Context) (proto.RoomManagerKey, error) {
	return &roomManagerKey{r.sec}, nil
}

func (r *memRoom) GenerateMessageKey(ctx scope.Context, kms security.KMS) (proto.RoomMessageKey, error) {
	nonce, err := kms.GenerateNonce(security.AES128.KeySize())
	if err != nil {
		return nil, err
	}

	mkey, err := kms.GenerateEncryptedKey(security.AES128, "room", r.name)
	if err != nil {
		return nil, err
	}

	r.messageKey = &roomMessageKey{
		timestamp: time.Now(),
		nonce:     nonce,
		key:       *mkey,
	}
	r.messageKey.id = fmt.Sprintf("%s", r.messageKey.timestamp)
	return r.messageKey, nil
}

func (r *memRoom) SaveCapability(ctx scope.Context, capability security.Capability) error {
	r.m.Lock()
	r.capabilities[capability.CapabilityID()] = capability
	r.m.Unlock()
	return nil
}

func (r *memRoom) GetCapability(ctx scope.Context, id string) (security.Capability, error) {
	return r.capabilities[id], nil
}

func (r *memRoom) BanAgent(ctx scope.Context, agentID string, until time.Time) error {
	r.m.Lock()
	r.agentBans[agentID] = until
	r.m.Unlock()
	return nil
}

func (r *memRoom) UnbanAgent(ctx scope.Context, agentID string) error {
	r.m.Lock()
	if _, ok := r.agentBans[agentID]; ok {
		delete(r.agentBans, agentID)
	}
	r.m.Unlock()
	return nil
}

func (r *memRoom) BanIP(ctx scope.Context, ip string, until time.Time) error {
	r.m.Lock()
	r.ipBans[ip] = until
	r.m.Unlock()
	return nil
}

func (r *memRoom) UnbanIP(ctx scope.Context, ip string) error {
	r.m.Lock()
	if _, ok := r.ipBans[ip]; ok {
		delete(r.ipBans, ip)
	}
	r.m.Unlock()
	return nil
}

func (r *memRoom) IsValidParent(id snowflake.Snowflake) (bool, error) {
	// TODO: actually check log to see if it is valid.
	return true, nil
}

func (r *memRoom) Managers(ctx scope.Context) ([]proto.Account, error) {
	r.m.Lock()
	managers := make([]proto.Account, 0, len(r.managerAccounts))
	for _, manager := range r.managerAccounts {
		managers = append(managers, manager)
	}
	r.m.Unlock()
	return managers, nil
}

func (r *memRoom) verifyManager(actor proto.Account, actorKey *security.ManagedKey) (
	*security.PublicKeyCapability, error) {

	// Verify that actorKey unlocks actor's keypair. In a real implementation,
	// we would take an additional step of verifying against a capability.
	kp := actor.KeyPair()
	if err := kp.Decrypt(actorKey); err != nil {
		return nil, err
	}

	// Verify actor is a manager.
	cid, ok := r.managers[actor.ID().String()]
	if !ok {
		return nil, proto.ErrAccessDenied
	}
	c, ok := r.capabilities[cid]
	if !ok {
		return nil, proto.ErrAccessDenied
	}

	return c.(*security.PublicKeyCapability), nil
}

func (r *memRoom) AddManager(
	ctx scope.Context, kms security.KMS, actor proto.Account, actorKey *security.ManagedKey,
	newManager proto.Account) error {

	r.m.Lock()
	defer r.m.Unlock()

	// Verify actor.
	pkCap, err := r.verifyManager(actor, actorKey)
	if err != nil {
		return err
	}

	// Add new manager.
	subjectKeyPair := r.sec.KeyPair.Clone()
	actorKeyPair, err := actor.Unlock(actorKey)
	if err != nil {
		return err
	}
	secretJSON, err := pkCap.DecryptPayload(&subjectKeyPair, actorKeyPair)
	if err != nil {
		return err
	}

	var secret []byte
	if err := json.Unmarshal(secretJSON, &secret); err != nil {
		return err
	}

	managerKey := &security.ManagedKey{
		KeyType:   security.AES128,
		Plaintext: secret,
	}
	unlockedSubjectKeyPair, err := r.sec.Unlock(managerKey)
	if err != nil {
		return err
	}

	newManagerKeyPair := newManager.KeyPair()

	nc, err := security.GrantPublicKeyCapability(
		kms, unlockedSubjectKeyPair, &newManagerKeyPair, nil, secret)
	if err != nil {
		return err
	}

	r.capabilities[nc.CapabilityID()] = nc
	r.managers[newManager.ID().String()] = nc.CapabilityID()
	r.managerAccounts[newManager.ID().String()] = newManager

	return nil
}

func (r *memRoom) RemoveManager(
	ctx scope.Context, actor proto.Account, actorKey *security.ManagedKey,
	formerManager proto.Account) error {

	r.m.Lock()
	defer r.m.Unlock()

	// Verify actor.
	if _, err := r.verifyManager(actor, actorKey); err != nil {
		return err
	}

	// Verify target is a manager.
	cid, ok := r.managers[formerManager.ID().String()]
	if !ok {
		return proto.ErrManagerNotFound
	}

	// Remove.
	delete(r.capabilities, cid)
	delete(r.managers, formerManager.ID().String())
	delete(r.managerAccounts, formerManager.ID().String())

	return nil
}

type roomMessageKey struct {
	id        string
	timestamp time.Time
	nonce     []byte
	key       security.ManagedKey
}

func (k *roomMessageKey) KeyID() string                   { return k.id }
func (k *roomMessageKey) Timestamp() time.Time            { return k.timestamp }
func (k *roomMessageKey) Nonce() []byte                   { return k.nonce }
func (k *roomMessageKey) ManagedKey() security.ManagedKey { return k.key.Clone() }

type roomManagerKey struct {
	*proto.RoomSecurity
}

func (r *roomManagerKey) KeyPair() security.ManagedKeyPair { return r.RoomSecurity.KeyPair }
