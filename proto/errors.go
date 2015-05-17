package proto

import "fmt"

var (
	ErrAccessDenied     = fmt.Errorf("access denied")
	ErrInvalidNick      = fmt.Errorf("invalid nick")
	ErrMessageNotFound  = fmt.Errorf("message not found")
	ErrEditInconsistent = fmt.Errorf("edit inconsistent")
	ErrInvalidParent    = fmt.Errorf("invalid parent ID")
)
