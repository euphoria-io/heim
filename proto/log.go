package proto

import (
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

// The Log provides slices of a Room's message tree, flattened and sorted
// chronologically.
type Log interface {
	GetMessage(scope.Context, snowflake.Snowflake) (*Message, error)
	Latest(scope.Context, int, snowflake.Snowflake) ([]Message, error)
}
