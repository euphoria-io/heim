package emails

import (
	"bufio"
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/textproto"
	"testing"

	"euphoria.io/heim/templates"

	. "github.com/smartystreets/goconvey/convey"
)

func parseEmail(data []byte) *templates.Email {
	r := bufio.NewReader(bytes.NewReader(data))
	hr := textproto.NewReader(r)
	h, err := hr.ReadMIMEHeader()
	So(err, ShouldBeNil)

	So(h.Get("Mime-Version"), ShouldEqual, "1.0")
	contentType, contentParams, err := mime.ParseMediaType(h.Get("Content-Type"))
	So(err, ShouldBeNil)
	So(contentType, ShouldEqual, "multipart/alternative")
	boundary := contentParams["boundary"]
	mpr := multipart.NewReader(r, boundary)

	part, err := mpr.NextPart()
	So(err, ShouldBeNil)
	So(part.Header.Get("Content-Type"), ShouldEqual, `text/plain; charset="utf-8"; format="fixed"`)
	text, err := ioutil.ReadAll(part)
	So(err, ShouldBeNil)

	part, err = mpr.NextPart()
	So(err, ShouldBeNil)
	So(part.Header.Get("Content-Type"), ShouldEqual, `text/html; charset="utf-8"`)
	html, err := ioutil.ReadAll(part)
	So(err, ShouldBeNil)

	email := &templates.Email{
		Header:      h,
		Text:        text,
		HTML:        html,
		Attachments: []templates.Attachment{},
	}

	for {
		part, err = mpr.NextPart()
		if err == io.EOF {
			break
		}
		So(err, ShouldBeNil)

		contentID := part.Header.Get("Content-ID")
		_, dispositionParams, err := mime.ParseMediaType(h.Get("Content-Disposition"))
		So(err, ShouldBeNil)
		filename := dispositionParams["filename"]
		content, err := ioutil.ReadAll(part)
		So(err, ShouldBeNil)

		email.Attachments = append(email.Attachments, templates.Attachment{
			Name:      filename,
			ContentID: contentID,
			Content:   content,
		})
	}

	return email
}

func TestTemplateEmailer(t *testing.T) {
	Convey("NewEmail", t, func() {
		tmpl, err := template.New("test").Parse(`
			{{define "test.html"}}html part{{end}}
			{{define "test.txt"}}text part{{end}}
			{{define "test.hdr"}}Subject: test
From: noreply@heim.invalid{{end}}`)
		So(err, ShouldBeNil)

		templater := &templates.Templater{Templates: map[string]*template.Template{"test": tmpl}}

		Convey("Send test email", func() {
			ref, err := NewEmail(templater, "<msgid@test>", "test@heim.invalid", "test", nil)
			So(err, ShouldBeNil)
			So(ref.ID, ShouldEqual, "<msgid@test>")
			So(ref.SendTo, ShouldEqual, "test@heim.invalid")
			So(ref.SendFrom, ShouldEqual, "noreply@heim.invalid")

			email := parseEmail(ref.Message)
			So(email.Header.Get("Message-ID"), ShouldEqual, ref.ID)
			So(email.Header.Get("Subject"), ShouldEqual, "test")
			So(string(email.Text), ShouldEqual, "text part")
			So(string(email.HTML), ShouldEqual, "html part")
		})
	})
}
