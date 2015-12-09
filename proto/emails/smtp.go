package emails

import (
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
	"time"

	"euphoria.io/scope"
)

func NewSMTPDeliverer(localAddr, serverAddr, sslHost string, auth smtp.Auth) *SMTPDeliverer {
	d := &SMTPDeliverer{
		addr:      serverAddr,
		localName: localAddr,
		auth:      auth,
	}
	if sslHost != "" {
		d.tlsConfig = &tls.Config{ServerName: sslHost}
	}
	return d
}

type SMTPDeliverer struct {
	addr      string
	localName string
	auth      smtp.Auth
	tlsConfig *tls.Config
}

func (s *SMTPDeliverer) String() string    { return fmt.Sprintf("smtp[%s]", s.addr) }
func (s *SMTPDeliverer) LocalName() string { return s.localName }

func (s *SMTPDeliverer) Deliver(ctx scope.Context, ref *EmailRef) error {
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
	if ref.SendFrom == "" {
		ref.SendFrom = "noreply@" + s.localName
	}
	sendFrom, err := mail.ParseAddress(ref.SendFrom)
	if err != nil {
		return fmt.Errorf("%s: from address error: %s", s, err)
	}
	if err := c.Mail(sendFrom.Address); err != nil {
		return fmt.Errorf("%s: mail error: %s", s, err)
	}

	sendTo, err := mail.ParseAddress(ref.SendTo)
	if err != nil {
		return fmt.Errorf("%s: to address error: %s", s, err)
	}
	if err := c.Rcpt(sendTo.Address); err != nil {
		return fmt.Errorf("%s: rcpt error: %s", s, err)
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("%s: data error: %s", s, err)
	}

	if _, err := wc.Write(ref.Message); err != nil {
		return fmt.Errorf("%s: write error: %s", s, err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("%s: close error: %s", s, err)
	}

	ref.Delivered = time.Now()
	return nil
}
