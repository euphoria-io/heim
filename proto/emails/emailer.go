package emails

import (
	"bytes"
	"time"

	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/heim/templates"
	"euphoria.io/scope"
)

type EmailRef struct {
	ID        string
	AccountID snowflake.Snowflake
	JobID     snowflake.Snowflake
	EmailType string
	SendTo    string
	SendFrom  string
	Message   []byte
	Created   time.Time
	Delivered time.Time
	Failed    time.Time

	data interface{}
}

type Emailer interface {
	Send(ctx scope.Context, to, templateName string, data interface{}) (*EmailRef, error)
}

func NewEmail(templater *templates.Templater, msgID, to, templateName string, data interface{}) (*EmailRef, error) {
	now := time.Now()
	ref := &EmailRef{
		ID:        msgID,
		EmailType: templateName,
		SendTo:    to,
		Created:   now,
		data:      data,
	}
	if templater == nil {
		return ref, nil
	}

	if cd, ok := data.(commonData); ok {
		cd.setAccountEmailAddress(to)
	}

	email, err := templates.EvaluateEmail(templater, templateName, data)
	if err != nil {
		return nil, err
	}

	email.Header.Set("To", to)
	email.Header.Set("Message-ID", msgID)
	ref.SendFrom = email.Header.Get("From")

	delivery := &bytes.Buffer{}
	if _, err := email.WriteTo(delivery); err != nil {
		return nil, err
	}

	ref.Message = delivery.Bytes()
	return ref, nil
}

type commonData interface {
	setAccountEmailAddress(string)
}

type CommonData struct {
	templates.StaticFiles
	AccountEmailAddress string
	LocalDomain         string
}

func (cd *CommonData) setAccountEmailAddress(addr string) { cd.AccountEmailAddress = addr }
