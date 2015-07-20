package emails

import (
	"fmt"
	"sync"

	"euphoria.io/scope"
)

type MessageID string

type Emailer interface {
	Send(ctx scope.Context, to string, templateName Template, data interface{}) (MessageID, error)
}

type MockEmailer interface {
	Emailer

	Inbox(addr string) <-chan TestMessage
}

type TestMessage struct {
	Template
	ID   MessageID
	Data interface{}
}

type TestEmailer struct {
	sync.Mutex
	counter  int
	channels map[string]chan TestMessage
}

func (e *TestEmailer) Send(
	ctx scope.Context, to string, templateName Template, data interface{}) (MessageID, error) {

	e.Lock()
	defer e.Unlock()

	e.counter++
	msg := TestMessage{
		ID:       MessageID(fmt.Sprintf("%08x", e.counter)),
		Template: templateName,
		Data:     data,
	}

	if ch, ok := e.channels[to]; ok {
		ch <- msg
	} else {
		fmt.Printf("sending %s to %s: %#v\n", templateName, to, data)
	}

	return msg.ID, nil
}

func (e *TestEmailer) Inbox(addr string) <-chan TestMessage {
	e.Lock()
	defer e.Unlock()

	if e.channels == nil {
		e.channels = map[string]chan TestMessage{}
	}
	if ch, ok := e.channels[addr]; ok {
		return ch
	}
	e.channels[addr] = make(chan TestMessage, 10)
	return e.channels[addr]
}
