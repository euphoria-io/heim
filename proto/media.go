package proto

import (
	"euphoria.io/scope"
)

type RawImage []byte

type MediaSet struct {
	Object       MediaObject
	Transcodings map[string]Transcoding
}

type MediaObject struct {
	ID       string        `json:"id"`
	Room     string        `json:"room"`
	Storage  string        `json:"-"`
	Uploader *IdentityView `json:"uploader"`
	Created  Time          `json:"created"`
	Uploaded Time          `json:"uploaded,omitempty"`
}

type Transcoding struct {
	MediaID     string `json:"media_id"`
	Name        string `json:"name"`
	URI         string `json:"-"`
	ContentType string `json:"content_type"`
	Size        uint64 `json:"size"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}

type MediaStore interface {
	Get(ctx scope.Context, mediaID string) (RawImage, error)

	Store(ctx scope.Context, id string, img RawImage) error
}
