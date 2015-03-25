package media

import (
	"io"
	"net/http"

	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type Store interface {
	Create(scope.Context, *security.ManagedKey) (*UploadHandle, error)
	Get(ctx scope.Context, id string, key *security.ManagedKey) (*DownloadHandle, error)
}

type UploadHandle struct {
	ID     string
	Header http.Header
	Method string
	URL    string
}

func (uh *UploadHandle) Upload(ctx scope.Context, r io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(uh.Method, uh.URL, r)
	if err != nil {
		return nil, err
	}
	for k, vs := range uh.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return do(ctx, req)
}

type DownloadHandle struct {
	Header http.Header
	URL    string
}

func (dh *DownloadHandle) Download(ctx scope.Context) (*http.Response, error) {
	req, err := http.NewRequest("GET", dh.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range dh.Header {
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
