package persist

import (
	"database/sql"
	"time"

	"heim/backend"
)

type Message struct {
	Room       string
	ID         string
	Parent     string
	Posted     time.Time
	SenderID   string `db:"sender_id"`
	SenderName string `db:"sender_name"`
	Content    string
}

func (Message) AfterCreateTable(db *sql.DB) error {
	_, err := db.Exec("CREATE INDEX message_room_parent ON message(room, parent)")
	return err
}

func NewMessage(
	room *Room, idView *backend.IdentityView, parent backend.Snowflake, content string) (
	*Message, error) {

	id, err := backend.NewSnowflake()
	if err != nil {
		return nil, err
	}

	return &Message{
		Room:       room.Name,
		ID:         id.String(),
		Parent:     parent.String(),
		Posted:     id.Time(),
		SenderID:   idView.ID,
		SenderName: idView.Name,
		Content:    content,
	}, nil
}

func (m *Message) ToBackend() backend.Message {
	msg := backend.Message{
		UnixTime: m.Posted.Unix(),
		Sender:   &backend.IdentityView{ID: m.SenderID, Name: m.SenderName},
		Content:  m.Content,
	}

	// ignore id parsing errors
	_ = msg.ID.FromString(m.ID)
	_ = msg.Parent.FromString(m.Parent)

	return msg
}
