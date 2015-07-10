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

type TestEmailer struct {
	sync.Mutex
	counter    int
	messages   map[MessageID]TestMessage
	deliveries map[string][]MessageID
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
	return msg.ID, nil
}

type TestMessage struct {
	ID MessageID
}
