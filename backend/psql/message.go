package psql

import (
	"database/sql"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/snowflake"
)

type Message struct {
	Room            string
	ID              string
	Parent          string
	Posted          time.Time
	SenderID        string `db:"sender_id"`
	SenderName      string `db:"sender_name"`
	ServerID        string `db:"server_id"`
	ServerEra       string `db:"server_era"`
	Content         string
	EncryptionKeyID sql.NullString `db:"encryption_key_id"`
}

func NewMessage(
	room *Room, idView *proto.IdentityView, id, parent snowflake.Snowflake, keyID, content string) (
	*Message, error) {

	msg := &Message{
		Room:    room.Name,
		ID:      id.String(),
		Parent:  parent.String(),
		Posted:  id.Time(),
		Content: content,
	}
	if idView != nil {
		msg.SenderID = idView.ID
		msg.SenderName = idView.Name
		msg.ServerID = idView.ServerID
		msg.ServerEra = idView.ServerEra
	}
	if keyID != "" {
		msg.EncryptionKeyID = sql.NullString{
			String: keyID,
			Valid:  true,
		}
	}
	return msg, nil
}

func (m *Message) ToBackend() proto.Message {
	msg := proto.Message{
		UnixTime: proto.Time(m.Posted),
		Sender: &proto.IdentityView{
			ID:        m.SenderID,
			Name:      m.SenderName,
			ServerID:  m.ServerID,
			ServerEra: m.ServerEra,
		},
		Content: m.Content,
	}

	// ignore id parsing errors
	_ = msg.ID.FromString(m.ID)
	_ = msg.Parent.FromString(m.Parent)
	if m.EncryptionKeyID.Valid {
		msg.EncryptionKeyID = m.EncryptionKeyID.String
	}

	return msg
}
