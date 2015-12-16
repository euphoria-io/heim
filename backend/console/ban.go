package console

import (
	"fmt"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/scope"
)

func init() {
	register("ban", ban{})
	register("unban", unban{})
}

type ban struct{}

func (ban) usage() string {
	return ("ban [-room <room>] [-duration <duration>] -agent <agent-id>\n" +
		"ban [-room <room>] [-duration <duration>] -ip <ip>")
}

func (ban) run(ctx scope.Context, c *console, args []string) error {
	roomName := c.String("room", "", "ban only in the given room")
	agent := c.String("agent", "", "agent ID to ban")
	ip := c.String("ip", "", "IP to ban")
	duration := c.Duration("duration", 0, "duration of ban (defaults to forever)")

	if err := c.Parse(args); err != nil {
		return err
	}

	var until time.Time
	var untilStr string
	switch *duration {
	case 0:
		until = time.Time{}
		untilStr = "forever"
	default:
		until = time.Now().Add(*duration)
		untilStr = fmt.Sprintf("until %s", until)
	}

	switch {
	case *agent != "":
		switch *roomName {
		case "":
			if err := c.backend.AgentTracker().BanAgent(ctx, *agent, until); err != nil {
				return err
			}
			c.Printf("agent %s banned globally %s\n", *agent, untilStr)
			return nil
		default:
			room, err := c.backend.GetRoom(ctx, *roomName)
			if err != nil {
				return err
			}
			if err := room.Ban(ctx, proto.Ban{ID: proto.UserID(*agent)}, until); err != nil {
				return err
			}
			c.Printf("agent %s banned in room %s %s\n", *agent, *roomName, untilStr)
			return nil
		}
	case *ip != "":
		switch *roomName {
		case "":
			if err := c.backend.BanIP(ctx, *ip, until); err != nil {
				return err
			}
			c.Printf("ip %s banned globally %s\n", *ip, untilStr)
			return nil
		default:
			room, err := c.backend.GetRoom(ctx, *roomName)
			if err != nil {
				return err
			}
			if err := room.Ban(ctx, proto.Ban{IP: *ip}, until); err != nil {
				return err
			}
			c.Printf("ip %s banned in room %s %s\n", *ip, *roomName, untilStr)
			return nil
		}
	default:
		return fmt.Errorf("-agent <agent-id> or -ip <ip> is required")
	}
}

type unban struct{}

func (unban) usage() string {
	return ("unban [-room <room>] -agent <agent-id>\n" +
		"unban [-room <room>] -ip <ip>")
}

func (unban) run(ctx scope.Context, c *console, args []string) error {
	roomName := c.String("room", "", "unban only in the given room")
	agent := c.String("agent", "", "agent ID to unban")
	ip := c.String("ip", "", "IP to unban")

	if err := c.Parse(args); err != nil {
		return err
	}

	switch {
	case *agent != "":
		switch *roomName {
		case "":
			if err := c.backend.AgentTracker().UnbanAgent(ctx, *agent); err != nil {
				return err
			}
			c.Printf("global ban of agent %s lifted\n", *agent)
			return nil
		default:
			room, err := c.backend.GetRoom(ctx, *roomName)
			if err != nil {
				return err
			}
			if err := room.Unban(ctx, proto.Ban{ID: proto.UserID(*agent)}); err != nil {
				return err
			}
			c.Printf("ban of agent %s in room %s lifted\n", *agent, *roomName)
			return nil
		}
	case *ip != "":
		switch *roomName {
		case "":
			if err := c.backend.UnbanIP(ctx, *ip); err != nil {
				return err
			}
			c.Printf("global ban of ip %s lifted\n", *ip)
			return nil
		default:
			room, err := c.backend.GetRoom(ctx, *roomName)
			if err != nil {
				return err
			}
			if err := room.Unban(ctx, proto.Ban{IP: *ip}); err != nil {
				return err
			}
			c.Printf("ban of ip %s in room %s lifted\n", *ip, *roomName)
			return nil
		}
	default:
		return fmt.Errorf("-agent <agent-id> or -ip <ip> is required")
	}
}
