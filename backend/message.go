package backend

import (
	"encoding/json"
)

type Message struct {
	ID       Snowflake     `json:"id"`
	Parent   Snowflake     `json:"parent"`
	UnixTime int64         `json:"time"`
	Sender   *IdentityView `json:"sender"`
	Content  string        `json:"content"`
}

func (msg *Message) Encode() ([]byte, error) { return json.Marshal(msg) }
