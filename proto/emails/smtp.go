package emails

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"

	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/heim/templates"
	"euphoria.io/scope"
)

func NewSMTPEmailer(templatesPath, localAddr, serverAddr, sslHost string, auth smtp.Auth) (*TemplateEmailer, error) {
	d := &SMTPDeliverer{
		addr:      serverAddr,
		localName: localAddr,
		auth:      auth,
	}

	emailer := &TemplateEmailer{
		Templater: &templates.Templater{},
		Deliverer: d,
	}

	if sslHost != "" {
		d.tlsConfig = &tls.Config{ServerName: sslHost}
	}

	if errs := emailer.Templater.Load(templatesPath); errs != nil {
		return nil, errs[0]
	}

	return emailer, nil
}

type SMTPDeliverer struct {
	addr      string
	localName string
	auth      smtp.Auth
	tlsConfig *tls.Config
}

func (s *SMTPDeliverer) String() string { return fmt.Sprintf("smtp[%s]", s.addr) }

func (s *SMTPDeliverer) MessageID() (string, error) {
	sf, err := snowflake.New()
	if err != nil {
		return "", fmt.Errorf("%s: snowflake error: %s", s, err)
	}
	return fmt.Sprintf("<%s@%s>", sf, s.localName), nil
}

func (s *SMTPDeliverer) Deliver(ctx scope.Context, from, to string, email io.WriterTo) error {
	// Connect and authenticate to SMTP server.
	c, err := smtp.Dial(s.addr)
	if err != nil {
		return fmt.Errorf("%s: dial error: %s", s, err)
	}
	defer c.Quit()

	if err := c.Hello(s.localName); err != nil {
		return fmt.Errorf("%s: ehlo error: %s", s, err)
	}

	if s.tlsConfig != nil {
		if err := c.StartTLS(s.tlsConfig); err != nil {
			return fmt.Errorf("%s: starttls error: %s", s, err)
		}
	}

	if s.auth != nil {
		if err := c.Auth(s.auth); err != nil {
			return fmt.Errorf("%s: auth error: %s", s, err)
		}
	}

	// Send email.
	if from == "" {
		from = "noreply@" + s.localName
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("%s: mail error: %s", s, err)
	}

	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("%s: rcpt error: %s", s, err)
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("%s: data error: %s", s, err)
	}

	if _, err := email.WriteTo(wc); err != nil {
		return fmt.Errorf("%s: write error: %s", s, err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("%s: close error: %s", s, err)
	}

	return nil
}
