package proto

import "fmt"

var (
	ErrAccessDenied = fmt.Errorf("access denied")
	ErrInvalidNick  = fmt.Errorf("invalid nick")
)
