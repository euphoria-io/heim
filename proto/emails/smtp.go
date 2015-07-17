package emails

import (
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"path/filepath"

	"encoding/base64"

	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

func NewSMTPEmailer(
	templatesPath, localAddr, serverAddr, sslHost string, auth smtp.Auth) (*SMTPEmailer, error) {

	templater, errs := LoadTemplates(templatesPath)
	if errs != nil {
		return nil, errs[0]
	}

	emailer := &SMTPEmailer{
		Templater: templater,
		addr:      serverAddr,
		localName: localAddr,
		auth:      auth,
	}

	if sslHost != "" {
		emailer.tlsConfig = &tls.Config{ServerName: sslHost}
	}

	return emailer, nil
}

type SMTPEmailer struct {
	*Templater

	addr      string
	localName string
	auth      smtp.Auth
	tlsConfig *tls.Config
}

func (s *SMTPEmailer) String() string { return fmt.Sprintf("smtp[%s]", s.addr) }

func (s *SMTPEmailer) Send(
	ctx scope.Context, to string, templateName Template, data interface{}) (MessageID, error) {

	// Construct email and assign a MessageID.
	sf, err := snowflake.New()
	if err != nil {
		return "", fmt.Errorf("%s: snowflake error: %s", s, err)
	}
	msgID := fmt.Sprintf("<%s@%s>", sf, s.localName)

	if ec, ok := data.(emailCommon); ok {
		ec.setToAddress(to)
	}

	result, err := s.Evaluate(templateName, data)
	if err != nil {
		return "", fmt.Errorf("%s: render error: %s", s, err)
	}
	result.Header.Set("Message-ID", msgID)

	// Connect and authenticate to SMTP server.
	c, err := smtp.Dial(s.addr)
	if err != nil {
		return "", fmt.Errorf("%s: dial error: %s", s, err)
	}
	defer c.Quit()

	if err := c.Hello(s.localName); err != nil {
		return "", fmt.Errorf("%s: ehlo error: %s", s, err)
	}

	if s.tlsConfig != nil {
		if err := c.StartTLS(s.tlsConfig); err != nil {
			return "", fmt.Errorf("%s: starttls error: %s", s, err)
		}
	}

	if s.auth != nil {
		if err := c.Auth(s.auth); err != nil {
			return "", fmt.Errorf("%s: auth error: %s", s, err)
		}
	}

	// Send email.
	from := result.Header.Get("From")
	if from == "" {
		from = "noreply@" + s.localName
	}

	if err := c.Mail(from); err != nil {
		return "", fmt.Errorf("%s: mail error: %s", s, err)
	}

	if err := c.Rcpt(to); err != nil {
		return "", fmt.Errorf("%s: rcpt error: %s", s, err)
	}

	wc, err := c.Data()
	if err != nil {
		return "", fmt.Errorf("%s: data error: %s", s, err)
	}

	if err := s.write(wc, result); err != nil {
		return "", fmt.Errorf("%s: write error: %s", s, err)
	}

	return MessageID(msgID), nil
}

func (s *SMTPEmailer) write(w io.Writer, result *TemplateResult) error {
	//w = io.MultiWriter(w, os.Stdout)
	mpw := multipart.NewWriter(w)

	// Write top-level headers.
	headers := result.Header
	headers.Set("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, mpw.Boundary()))
	headers.Set("MIME-Version", "1.0")
	for k, vv := range headers {
		for _, v := range vv {
			fmt.Fprintf(w, "%s: %s\n", k, v)
		}
	}
	fmt.Fprintln(w)

	// Write text part.
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", `text/plain; charset="utf-8"; format="fixed"`)
	// TODO: Content-Transfer-Encoding
	pw, err := mpw.CreatePart(textHeader)
	if err != nil {
		return fmt.Errorf("create text part: %s", err)
	}
	if _, err := pw.Write(result.Text); err != nil {
		return fmt.Errorf("write text part: %s", err)
	}

	// Write html part.
	textHeader = textproto.MIMEHeader{}
	textHeader.Set("Content-Type", `text/html; charset="utf-8"`)
	// TODO: Content-Transfer-Encoding
	pw, err = mpw.CreatePart(textHeader)
	if err != nil {
		return fmt.Errorf("create html part: %s", err)
	}
	if _, err := pw.Write(result.HTML); err != nil {
		return fmt.Errorf("write html part: %s", err)
	}

	// Write attachments.
	for name, cid := range result.Attachments {
		content, ok := s.staticFiles[name]
		if !ok {
			return fmt.Errorf("create attachment %s: content not found", name)
		}
		textHeader = textproto.MIMEHeader{}
		textHeader.Set("Content-ID", fmt.Sprintf("<%s>", cid))
		textHeader.Set("Content-Type", mime.TypeByExtension(filepath.Ext(name)))
		textHeader.Set("Content-Transfer-Encoding", "base64")
		textHeader.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, name))
		pw, err = mpw.CreatePart(textHeader)
		if err != nil {
			return fmt.Errorf("create attachment %s: %s", name, err)
		}
		b64w := base64.NewEncoder(base64.StdEncoding, pw)
		if _, err := b64w.Write(content); err != nil {
			return fmt.Errorf("write attachment %s: %s", name, err)
		}
		if err := b64w.Close(); err != nil {
			return fmt.Errorf("close attachment %s: %s", name, err)
		}
	}

	// Finalize.
	if err := mpw.Close(); err != nil {
		return fmt.Errorf("multipart close: %s", err)
	}

	return nil
}
