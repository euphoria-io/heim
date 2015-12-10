package templates

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func splitEmail(data []byte) (textproto.MIMEHeader, []byte) {
	r := bufio.NewReader(bytes.NewReader(data))
	tr := textproto.NewReader(r)
	header, err := tr.ReadMIMEHeader()
	So(err, ShouldBeNil)
	content, err := ioutil.ReadAll(r)
	So(err, ShouldBeNil)
	return header, content
}

func TestEmail(t *testing.T) {
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

	Convey("WriteTo", t, func() {
		e := &Email{
			Header: textproto.MIMEHeader{"Subject": []string{"test"}},
			Text:   []byte("text"),
			HTML:   []byte("html"),
			Attachments: []Attachment{
				{
					Name:      "a.png",
					ContentID: "a.png@localhost",
					Content:   []byte("lmao"),
				},
				{
					Name:      "b.png",
					ContentID: "b.png@localhost",
					Content:   []byte("b"),
				},
			},
		}

		buf := &bytes.Buffer{}
		n, err := e.WriteTo(buf)
		So(err, ShouldBeNil)
		Printf(buf.String())
		So(n, ShouldEqual, buf.Len())

		header, content := splitEmail(buf.Bytes())
		So(header, ShouldResemble, textproto.MIMEHeader{
			"Subject":      []string{"test"},
			"Mime-Version": []string{"1.0"},
			"Content-Type": []string{header.Get("Content-Type")},
		})
		contentType, contentParams, err := mime.ParseMediaType(header.Get("Content-Type"))
		So(err, ShouldBeNil)
		So(contentType, ShouldEqual, "multipart/alternative")
		boundary := contentParams["boundary"]
		mpr := multipart.NewReader(bytes.NewReader(content), boundary)

		// Verify text part.
		part, err := mpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Type": []string{`text/plain; charset="utf-8"; format="fixed"`},
		})
		data, err := ioutil.ReadAll(part)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, "text")

		// Verify html mime/multipart part.
		part, err = mpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Type": []string{part.Header.Get("Content-Type")},
			"Mime-Version": []string{"1.0"},
		})
		innerContentType, innerContentParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		So(err, ShouldBeNil)
		So(innerContentType, ShouldEqual, "multipart/related")
		innerBoundary := innerContentParams["boundary"]
		htmlmpr := multipart.NewReader(part, innerBoundary)

		// Verify html part.
		part, err = htmlmpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Type": []string{`text/html; charset="utf-8"`},
		})
		data, err = ioutil.ReadAll(part)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, "html")

		// Verify attachments.
		part, err = htmlmpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Id":                []string{"<a.png@localhost>"},
			"Content-Type":              []string{"image/png"},
			"Content-Transfer-Encoding": []string{"base64"},
			"Content-Disposition":       []string{`inline; filename="a.png"`},
		})
		decoder := base64.NewDecoder(base64.StdEncoding, part)
		data, err = ioutil.ReadAll(decoder)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, "lmao")

		part, err = htmlmpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Id":                []string{"<b.png@localhost>"},
			"Content-Type":              []string{"image/png"},
			"Content-Transfer-Encoding": []string{"base64"},
			"Content-Disposition":       []string{`inline; filename="b.png"`},
		})
		decoder = base64.NewDecoder(base64.StdEncoding, part)
		data, err = ioutil.ReadAll(decoder)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, "b")
	})

	Convey("EvaluateEmail", t, func() {
		td := tempdir()
		defer os.RemoveAll(td)

		write(td, "test.hdr", "Subject: test")
		write(td, "test.html", `<img src="{{.File "a.png"}}">`)
		write(td, "test.txt", "text")
		write(filepath.Join(td, "static"), "a.png", "lmao")

		templater := &Templater{}
		So(templater.Load(td), ShouldBeNil)

		e, err := EvaluateEmail(templater, "test", &StaticFiles{})
		So(err, ShouldBeNil)
		So(e, ShouldResemble, &Email{
			Header: textproto.MIMEHeader{"Subject": []string{"test"}},
			Text:   []byte("text"),
			HTML:   []byte(`<img src="cid:a.png@localhost">`),
			Attachments: []Attachment{
				{
					Name:      "a.png",
					ContentID: "a.png@localhost",
					Content:   []byte("lmao"),
				},
			},
		})
	})
}
