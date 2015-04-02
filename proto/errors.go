package proto

import "fmt"

var (
	ErrAccessDenied     = fmt.Errorf("access denied")
	ErrEditInconsistent = fmt.Errorf("edit inconsistent")
	ErrInvalidNick      = fmt.Errorf("invalid nick")
	ErrMediaNotFound    = fmt.Errorf("media not found")
	ErrMessageNotFound  = fmt.Errorf("message not found")
	ErrInvalidParent    = fmt.Errorf("invalid parent ID")
)
