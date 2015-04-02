package console

import (
	"fmt"
	"strings"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

func init() {
	register("delete-message", deleteMessage{})
	register("undelete-message", undeleteMessage{})
}

type deleteMessage struct{}

func (deleteMessage) usage() string { return "delete-message [-quiet] <room>:<message-id>" }

func (deleteMessage) run(ctx scope.Context, c *console, args []string) error {
	return setDeleted(ctx, c, args, true)
}

type undeleteMessage struct{}

func (undeleteMessage) usage() string { return "undelete-message [-quiet] <room>:<message-id>" }

func (undeleteMessage) run(ctx scope.Context, c *console, args []string) error {
	return setDeleted(ctx, c, args, false)
}

func parseDeleteMessageArg(arg string) (string, snowflake.Snowflake, error) {
	parts := strings.SplitN(arg, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("format should be <room>:<message-id>")
	}

	var msgID snowflake.Snowflake
	if err := msgID.FromString(parts[1]); err != nil {
		return "", 0, err
	}

	return parts[0], msgID, nil
}

func setDeleted(ctx scope.Context, c *console, args []string, deleted bool) error {
	quiet := c.Bool("quiet", false, "suppress edit-message-event broadcast")

	if err := c.Parse(args); err != nil {
		return err
	}

	if len(c.Args()) < 1 {
		return fmt.Errorf("one or more message ids required")
	}

	for _, arg := range c.Args() {
		roomName, msgID, err := parseDeleteMessageArg(arg)
		if err != nil {
			return err
		}

		room, err := c.backend.GetRoom(roomName, false)
		if err != nil {
			return fmt.Errorf("%s: %s", arg, err)
		}

		msg, err := room.GetMessage(ctx, msgID)
		if err != nil {
			return fmt.Errorf("%s: %s", arg, err)
		}

		var action string
		if deleted {
			action = "Deleting"
		} else {
			action = "Undeleting"
		}
		c.Printf("%s message %s in room %s...\n", action, msgID.String(), roomName)
		edit := proto.EditMessageCommand{
			ID:             msgID,
			PreviousEditID: msg.PreviousEditID,
			Delete:         deleted,
			Announce:       !*quiet,
		}
		if err := room.EditMessage(ctx, c, edit); err != nil {
			return fmt.Errorf("%s: %s", arg, err)
		}
		if deleted {
			c.Printf("Deleted!\n")
		} else {
			c.Printf("Undeleted!\n")
		}
	}
	return nil
}
