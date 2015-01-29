package mock

import (
	"sync"

	"heim/proto"

	"golang.org/x/net/context"
)

type memLog struct {
	sync.Mutex
	msgs []*proto.Message
}

func newMemLog() *memLog { return &memLog{msgs: []*proto.Message{}} }

func (log *memLog) post(msg *proto.Message) {
	log.Lock()
	defer log.Unlock()

	log.msgs = append(log.msgs, msg)
}

func (log *memLog) Latest(ctx context.Context, n int, before proto.Snowflake) (
	[]proto.Message, error) {

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
		return []proto.Message{}, nil
	}

	messages := make([]proto.Message, len(slice))
	for i, msg := range slice {
		messages[i] = *msg
	}
	return messages, nil
}
