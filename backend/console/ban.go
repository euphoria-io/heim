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

	ban := proto.Ban{}

	switch {
	case *agent != "":
		ban.ID = proto.UserID(*agent)
	case *ip != "":
		ban.IP = *ip
	default:
		return fmt.Errorf("-agent <agent-id> or -ip <ip> is required")
	}

	if *roomName == "" {
		if err := c.backend.Ban(ctx, ban, until); err != nil {
			return err
		}
		c.Printf("banned globally for %s: %#v\n", untilStr, ban)
	} else {
		room, err := c.backend.GetRoom(ctx, *roomName)
		if err != nil {
			return err
		}
		if err := room.Ban(ctx, ban, until); err != nil {
			return err
		}
		c.Printf("banned in room %s for %s: %#v\n", *roomName, untilStr, ban)
	}

	return nil
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

	ban := proto.Ban{}

	switch {
	case *agent != "":
		ban.ID = proto.UserID(*agent)
	case *ip != "":
		ban.IP = *ip
	default:
		return fmt.Errorf("-agent <agent-id> or -ip <ip> is required")
	}

	if *roomName == "" {
		if err := c.backend.Unban(ctx, ban); err != nil {
			return err
		}
		c.Printf("global unban: %#v\n", ban)
	} else {
		room, err := c.backend.GetRoom(ctx, *roomName)
		if err != nil {
			return err
		}
		if err := room.Unban(ctx, ban); err != nil {
			return err
		}
		c.Printf("unban in room %s: %#v\n", *roomName, ban)
	}

	return nil
}
