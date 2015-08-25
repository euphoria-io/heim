package psql

import (
	"database/sql"
	"time"

	"gopkg.in/gorp.v1"
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

type BannedIP struct {
	IP      string `db:"ip"`
	Room    sql.NullString
	Created time.Time
	Expires gorp.NullTime
	Reason  string
}
