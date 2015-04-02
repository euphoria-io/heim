package psql

import "time"

type MediaObject struct {
	ID              string
	Room            string
	AgentID         string `db:"agent_id"`
	Created         time.Time
	Updated         time.Time
	EncryptionKeyID string `db:"encryption_key_id"`
}

type MediaTranscoding struct {
	MediaID     string `db:"media_id"`
	Name        string
	URI         string
	ContentType string `db:"content_type"`
	Size        uint64
	Width       int
	Height      int
}
