package psql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"encoding/json"

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

func (r *Room) generateMessageKey(kms security.KMS) (*RoomMessageKeyBinding, error) {
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

	rmkb := &RoomMessageKeyBinding{
		MessageKey: MessageKey{
			ID:           keyID.String(),
			EncryptedKey: mkey.Ciphertext,
			IV:           mkey.IV,
			Nonce:        nonce,
		},
		RoomMessageKey: RoomMessageKey{
			Room:      r.Name,
			KeyID:     keyID.String(),
			Activated: time.Now(),
		},
	}
	return rmkb, nil
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
	ctx scope.Context, session proto.Session, edit proto.EditMessageCommand) error {

	editID, err := snowflake.New()
	if err != nil {
		return err
	}

	t, err := rb.DbMap.Begin()
	if err != nil {
		return err
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
		return err
	}

	if msg.PreviousEditID.Valid && msg.PreviousEditID.String != edit.PreviousEditID.String() {
		rollback()
		return proto.ErrEditInconsistent
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
		return err
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
		return err
	}

	if err := t.Commit(); err != nil {
		return err
	}

	if edit.Announce {
		event := &proto.EditMessageEvent{
			EditID:  editID,
			Message: msg.ToBackend(),
		}
		return rb.Backend.broadcast(ctx, rb.Room, proto.EditMessageEventType, event, session)
	}

	return nil
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

	rmkb, err := rb.Room.generateMessageKey(kms)
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
	rmkb := &RoomMessageKeyBinding{}
	err := rb.DbMap.SelectOne(
		rmkb,
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
	return rmkb, nil
}

func (rb *RoomBinding) ManagerKey(ctx scope.Context) (proto.RoomManagerKey, error) {
	return &RoomManagerKeyBinding{rb.Room}, nil
}

func (rb *RoomBinding) SaveCapability(ctx scope.Context, capability security.Capability) error {
	transaction, err := rb.DbMap.Begin()
	if err != nil {
		return err
	}

	rcb := &RoomCapabilityBinding{
		Capability: Capability{
			ID:                   capability.CapabilityID(),
			EncryptedPrivateData: capability.EncryptedPayload(),
			PublicData:           capability.PublicPayload(),
		},
		RoomCapability: RoomCapability{
			Room:         rb.Name,
			CapabilityID: capability.CapabilityID(),
			Granted:      time.Now(),
		},
	}

	if err := transaction.Insert(&rcb.Capability, &rcb.RoomCapability); err != nil {
		if rerr := transaction.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return err
	}

	if err := transaction.Commit(); err != nil {
		return err
	}

	backend.Logger(ctx).Printf("added capability %s to room %s", capability.CapabilityID(), rb.Name)
	return nil
}

func (rb *RoomBinding) GetCapability(ctx scope.Context, id string) (security.Capability, error) {
	rcb := &RoomCapabilityBinding{}

	backend.Logger(ctx).Printf("looking up capability %s in room %s", id, rb.Name)
	err := rb.DbMap.SelectOne(
		rcb,
		"SELECT c.id, c.nonce, c.encrypted_private_data, c.public_data,"+
			" r.room, r.capability_id, r.granted, r.revoked"+
			" FROM capability c, room_capability r"+
			" WHERE r.room = $1 AND r.capability_id = $2 AND c.id = $2 AND r.revoked < r.granted",
		rb.Name, id)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return rcb, nil
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
			" FROM account a, room_manager m"+
			" WHERE m.room = $1 AND m.account_id = a.id",
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

func (rb *RoomBinding) getManagerCapability(account proto.Account) (
	*security.PublicKeyCapability, error) {

	var c Capability
	err := rb.SelectOne(
		&c,
		"SELECT c.id, c.nonce, c.encrypted_private_data, c.public_data"+
			" FROM capability c, room_capability rc, room_manager rm"+
			" WHERE rm.room = $1 AND rm.account_id = $2 AND rm.capability_id = rc.capability_id"+
			" AND rc.revoked < rc.granted AND c.id = rc.capability_id",
		rb.Name, account.ID().String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrAccessDenied
		}
	}
	return &security.PublicKeyCapability{&c}, nil
}

func (rb *RoomBinding) AddManager(
	ctx scope.Context, kms security.KMS, actor proto.Account, actorKey *security.ManagedKey,
	newManager proto.Account) error {

	// Get the actor's manager capability.
	pkCap, err := rb.getManagerCapability(actor)
	if err != nil {
		return err
	}

	// Extract the manager key from the actor's capability.
	rmkb := &RoomManagerKeyBinding{rb.Room}
	subjectKeyPair := rmkb.KeyPair()
	actorKeyPair, err := actor.Unlock(actorKey)
	if err != nil {
		return err
	}
	secretJSON, err := pkCap.DecryptPayload(&subjectKeyPair, actorKeyPair)
	if err != nil {
		return err
	}
	managerKey := &security.ManagedKey{
		KeyType: proto.RoomManagerKeyType,
	}
	if err := json.Unmarshal(secretJSON, &managerKey.Plaintext); err != nil {
		return err
	}

	// Verify manager key is correct.
	unlockedSubjectKeyPair, err := rmkb.Unlock(managerKey)
	if err != nil {
		return fmt.Errorf("room security unlock error: %s", err)
	}

	// Grant capability for new manager.
	newManagerKeyPair := newManager.KeyPair()
	capability, err := security.GrantPublicKeyCapability(
		kms, rb.Room.Nonce, unlockedSubjectKeyPair, &newManagerKeyPair, nil, managerKey.Plaintext)
	if err != nil {
		return err
	}

	// Save the results.
	c := &Capability{
		ID:                   capability.CapabilityID(),
		NonceBytes:           capability.Nonce(),
		EncryptedPrivateData: capability.EncryptedPayload(),
		PublicData:           capability.PublicPayload(),
	}
	rc := &RoomCapability{
		Room:         rb.Name,
		CapabilityID: capability.CapabilityID(),
		Granted:      time.Now(),
	}
	rm := &RoomManager{
		Room:         rb.Name,
		AccountID:    newManager.ID().String(),
		CapabilityID: capability.CapabilityID(),
	}
	t, err := rb.DbMap.Begin()
	if err != nil {
		return err
	}
	if err := t.Insert(c, rc, rm); err != nil {
		if rerr := t.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		if strings.HasPrefix(err.Error(), "pq: duplicate key value") {
			return nil
		}
		return err
	}
	if err := t.Commit(); err != nil {
		return err
	}
	return nil
}

func (rb *RoomBinding) RemoveManager(
	ctx scope.Context, actor proto.Account, actorKey *security.ManagedKey,
	formerManager proto.Account) error {

	// Get the actor's manager capability.
	pkCap, err := rb.getManagerCapability(actor)
	if err != nil {
		return err
	}

	// Extract the manager key from the actor's capability.
	rmkb := &RoomManagerKeyBinding{rb.Room}
	subjectKeyPair := rmkb.KeyPair()
	actorKeyPair, err := actor.Unlock(actorKey)
	if err != nil {
		return err
	}
	secretJSON, err := pkCap.DecryptPayload(&subjectKeyPair, actorKeyPair)
	if err != nil {
		return fmt.Errorf("public key capability error: %s", err)
	}
	managerKey := &security.ManagedKey{
		KeyType: proto.RoomManagerKeyType,
	}
	if err := json.Unmarshal(secretJSON, &managerKey.Plaintext); err != nil {
		return err
	}

	// Verify manager key is correct.
	if _, err = rmkb.Unlock(managerKey); err != nil {
		return fmt.Errorf("room security unlock error: %s", err)
	}

	// Look up and remove the target capability.
	t, err := rb.DbMap.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	var rm RoomManager
	err = t.SelectOne(
		&rm,
		"SELECT room, account_id, capability_id FROM room_manager WHERE room = $1 AND account_id = $2",
		rb.Name, formerManager.ID().String())
	if err != nil {
		rollback()
		if err == sql.ErrNoRows {
			return proto.ErrManagerNotFound
		}
		return err
	}

	result, err := t.Exec("DELETE FROM capability WHERE id = $1", rm.CapabilityID)
	if err != nil {
		rollback()
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		rollback()
		return fmt.Errorf("unable to determine number of affected rows: %s", err)
	}
	if n < 1 {
		rollback()
		return fmt.Errorf("no rows were affected")
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}
