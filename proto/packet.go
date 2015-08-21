package proto

import (
	"encoding/json"
	"fmt"
	"reflect"

	"euphoria.io/heim/proto/snowflake"
)

type PacketType string

func (c PacketType) Event() PacketType { return c + "-event" }
func (c PacketType) Reply() PacketType { return c + "-reply" }

var (
	AuthType      = PacketType("auth")
	AuthReplyType = AuthType.Reply()

	BanType        = PacketType("ban")
	BanReplyType   = BanType.Reply()
	UnbanType      = PacketType("unban")
	UnbanReplyType = UnbanType.Reply()

	SendType      = PacketType("send")
	SendEventType = SendType.Event()
	SendReplyType = SendType.Reply()

	ChangePasswordType      = PacketType("change-password")
	ChangePasswordReplyType = ChangePasswordType.Reply()

	EditMessageType      = PacketType("edit-message")
	EditMessageEventType = EditMessageType.Event()
	EditMessageReplyType = EditMessageType.Reply()

	GrantAccessType      = PacketType("grant-access")
	GrantAccessReplyType = GrantAccessType.Reply()

	GrantManagerType      = PacketType("grant-manager")
	GrantManagerReplyType = GrantManagerType.Reply()

	JoinType      = PacketType("join")
	JoinEventType = JoinType.Event()
	PartType      = PacketType("part")
	PartEventType = PartType.Event()

	LogType      = PacketType("log")
	LogReplyType = LogType.Reply()

	LoginType      = PacketType("login")
	LoginReplyType = LoginType.Reply()

	LogoutType      = PacketType("logout")
	LogoutReplyType = LogoutType.Reply()

	NickType      = PacketType("nick")
	NickEventType = NickType.Event()
	NickReplyType = NickType.Reply()

	PingType      = PacketType("ping")
	PingEventType = PingType.Event()
	PingReplyType = PingType.Reply()

	RegisterAccountType      = PacketType("register-account")
	RegisterAccountReplyType = RegisterAccountType.Reply()

	ResetPasswordType      = PacketType("reset-password")
	ResetPasswordReplyType = ResetPasswordType.Reply()

	RevokeAccessType      = PacketType("revoke-access")
	RevokeAccessReplyType = RevokeAccessType.Reply()

	RevokeManagerType      = PacketType("revoke-manager")
	RevokeManagerReplyType = RevokeManagerType.Reply()

	StaffCreateRoomType      = PacketType("staff-create-room")
	StaffCreateRoomReplyType = StaffCreateRoomType.Reply()

	StaffGrantManagerType      = PacketType("staff-grant-manager")
	StaffGrantManagerReplyType = StaffGrantManagerType.Reply()

	StaffLockRoomType      = PacketType("staff-lock-room")
	StaffLockRoomReplyType = StaffLockRoomType.Reply()

	StaffRevokeAccessType      = PacketType("staff-revoke-access")
	StaffRevokeAccessReplyType = StaffRevokeAccessType.Reply()

	StaffRevokeManagerType      = PacketType("staff-revoke-manager")
	StaffRevokeManagerReplyType = StaffRevokeManagerType.Reply()

	UnlockStaffCapabilityType      = PacketType("unlock-staff-capability")
	UnlockStaffCapabilityReplyType = UnlockStaffCapabilityType.Reply()

	WhoType      = PacketType("who")
	WhoReplyType = WhoType.Reply()

	BounceEventType     = PacketType("bounce").Event()
	DisconnectEventType = PacketType("disconnect").Event()
	HelloEventType      = PacketType("hello").Event()
	NetworkEventType    = PacketType("network").Event()
	SnapshotEventType   = PacketType("snapshot").Event()

	ErrorReplyType = PacketType("error").Reply()

	payloadMap = map[PacketType]reflect.Type{
		SendType:      reflect.TypeOf(SendCommand{}),
		SendReplyType: reflect.TypeOf(SendReply{}),
		SendEventType: reflect.TypeOf(SendEvent{}),

		ChangePasswordType:      reflect.TypeOf(ChangePasswordCommand{}),
		ChangePasswordReplyType: reflect.TypeOf(ChangePasswordReply{}),

		EditMessageType:      reflect.TypeOf(EditMessageCommand{}),
		EditMessageEventType: reflect.TypeOf(EditMessageEvent{}),
		EditMessageReplyType: reflect.TypeOf(EditMessageReply{}),

		GrantAccessType:      reflect.TypeOf(GrantAccessCommand{}),
		GrantAccessReplyType: reflect.TypeOf(GrantAccessReply{}),

		GrantManagerType:      reflect.TypeOf(GrantManagerCommand{}),
		GrantManagerReplyType: reflect.TypeOf(GrantManagerReply{}),

		LogType:      reflect.TypeOf(LogCommand{}),
		LogReplyType: reflect.TypeOf(LogReply{}),

		JoinEventType: reflect.TypeOf(PresenceEvent{}),
		PartEventType: reflect.TypeOf(PresenceEvent{}),

		NickType:      reflect.TypeOf(NickCommand{}),
		NickReplyType: reflect.TypeOf(NickReply{}),
		NickEventType: reflect.TypeOf(NickEvent{}),

		PingType:      reflect.TypeOf(PingCommand{}),
		PingEventType: reflect.TypeOf(PingEvent{}),
		PingReplyType: reflect.TypeOf(PingReply{}),

		StaffCreateRoomType:      reflect.TypeOf(StaffCreateRoomCommand{}),
		StaffCreateRoomReplyType: reflect.TypeOf(StaffCreateRoomReply{}),

		StaffGrantManagerType:      reflect.TypeOf(StaffGrantManagerCommand{}),
		StaffGrantManagerReplyType: reflect.TypeOf(StaffGrantManagerReply{}),

		StaffLockRoomType:      reflect.TypeOf(StaffLockRoomCommand{}),
		StaffLockRoomReplyType: reflect.TypeOf(StaffLockRoomReply{}),

		StaffRevokeAccessType:      reflect.TypeOf(StaffRevokeAccessCommand{}),
		StaffRevokeAccessReplyType: reflect.TypeOf(StaffRevokeAccessReply{}),

		StaffRevokeManagerType:      reflect.TypeOf(StaffRevokeManagerCommand{}),
		StaffRevokeManagerReplyType: reflect.TypeOf(StaffRevokeManagerReply{}),

		AuthType:      reflect.TypeOf(AuthCommand{}),
		AuthReplyType: reflect.TypeOf(AuthReply{}),

		BanType:        reflect.TypeOf(BanCommand{}),
		BanReplyType:   reflect.TypeOf(BanReply{}),
		UnbanType:      reflect.TypeOf(UnbanCommand{}),
		UnbanReplyType: reflect.TypeOf(UnbanReply{}),

		BounceEventType:     reflect.TypeOf(BounceEvent{}),
		DisconnectEventType: reflect.TypeOf(DisconnectEvent{}),
		HelloEventType:      reflect.TypeOf(HelloEvent{}),
		NetworkEventType:    reflect.TypeOf(NetworkEvent{}),
		SnapshotEventType:   reflect.TypeOf(SnapshotEvent{}),

		LoginType:      reflect.TypeOf(LoginCommand{}),
		LoginReplyType: reflect.TypeOf(LoginReply{}),

		LogoutType:      reflect.TypeOf(LogoutCommand{}),
		LogoutReplyType: reflect.TypeOf(LogoutReply{}),

		RegisterAccountType:      reflect.TypeOf(RegisterAccountCommand{}),
		RegisterAccountReplyType: reflect.TypeOf(RegisterAccountReply{}),

		ResetPasswordType:      reflect.TypeOf(ResetPasswordCommand{}),
		ResetPasswordReplyType: reflect.TypeOf(ResetPasswordReply{}),

		RevokeManagerType:      reflect.TypeOf(RevokeManagerCommand{}),
		RevokeManagerReplyType: reflect.TypeOf(RevokeManagerReply{}),

		RevokeAccessType:      reflect.TypeOf(RevokeAccessCommand{}),
		RevokeAccessReplyType: reflect.TypeOf(RevokeAccessReply{}),

		UnlockStaffCapabilityType:      reflect.TypeOf(UnlockStaffCapabilityCommand{}),
		UnlockStaffCapabilityReplyType: reflect.TypeOf(UnlockStaffCapabilityReply{}),

		WhoType:      reflect.TypeOf(WhoCommand{}),
		WhoReplyType: reflect.TypeOf(WhoReply{}),
	}
)

