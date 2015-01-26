package persist

import (
	"database/sql"
	"fmt"
	"time"

	"heim/backend"
	"heim/backend/proto"
	"heim/backend/proto/security"

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

type RoomKey struct {
	Room         string
	Timestamp    time.Time
	Nonce        []byte
	IV           []byte
	EncryptedKey []byte `db:"encrypted_key"`
}

type RoomBinding struct {
	*Backend
	*Room
	key *RoomKey
}

func (rb *RoomBinding) Latest(ctx context.Context, n int, before proto.Snowflake) (
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

	nonce, err := kms.GenerateNonce(security.AES128.KeySize())
	if err != nil {
		return nil, err
	}

	mkey, err := kms.GenerateEncryptedKey(security.AES256)
	if err != nil {
		return nil, err
	}

	roomKey := &RoomKey{
		Room:         rb.Name,
		Timestamp:    time.Now(),
		Nonce:        nonce,
		IV:           mkey.IV,
		EncryptedKey: mkey.Ciphertext,
	}

	if err := rb.DbMap.Insert(roomKey); err != nil {
		return nil, err
	}

	rb.key = roomKey
	return rb, nil
}

func (rb *RoomBinding) RoomKey() proto.RoomKey { return rb }

func (rb *RoomBinding) Timestamp() time.Time { return rb.key.Timestamp }
func (rb *RoomBinding) Nonce() []byte        { return rb.key.Nonce }

func (rb *RoomBinding) ManagedKey() *security.ManagedKey {
	dup := func(v []byte) []byte {
		w := make([]byte, len(v))
		copy(w, v)
		return w
	}

	return &security.ManagedKey{
		KeyType:    security.AES256,
		IV:         dup(rb.key.IV),
		Ciphertext: dup(rb.key.EncryptedKey),
	}
}
