package psql

import (
	"fmt"
	"heim/proto"

	"golang.org/x/net/context"
)

type ListenerMap map[string]proto.Session

func (lm ListenerMap) Broadcast(ctx context.Context, event *proto.Packet, exclude ...string) error {
	payload, err := event.Payload()
	if err != nil {
		return err
	}

	excludeSet := map[string]struct{}{}
	for _, exc := range exclude {
		excludeSet[exc] = struct{}{}
	}

	for sessionID, session := range lm {
		if _, ok := excludeSet[sessionID]; !ok {
			if err := session.Send(ctx, event.Type, payload); err != nil {
				// TODO: accumulate errors
				return fmt.Errorf("send message to %s: %s", session.ID(), err)
			}
		}
	}

	return nil
}