func PacketsByType() map[PacketType]string {
	templates := map[PacketType]string{}
	for name, templateType := range payloadMap {
		templates[name] = templateType.Name()
	}
	return templates
}

type ErrorReply struct {
	Error string `json:"error"`
}

// The `send` command sends a message to a room. The session must be
// successfully joined with the room. This message will be broadcast to
// all sessions joined with the room.
//
// If the room is private, then the message content will be encrypted
// before it is stored and broadcast to the rest of the room.
//
// The caller of this command will not receive the corresponding
// `send-event`, but will receive the same information in the `send-reply`.
type SendCommand struct {
	Content string              `json:"content"`          // the content of the message (client-defined)
	Parent  snowflake.Snowflake `json:"parent,omitempty"` // the id of the parent message, if any
}

// A `send-event` indicates a message received by the room from another session.
type SendEvent Message

// `send-reply` returns the message that was sent. This includes the message id,
// which was populated by the server.
type SendReply SendEvent

// The `edit-message` command can be used by active room managers to modify the
// content or display of a message.
//
// A message deleted by this command is still stored in the database. Deleted
// messages may be undeleted by this command. (Messages that have expired from
// the database due to the room's retention policy are no longer available and
// cannot be restored by this or any command).
//
// If the `announce` field is set to true, then an edit-message-event will be
// broadcast to the room.
//
// TODO: support content editing
// TODO: support reparenting
type EditMessageCommand struct {
	ID             snowflake.Snowflake `json:"id"`                // the id of the message to edit
	PreviousEditID snowflake.Snowflake `json:"previous_edit_id"`  // the `previous_edit_id` of the message; if this does not match, the edit will fail (basic conflict resolution)
	Parent         snowflake.Snowflake `json:"parent,omitempty"`  // the new parent of the message (*not yet implemented*)
	Content        string              `json:"content,omitempty"` // the new content of the message (*not yet implemented*)
	Delete         bool                `json:"delete"`            // the new deletion status of the message
	Announce       bool                `json:"announce"`          // if true, broadcast an `edit-message-event` to the room
}

