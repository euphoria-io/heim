package emails

import (
	"bytes"
	"fmt"
	"sync"

	"euphoria.io/heim/templates"
	"euphoria.io/scope"
)

type Emailer interface {
	Send(ctx scope.Context, to, templateName string, data interface{}) (messageID string, err error)
}

type MockEmailer interface {
	Emailer

	Inbox(addr string) <-chan TestMessage
}

type TestMessage struct {
	TemplateName string
	MessageID    string
	Data         interface{}
	Delivery     []byte
}

type TestEmailer struct {
	sync.Mutex
	Templater *templates.Templater

	counter      int
	sendChannels map[string]chan TestMessage
}

func (e *TestEmailer) Send(ctx scope.Context, to, templateName string, data interface{}) (string, error) {
	e.Lock()
	defer e.Unlock()

	e.counter++
	msgID := fmt.Sprintf("%08x", e.counter)

	delivery := &bytes.Buffer{}
	if e.Templater != nil {
		email, err := templates.EvaluateEmail(e.Templater, templateName, data)
		if err != nil {
			return "", err
		}
		if _, err := email.WriteTo(delivery); err != nil {
			return "", err
		}
	}

	if ch, ok := e.sendChannels[to]; ok {
		ch <- TestMessage{
			TemplateName: templateName,
			MessageID:    msgID,
			Data:         data,
			Delivery:     delivery.Bytes(),
		}
	} else {
		fmt.Printf("sending %s to %s: %#v\n", templateName, to, data)
	}

	return msgID, nil
}

func (e *TestEmailer) Inbox(addr string) <-chan TestMessage {
	e.Lock()
	defer e.Unlock()

	if e.sendChannels == nil {
		e.sendChannels = map[string]chan TestMessage{}
	}
	if ch, ok := e.sendChannels[addr]; ok {
		return ch
	}
	e.sendChannels[addr] = make(chan TestMessage, 10)
	return e.sendChannels[addr]
}

type TemplateEmailer struct {
	Templater *templates.Templater
	Deliverer Deliverer
}

func (e *TemplateEmailer) Send(ctx scope.Context, to, templateName string, data interface{}) (string, error) {
	msgID, err := e.Deliverer.MessageID()
	if err != nil {
		return "", err
	}

	if cd, ok := data.(commonData); ok {
		cd.setAccountEmailAddress(to)
	}
	email, err := templates.EvaluateEmail(e.Templater, templateName, data)
	if err != nil {
		return "", err
	}

	email.Header.Set("Message-ID", msgID)
	from := email.Header.Get("From")
	if err := e.Deliverer.Deliver(ctx, from, to, email); err != nil {
		return "", err
	}
	return msgID, nil
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
