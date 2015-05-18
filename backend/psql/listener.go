package psql

import (
	"fmt"
	"strings"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/scope"
)

type Listener struct {
	proto.Session
	*proto.Client
}

type ListenerMap map[string]Listener

func (lm ListenerMap) Broadcast(ctx scope.Context, event *proto.Packet, exclude ...string) error {
	payload, err := event.Payload()
	if err != nil {
		return err
	}

	excludeSet := map[string]struct{}{}
	for _, exc := range exclude {
		excludeSet[exc] = struct{}{}
	}

	// Inspect packet to see if it's a bounce event. If so, we'll deliver it
	// only to the bounced parties.
	bounceAgentID := ""
	bounceIP := ""
	if event.Type == proto.BounceEventType {
		if bounceEvent, ok := payload.(*proto.BounceEvent); ok {
			bounceAgentID = bounceEvent.AgentID
			bounceIP = bounceEvent.IP
		} else {
			backend.Logger(ctx).Printf("wtf? expected *proto.BounceEvent, got %T", payload)
		}
	}

	// Inspect packet to see if it's a join event. If so, we'll look for aliased
	// sessions to kick into fast-keepalive mode.
	fastKeepaliveAgentID := ""
	if event.Type == proto.JoinEventType {
		if presence, ok := payload.(*proto.PresenceEvent); ok {
			if idx := strings.IndexRune(presence.ID, '-'); idx >= 0 {
				fastKeepaliveAgentID = presence.ID[:idx]
			}
		}
	}

	for sessionID, listener := range lm {
		if _, ok := excludeSet[sessionID]; !ok {
			if bounceAgentID != "" {
				if strings.HasPrefix(sessionID, bounceAgentID) {
					backend.Logger(ctx).Printf("sending bounce to %s: %#v", listener.ID(), payload)
					if err := listener.Send(ctx, event.Type, payload); err != nil {
						backend.Logger(ctx).Printf("error sending bounce event to %s: %s",
							listener.ID(), err)
					}
					listener.Close()
				}
				continue
			}
			if bounceIP != "" {
				if listener.Client.IP == bounceIP {
					backend.Logger(ctx).Printf("sending bounce to %s: %#v", listener.ID(), payload)
					if err := listener.Send(ctx, event.Type, payload); err != nil {
						backend.Logger(ctx).Printf("error sending bounce event to %s: %s",
							listener.ID(), err)
					}
					listener.Close()
				}
				continue
			}
			if fastKeepaliveAgentID != "" && strings.HasPrefix(sessionID, fastKeepaliveAgentID) {
				if err := listener.CheckAbandoned(); err != nil {
					fmt.Errorf("fast keepalive to %s: %s", listener.ID(), err)
				}
			}
			if err := listener.Send(ctx, event.Type, payload); err != nil {
				// TODO: accumulate errors
				return fmt.Errorf("send message to %s: %s", listener.ID(), err)
			}
		}
	}

	return nil
}
