package psql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"gopkg.in/gorp.v1"
)

var notImpl = fmt.Errorf("not implemented")
var global *RoomBinding

type Room struct {
	Name                   string
	FoundedBy              string `db:"founded_by"`
	RetentionDays          int    `db:"retention_days"`
	Nonce                  []byte `db:"pk_nonce"`
	MAC                    []byte `db:"pk_mac"`
	IV                     []byte `db:"pk_iv"`
	EncryptedManagementKey []byte `db:"encrypted_management_key"`
	EncryptedPrivateKey    []byte `db:"encrypted_private_key"`
	PublicKey              []byte `db:"public_key"`
	MinAgentAge            int64  `db:"min_agent_age"`
}

func (r *Room) Bind(b *Backend) *ManagedRoomBinding {
	return &ManagedRoomBinding{
		RoomBinding: RoomBinding{
			RoomName: r.Name,
			Backend:  b,
		},
		Room: r,
	}
}

func (r *Room) generateMessageKey(b *Backend, kms security.KMS) (*RoomMessageKeyBinding, error) {
	// Generate unique ID for storing new key in DB.
	keyID, err := snowflake.New()
	if err != nil {
		return nil, err
	}

	// Use KMS to generate nonce and key.
	nonce, err := kms.GenerateNonce(proto.RoomManagerKeyType.KeySize())
	if err != nil {
		return nil, err
	}

	mkey, err := kms.GenerateEncryptedKey(proto.RoomManagerKeyType, "room", r.Name)
	if err != nil {
		return nil, err
	}

	return NewRoomMessageKeyBinding(r.Bind(b), keyID, mkey, nonce), nil
}

type RoomBinding struct {
	*Backend
	RoomName     string
	messageKeyID sql.NullString
}

func (rb *RoomBinding) broadcast(
	ctx scope.Context, db gorp.SqlExecutor, packetType proto.PacketType, payload interface{}, exclude ...proto.Session) error {

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	packet := &proto.Packet{Type: packetType, Data: json.RawMessage(encodedPayload)}
	broadcastMsg := BroadcastMessage{
		Event:   packet,
		Exclude: make([]string, 0, len(exclude)),
	}
	if rb != nil {
		broadcastMsg.Room = rb.RoomName
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
	_, err = db.Exec(fmt.Sprintf("NOTIFY broadcast, '%s'", escaped))
	return err
}

func (rb *RoomBinding) ID() string { return rb.RoomName }

func (rb *RoomBinding) GetMessage(ctx scope.Context, id snowflake.Snowflake) (*proto.Message, error) {
	var msg Message

	nDays, err := rb.DbMap.SelectInt("SELECT retention_days FROM room WHERE name = $1", rb.RoomName)
	if err != nil {
		return nil, err
	}

	cols, err := allColumns(rb.DbMap, Message{}, "")
	if err != nil {
		return nil, err
	}
	err = rb.DbMap.SelectOne(&msg, fmt.Sprintf("SELECT %s FROM message WHERE room = $1 AND id = $2", cols), rb.RoomName, id.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrMessageNotFound
		}
		return nil, err
	}
	if nDays > 0 {
		threshold := time.Now().Add(time.Duration(-nDays) * 24 * time.Hour)
		if msg.Posted.Before(threshold) {
			return nil, proto.ErrMessageNotFound
		}
	}
	m := msg.ToBackend()
	return &m, nil
}

func (rb *RoomBinding) getParentPostTime(id snowflake.Snowflake) (time.Time, error) {
	var row struct {
		Posted time.Time
	}
	err := rb.DbMap.SelectOne(&row,
		"SELECT posted FROM message WHERE room = $1 AND id = $2",
		rb.RoomName, id.String())
	if err != nil {
		return time.Time{}, err
	}
	return row.Posted, nil
}

