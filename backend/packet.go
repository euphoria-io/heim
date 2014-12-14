package backend

import (
	"encoding/json"
	"fmt"
)

type CommandType string

const (
	SendType CommandType = "send"
	LogType              = "log"
	NickType             = "nick"
	WhoType              = "who"
)

type SendCommand struct {
	Content string `json:"content"`
}

type LogCommand struct {
	N int `json:"n"`
}

type NickCommand struct {
	Name string `json:"name"`
	From string `json:"from"`
}

type WhoCommand struct{}

type Packet struct {
	ID   string          `json:"id"`
	Type CommandType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (cmd *Packet) Payload() (interface{}, error) {
	var payload interface{}

	switch cmd.Type {
	case SendType:
		payload = &SendCommand{}
	case LogType:
		payload = &LogCommand{}
	case NickType:
		payload = &NickCommand{}
	case WhoType:
		payload = &WhoCommand{}
	default:
		return nil, fmt.Errorf("invalid command type: %s", cmd.Type)
	}

	if err := json.Unmarshal(cmd.Data, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func (cmd *Packet) Encode() ([]byte, error) { return json.Marshal(cmd) }

func Response(refID string, msgType CommandType, payload interface{}) (*Packet, error) {
	cmd := &Packet{
		ID:   refID,
		Type: msgType,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := cmd.Data.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return cmd, nil
}

func ParseRequest(data []byte) (*Packet, error) {
	cmd := &Packet{}
	if err := json.Unmarshal(data, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}
