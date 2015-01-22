package proto

import (
	"encoding/json"
)

// A Message is a node in a Room's Log. It corresponds to a chat message, or
// a post, or any broadcasted event in a room that should appear in the log.
type Message struct {
	ID       Snowflake     `json:"id"`
	Parent   Snowflake     `json:"parent"`
	UnixTime int64         `json:"time"`
	Sender   *IdentityView `json:"sender"`
	Content  string        `json:"content"`
}

func (msg *Message) Encode() ([]byte, error) { return json.Marshal(msg) }