// The `change-password` command changes the password of the signed in account.
type ChangePasswordCommand struct {
	OldPassword string `json:"old_password"` // the current (and soon-to-be former) password
	NewPassword string `json:"new_password"` // the new password
}

// The `change-password-reply` packet returns the outcome of changing the password.
type ChangePasswordReply struct{}

// `edit-message-reply` returns the id of a successful edit.
type EditMessageReply struct {
	EditID  snowflake.Snowflake `json:"edit_id"`           // the unique id of the edit that was applied
	Deleted bool                `json:"deleted,omitempty"` // the new deletion status of the edited message
}

// An `edit-message-event` indicates that a message in the room has been
// modified or deleted. If the client offers a user interface and the
// indicated message is currently displayed, it should update its display
// accordingly.
//
// The event packet includes a snapshot of the message post-edit.
type EditMessageEvent struct {
	Message
	EditID snowflake.Snowflake `json:"edit_id"` // the id of the edit
}

// The `grant-access` command may be used by an active manager in a private room
// to create a new capability for access. Access may be granted to either a
// passcode or an account.
//
// If the room is not private, or if the requested access grant already exists,
// an error will be returned.
type GrantAccessCommand struct {
	AccountID snowflake.Snowflake `json:"account_id,omitempty"` // the id of an account to grant access to
	Passcode  string              `json:"passcode,omitempty"`   // a passcode to grant access to; anyone presenting the same passcode can access the room
}

// `grant-access-reply` confirms that access was granted.
type GrantAccessReply struct{}

// The `grant-manager` command may be used by an active room manager to make
// another account a manager in the same room.
//
// An error is returned if the account can't be found.
type GrantManagerCommand struct {
	AccountID snowflake.Snowflake `json:"account_id"` // the id of an account to grant manager status to
}

// `grant-manager-reply` confirms that manager status was granted.
type GrantManagerReply struct{}

// The `staff-grant-manager` command is a version of the [grant-manager](#grant-manager)
// command that is available to staff. The staff account does not need to be a manager
// of the room to use this command.
type StaffGrantManagerCommand GrantManagerCommand

// `staff-grant-manager-reply` confirms that requested manager change was granted.
type StaffGrantManagerReply GrantManagerReply