func (rb *RoomBinding) IsValidParent(id snowflake.Snowflake) (bool, error) {
	if id.String() == "" {
		return true, nil
	}
	if _, err := rb.getParentPostTime(id); err != nil {
		// check for nonexistant parent
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (rb *RoomBinding) Latest(ctx scope.Context, n int, before snowflake.Snowflake) (
	[]proto.Message, error) {

	return rb.Backend.latest(ctx, rb, n, before)
}

func (rb *RoomBinding) Join(ctx scope.Context, session proto.Session) error {
	return rb.Backend.join(ctx, rb, session)
}

func (rb *RoomBinding) Part(ctx scope.Context, session proto.Session) error {
	return rb.Backend.part(ctx, rb, session)
}

func (rb *RoomBinding) Send(ctx scope.Context, session proto.Session, msg proto.Message) (
	proto.Message, error) {

	return rb.Backend.sendMessageToRoom(ctx, rb, msg, session)
}

func (rb *RoomBinding) EditMessage(
	ctx scope.Context, session proto.Session, edit proto.EditMessageCommand) (
	proto.EditMessageReply, error) {

	var reply proto.EditMessageReply

	editID, err := snowflake.New()
	if err != nil {
		return reply, err
	}

	cols, err := allColumns(rb.DbMap, Message{}, "")
	if err != nil {
		return reply, err
	}

	t, err := rb.DbMap.Begin()
	if err != nil {
		return reply, err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			logging.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	var msg Message
	err = t.SelectOne(&msg, fmt.Sprintf("SELECT %s FROM message WHERE room = $1 AND id = $2", cols), rb.RoomName, edit.ID.String())
	if err != nil {
		rollback()
		return reply, err
	}

	if msg.PreviousEditID.Valid && msg.PreviousEditID.String != edit.PreviousEditID.String() {
		rollback()
		return reply, proto.ErrEditInconsistent
	}

	entry := &MessageEditLog{
		EditID:          editID.String(),
		Room:            rb.RoomName,
		MessageID:       edit.ID.String(),
		PreviousEditID:  msg.PreviousEditID,
		PreviousContent: msg.Content,
		PreviousParent: sql.NullString{
			String: msg.Parent,
			Valid:  true,
		},
	}
	// TODO: tests pass in a nil session, until we add support for the edit command
	if session != nil {
		entry.EditorID = sql.NullString{
			String: string(session.Identity().ID()),
			Valid:  true,
		}
	}
	if err := t.Insert(entry); err != nil {
		rollback()
		return reply, err
	}

	now := time.Time(proto.Now())
	sets := []string{"edited = $3", "previous_edit_id = $4"}
	args := []interface{}{rb.RoomName, edit.ID.String(), now, editID.String()}
	msg.Edited = gorp.NullTime{Valid: true, Time: now}
	if edit.Content != "" {
		args = append(args, edit.Content)
		sets = append(sets, fmt.Sprintf("content = $%d", len(args)))
		msg.Content = edit.Content
	}
	if edit.Parent != 0 {
		args = append(args, edit.Parent.String())
		sets = append(sets, fmt.Sprintf("parent = $%d", len(args)))
		msg.Parent = edit.Parent.String()
	}
	if edit.Delete != msg.Deleted.Valid {
		if edit.Delete {
			args = append(args, now)
			sets = append(sets, fmt.Sprintf("deleted = $%d", len(args)))
			msg.Deleted = gorp.NullTime{Valid: true, Time: now}
		} else {
			sets = append(sets, "deleted = NULL")
			msg.Deleted.Valid = false
		}
	}
	query := fmt.Sprintf("UPDATE message SET %s WHERE room = $1 AND id = $2", strings.Join(sets, ", "))
	if _, err := t.Exec(query, args...); err != nil {
		rollback()
		return reply, err
	}

	if edit.Announce {
		event := &proto.EditMessageEvent{
			EditID:  editID,
			Message: msg.ToTransmission(),
		}
		err = rb.broadcast(ctx, t, proto.EditMessageEventType, event, session)
		if err != nil {
			rollback()
			return reply, err
		}
	}

	if err := t.Commit(); err != nil {
		return reply, err
	}

	reply.EditID = editID
	reply.Message = msg.ToTransmission()
	return reply, nil
}

func (rb *RoomBinding) Listing(ctx scope.Context) (proto.Listing, error) {
	return rb.Backend.listing(ctx, rb)
}

func (rb *RoomBinding) RenameUser(ctx scope.Context, session proto.Session, formerName string) (
	*proto.NickEvent, error) {

	presence := &Presence{
		Room:      rb.RoomName,
		ServerID:  rb.desc.ID,
		ServerEra: rb.desc.Era,
		SessionID: session.ID(),
		Updated:   time.Now(),
	}
	err := presence.SetFact(&proto.Presence{
		SessionView:    *session.View(),
		LastInteracted: presence.Updated,
	})
	if err != nil {
		return nil, fmt.Errorf("presence marshal error: %s", err)
	}

	t, err := rb.DbMap.Begin()
	if err != nil {
		return nil, err
	}

	if _, err := t.Update(presence); err != nil {
		rollback(ctx, t)
		return nil, fmt.Errorf("presence update error: %s", err)
	}

	event := &proto.NickEvent{
		SessionID: session.ID(),
		ID:        session.Identity().ID(),
		From:      formerName,
		To:        session.Identity().Name(),
	}
	if err := rb.broadcast(ctx, t, proto.NickEventType, event, session); err != nil {
		rollback(ctx, t)
		return nil, err
	}

	if err := t.Commit(); err != nil {
		return nil, err
	}

	return event, nil
}

func (rb *RoomBinding) MessageKeyID(ctx scope.Context) (string, bool, error) { return "", false, nil }

func (rb *ManagedRoomBinding) GenerateMessageKey(ctx scope.Context, kms security.KMS) (
	proto.RoomMessageKey, error) {

	rmkb, err := rb.Room.generateMessageKey(rb.Backend, kms)
	if err != nil {
		return nil, err
	}

	// Insert key and room association into the DB.
	transaction, err := rb.DbMap.Begin()
	if err != nil {
		return nil, err
	}

	if err := transaction.Insert(&rmkb.MessageKey); err != nil {
		if rerr := transaction.Rollback(); rerr != nil {
			logging.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return nil, err
	}

	if err := transaction.Insert(&rmkb.RoomMessageKey); err != nil {
		if rerr := transaction.Rollback(); rerr != nil {
			logging.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return nil, err
	}

	if err := transaction.Commit(); err != nil {
		return nil, err
	}

	return rmkb, nil
}

func (rb *ManagedRoomBinding) MessageKeyID(ctx scope.Context) (string, bool, error) {
	var row struct {
		KeyID string `db:"key_id"`
	}
	err := rb.DbMap.SelectOne(
		&row,
		"SELECT key_id FROM room_master_key WHERE room = $1 AND expired < activated ORDER BY activated DESC LIMIT 1",
		rb.RoomName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return row.KeyID, true, nil
}

func (rb *ManagedRoomBinding) MessageKey(ctx scope.Context) (proto.RoomMessageKey, error) {
	var row struct {
		MessageKey
		RoomMessageKey
	}

	mkCols, err := allColumns(rb.DbMap, row.MessageKey, "mk")
	if err != nil {
		return nil, err
	}
	rCols, err := allColumns(rb.DbMap, row.RoomMessageKey, "r")
	if err != nil {
		return nil, err
	}

	err = rb.DbMap.SelectOne(
		&row,
		fmt.Sprintf("SELECT %s, %s FROM master_key mk, room_master_key r"+
			" WHERE r.room = $1 AND mk.id = r.key_id AND r.expired < r.activated"+
			" ORDER BY r.activated DESC LIMIT 1",
			mkCols, rCols),
		rb.RoomName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	msgKey := &security.ManagedKey{
		KeyType:      proto.RoomMessageKeyType,
		IV:           row.MessageKey.IV,
		Ciphertext:   row.MessageKey.EncryptedKey,
		ContextKey:   "room",
		ContextValue: rb.RoomName,
	}
	var keyID snowflake.Snowflake
	if err := keyID.FromString(row.KeyID); err != nil {
		return nil, err
	}
	return NewRoomMessageKeyBinding(rb, keyID, msgKey, row.Nonce), nil
}

func (rb *RoomBinding) WaitForPart(sessionID string) error {
	rb.Backend.Lock()
	defer rb.Backend.Unlock()

	stillPresent := func() (bool, error) {
		var count int
		err := rb.Backend.SelectOne(&count, "SELECT COUNT(*) FROM presence WHERE session_id = $1", sessionID)
		if err != nil {
			return false, err
		}
		if count == 0 {
			return false, nil
		}
		return true, nil
	}

	p, err := stillPresent()
	if err != nil {
		return err
	}
	if !p {
		return nil
	}

	if rb.Backend.partWaiters == nil {
		rb.Backend.partWaiters = map[string]chan struct{}{}
	}
	c := make(chan struct{}, 1)
	rb.Backend.partWaiters[sessionID] = c
	rb.Backend.Unlock()

	defer func() {
		rb.Backend.Lock()
		delete(rb.Backend.partWaiters, sessionID)
	}()

	for {
		select {
		case <-c:
			return nil
		case <-time.After(10 * time.Millisecond):
			p, err := stillPresent()
			if err != nil {
				return err
			}
			if !p {
				return nil
			}
		}
	}
}

type ManagedRoomBinding struct {
	RoomBinding
	*Room
}

func (rb *ManagedRoomBinding) ManagerKey(ctx scope.Context) (proto.RoomManagerKey, error) {
	return NewRoomManagerKeyBinding(rb), nil
}

func (rb *ManagedRoomBinding) Ban(ctx scope.Context, ban proto.Ban, until time.Time) error {
	switch {
	case ban.ID != "":
		return rb.banAgent(ctx, ban.ID, until)
	case ban.IP != "":
		return rb.banIP(ctx, ban.IP, until)
	default:
		return fmt.Errorf("id or ip must be given")
	}
}

func (rb *ManagedRoomBinding) Unban(ctx scope.Context, ban proto.Ban) error {
	switch {
	case ban.ID != "":
		return rb.unbanAgent(ctx, ban.ID)
	case ban.IP != "":
		return rb.unbanIP(ctx, ban.IP)
	default:
		return fmt.Errorf("id or ip must be given")
	}
}

func (rb *ManagedRoomBinding) banAgent(ctx scope.Context, agentID proto.UserID, until time.Time) error {
	ban := &BannedAgent{
		AgentID: agentID.String(),
		Room: sql.NullString{
			String: rb.Name,
			Valid:  true,
		},
		Created: time.Now(),
		Expires: gorp.NullTime{
			Time:  until,
			Valid: !until.IsZero(),
		},
	}

	// Loop within transaction in read committed mode to simulate UPSERT.
	t, err := rb.DbMap.Begin()
	if err != nil {
		return err
	}
	for {
		// Try to insert; if this fails due to duplicate key value, try to update.
		if err := rb.DbMap.Insert(ban); err != nil {
			if !strings.HasPrefix(err.Error(), "pq: duplicate key value") {
				rollback(ctx, t)
				return err
			}
		} else {
			break
		}
		n, err := rb.DbMap.Update(ban)
		if err != nil {
			rollback(ctx, t)
			return err
		}
		if n > 0 {
			break
		}
	}

	bounceEvent := &proto.BounceEvent{Reason: "banned", AgentID: agentID.String()}
	if err := rb.broadcast(ctx, t, proto.BounceEventType, bounceEvent); err != nil {
		rollback(ctx, t)
		return err
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}

func (rb *ManagedRoomBinding) unbanAgent(ctx scope.Context, agentID proto.UserID) error {
	_, err := rb.DbMap.Exec(
		"DELETE FROM banned_agent WHERE agent_id = $1 AND room = $2", agentID.String(), rb.Name)
	return err
}

func (rb *ManagedRoomBinding) banIP(ctx scope.Context, ip string, until time.Time) error {
	ban := &BannedIP{
		IP: ip,
		Room: sql.NullString{
			String: rb.Name,
			Valid:  true,
		},
		Created: time.Now(),
		Expires: gorp.NullTime{
			Time:  until,
			Valid: !until.IsZero(),
		},
	}

	t, err := rb.DbMap.Begin()
	if err != nil {
		return err
	}

	if err := t.Insert(ban); err != nil {
		rollback(ctx, t)
		return err
	}

	bounceEvent := &proto.BounceEvent{Reason: "banned", IP: ip}
	if err := rb.broadcast(ctx, t, proto.BounceEventType, bounceEvent); err != nil {
		rollback(ctx, t)
		return err
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}

func (rb *ManagedRoomBinding) unbanIP(ctx scope.Context, ip string) error {
	_, err := rb.DbMap.Exec(
		"DELETE FROM banned_ip WHERE ip = $1 AND room = $2", ip, rb.Name)
	return err
}

func (rb *ManagedRoomBinding) Managers(ctx scope.Context) ([]proto.Account, error) {
	type RoomManagerAccount struct {
		Account
	}

	cols, err := allColumns(rb.DbMap, Account{}, "a")
	if err != nil {
		return nil, err
	}

	rows, err := rb.Select(
		Account{},
		fmt.Sprintf(
			"SELECT %s FROM account a, room_manager_capability m WHERE m.room = $1 AND m.account_id = a.id AND revoked < granted",
			cols),
		rb.Name)
	if err != nil {
		return nil, err
	}

	accounts := make([]proto.Account, 0, len(rows))
	for _, row := range rows {
		account, ok := row.(*Account)
		if !ok {
			return nil, fmt.Errorf("expected row of type *Account, got %T", row)
		}
		accounts = append(accounts, account.Bind(rb.Backend))
	}
	return accounts, nil
}

func (rb *ManagedRoomBinding) ManagerCapability(ctx scope.Context, manager proto.Account) (
	security.Capability, error) {

	var c Capability
	cols, err := allColumns(rb.DbMap, c, "c")
	if err != nil {
		return nil, err
	}
	err = rb.SelectOne(
		&c,
		fmt.Sprintf("SELECT %s FROM capability c, room_manager_capability rm"+
			" WHERE rm.room = $1 AND c.id = rm.capability_id AND c.account_id = $2"+
			" AND rm.revoked < rm.granted",
			cols),
		rb.Name, manager.ID().String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrManagerNotFound
		}
	}
	return &c, nil
}

func (rb *ManagedRoomBinding) AddManager(
	ctx scope.Context, kms security.KMS, actor proto.Account, actorKey *security.ManagedKey,
	newManager proto.Account) error {

	rmkb := NewRoomManagerKeyBinding(rb)
	if err := rmkb.GrantToAccount(ctx, kms, actor, actorKey, newManager); err != nil {
		if err == proto.ErrCapabilityNotFound {
			return proto.ErrAccessDenied
		}
		if err.Error() == `pq: duplicate key value violates unique constraint "capability_pkey"` {
			return nil
		}
		return err
	}
	return nil
}

func (rb *ManagedRoomBinding) RemoveManager(
	ctx scope.Context, actor proto.Account, actorKey *security.ManagedKey,
	formerManager proto.Account) error {

	t, err := rb.Backend.DbMap.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			logging.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	rmkb := NewRoomManagerKeyBinding(rb)
	rmkb.SetExecutor(t)

	if _, _, _, err := rmkb.Authority(ctx, actor, actorKey); err != nil {
		rollback()
		if err == proto.ErrCapabilityNotFound {
			return proto.ErrAccessDenied
		}
		return err
	}

	if err := rmkb.RevokeFromAccount(ctx, formerManager); err != nil {
		rollback()
		if err == proto.ErrCapabilityNotFound || err == proto.ErrAccessDenied {
			return proto.ErrManagerNotFound
		}
		return err
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}

func (rb *ManagedRoomBinding) MinAgentAge() time.Duration {
	return time.Duration(time.Duration(rb.Room.MinAgentAge) * time.Second)
}

func (rb *ManagedRoomBinding) IsValidParent(id snowflake.Snowflake) (bool, error) {
	if id.String() == "" || rb.RetentionDays == 0 {
		return true, nil
	}
	posted, err := rb.getParentPostTime(id)
	if err != nil {
		// check for nonexistant parent
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	threshold := time.Now().Add(time.Duration(-rb.RetentionDays) * 24 * time.Hour)
	if posted.Before(threshold) {
		return false, nil
	}
	return true, nil
}
