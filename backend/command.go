package backend

import (
	"encoding/json"
	"fmt"
)

type CommandType string

const (
	SendType CommandType = "send"
	LogType              = "log"
)

type SendCommand struct {
	Content string `json:"content"`
}

type LogCommand struct {
	N int `json:"n"`
}

type Command struct {
	ID   string      `json:"id"`
	Type CommandType `json:"type"`
	Data json.RawMessage
}

func (cmd *Command) Payload() (interface{}, error) {
	var payload interface{}

	switch cmd.Type {
	case SendType:
		payload = &SendCommand{}
	case LogType:
		payload = &LogCommand{}
	default:
		return nil, fmt.Errorf("invalid command type: %s", cmd.Type)
	}

	if err := json.Unmarshal(cmd.Data, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func (cmd *Command) Encode() ([]byte, error) { return json.Marshal(cmd) }

func Response(refID string, msgType CommandType, payload interface{}) (*Command, error) {
	cmd := &Command{
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

func ParseRequest(data []byte) (*Command, error) {
	cmd := &Command{}
	if err := json.Unmarshal(data, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}
