package proto

import "euphoria.io/scope"

// A Session is a connection between a client and a Room.
type Session interface {
	// ID returns the globally unique identifier for the Session.
	ID() string

	// ServerID returns the globally unique identifier of the server hosting
	// the Session.
	ServerID() string

	// Identity returns the Identity associated with the Session.
	Identity() Identity

	// SetName sets the acting nickname for the Session.
	SetName(name string)

	// Send sends a packet to the Session's client.
	Send(scope.Context, PacketType, interface{}) error

	// Close terminates the Session and disconnects the client.
	Close()

	// CheckAbandoned() issues an immediate ping to the session with a short
	// timeout.
	CheckAbandoned() error

	View() *SessionView
}

// SessionView describes a session and its identity.
type SessionView struct {
	*IdentityView
	SessionID string `json:"session_id"`           // id of the session, unique across all sessions globally
	IsStaff   bool   `json:"is_staff,omitempty"`   // if true, this session belongs to a member of staff
	IsManager bool   `json:"is_manager,omitempty"` // if true, this session belongs to a manager of the room
}
