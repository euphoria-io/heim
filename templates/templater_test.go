package templates

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type errorData struct{}

func (errorData) Error(msg string) (string, error) { return "", errors.New(msg) }

func TestTemplater(t *testing.T) {
	tempdir := func() string {
		td, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(td, "static"), 0755); err != nil {
			t.Fatal(err)
		}
		return td
	}

	write := func(tmpDir, name, content string) {
		path := filepath.Join(tmpDir, name)
		So(ioutil.WriteFile(path, []byte(content), 0644), ShouldBeNil)
	}

	Convey("Pair of templates", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "welcome.hdr", "Subject: {{.Subject}}\nReply-To: max@euphoria.io")
		write(td, "welcome.txt", "subject was {{.Subject}}")
		write(td, "welcome.html", "<h1>{{.Subject}}</h1>")

		write(td, "alert.hdr", "Subject: alert!")
		write(td, "alert.html", "alert!")

		templater := &Templater{}
		So(templater.Load(td), ShouldBeNil)

		_, ok := templater.Templates["alert"]
		So(ok, ShouldBeTrue)
		_, ok = templater.Templates["welcome"]
		So(ok, ShouldBeTrue)
	})

	Convey("Static files", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(filepath.Join(td, "static"), "a.png", "ayyy")
		write(filepath.Join(td, "static"), "b.png", "lmao")

		templater := &Templater{}
		So(templater.Load(td), ShouldBeNil)

		content, ok := templater.staticFiles["a.png"]
		So(ok, ShouldBeTrue)
		So(string(content), ShouldEqual, "ayyy")
		content, ok = templater.staticFiles["b.png"]
		So(ok, ShouldBeTrue)
		So(string(content), ShouldEqual, "lmao")
	})

	Convey("Base directory not found", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		templater := &Templater{}
		errs := templater.Load(filepath.Join(td, "notfound"))
		So(len(errs), ShouldEqual, 1)
		So(errs[0].Error(), ShouldEndWith, "/notfound: no such file or directory")
	})

	Convey("Template unreadable", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "c.html", "c")
		So(os.Symlink(filepath.Join(td, "nofile"), filepath.Join(td, "a.html")), ShouldBeNil)
		So(os.Symlink(filepath.Join(td, "nofile"), filepath.Join(td, "b.html")), ShouldBeNil)

		templater := &Templater{}
		errs := templater.Load(td)
		So(len(errs), ShouldEqual, 2)
		So(errs[0].Error(), ShouldEndWith, ": no such file or directory")
		So(errs[1].Error(), ShouldEndWith, ": no such file or directory")
	})

	Convey("Static directory not found", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		So(os.RemoveAll(filepath.Join(td, "static")), ShouldBeNil)

		templater := &Templater{}
		errs := templater.Load(td)
		So(len(errs), ShouldEqual, 1)
		So(errs[0].Error(), ShouldEndWith, "/static: no such file or directory")
	})

	Convey("Static file unreadable", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(filepath.Join(td, "static"), "c.png", "c")
		So(os.Symlink(filepath.Join(td, "static", "nofile"), filepath.Join(td, "static", "a.png")), ShouldBeNil)
		So(os.Symlink(filepath.Join(td, "static", "nofile"), filepath.Join(td, "static", "b.png")), ShouldBeNil)

		templater := &Templater{}
		errs := templater.Load(td)
		So(len(errs), ShouldEqual, 2)
		So(errs[0].Error(), ShouldEndWith, ": no such file or directory")
		So(errs[1].Error(), ShouldEndWith, ": no such file or directory")
	})

	Convey("Evaluate with attachments", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(filepath.Join(td, "static"), "a.png", "lmao")
		write(td, "test.html", `<img src="{{.File "a.png"}}">`)

		templater := &Templater{}
		So(templater.Load(td), ShouldBeNil)

		data := &StaticFiles{}
		content, err := templater.Evaluate("test.html", data)
		So(err, ShouldBeNil)
		So(string(content), ShouldEqual, `<img src="cid:a.png@localhost">`)
		So(data.Attachments(), ShouldResemble, map[string]Attachment{
			"a.png": Attachment{
				Name:      "a.png",
				ContentID: "a.png@localhost",
				Content:   []byte("lmao"),
			},
		})
	})

	Convey("Static file not found", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "test.html", `<img src="{{.File "a.png"}}">`)

		templater := &Templater{}
		So(templater.Load(td), ShouldBeNil)

		_, err := templater.Evaluate("test.html", &StaticFiles{})
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEndWith, "error calling File: a.png: file not available")
	})

	Convey("Evaluate headers", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "test.html", "")
		write(td, "test.hdr", "Subject: test\nReply-To: noreply@test.invalid")

		templater := &Templater{}
		So(templater.Load(td), ShouldBeNil)

		content, err := templater.Evaluate("test.hdr", nil)
		So(err, ShouldBeNil)
		So(string(content), ShouldEqual, "Subject: test\nReply-To: noreply@test.invalid\n\n")
	})

	Convey("Template not found", t, func() {
		t := &Templater{}
		_, err := t.Evaluate("test.html", nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "no template found for test.html")
	})

	Convey("Evaluation error", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "test.html", `{{.Error "test"}}`)

		templater := &Templater{}
		So(templater.Load(td), ShouldBeNil)

		_, err := templater.Evaluate("test.html", errorData{})
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEndWith, "error calling Error: test")
	})
}
