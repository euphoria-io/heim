package filestore

import (
	"crypto/rand"
	"heim/proto/security"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestStore(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "heim-filestore")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Need some indirection to deal with a chicken-egg problem.
	var fs *Store
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
	defer server.Close()

	fs, err = Open(tempDir, server.URL)
	if err != nil {
		t.Fatal(err)
	}

	Convey("Unencrypted file store", t, func() {
		uh, err := fs.Create(nil)
		So(err, ShouldBeNil)

		_, err = os.Stat(fs.path(uh.ID))
		So(err, ShouldNotBeNil)
		So(os.IsNotExist(err), ShouldBeTrue)

		content := "content"
		resp, err := uh.Upload(strings.NewReader(content))
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, http.StatusCreated)

		f, err := os.Open(fs.path(uh.ID))
		So(err, ShouldBeNil)
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, content)

		dh, err := fs.Get(uh.ID, nil)
		So(err, ShouldBeNil)
		resp, err = dh.Download()
		So(err, ShouldBeNil)
		data, err = ioutil.ReadAll(resp.Body)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, content)
	})

	Convey("Encrypted file store", t, func() {
		keyBytes := make([]byte, security.AES256.KeySize())
		_, err := rand.Read(keyBytes)
		So(err, ShouldBeNil)
		key := &security.ManagedKey{KeyType: security.AES256, Plaintext: keyBytes}

		uh, err := fs.Create(key)
		So(err, ShouldBeNil)

		content := "secret content"
		resp, err := uh.Upload(strings.NewReader(content))
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, http.StatusCreated)

		f, err := os.Open(fs.path(uh.ID))
		So(err, ShouldBeNil)
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		So(err, ShouldBeNil)
		So(string(data), ShouldNotEqual, content)

		dh, err := fs.Get(uh.ID, key)
		So(err, ShouldBeNil)
		resp, err = dh.Download()
		So(err, ShouldBeNil)
		data, err = ioutil.ReadAll(resp.Body)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, content)
	})
}
