package psql

import (
	"database/sql"
	"fmt"
	"time"

	"heim/backend"
	"heim/proto"
	"heim/proto/security"
	"heim/proto/snowflake"

	"golang.org/x/net/context"
)

var notImpl = fmt.Errorf("not implemented")
var logger = backend.Logger

type Room struct {
	Name      string
	FoundedBy string `db:"founded_by"`
}

func (Room) AfterCreateTable(db *sql.DB) error {
	_, err := db.Exec("CREATE INDEX room_founded_by ON room(founded_by)")
	return err
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

func (rb *RoomBinding) Latest(ctx context.Context, n int, before snowflake.Snowflake) (
	[]proto.Message, error) {

	return rb.Backend.latest(ctx, rb.Room, n, before)
}

func (rb *RoomBinding) Join(ctx context.Context, session proto.Session) error {
	return rb.Backend.join(ctx, rb.Room, session)
}

func (rb *RoomBinding) Part(ctx context.Context, session proto.Session) error {
	return rb.Backend.part(ctx, rb.Room, session)
}

func (rb *RoomBinding) Send(ctx context.Context, session proto.Session, msg proto.Message) (
	proto.Message, error) {

	logger(ctx).Printf("Send\n")
	return rb.Backend.sendMessageToRoom(ctx, rb.Room, session, msg, session)
}

func (rb *RoomBinding) Listing(ctx context.Context) (proto.Listing, error) {
	return rb.Backend.listing(ctx, rb.Room)
}

func (rb *RoomBinding) RenameUser(ctx context.Context, session proto.Session, formerName string) (
	*proto.NickEvent, error) {

	event := &proto.NickEvent{
		ID:   session.Identity().ID(),
		From: formerName,
		To:   session.Identity().Name(),
	}
	return event, rb.Backend.broadcast(ctx, rb.Room, session, proto.NickEventType, event, session)
}

func (rb *RoomBinding) GenerateMasterKey(
	ctx context.Context, kms security.KMS) (proto.RoomKey, error) {

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

	mkey, err := kms.GenerateEncryptedKey(security.AES256)
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

func (rb *RoomBinding) MasterKey(ctx context.Context) (proto.RoomKey, error) {
	rmkb := &RoomMasterKeyBinding{}
	err := rb.DbMap.SelectOne(
		rmkb,
		"SELECT * FROM master_key mk, room_master_key r"+
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

func (rb *RoomBinding) SaveCapability(ctx context.Context, capability security.Capability) error {
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

func (rb *RoomBinding) GetCapability(ctx context.Context, id string) (security.Capability, error) {
	rcb := &RoomCapabilityBinding{}

	backend.Logger(ctx).Printf("looking up capability %s in room %s", id, rb.Name)
	err := rb.DbMap.SelectOne(
		rcb,
		"SELECT * FROM capability c, room_capability r"+
			" WHERE r.room = $1 AND c.id = $2 AND r.revoked < r.granted",
		rb.Name, id)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return rcb, nil
}
