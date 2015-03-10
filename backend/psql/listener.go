package psql

import (
	"fmt"
	"strings"

	"euphoria.io/heim/proto"
	"euphoria.io/scope"
)

type ListenerMap map[string]proto.Session

func (lm ListenerMap) Broadcast(ctx scope.Context, event *proto.Packet, exclude ...string) error {
	payload, err := event.Payload()
	if err != nil {
		return err
	}

	excludeSet := map[string]struct{}{}
	for _, exc := range exclude {
		excludeSet[exc] = struct{}{}
	}

	// Inspect packet to see if it's a join event. If so, we'll look for aliased
	// sessions to kick into fast-keepalive mode.
	agentID := ""
	if event.Type == proto.JoinEventType {
		if presence, ok := payload.(*proto.PresenceEvent); ok {
			if idx := strings.IndexRune(presence.ID, '-'); idx >= 0 {
				agentID = presence.ID[:idx]
			}
		}
	}

	for sessionID, session := range lm {
		if _, ok := excludeSet[sessionID]; !ok {
			if agentID != "" && strings.HasPrefix(sessionID, agentID) {
				if err := session.CheckAbandoned(); err != nil {
					fmt.Errorf("fast keepalive to %s: %s", session.ID(), err)
				}
			}
			if err := session.Send(ctx, event.Type, payload); err != nil {
				// TODO: accumulate errors
				return fmt.Errorf("send message to %s: %s", session.ID(), err)
			}
		}
	}

	return nil
}
