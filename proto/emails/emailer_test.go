package emails

import (
	"bufio"
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"

	"euphoria.io/heim/templates"
	"euphoria.io/scope"

	. "github.com/smartystreets/goconvey/convey"
)

func parseEmail(data []byte) *templates.Email {
	r := bufio.NewReader(bytes.NewReader(data))
	hr := textproto.NewReader(r)
	h, err := hr.ReadMIMEHeader()
	So(err, ShouldBeNil)

	So(h.Get("Mime-Version"), ShouldEqual, "1.0")
	ctype := h.Get("Content-Type")
	So(ctype, ShouldStartWith, "multipart/alternative;")
	So(ctype, ShouldEndWith, `"`)
	idx := strings.Index(ctype, `boundary="`)
	So(idx, ShouldBeGreaterThan, -1)
	mpr := multipart.NewReader(r, ctype[idx+len(`boundary="`):len(ctype)-1])

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
		disposition := part.Header.Get("Content-Disposition")
		idx = strings.Index(disposition, `filename="`)
		So(idx, ShouldBeGreaterThan, -1)
		filename := disposition[idx+len(`filename="`):]
		idx = strings.IndexRune(filename, '"')
		So(idx, ShouldBeGreaterThan, -1)
		filename = filename[:idx]
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
	Convey("TemplateEmailer", t, func() {
		tmpl, err := template.New("test").Parse(`
			{{define "test.html"}}html part{{end}}
			{{define "test.txt"}}text part{{end}}
			{{define "test.hdr"}}Subject: test{{end}}`)
		So(err, ShouldBeNil)

		e := &TemplateEmailer{
			Deliverer: &TestDeliverer{},
			Templater: &templates.Templater{Templates: map[string]*template.Template{"test": tmpl}},
		}

		Convey("Send test email", func() {
			c := e.Deliverer.(MockDeliverer).Inbox("test@heim.invalid")
			msgID, err := e.Send(scope.New(), "test@heim.invalid", "test", nil)
			So(err, ShouldBeNil)

			email := parseEmail(<-c)
			So(email.Header.Get("Message-ID"), ShouldEqual, msgID)
			So(email.Header.Get("Subject"), ShouldEqual, "test")
			So(string(email.Text), ShouldEqual, "text part")
			So(string(email.HTML), ShouldEqual, "html part")
		})
	})
}

func TestTestEmailer(t *testing.T) {
	Convey("TemplateEmailer", t, func() {
		tmpl, err := template.New("test").Parse(`
			{{define "test.html"}}html part{{end}}
			{{define "test.txt"}}text part{{end}}
			{{define "test.hdr"}}Subject: test{{end}}`)
		So(err, ShouldBeNil)

		e := &TestEmailer{
			Templater: &templates.Templater{Templates: map[string]*template.Template{"test": tmpl}},
		}

		Convey("Send test email", func() {
			c := e.Inbox("test@heim.invalid")
			msgID, err := e.Send(scope.New(), "test@heim.invalid", "test", nil)
			So(err, ShouldBeNil)

			msg := <-c
			So(msg.MessageID, ShouldEqual, msgID)
			So(msg.TemplateName, ShouldEqual, "test")

			email := parseEmail(msg.Delivery)
			So(email.Header.Get("Subject"), ShouldEqual, "test")
			So(string(email.Text), ShouldEqual, "text part")
			So(string(email.HTML), ShouldEqual, "html part")
		})
	})
}
