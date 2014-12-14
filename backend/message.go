package backend

import (
	"encoding/json"
	"time"
)

type Message struct {
	Timestamp time.Time `json:"time"`
	Sender    Identity  `json:"sender"`
	Content   string    `json:"content"`
}

func (msg *Message) Encode() ([]byte, error) { return json.Marshal(msg) }
