package psql

import "time"

type SessionLog struct {
	SessionID string `db:"session_id"`
	IP        string
	Room      string
	UserAgent string `db:"user_agent"`
	Connected time.Time
}
