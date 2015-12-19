package proto

import (
	"encoding/json"

	"euphoria.io/heim/proto/snowflake"
)

const (
	MaxMessageLength             = 1 << 20
	MaxMessageTransmissionLength = 4096
)

// A Message is a node in a Room's Log. It corresponds to a chat message, or
// a post, or any broadcasted event in a room that should appear in the log.
type Message struct {
	ID              snowflake.Snowflake `json:"id"`                          // the id of the message (unique within a room)
	Parent          snowflake.Snowflake `json:"parent,omitempty"`            // the id of the message's parent, or null if top-level
	PreviousEditID  snowflake.Snowflake `json:"previous_edit_id,omitempty"`  // the edit id of the most recent edit of this message, or null if it's never been edited
	UnixTime        Time                `json:"time"`                        // the unix timestamp of when the message was posted
	Sender          SessionView         `json:"sender"`                      // the view of the sender's session
	Content         string              `json:"content"`                     // the content of the message (client-defined)
	EncryptionKeyID string              `json:"encryption_key_id,omitempty"` // the id of the key that encrypts the message in storage
	Edited          Time                `json:"edited,omitempty"`            // the unix timestamp of when the message was last edited
	Deleted         Time                `json:"deleted,omitempty"`           // the unix timestamp of when the message was deleted
	Truncated       bool                `json:"truncated,omitempty"`         // if true, then the full content of this message is not included (see `get-message` to obtain the message with full content)
}

func (msg *Message) Encode() ([]byte, error) { return json.Marshal(msg) }