// A `presence-event` describes a session joining into or parting from a room.
type PresenceEvent SessionView

// The `log` command requests messages from the room's message log. This can be used
// to supplement the log provided by `snapshot-event` (for example, when scrolling
// back further in history).
type LogCommand struct {
	N      int                 `json:"n"`                // maximum number of messages to return (up to 1000)
	Before snowflake.Snowflake `json:"before,omitempty"` // return messages prior to this snowflake
}

// The `log-reply` packet returns a list of messages from the room's message log.
type LogReply struct {
	Log    []Message           `json:"log"`              // list of messages returned
	Before snowflake.Snowflake `json:"before,omitempty"` // messages prior to this snowflake were returned
}

// The `nick` command sets the name you present to the room. This name applies
// to all messages sent during this session, until the `nick` command is called
// again.
type NickCommand struct {
	Name string `json:"name"` // the requested name (maximum length 36 bytes)
}

// `nick-reply` confirms the `nick` command. It returns the session's former
// and new names (the server may modify the requested nick).
type NickReply struct {
	SessionID string `json:"session_id"` // the id of the session this name applies to
	ID        UserID `json:"id"`         // the id of the agent or account logged into the session
	From      string `json:"from"`       // the previous name associated with the session
	To        string `json:"to"`         // the name associated with the session henceforth
}

// `nick-event` announces a nick change by another session in the room.
type NickEvent NickReply

// The `ping` command initiates a client-to-server ping. The server will send
// back a `ping-reply` with the same timestamp as soon as possible.
type PingCommand struct {
	UnixTime Time `json:"time"` // an arbitrary value, intended to be a unix timestamp
}

// A `ping-event` represents a server-to-client ping. The client should send back
// a `ping-reply` with the same value for the time field as soon as possible
// (or risk disconnection).
type PingEvent struct {
	UnixTime     Time `json:"time"` // a unix timestamp according to the server's clock
	NextUnixTime Time `json:"next"` // the expected time of the next ping-event, according to the server's clock
}

// `ping-reply` is a response to a `ping` command or `ping-event`.
type PingReply struct {
	UnixTime Time `json:"time,omitempty"` // the timestamp of the ping being replied to
}

// The `auth` command attempts to join a private room. It should be sent in response
// to a `bounce-event` at the beginning of a session.
type AuthCommand struct {
	Type     AuthOption `json:"type"`               // the method of authentication
	Passcode string     `json:"passcode,omitempty"` // use this field for `passcode` authentication
}

// The `auth-reply` packet reports whether the `auth` command succeeded.
type AuthReply struct {
	Success bool   `json:"success"`          // true if authentication succeeded
	Reason  string `json:"reason,omitempty"` // if `success` was false, the reason for failure
}

// `Ban` describes an entry in a ban list. When incoming sessions match one of
// these entries, they are rejected.
type Ban struct {
	ID UserID `json:"id,omitempty"` // if given, select for the given agent or account
	IP string `json:"ip,omitempty"` // if given, select for the given IP address
}

// The `ban` command adds an entry to the room's ban list. Any joined sessions
// that match this entry will be disconnected. New sessions matching the entry
// will be unable to join the room.
//
// The command is a no-op if an identical entry already exists in the ban list.
type BanCommand struct {
	Ban
	Seconds int `json:"seconds,omitempty"` // if given, the ban is temporary
}

// The `ban-reply` packet indicates that the `ban` command succeeded.
type BanReply struct{}

// The `unban` command removes an entry from the room's ban list.
type UnbanCommand struct {
	Ban
}

// The `unban-reply` packet indicates that the `unban` command succeeded.
type UnbanReply struct{}

// A `bounce-event` indicates that access to a room is denied.
type BounceEvent struct {
	Reason      string       `json:"reason,omitempty"`       // the reason why access was denied
	AuthOptions []AuthOption `json:"auth_options,omitempty"` // authentication options that may be used; see [auth](#auth)
	AgentID     string       `json:"agent_id,omitempty"`     // internal use only
	IP          string       `json:"ip,omitempty"`           // internal use only
}

// A `disconnect-event` indicates that the session is being closed. The client
// will subsequently be disconnected.
//
// If the disconnect reason is "authentication changed", the client should
// immediately reconnect.
type DisconnectEvent struct {
	Reason string `json:"reason"` // the reason for disconnection
}

