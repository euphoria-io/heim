package templates

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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
	// TODO: Content-Transfer-Encoding
	pw, err := mpw.CreatePart(textHeader)
	if err != nil {
		return wc.n, fmt.Errorf("create text part: %s", err)
	}
	if _, err := pw.Write(e.Text); err != nil {
		return wc.n, fmt.Errorf("write text part: %s", err)
	}

	// Write html part.
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", `text/html; charset="utf-8"`)
	// TODO: Content-Transfer-Encoding
	pw, err = mpw.CreatePart(htmlHeader)
	if err != nil {
		return wc.n, fmt.Errorf("create html part: %s", err)
	}
	if _, err := pw.Write(e.HTML); err != nil {
		return wc.n, fmt.Errorf("write html part: %s", err)
	}

	// Write attachments.
	for _, att := range e.Attachments {
		attHeader := textproto.MIMEHeader{}
		attHeader.Set("Content-ID", fmt.Sprintf("<%s>", att.ContentID))
		attHeader.Set("Content-Type", mime.TypeByExtension(filepath.Ext(att.Name)))
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, att.Name))
		pw, err := mpw.CreatePart(attHeader)
		if err != nil {
			return wc.n, fmt.Errorf("create attachment %s: %s", att.Name, err)
		}
		b64w := base64.NewEncoder(base64.StdEncoding, pw)
		if _, err := b64w.Write(att.Content); err != nil {
			return wc.n, fmt.Errorf("write attachment %s: %s", att.Name, err)
		}
		if err := b64w.Close(); err != nil {
			return wc.n, fmt.Errorf("close attachment %s: %s", att.Name, err)
		}
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
