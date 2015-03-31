package proto

import (
	"io"
	"net/http"

	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type MediaSet struct {
	Object       MediaObject
	Transcodings map[string]Transcoding
	Handles      map[string]DownloadHandle
}

type MediaObject struct {
	ID       string        `json:"id"`
	Room     string        `json:"room"`
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
	Create(scope.Context, *security.ManagedKey) (*UploadHandle, error)
	Get(ctx scope.Context, uri string, key *security.ManagedKey) (*DownloadHandle, error)
}

type UploadHandle struct {
	ID      string      `json:"id"`
	Headers http.Header `json:"headers"`
	Method  string      `json:"method"`
	URL     string      `json:"url"`
}

func (uh *UploadHandle) Upload(ctx scope.Context, r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(uh.Method, uh.URL, r)
	if err != nil {
		return nil, err
	}
	for k, vs := range uh.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return do(ctx, req)
}

type DownloadHandle struct {
	Headers http.Header `json:"headers"`
	URL     string      `json:"url"`
}

func (dh *DownloadHandle) Download(ctx scope.Context) (*http.Response, error) {
	req, err := http.NewRequest("GET", dh.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range dh.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return do(ctx, req)
}

func do(ctx scope.Context, req *http.Request) (*http.Response, error) {
	c := http.DefaultClient
	t := c.Transport
	if t == nil {
		t = http.DefaultTransport
	}

	type result struct {
		r   *http.Response
		err error
	}
	ch := make(chan result)
	go func() {
		res := result{}
		res.r, res.err = http.DefaultClient.Do(req)
		ch <- res
	}()

	select {
	case <-ctx.Done():
		type requestCanceller interface {
			CancelRequest(*http.Request)
		}
		if rc, ok := t.(requestCanceller); ok {
			rc.CancelRequest(req)
		}
		return nil, ctx.Err()
	case res := <-ch:
		return res.r, res.err
	}
}
