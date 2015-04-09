package proto

import (
	"encoding/json"
	"fmt"

	"euphoria.io/heim/proto/snowflake"
)

type PacketType string

func (c PacketType) Event() PacketType { return c + "-event" }
func (c PacketType) Reply() PacketType { return c + "-reply" }

var (
	AuthType      = PacketType("auth")
	AuthEventType = AuthType.Event()
	AuthReplyType = AuthType.Reply()

	SendType      = PacketType("send")
	SendEventType = SendType.Event()
	SendReplyType = SendType.Reply()

	EditMessageType      = PacketType("edit-message")
	EditMessageEventType = EditMessageType.Event()
	EditMessageReplyType = EditMessageType.Reply()

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

	PingType      = PacketType("ping")
	PingEventType = PingType.Event()
	PingReplyType = PingType.Reply()

	WhoType      = PacketType("who")
	WhoEventType = WhoType.Event()
	WhoReplyType = WhoType.Reply()

	BounceEventType   = PacketType("bounce").Event()
	NetworkEventType  = PacketType("network").Event()
	SnapshotEventType = PacketType("snapshot").Event()

	ErrorReplyType = PacketType("error").Reply()
)

type ErrorReply struct {
	Error string `json:"error"`
}

type SendCommand struct {
	Content string              `json:"content"`
	Parent  snowflake.Snowflake `json:"parent"`
}

type SendEvent Message
type SendReply SendEvent

type EditMessageCommand struct {
	ID             snowflake.Snowflake `json:"id"`
	PreviousEditID snowflake.Snowflake `json:"previous_edit_id"`
	Parent         snowflake.Snowflake `json:"parent"`
	Content        string              `json:"content"`
	Delete         bool                `json:"delete"`
	Announce       bool                `json:"announce"`
}

type EditMessageReply struct {
	EditID  snowflake.Snowflake `json:"edit_id"`
	Deleted bool                `json:"deleted,omitempty"`
}

type EditMessageEvent struct {
	Message
	EditID snowflake.Snowflake `json:"edit_id"`
}

type PresenceEvent SessionView

type LogCommand struct {
	N      int                 `json:"n"`
	Before snowflake.Snowflake `json:"before"`
}

type LogReply struct {
	Log    []Message           `json:"log"`
	Before snowflake.Snowflake `json:"before"`
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

type PingCommand struct {
	UnixTime     int64 `json:"time"`
	NextUnixTime int64 `json:"next"`
}

type PingEvent PingCommand

type PingReply struct {
	UnixTime int64 `json:"time,omitempty"`
}

type AuthCommand struct {
	Type     AuthOption `json:"type"`
	Passcode string     `json:"passcode,omitempty"`
}

type AuthReply struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason,omitempty"`
}

type AuthEvent AuthReply

type BounceEvent struct {
	Reason      string       `json:"reason,omitempty"`
	AuthOptions []AuthOption `json:"auth_options,omitempty"`
	AgentID     string       `json:"agent_id,omitempty"`
	IP          string       `json:"ip,omitempty"`
}

type SnapshotEvent struct {
	Identity  string    `json:"identity"`
	SessionID string    `json:"session_id"`
	Version   string    `json:"version"`
	Listing   Listing   `json:"listing"`
	Log       []Message `json:"log"`
}

type NetworkEvent struct {
	Type      string `json:"type"` // for now, always "partition"
	ServerID  string `json:"server_id"`
	ServerEra string `json:"server_era"`
}

type WhoCommand struct{}

type WhoReply struct {
	Listing `json:"listing"`
}

type WhoEvent WhoReply

type Packet struct {
	ID    string          `json:"id,omitempty"`
	Type  PacketType      `json:"type"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func (cmd *Packet) Payload() (interface{}, error) {
	if cmd.Error != "" {
		return &ErrorReply{Error: cmd.Error}, nil
	}

	var payload interface{}

	// TODO: use reflect + a map
	switch cmd.Type {
	case SendType:
		payload = &SendCommand{}
	case SendReplyType:
		payload = &SendReply{}
	case SendEventType:
		payload = &SendEvent{}
	case EditMessageType:
		payload = &EditMessageCommand{}
	case EditMessageEventType:
		payload = &EditMessageEvent{}
	case EditMessageReplyType:
		payload = &EditMessageReply{}
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
	case PingType:
		payload = &PingCommand{}
	case PingEventType:
		payload = &PingEvent{}
	case PingReplyType:
		payload = &PingReply{}
	case AuthType:
		payload = &AuthCommand{}
	case AuthEventType:
		payload = &AuthEvent{}
	case AuthReplyType:
		payload = &AuthReply{}
	case BounceEventType:
		payload = &BounceEvent{}
	case NetworkEventType:
		payload = &NetworkEvent{}
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

	if err, ok := payload.(error); ok {
		msgType = ErrorReplyType
		packet.Error = err.Error()
		payload = nil
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
	case *BounceEvent:
		packet.Type = BounceEventType
	case *PingEvent:
		packet.Type = PingEventType
	case *NetworkEvent:
		packet.Type = NetworkEventType
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