// A `hello-event` is sent by the server to the client when a session is started.
// It includes information about the client's authentication and associated identity.
type HelloEvent struct {
	SessionView
	Version string `json:"version"`
}

// A `snapshot-event` indicates that a session has successfully joined a room.
// It also offers a snapshot of the room's state and recent history.
type SnapshotEvent struct {
	Identity  UserID    `json:"identity"`   // the id of the agent or account logged into this session
	SessionID string    `json:"session_id"` // the globally unique id of this session
	Version   string    `json:"version"`    // the server's version identifier
	Listing   Listing   `json:"listing"`    // the list of all other sessions joined to the room (excluding this session)
	Log       []Message `json:"log"`        // the most recent messages posted to the room (currently up to 100)
}

// A `network-event` indicates some server-side event that impacts the presence
// of sessions in a room.
//
// If the network event type is `partition`, then this should be treated as
// a [part-event](#part-event) for all sessions connected to the same server
// id/era combo.
type NetworkEvent struct {
	Type      string `json:"type"`       // the type of network event; for now, always `partition`
	ServerID  string `json:"server_id"`  // the id of the affected server
	ServerEra string `json:"server_era"` // the era of the affected server
}

// The `login` command attempts to log an anonymous session into an account.
// It will return an error if the session is already logged in.
//
// If the login succeeds, the client should expect to receive a
// `disconnect-event` shortly after. The next connection the client makes
// will be a logged in session.
type LoginCommand struct {
	Namespace string `json:"namespace"` // the namespace of a personal identifier
	ID        string `json:"id"`        // the id of a personal identifier
	Password  string `json:"password"`  // the password for unlocking the account
}

// The `login-reply` packet returns whether the session successfully logged
// into an account.
//
// If this reply returns success, the client should expect to receive a
// `disconnect-event` shortly after. The next connection the client makes
// will be a logged in session.
type LoginReply struct {
	Success   bool                `json:"success"`              // true if the session is now logged in
	Reason    string              `json:"reason,omitempty"`     // if `success` was false, the reason why
	AccountID snowflake.Snowflake `json:"account_id,omitempty"` // if `success` was true, the id of the account the session logged into.
}

// The `logout` command logs a session out of an account. It will return an error
// if the session is not logged in.
//
// If the logout is successful, the client should expect to receive a
// `disconnect-event` shortly after. The next connection the client
// makes will be a logged out session.
type LogoutCommand struct{}

// The `logout-reply` packet confirms a logout.
type LogoutReply struct{}

// The `register-account` command creates a new account and logs into it.
// It will return an error if the session is already logged in.
//
// If the account registration succeeds, the client should expect to receive a
// `disconnect-event` shortly after. The next connection the client makes will be
// a logged in session using the new account.
type RegisterAccountCommand LoginCommand

// The `register-account-reply` packet returns whether the new account was
// registered.
//
// If this reply returns success, the client should expect to receive a
// disconnect-event shortly after. The next connection the client makes
// will be a logged in session, using the newly created account.
type RegisterAccountReply LoginReply

