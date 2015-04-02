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
	PreviousEditID  string `db:"previous_edit_id"`
	Parent          string
	Posted          time.Time
	Edited          time.Time
	Deleted         time.Time
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

type MessageEditLog struct {
	EditID          string `db:"edit_id"`
	Room            string
	MessageID       string         `db:"message_id"`
	EditorID        sql.NullString `db:"editor_id"`
	PreviousEditID  sql.NullString `db:"previous_edit_id"`
	PreviousContent string         `db:"previous_content"`
	PreviousParent  sql.NullString `db:"previous_parent"`
}
