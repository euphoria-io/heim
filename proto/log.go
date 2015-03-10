package proto

import (
	"heim/proto/snowflake"

	"euphoria.io/scope"
)

// The Log provides slices of a Room's message tree, flattened and sorted
// chronologically.
type Log interface {
	Latest(scope.Context, int, snowflake.Snowflake) ([]Message, error)
}
