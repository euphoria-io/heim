package templates

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"path/filepath"
	"sort"
)

type writeCounter struct {
	w io.Writer
	n int64
}

func (wc *writeCounter) Write(data []byte) (int, error) {
	n, err := wc.w.Write(data)
	wc.n += int64(n)
	return n, err
}

type Email struct {
	Header      textproto.MIMEHeader
	Text        []byte
	HTML        []byte
	Attachments []Attachment
}

func (e *Email) WriteTo(w io.Writer) (int64, error) {
	wc := &writeCounter{w: w}
	w = wc
	mpw := multipart.NewWriter(wc)

	// Write top-level headers.
	headers := e.Header
	headers.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, mpw.Boundary()))
	headers.Set("MIME-Version", "1.0")
	for k, vv := range headers {
		for _, v := range vv {
			if _, err := fmt.Fprintf(w, "%s: %s\n", k, v); err != nil {
				return wc.n, err
			}
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return wc.n, err
	}

	// Write text part.
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", `text/plain; charset="utf-8"; format="fixed"`)
	textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := mpw.CreatePart(textHeader)
	if err != nil {
		return wc.n, fmt.Errorf("create text part: %s", err)
	}
	qpw := quotedprintable.NewWriter(pw)
	if _, err := qpw.Write(e.Text); err != nil {
		return wc.n, fmt.Errorf("write text part: %s", err)
	}
	if err := qpw.Close(); err != nil {
		return wc.n, fmt.Errorf("close text part: %s", err)
	}

	// Write top-level HTML multipart/related headers.

	// Chicken-and-egg: we need the inner multipart boundary to write the outer
	// part headers, but we can't get the boundary until we've constructed
	// multipart.Writer with a part writer. We'll generate the boundary ahead
	// of time and then manually supply it to the inner multipart.Writer.
	htmlRelatedBoundary := multipart.NewWriter(nil).Boundary()

	htmlRelatedHeader := textproto.MIMEHeader{}
	htmlRelatedHeader.Set("Content-Type", fmt.Sprintf(`multipart/related; boundary="%s"`, htmlRelatedBoundary))
	htmlRelatedHeader.Set("MIME-Version", "1.0")
	pw, err = mpw.CreatePart(htmlRelatedHeader)
	if err != nil {
		return wc.n, fmt.Errorf("create html multipart part: %s", err)
	}

	// Create inner multipart data for HTML with attachments.
	htmlmpw := multipart.NewWriter(pw)
	htmlmpw.SetBoundary(htmlRelatedBoundary)
	if err != nil {
		return wc.n, fmt.Errorf("set html multipart boundary: %s", err)
	}

	// Write html part.
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", `text/html; charset="utf-8"`)
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlpw, err := htmlmpw.CreatePart(htmlHeader)
	if err != nil {
		return wc.n, fmt.Errorf("create html part: %s", err)
	}
	qpw = quotedprintable.NewWriter(htmlpw)
	if _, err := qpw.Write(e.HTML); err != nil {
		return wc.n, fmt.Errorf("write html part: %s", err)
	}
	if err := qpw.Close(); err != nil {
		return wc.n, fmt.Errorf("close html part: %s", err)
	}

	// Write attachments.
	for _, att := range e.Attachments {
		attHeader := textproto.MIMEHeader{}
		attHeader.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
		attHeader.Set("Content-Type", mime.TypeByExtension(filepath.Ext(att.Name)))
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, att.Name))
		htmlpw, err := htmlmpw.CreatePart(attHeader)
		if err != nil {
			return wc.n, fmt.Errorf("create attachment %s: %s", att.Name, err)
		}
		b64w := base64.NewEncoder(base64.StdEncoding, htmlpw)
		if _, err := b64w.Write(att.Content); err != nil {
			return wc.n, fmt.Errorf("write attachment %s: %s", att.Name, err)
		}
		if err := b64w.Close(); err != nil {
			return wc.n, fmt.Errorf("close attachment %s: %s", att.Name, err)
		}
	}

	// Finalize HTML multipart/related.
	if err := htmlmpw.Close(); err != nil {
		return wc.n, fmt.Errorf("multipart close: %s", err)
	}

	// Finalize.
	if err := mpw.Close(); err != nil {
		return wc.n, fmt.Errorf("multipart close: %s", err)
	}

	return wc.n, nil
}

func EvaluateEmail(t *Templater, baseName string, context interface{}) (*Email, error) {
	email := &Email{}

	headerBytes, err := t.Evaluate(baseName+".hdr", context)
	if err != nil {
		return nil, fmt.Errorf("%s.hdr: %s", baseName, err)
	}

	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(headerBytes)))
	email.Header, err = r.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("%s.hdr: %s", baseName, err)
	}

	if email.Text, err = t.Evaluate(baseName+".txt", context); err != nil {
		return nil, fmt.Errorf("%s.txt: %s", baseName, err)
	}

	if email.HTML, err = t.Evaluate(baseName+".html", context); err != nil {
		return nil, fmt.Errorf("%s.html: %s", baseName, err)
	}

	if sf, ok := context.(staticFiles); ok {
		attachments := sf.Attachments()
		email.Attachments = make([]Attachment, 0, len(attachments))
		for _, attachment := range attachments {
			email.Attachments = append(email.Attachments, attachment)
		}
		sort.Sort(attachmentList(email.Attachments))
	}

	return email, nil
}
