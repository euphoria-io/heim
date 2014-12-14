package backend

import (
	"sync"

	"golang.org/x/net/context"
)

type Log interface {
	Latest(context.Context, int) ([]Message, error)
}

type memLog struct {
	sync.Mutex
	msgs []*Message
}

func newMemLog() *memLog { return &memLog{msgs: []*Message{}} }

func (log *memLog) post(msg *Message) {
	log.Lock()
	defer log.Unlock()

	log.msgs = append(log.msgs, msg)
}

func (log *memLog) Latest(ctx context.Context, n int) ([]Message, error) {
	log.Lock()
	defer log.Unlock()

	start := len(log.msgs) - n
	if start < 0 {
		start = 0
	}

	slice := log.msgs[start:]
	if len(slice) == 0 {
		return nil, nil
	}

	messages := make([]Message, len(slice))
	for i, msg := range slice {
		messages[i] = *msg
	}
	return messages, nil
}
