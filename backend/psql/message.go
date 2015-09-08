package psql

import (
	"database/sql"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/snowflake"

	"gopkg.in/gorp.v1"
)

type Message struct {
	Room            string
	ID              string
	PreviousEditID  sql.NullString `db:"previous_edit_id"`
	Parent          string
	Posted          time.Time
	Edited          gorp.NullTime
	Deleted         gorp.NullTime
	SessionID       string `db:"session_id"`
	SenderID        string `db:"sender_id"`
	SenderName      string `db:"sender_name"`
	SenderIsManager bool   `db:"sender_is_manager"`
	SenderIsStaff   bool   `db:"sender_is_staff"`
	ServerID        string `db:"server_id"`
	ServerEra       string `db:"server_era"`
	Content         string
	EncryptionKeyID sql.NullString `db:"encryption_key_id"`
}

func NewMessage(
	room *Room, sessionView *proto.SessionView, id, parent snowflake.Snowflake, keyID, content string) (
	*Message, error) {

	msg := &Message{
		Room:    room.Name,
		ID:      id.String(),
		Parent:  parent.String(),
		Posted:  id.Time(),
		Content: content,
	}
	if sessionView != nil {
		msg.SessionID = sessionView.SessionID
		msg.SenderID = string(sessionView.ID)
		msg.SenderName = sessionView.Name
		msg.ServerID = sessionView.ServerID
		msg.ServerEra = sessionView.ServerEra
		msg.SenderIsManager = sessionView.IsManager
		msg.SenderIsStaff = sessionView.IsStaff
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
		Sender: &proto.SessionView{
			IdentityView: &proto.IdentityView{
				ID:        proto.UserID(m.SenderID),
				Name:      m.SenderName,
				ServerID:  m.ServerID,
				ServerEra: m.ServerEra,
			},
			SessionID: m.SessionID,
			IsManager: m.SenderIsManager,
			IsStaff:   m.SenderIsStaff,
		},
		Content: m.Content,
	}

	// ignore id parsing errors
	_ = msg.ID.FromString(m.ID)
	_ = msg.Parent.FromString(m.Parent)
	if m.PreviousEditID.Valid {
		_ = msg.PreviousEditID.FromString(m.PreviousEditID.String)
	}

	// other optionals
	if m.EncryptionKeyID.Valid {
		msg.EncryptionKeyID = m.EncryptionKeyID.String
	}
	if m.Deleted.Valid {
		msg.Deleted = proto.Time(m.Deleted.Time)
	}
	if m.Edited.Valid {
		msg.Edited = proto.Time(m.Edited.Time)
	}

	return msg
}

func (m *Message) ToTransmission() proto.Message {
	msg := m.ToBackend()
	if len(msg.Content) > proto.MaxMessageTransmissionLength {
		msg.Content = msg.Content[:proto.MaxMessageTransmissionLength]
		msg.Truncated = true
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
