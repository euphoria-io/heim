package psql

import (
	"time"

	"github.com/go-gorp/gorp"
)

type MediaObject struct {
	MediaID  string `db:"media_id"`
	Room     string
	Storage  string
	Uploader string
	Created  time.Time
	Uploaded gorp.NullTime
}

type Transcoding struct {
	MediaID     string `db:"media_id"`
	ParentID    string `db:"parent_id"`
	Name        string
	URI         string
	ContentType string `db:"content_type"`
	Size        uint64
	Width       int
	Height      int
}

func NewMediaObject(mediaID string, uploader string, room string, storage string) *MediaObject {
	return &MediaObject{MediaID: mediaID,
		Room:     room,
		Storage:  storage,
		Uploader: uploader,
		Created:  time.Now(),
	}
}
