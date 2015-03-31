package psql

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
)

type BannedAgent struct {
	AgentID       string `db:"agent_id"`
	Room          sql.NullString
	Created       time.Time
	Expires       gorp.NullTime
	RoomReason    string `db:"room_reason"`
	AgentReason   string `db:"agent_reason"`
	PrivateReason string `db:"private_reason"`
}
