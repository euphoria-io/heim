package backend

import (
	"sync"

	"golang.org/x/net/context"
)

type Log interface {
	Latest(context.Context, int, Snowflake) ([]Message, error)
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

func (log *memLog) Latest(ctx context.Context, n int, before Snowflake) ([]Message, error) {
	log.Lock()
	defer log.Unlock()

	end := len(log.msgs)
	if !before.IsZero() {
		for end > 0 && !log.msgs[end-1].ID.Before(before) {
			end--
		}
	}

	start := end - n
	if start < 0 {
		start = 0
	}

	slice := log.msgs[start:end]
	if len(slice) == 0 {
		return []Message{}, nil
	}

	messages := make([]Message, len(slice))
	for i, msg := range slice {
		messages[i] = *msg
	}
	return messages, nil
}
