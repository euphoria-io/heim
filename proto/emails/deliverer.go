package emails

import (
	"fmt"
	"sync"
	"time"

	"euphoria.io/scope"
)

type Deliverer interface {
	Deliver(ctx scope.Context, ref *EmailRef) error
	LocalName() string
}

type TestMessage struct {
	EmailRef
	Data interface{}
}

type MockDeliverer interface {
	Deliverer

	Inbox(addr string) <-chan *TestMessage
}

type TestDeliverer struct {
	sync.Mutex
	counter  int
	channels map[string]chan *TestMessage
}

func (td *TestDeliverer) LocalName() string { return "test" }

func (td *TestDeliverer) Deliver(ctx scope.Context, ref *EmailRef) error {
	td.Lock()
	defer td.Unlock()

	ref.Delivered = time.Now()
	if ch, ok := td.channels[ref.SendTo]; ok {
		ch <- &TestMessage{
			EmailRef: *ref,
			Data:     ref.data,
		}
	} else {
		fmt.Printf("delivered:\n%s\n", string(ref.Message))
	}

	return nil
}

func (td *TestDeliverer) Inbox(addr string) <-chan *TestMessage {
	td.Lock()
	defer td.Unlock()

	if td.channels == nil {
		td.channels = map[string]chan *TestMessage{}
	}
	td.channels[addr] = make(chan *TestMessage, 10)
	return td.channels[addr]
}
