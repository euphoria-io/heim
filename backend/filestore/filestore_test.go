package filestore

import (
	"crypto/rand"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

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

	newID := func() string {
		sf, err := snowflake.New()
		if err != nil {
			t.Fatal(err)
		}
		return sf.String()
	}

	// Compile-time assertion that FileStore implements proto.MediaResolver
	_ = proto.MediaResolver(fs)

	Convey("Unencrypted file store", t, func() {
		ctx := scope.New()

		uh, err := fs.Create(ctx, newID(), nil)
		So(err, ShouldBeNil)

		_, err = os.Stat(fs.path(uh.ID))
		So(err, ShouldNotBeNil)
		So(os.IsNotExist(err), ShouldBeTrue)

		content := "content"
		resp, err := uh.Upload(ctx, strings.NewReader(content))
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, http.StatusCreated)

		f, err := os.Open(fs.path(uh.ID))
		So(err, ShouldBeNil)
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, content)

		dh, err := fs.Get(ctx, uh.ID, nil)
		So(err, ShouldBeNil)
		resp, err = dh.Download(ctx)
		So(err, ShouldBeNil)
		data, err = ioutil.ReadAll(resp.Body)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, content)
	})

	Convey("Encrypted file store", t, func() {
		ctx := scope.New()

		keyBytes := make([]byte, security.AES256.KeySize())
		_, err := rand.Read(keyBytes)
		So(err, ShouldBeNil)
		key := &security.ManagedKey{KeyType: security.AES256, Plaintext: keyBytes}

		uh, err := fs.Create(ctx, newID(), key)
		So(err, ShouldBeNil)

		content := "secret content"
		resp, err := uh.Upload(ctx, strings.NewReader(content))
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, http.StatusCreated)

		f, err := os.Open(fs.path(uh.ID))
		So(err, ShouldBeNil)
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		So(err, ShouldBeNil)
		So(string(data), ShouldNotEqual, content)

		dh, err := fs.Get(ctx, uh.ID, key)
		So(err, ShouldBeNil)
		resp, err = dh.Download(ctx)
		So(err, ShouldBeNil)
		data, err = ioutil.ReadAll(resp.Body)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, content)
	})
}
