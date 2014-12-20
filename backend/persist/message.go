package persist

import (
	"time"

	"heim/backend"
)

type Message struct {
	Room       string
	Posted     time.Time
	SenderID   string `db:"sender_id"`
	SenderName string `db:"sender_name"`
	Content    string
}

func NewMessage(
	room *Room, sentAt time.Time, idView *backend.IdentityView, content string) *Message {

	return &Message{
		Room:       room.Name,
		Posted:     sentAt,
		SenderID:   idView.ID,
		SenderName: idView.Name,
		Content:    content,
	}
}

func (m *Message) ToBackend() backend.Message {
	return backend.Message{
		UnixTime: m.Posted.Unix(),
		Sender:   &backend.IdentityView{ID: m.SenderID, Name: m.SenderName},
		Content:  m.Content,
	}
}
