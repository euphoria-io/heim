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
	Name          string
	FoundedBy     string `db:"founded_by"`
	RetentionDays int    `db:"retention_days"`
}

func (r *Room) Bind(b *Backend) *RoomBinding {
	return &RoomBinding{
		Backend: b,
		Room:    r,
	}
}

type RoomBinding struct {
	*Backend
	*Room
}

func (rb *RoomBinding) GetMessage(ctx scope.Context, id snowflake.Snowflake) (*proto.Message, error) {
	var msg Message
	err := rb.DbMap.SelectOne(
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
	m := msg.ToBackend()
	return &m, nil
}

func (rb *RoomBinding) IsValidParent(id snowflake.Snowflake) (bool, error) {
	if id.String() == "" || rb.RetentionDays == 0 {
		return true, nil
	}
	var parentTime time.Time
	err := rb.DbMap.SelectOne(&parentTime,
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
	if parentTime.Before(threshold) {
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

func (rb *RoomBinding) GenerateMasterKey(ctx scope.Context, kms security.KMS) (proto.RoomKey, error) {

	// Generate unique ID for storing new key in DB.
	keyID, err := snowflake.New()
	if err != nil {
		return nil, err
	}

	// Use KMS to generate nonce and key.
	nonce, err := kms.GenerateNonce(security.AES128.KeySize())
	if err != nil {
		return nil, err
	}

	mkey, err := kms.GenerateEncryptedKey(security.AES128, "room", rb.Name)
	if err != nil {
		return nil, err
	}

	// Insert key and room association into the DB.
	transaction, err := rb.DbMap.Begin()
	if err != nil {
		return nil, err
	}

	rmkb := &RoomMasterKeyBinding{
		MasterKey: MasterKey{
			ID:           keyID.String(),
			EncryptedKey: mkey.Ciphertext,
			IV:           mkey.IV,
			Nonce:        nonce,
		},
		RoomMasterKey: RoomMasterKey{
			Room:      rb.Name,
			KeyID:     keyID.String(),
			Activated: time.Now(),
		},
	}
	if err := transaction.Insert(&rmkb.MasterKey); err != nil {
		if rerr := transaction.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", rerr)
		}
		return nil, err
	}

	if err := transaction.Insert(&rmkb.RoomMasterKey); err != nil {
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

func (rb *RoomBinding) MasterKey(ctx scope.Context) (proto.RoomKey, error) {
	rmkb := &RoomMasterKeyBinding{}
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
		"SELECT c.id, c.encrypted_private_data, c.public_data,"+
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
