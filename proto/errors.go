package proto

import "fmt"

var (
	ErrAccessDenied             = fmt.Errorf("access denied")
	ErrAccountIdentityInUse     = fmt.Errorf("account identity already in use")
	ErrAccountNotFound          = fmt.Errorf("account not found")
	ErrAgentAlreadyExists       = fmt.Errorf("agent already exists")
	ErrAgentNotFound            = fmt.Errorf("agent not found")
	ErrCapabilityNotFound       = fmt.Errorf("capability not found")
	ErrClientKeyNotFound        = fmt.Errorf("client key not found")
	ErrEditInconsistent         = fmt.Errorf("edit inconsistent")
	ErrInvalidNick              = fmt.Errorf("invalid nick")
	ErrInvalidParent            = fmt.Errorf("invalid parent ID")
	ErrInvalidVerificationToken = fmt.Errorf("invalid verification token")
	ErrNotLoggedIn              = fmt.Errorf("not logged in")
	ErrManagerNotFound          = fmt.Errorf("manager not found")
	ErrMessageNotFound          = fmt.Errorf("message not found")
	ErrPersonalIdentityInUse    = fmt.Errorf("personal identity already in use")
	ErrRoomNotFound             = fmt.Errorf("room not found")
)