// The `reset-password` command generates a password reset request. An email
// will be sent to the owner of the given personal identifier, with
// instructions and a confirmation code for resetting the password.
type ResetPasswordCommand struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`
}

// `reset-password-reply` confirms that the password reset is in progress.
type ResetPasswordReply struct{}

// The `revoke-access` command disables an access grant to a private room.
// The grant may be to an account or to a passcode.
//
// TODO: all live sessions using the revoked grant should be disconnected
// TODO: support revocation by capability_id, in case a manager doesn't know the passcode
type RevokeAccessCommand struct {
	AccountID snowflake.Snowflake `json:"account_id,omitempty"` // the id of the account to revoke access from
	Passcode  string              `json:"passcode",omitempty`   // the passcode to revoke access from
}

// `revoke-access-reply` confirms that the access grant was revoked.
type RevokeAccessReply struct{}

// The `revoke-manager` command removes an account as manager of the room.
// This command can be applied to oneself, so be careful not to orphan
// your room!
type RevokeManagerCommand struct {
	AccountID snowflake.Snowflake `json:"account_id"` // the id of the account to remove as manager
}

// `revoke-manager-reply` confirms that the manager grant was revoked.
type RevokeManagerReply struct{}

// The `staff-create-room` command creates a new room.
type StaffCreateRoomCommand struct {
	Name     string                `json:"name"`              // the name of the new rom
	Managers []snowflake.Snowflake `json:"managers"`          // ids of manager accounts for this room (there must be at least one)
	Private  bool                  `json:"private,omitempty"` // if true, create a private room (all managers will be granted access)
}

// `staff-create-room-reply` returns the outcome of a room creation request.
type StaffCreateRoomReply struct {
	Success       bool   `json:"success"`                  // whether the room was created
	FailureReason string `json:"failure_reason,omitempty"` // if `success` was false, the reason why
}

// The `staff-revoke-access` command is a version of the [revoke-access](#revoke-access)
// command that is available to staff. The staff account does not need to be a manager
// of the room to use this command.
type StaffRevokeAccessCommand RevokeAccessCommand

// `staff-revoke-access-reply` confirms that requested access capability was revoked.
type StaffRevokeAccessReply RevokeAccessReply

// The `staff-revoke-manager` command is a version of the [revoke-manager](#revoke-access)
// command that is available to staff. The staff account does not need to be a manager
// of the room to use this command.
type StaffRevokeManagerCommand RevokeManagerCommand

// `staff-revoke-manager-reply` confirms that requested manager capability was revoked.
type StaffRevokeManagerReply RevokeManagerReply

// The `staff-lock-room` command makes a room private. If the room is already private,
// then it generates a new message key (which currently invalidates all access grants).
type StaffLockRoomCommand struct{}

// `staff-lock-room-reply` confirms that the room has been made newly private.
type StaffLockRoomReply struct{}

// The `unlock-staff-capability` command may be called by a staff account to gain access to
// staff commands.
type UnlockStaffCapabilityCommand struct {
	Password string `json:"password"` // the account's password
}

// `unlock-staff-capability-reply` returns the outcome of unlocking the staff
// capability.
type UnlockStaffCapabilityReply struct {
	Success       bool   `json:"success"`                  // whether staff capability was unlocked
	FailureReason string `json:"failure_reason,omitempty"` // if `success` was false, the reason why
}

// The `who` command requests a list of sessions currently joined in the room.
type WhoCommand struct{}

// The `who-reply` packet lists the sessions currently joined in the room.
type WhoReply struct {
	Listing `json:"listing"` // a list of session views
}

type Packet struct {
	ID    string          `json:"id,omitempty"`    // client-generated id for associating replies with commands
	Type  PacketType      `json:"type"`            // the name of the command, reply, or event
	Data  json.RawMessage `json:"data,omitempty"`  // the payload of the command, reply, or event
	Error string          `json:"error,omitempty"` // this field appears in replies if a command fails

	Throttled       bool   `json:"throttled,omitempty"`        // this field appears in replies to warn the client that it may be flooding; the client should slow down its command rate
	ThrottledReason string `json:"throttled_reason,omitempty"` // if throttled is true, this field describes why
}

func (cmd *Packet) Payload() (interface{}, error) {
	if cmd.Error != "" {
		return &ErrorReply{Error: cmd.Error}, nil
	}
	payloadType, ok := payloadMap[cmd.Type]
	if !ok {
		return nil, fmt.Errorf("invalid command type: %s", cmd.Type)
	}
	payload := reflect.New(payloadType).Interface()
	if payload != nil && payloadType.NumField() > 0 {
		if err := json.Unmarshal(cmd.Data, payload); err != nil {
			return nil, err
		}
	}
	return payload, nil
}

func (cmd *Packet) Encode() ([]byte, error) { return json.Marshal(cmd) }

func MakeResponse(
	refID string, msgType PacketType, payload interface{}, throttled bool) (*Packet, error) {

	packet := &Packet{
		ID:   refID,
		Type: msgType.Reply(),
	}

	if throttled {
		packet.Throttled = true
		packet.ThrottledReason = "woah, slow down there"
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
	case *DisconnectEvent:
		packet.Type = DisconnectEventType
	case *PingEvent:
		packet.Type = PingEventType
	case *NetworkEvent:
		packet.Type = NetworkEventType
	case *SnapshotEvent:
		packet.Type = SnapshotEventType
	case *HelloEvent:
		packet.Type = HelloEventType
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
