package console

import (
	"fmt"
	"time"

	"euphoria.io/scope"
)

func init() {
	register("ban", ban{})
	register("unban", unban{})
}

type ban struct{}

func (ban) usage() string {
	return ("ban [-room <room>] [-until <duration>] -agent <agent-id>\n" +
		"ban [-room <room>] [-until <duration>] -ip <ip>")
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
			if err := c.backend.BanAgent(ctx, *agent, until); err != nil {
				return err
			}
			c.Printf("agent %s banned globally %s\n", *agent, untilStr)
			return nil
		default:
			room, err := c.backend.GetRoom(*roomName, false)
			if err != nil {
				return err
			}
			if err := room.BanAgent(ctx, *agent, until); err != nil {
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
			room, err := c.backend.GetRoom(*roomName, false)
			if err != nil {
				return err
			}
			if err := room.BanIP(ctx, *ip, until); err != nil {
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
			if err := c.backend.UnbanAgent(ctx, *agent); err != nil {
				return err
			}
			c.Printf("global ban of agent %s lifted\n", *agent)
			return nil
		default:
			room, err := c.backend.GetRoom(*roomName, false)
			if err != nil {
				return err
			}
			if err := room.UnbanAgent(ctx, *agent); err != nil {
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
			room, err := c.backend.GetRoom(*roomName, false)
			if err != nil {
				return err
			}
			if err := room.UnbanIP(ctx, *ip); err != nil {
				return err
			}
			c.Printf("ban of ip %s in room %s lifted\n", *ip, *roomName)
			return nil
		}
	default:
		return fmt.Errorf("-agent <agent-id> or -ip <ip> is required")
	}
}
