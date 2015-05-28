package proto

import "fmt"

var (
	ErrAccessDenied          = fmt.Errorf("access denied")
	ErrAccountIdentityInUse  = fmt.Errorf("account identity already in use")
	ErrAccountNotFound       = fmt.Errorf("account not found")
	ErrPersonalIdentityInUse = fmt.Errorf("personal identity already in use")
	ErrRoomNotFound          = fmt.Errorf("room not found")
	ErrInvalidNick           = fmt.Errorf("invalid nick")
	ErrInvalidParent         = fmt.Errorf("invalid parent ID")
	ErrMessageNotFound       = fmt.Errorf("message not found")
	ErrEditInconsistent      = fmt.Errorf("edit inconsistent")
)
