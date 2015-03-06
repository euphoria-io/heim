package filestore

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"heim/objstore"
	"heim/proto/security"
)

func verifyPath(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %s", path, err)
			}
			return nil
		}
		return fmt.Errorf("stat %s: %s", path, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func newID() (string, error) {
	// Make ID lengths equal to AES256 block size so we can use as IV.
	buf := make([]byte, security.AES256.BlockSize())
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("rand: %s", err)
	}
	return hex.EncodeToString(buf), nil
}

func keyHeaders(key *security.ManagedKey) (http.Header, error) {
	header := http.Header{}

	if key != nil {
		if key.Encrypted() {
			return nil, security.ErrKeyMustBeDecrypted
		}
		switch key.KeyType {
		case security.AES256:
			header.Set("x-heim-dev-key-type", "AES256")
			header.Set("x-heim-dev-key", base64.URLEncoding.EncodeToString(key.Plaintext))
		default:
			return nil, fmt.Errorf("key type %s not supported", key.KeyType)
		}
	}

	return header, nil
}

func parseID(uri string) ([]byte, error) {
	if len(uri) != security.AES256.BlockSize()*2+1 || !strings.HasPrefix(uri, "/") {
		return nil, fmt.Errorf("invalid id")
	}
	return hex.DecodeString(uri[1:])
}

func parseRequest(r *http.Request) (string, *security.ManagedKey, error) {
	idBytes, err := parseID(r.URL.Path)
	if err != nil {
		return "", nil, err
	}
	id := r.URL.Path[1:]

	switch keyType := r.Header.Get("x-heim-dev-key-type"); keyType {
	case "":
		return id, nil, nil
	case "AES256":
		data, err := base64.URLEncoding.DecodeString(r.Header.Get("x-heim-dev-key"))
		if err != nil {
			return "", nil, err
		}
		if len(data) != security.AES256.KeySize() {
			return "", nil, security.ErrInvalidKey
		}
		key := &security.ManagedKey{
			KeyType:   security.AES256,
			Plaintext: data,
			IV:        idBytes,
		}
		return id, key, nil
	default:
		return "", nil, fmt.Errorf("key type %s not supported", keyType)
	}
}

func Open(path, baseURL string) (*Store, error) {
	if err := verifyPath(path); err != nil {
		return nil, err
	}

	store := &Store{
		root:    path,
		baseURL: baseURL,
	}

	return store, nil
}

type Store struct {
	root    string
	baseURL string
}

func (s *Store) Create(key *security.ManagedKey) (*objstore.UploadHandle, error) {
	header, err := keyHeaders(key)
	if err != nil {
		return nil, fmt.Errorf("filestore create: %s", err)
	}

	id, err := newID()
	if err != nil {
		return nil, fmt.Errorf("filestore create: %s", err)
	}

	handle := &objstore.UploadHandle{
		ID:     id,
		Header: header,
		Method: "PUT",
		URL:    s.baseURL + "/" + id,
	}
	return handle, nil
}

func (s *Store) Get(id string, key *security.ManagedKey) (*objstore.DownloadHandle, error) {
	header, err := keyHeaders(key)
	if err != nil {
		return nil, fmt.Errorf("filestore get: %s", err)
	}

	handle := &objstore.DownloadHandle{
		Header: header,
		URL:    s.baseURL + "/" + id,
	}
	return handle, nil
}

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.serveGet(w, r)
	case "PUT":
		s.servePut(w, r)
	default:
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
	}
}

func (s *Store) path(id string) string { return filepath.Join(s.root, id[:2], id[2:4], id) }

func (s *Store) serveGet(w http.ResponseWriter, r *http.Request) {
	id, key, err := parseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path := s.path(id)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if key != nil {
		if err := key.BlockCrypt(key.IV, key.Plaintext, data, false); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = key.Unpad(data)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Store) servePut(w http.ResponseWriter, r *http.Request) {
	id, key, err := parseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if key != nil {
		data = key.Pad(data)
		if err := key.BlockCrypt(key.IV, key.Plaintext, data, true); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	path := s.path(id)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := f.Write(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := f.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
