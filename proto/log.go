package proto

import (
	"heim/proto/snowflake"

	"golang.org/x/net/context"
)

// The Log provides slices of a Room's message tree, flattened and sorted
// chronologically.
type Log interface {
	Latest(context.Context, int, snowflake.Snowflake) ([]Message, error)
}
