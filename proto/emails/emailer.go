package emails

import (
	"fmt"
	"sync"

	"euphoria.io/scope"
)

type MessageID string
type Template string

type Emailer interface {
	Send(
		ctx scope.Context, to string, templateName Template, data map[string]interface{}) (
		MessageID, error)
}

type MockEmailer interface {
	Emailer

	Inbox(addr string) <-chan Template
}

type TestEmailer struct {
	sync.Mutex
	counter    int
	messages   map[MessageID]TestMessage
	deliveries map[string][]MessageID
	channels   map[string]chan Template
}

func (e *TestEmailer) Send(
	ctx scope.Context, to string, templateName Template, data map[string]interface{}) (MessageID, error) {

	e.Lock()
	defer e.Unlock()

	e.counter++
	msg := TestMessage{
		ID: MessageID(fmt.Sprintf("%08x", e.counter)),
	}

	if e.messages == nil {
		e.messages = map[MessageID]TestMessage{msg.ID: msg}
	} else {
		e.messages[msg.ID] = msg
	}

	if e.deliveries == nil {
		e.deliveries = map[string][]MessageID{}
	}
	e.deliveries[to] = append(e.deliveries[to], msg.ID)

	if ch, ok := e.channels[to]; ok {
		ch <- templateName
	}

	return msg.ID, nil
}

func (e *TestEmailer) Inbox(addr string) <-chan Template {
	e.Lock()
	defer e.Unlock()

	if e.channels == nil {
		e.channels = map[string]chan Template{}
	}
	if ch, ok := e.channels[addr]; ok {
		return ch
	}
	e.channels[addr] = make(chan Template, 10)
	return e.channels[addr]
}

type TestMessage struct {
	ID MessageID
}
