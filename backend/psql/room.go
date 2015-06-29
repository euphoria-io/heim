package psql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"github.com/go-gorp/gorp"
)

var notImpl = fmt.Errorf("not implemented")
var logger = backend.Logger

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
}

func (r *Room) Bind(b *Backend) *RoomBinding {
	return &RoomBinding{
		Backend: b,
		Room:    r,
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
	*Room
}

func (rb *RoomBinding) GetMessage(ctx scope.Context, id snowflake.Snowflake) (*proto.Message, error) {
	var msg Message

	nDays, err := rb.DbMap.SelectInt("SELECT retention_days FROM room WHERE name = $1", rb.Name)
	if err != nil {
		return nil, err
	}

	err = rb.DbMap.SelectOne(
		&msg,
		"SELECT room, id, previous_edit_id, parent, posted, edited, deleted,"+
			" session_id, sender_id, sender_name, server_id, server_era, content, encryption_key_id"+
			" FROM message WHERE room = $1 AND id = $2",
		rb.Name, id.String())
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

func (rb *RoomBinding) IsValidParent(id snowflake.Snowflake) (bool, error) {
	if id.String() == "" || rb.RetentionDays == 0 {
		return true, nil
	}
	var row struct {
		Posted time.Time
	}
	err := rb.DbMap.SelectOne(&row,
		"SELECT posted FROM message WHERE room = $1 AND id = $2",
		rb.Name, id.String())
	if err != nil {
		// check for nonexistant parent
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	threshold := time.Now().Add(time.Duration(-rb.RetentionDays) * 24 * time.Hour)
	if row.Posted.Before(threshold) {
		return false, nil
	}
	return true, nil
}

func (rb *RoomBinding) Latest(ctx scope.Context, n int, before snowflake.Snowflake) (
	[]proto.Message, error) {

	return rb.Backend.latest(ctx, rb.Room, n, before)
}

func (rb *RoomBinding) Join(ctx scope.Context, session proto.Session) error {
	return rb.Backend.join(ctx, rb.Room, session)
}

func (rb *RoomBinding) Part(ctx scope.Context, session proto.Session) error {
	return rb.Backend.part(ctx, rb.Room, session)
}

func (rb *RoomBinding) Send(ctx scope.Context, session proto.Session, msg proto.Message) (
	proto.Message, error) {

	return rb.Backend.sendMessageToRoom(ctx, rb.Room, msg, session)
}

func (rb *RoomBinding) EditMessage(
	ctx scope.Context, session proto.Session, edit proto.EditMessageCommand) (
	proto.EditMessageReply, error) {

	var reply proto.EditMessageReply

	editID, err := snowflake.New()
	if err != nil {
		return reply, err
	}

	t, err := rb.DbMap.Begin()
	if err != nil {
		return reply, err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	var msg Message
	err = t.SelectOne(
		&msg,
		"SELECT room, id, previous_edit_id, parent, posted, edited, deleted,"+
			" session_id, sender_id, sender_name, server_id, server_era, content, encryption_key_id"+
			" FROM message WHERE room = $1 AND id = $2",
		rb.Name, edit.ID.String())
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
		Room:            rb.Name,
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
			String: session.Identity().ID(),
			Valid:  true,
		}
	}
	if err := t.Insert(entry); err != nil {
		rollback()
		return reply, err
	}

	now := time.Time(proto.Now())
	sets := []string{"edited = $3", "previous_edit_id = $4"}
	args := []interface{}{rb.Name, edit.ID.String(), now, editID.String()}
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

	if err := t.Commit(); err != nil {
		return reply, err
	}

	if edit.Announce {
		event := &proto.EditMessageEvent{
			EditID:  editID,
			Message: msg.ToBackend(),
		}
		err = rb.Backend.broadcast(ctx, rb.Room, proto.EditMessageEventType, event, session)
		if err != nil {
			return reply, err
		}
	}

	reply.EditID = editID
	reply.Deleted = edit.Delete
	return reply, nil
}

func (rb *RoomBinding) Listing(ctx scope.Context) (proto.Listing, error) {
	return rb.Backend.listing(ctx, rb.Room)
}

func (rb *RoomBinding) RenameUser(ctx scope.Context, session proto.Session, formerName string) (
	*proto.NickEvent, error) {

	presence := &Presence{
		Room:      rb.Name,
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
	if _, err := rb.DbMap.Update(presence); err != nil {
		return nil, fmt.Errorf("presence update error: %s", err)
	}

	event := &proto.NickEvent{
		SessionID: session.ID(),
		ID:        session.Identity().ID(),
		From:      formerName,
		To:        session.Identity().Name(),
	}
	return event, rb.Backend.broadcast(ctx, rb.Room, proto.NickEventType, event, session)
}

func (rb *RoomBinding) GenerateMessageKey(ctx scope.Context, kms security.KMS) (
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
			backend.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return nil, err
	}

	if err := transaction.Insert(&rmkb.RoomMessageKey); err != nil {
		if rerr := transaction.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return nil, err
	}

	if err := transaction.Commit(); err != nil {
		return nil, err
	}

	return rmkb, nil
}

func (rb *RoomBinding) MessageKey(ctx scope.Context) (proto.RoomMessageKey, error) {
	var row struct {
		MessageKey
		RoomMessageKey
	}
	err := rb.DbMap.SelectOne(
		&row,
		"SELECT mk.id, mk.encrypted_key, mk.iv, mk.nonce,"+
			" r.room, r.key_id, r.activated, r.expired, r.comment"+
			" FROM master_key mk, room_master_key r"+
			" WHERE r.room = $1 AND mk.id = r.key_id AND r.expired < r.activated"+
			" ORDER BY r.activated DESC LIMIT 1",
		rb.Name)
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
		ContextValue: rb.Room.Name,
	}
	var keyID snowflake.Snowflake
	if err := keyID.FromString(row.KeyID); err != nil {
		return nil, err
	}
	return NewRoomMessageKeyBinding(rb, keyID, msgKey, row.Nonce), nil
}

func (rb *RoomBinding) ManagerKey(ctx scope.Context) (proto.RoomManagerKey, error) {
	return NewRoomManagerKeyBinding(rb), nil
}

func (rb *RoomBinding) BanAgent(ctx scope.Context, agentID string, until time.Time) error {
	ban := &BannedAgent{
		AgentID: agentID,
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

	if err := rb.DbMap.Insert(ban); err != nil {
		return err
	}

	bounceEvent := &proto.BounceEvent{Reason: "banned", AgentID: agentID}
	return rb.broadcast(ctx, rb.Room, proto.BounceEventType, bounceEvent)
}

func (rb *RoomBinding) UnbanAgent(ctx scope.Context, agentID string) error {
	_, err := rb.DbMap.Exec(
		"DELETE FROM banned_agent WHERE agent_id = $1 AND room = $2", agentID, rb.Name)
	return err
}

func (rb *RoomBinding) BanIP(ctx scope.Context, ip string, until time.Time) error {
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

	if err := rb.DbMap.Insert(ban); err != nil {
		return err
	}

	bounceEvent := &proto.BounceEvent{Reason: "banned", IP: ip}
	return rb.broadcast(ctx, rb.Room, proto.BounceEventType, bounceEvent)
}

func (rb *RoomBinding) UnbanIP(ctx scope.Context, ip string) error {
	_, err := rb.DbMap.Exec(
		"DELETE FROM banned_ip WHERE ip = $1 AND room = $2", ip, rb.Name)
	return err
}

func (rb *RoomBinding) Managers(ctx scope.Context) ([]proto.Account, error) {
	type RoomManagerAccount struct {
		Account
	}

	rows, err := rb.Select(
		Account{},
		"SELECT a.id, a.nonce, a.mac, a.encrypted_system_key, a.encrypted_user_key,"+
			" a.encrypted_private_key, a.public_key"+
			" FROM account a, room_manager_capability m"+
			" WHERE m.room = $1 AND m.account_id = a.id AND revoked < granted",
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

func (rb *RoomBinding) getManagerCapability(ctx scope.Context, account proto.Account) (
	*security.PublicKeyCapability, error) {

	c, err := rb.ManagerCapability(ctx, account)
	if err != nil {
		if err == proto.ErrManagerNotFound {
			return nil, proto.ErrAccessDenied
		}
		return nil, err
	}
	return &security.PublicKeyCapability{Capability: c}, nil
}

func (rb *RoomBinding) ManagerCapability(ctx scope.Context, manager proto.Account) (
	security.Capability, error) {

	var c Capability
	err := rb.SelectOne(
		&c,
		"SELECT c.id, c.nonce, c.encrypted_private_data, c.public_data"+
			" FROM capability c, room_manager_capability rm"+
			" WHERE rm.room = $1 AND c.id = rm.capability_id AND c.account_id = $2"+
			" AND rm.revoked < rm.granted",
		rb.Name, manager.ID().String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrManagerNotFound
		}
	}
	return &c, nil
}

func (rb *RoomBinding) AddManager(
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

func (rb *RoomBinding) RemoveManager(
	ctx scope.Context, actor proto.Account, actorKey *security.ManagedKey,
	formerManager proto.Account) error {

	t, err := rb.Backend.DbMap.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
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

func (rb *RoomBinding) UpgradeRoom(ctx scope.Context, kms security.KMS) error {
	if len(rb.Room.Nonce) > 0 {
		return fmt.Errorf("already upgraded")
	}

	msgKey, err := rb.MessageKey(ctx)
	if err != nil {
		return err
	}

	sec, err := proto.NewRoomSecurity(kms, rb.Room.Name)
	if err != nil {
		return err
	}

	nonce := sec.Nonce
	if msgKey != nil {
		nonce = msgKey.Nonce()
	}

	resp, err := rb.Backend.DbMap.Exec(
		`UPDATE room SET pk_nonce = $1, pk_mac = $2, pk_iv = $3,`+
			` encrypted_management_key = $4, encrypted_private_key = $5, public_key = $6`+
			` WHERE name = $7`,
		nonce, sec.MAC, sec.KeyPair.IV,
		sec.KeyEncryptingKey.Ciphertext, sec.KeyPair.EncryptedPrivateKey, sec.KeyPair.PublicKey,
		rb.Room.Name)
	if err != nil {
		return err
	}
	n, err := resp.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return proto.ErrRoomNotFound
	}
	return nil
}
