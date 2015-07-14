package emails

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"strings"
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

func TestSMTPEmailer(t *testing.T) {
	Convey("Result serializer", t, func() {
		s := &SMTPEmailer{
			Templater: &Templater{
				staticFiles: map[string][]byte{
					"test.png": []byte("test image"),
				},
			},
		}
		result := &TemplateResult{
			Header: textproto.MIMEHeader{
				"Subject": []string{"Hi"},
			},
			Text:        []byte("text part"),
			HTML:        []byte("html part"),
			Attachments: map[string]string{"test.png": "test.png@localhost"},
		}
		buf := &bytes.Buffer{}
		So(s.write(buf, result), ShouldBeNil)

		header, content := splitEmail(buf.Bytes())
		ctype := header.Get("Content-Type")
		So(ctype, ShouldStartWith, "multipart/alternative")
		So(ctype, ShouldEndWith, `"`)
		idx := strings.Index(ctype, `boundary="`)
		So(idx, ShouldBeGreaterThan, 0)
		boundary := ctype[idx+len(`boundary="`) : len(ctype)-1]

		So(header, ShouldResemble, textproto.MIMEHeader{
			"Subject":      []string{"Hi"},
			"Mime-Version": []string{"1.0"},
			"Content-Type": []string{header.Get("Content-Type")},
		})

		Printf("boundary: %s\n", boundary)
		Printf("content: %s\n", string(content))

		mpr := multipart.NewReader(bytes.NewReader(content), boundary)

		// Verify text part.
		part, err := mpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Type": []string{`text/plain; charset="utf-8"; format="fixed"`},
		})
		data, err := ioutil.ReadAll(part)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, "text part")

		// Verify html part.
		part, err = mpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Type": []string{`text/html; charset="utf-8"`},
		})
		data, err = ioutil.ReadAll(part)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, "html part")

		// Verify attachment.
		part, err = mpr.NextPart()
		So(err, ShouldBeNil)
		So(part.Header, ShouldResemble, textproto.MIMEHeader{
			"Content-Id":                []string{"<test.png@localhost>"},
			"Content-Type":              []string{"image/png"},
			"Content-Transfer-Encoding": []string{"base64"},
			"Content-Disposition":       []string{`inline; filename="test.png"`},
		})
		decoder := base64.NewDecoder(base64.StdEncoding, part)
		data, err = ioutil.ReadAll(decoder)
		So(err, ShouldBeNil)
		So(string(data), ShouldEqual, "test image")
	})
}
