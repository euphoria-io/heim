package backend

import (
	"fmt"
	"net/http"
	"strings"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type resolverEntry struct {
	resolver proto.MediaResolver
	key      string
	secret   string
}

type MediaDispatcher struct {
	resolvers       map[string]resolverEntry
	defaultResolver string
}

func (d *MediaDispatcher) Add(name string, resolver proto.MediaResolver, key, secret string) {
	r := resolverEntry{
		resolver: resolver,
		key:      key,
		secret:   secret,
	}
	if d.resolvers == nil {
		d.resolvers = map[string]resolverEntry{name: r}
	} else {
		d.resolvers[name] = r
	}
}

func (d *MediaDispatcher) SetDefault(name string) error {
	if _, err := d.Get(name); err != nil {
		return err
	}
	d.defaultResolver = name
	return nil
}

func (d *MediaDispatcher) Default() (string, proto.MediaResolver, error) {
	if d.resolvers == nil || d.defaultResolver == "" {
		return "", nil, fmt.Errorf("media uploads/downloads not configured")
	}
	resolver, err := d.Get(d.defaultResolver)
	if err != nil {
		return "", nil, err
	}
	return d.defaultResolver, resolver, nil
}

func (d *MediaDispatcher) Get(name string) (proto.MediaResolver, error) {
	entry, ok := d.resolvers[name]
	if !ok {
		return nil, fmt.Errorf("no such media resolver: %s", name)
	}
	return entry.resolver, nil
}

func (d *MediaDispatcher) Auth(w http.ResponseWriter, r *http.Request, name string) error {
	entry, ok := d.resolvers[name]
	if !ok {
		return fmt.Errorf("no such media resolver: %s", name)
	}

	key, secret, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "authorization required", http.StatusUnauthorized)
		return fmt.Errorf("authorization required")
	}

	if key != entry.key || secret != entry.secret {
		http.Error(w, "forbidden", http.StatusForbidden)
		return fmt.Errorf("forbidden")
	}

	return nil
}

type simpleMediaResolver string

func (url simpleMediaResolver) Create(ctx scope.Context, mediaID string, key *security.ManagedKey) (
	*proto.UploadHandle, error) {

	// TODO: add encryption header
	handle := &proto.UploadHandle{
		ID:     mediaID,
		Method: "PUT",
		URL:    strings.TrimRight(string(url), "/") + "/" + mediaID,
	}
	return handle, nil
}

func (url simpleMediaResolver) Get(ctx scope.Context, mediaID string, key *security.ManagedKey) (
	*proto.DownloadHandle, error) {

	// TODO: add encryption header
	handle := &proto.DownloadHandle{
		URL: strings.TrimRight(string(url), "/") + "/" + mediaID,
	}
	return handle, nil
}
