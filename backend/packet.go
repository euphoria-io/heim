package backend

import (
	"encoding/json"
	"fmt"
)

type PacketType string

func (c PacketType) Event() PacketType { return c + "-event" }
func (c PacketType) Reply() PacketType { return c + "-reply" }

var (
	SendType      = PacketType("send")
	SendEventType = SendType.Event()
	SendReplyType = SendType.Reply()

	JoinType      = PacketType("join")
	JoinEventType = JoinType.Event()
	PartType      = PacketType("part")
	PartEventType = PartType.Event()

	LogType      = PacketType("log")
	LogEventType = LogType.Event()
	LogReplyType = LogType.Reply()

	NickType      = PacketType("nick")
	NickEventType = NickType.Event()
	NickReplyType = NickType.Reply()

	WhoType      = PacketType("who")
	WhoEventType = WhoType.Event()
	WhoReplyType = WhoType.Reply()

	SnapshotEventType = PacketType("snapshot").Event()
)

type ErrorReply struct {
	Error string
}

type SendCommand struct {
	Content string    `json:"content"`
	Parent  Snowflake `json:"parent"`
}

type SendEvent Message
type SendReply SendEvent

type PresenceEvent IdentityView

type LogCommand struct {
	N      int       `json:"n"`
	Before Snowflake `json:"before"`
}

type LogReply struct {
	Log    []Message `json:"log"`
	Before Snowflake `json:"before"`
}

type LogEvent LogReply

type NickCommand struct {
	Name string `json:"name"`
}

type NickReply struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type NickEvent NickReply

type SnapshotEvent struct {
	Version string    `json:"version"`
	Listing Listing   `json:"listing"`
	Log     []Message `json:"log"`
}

type WhoCommand struct{}

type WhoReply struct {
	Listing `json:"listing"`
}

type WhoEvent WhoReply

type Packet struct {
	ID   string          `json:"id"`
	Type PacketType      `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (cmd *Packet) Payload() (interface{}, error) {
	var payload interface{}

	// TODO: use reflect + a map
	switch cmd.Type {
	case SendType:
		payload = &SendCommand{}
	case SendReplyType:
		payload = &SendReply{}
	case SendEventType:
		payload = &SendEvent{}
	case LogType:
		payload = &LogCommand{}
	case LogEventType:
		payload = &LogEvent{}
	case LogReplyType:
		payload = &LogReply{}
	case JoinEventType, PartEventType:
		payload = &PresenceEvent{}
	case NickType:
		payload = &NickCommand{}
	case NickReplyType:
		payload = &NickReply{}
	case NickEventType:
		payload = &NickEvent{}
	case SnapshotEventType:
		payload = &SnapshotEvent{}
	case WhoType:
		return &WhoCommand{}, nil
	case WhoEventType:
		payload = &WhoEvent{}
	case WhoReplyType:
		payload = &WhoReply{}
	default:
		return nil, fmt.Errorf("invalid command type: %s", cmd.Type)
	}

	if payload != nil {
		if err := json.Unmarshal(cmd.Data, payload); err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func (cmd *Packet) Encode() ([]byte, error) { return json.Marshal(cmd) }

func MakeResponse(refID string, msgType PacketType, payload interface{}) (*Packet, error) {
	packet := &Packet{
		ID:   refID,
		Type: msgType.Reply(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := packet.Data.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return packet, nil
}

func MakeEvent(payload interface{}) (*Packet, error) {
	packet := &Packet{}
	switch payload.(type) {
	case *SnapshotEvent:
		packet.Type = SnapshotEventType
	default:
		return nil, fmt.Errorf("don't know how to make event from %T", payload)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if err := packet.Data.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return packet, nil
}

func ParseRequest(data []byte) (*Packet, error) {
	cmd := &Packet{}
	if err := json.Unmarshal(data, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}
