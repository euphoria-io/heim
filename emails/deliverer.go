package emails

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"euphoria.io/scope"
)

type Deliverer interface {
	Deliver(ctx scope.Context, from, to string, email io.WriterTo) error
	MessageID() (string, error)
}

type MockDeliverer interface {
	Deliverer

	Inbox(addr string) <-chan []byte
}

type TestDeliverer struct {
	sync.Mutex
	counter  int
	channels map[string]chan []byte
}

func (td *TestDeliverer) MessageID() (string, error) {
	td.Lock()
	defer td.Unlock()

	td.counter++
	return fmt.Sprintf("<%08x@test>", td.counter), nil
}

func (td *TestDeliverer) Deliver(ctx scope.Context, from, to string, email io.WriterTo) error {
	td.Lock()
	defer td.Unlock()

	buf := &bytes.Buffer{}
	if _, err := email.WriteTo(buf); err != nil {
		return err
	}

	if ch, ok := td.channels[to]; ok {
		ch <- buf.Bytes()
	} else {
		fmt.Printf("delivered:\n%s\n", buf.String())
	}

	return nil
}

func (td *TestDeliverer) Inbox(addr string) <-chan []byte {
	td.Lock()
	defer td.Unlock()

	if td.channels == nil {
		td.channels = map[string]chan []byte{}
	}
	td.channels[addr] = make(chan []byte, 10)
	return td.channels[addr]
}
