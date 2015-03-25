package media

import (
	"io"
	"net/http"
)

type Store interface {
	Create() (*UploadHandle, error)
	Get(id string) (*DownloadHandle, error)
}

type UploadHandle struct {
	ID     string
	Header http.Header
	Method string
	URL    string
}

func (uh *UploadHandle) Upload(r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(uh.Method, uh.URL, r)
	if err != nil {
		return nil, err
	}
	for k, vs := range uh.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return http.DefaultClient.Do(req)
}

type DownloadHandle struct {
	Header http.Header
	URL    string
}

func (dh *DownloadHandle) Download() (*http.Response, error) {
	req, err := http.NewRequest("GET", dh.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range dh.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return http.DefaultClient.Do(req)
}
