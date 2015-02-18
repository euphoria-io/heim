package console

import (
	"fmt"
)

func init() { register("lock-room", lockRoom{}) }

type lockRoom struct{}

func (lockRoom) usage() string { return "usage: lock-room [OPTIONS] ROOM" }

func (lockRoom) run(c *console, args []string) error {
	// TODO: --retroactive: encrypt all existing cleartext messages
	// TODO: --upgrade: re-encrypt all messages encrypted with previous key
	force := c.Bool("force", false, "relock room if already locked, invalidating all previous grants")

	if err := c.Parse(args); err != nil {
		return err
	}

	if len(c.Args()) < 1 {
		return usageError("room name must be given")
	}

	roomName := c.Arg(0)

	room, err := c.backend.GetRoom(roomName)
	if err != nil {
		return err
	}

	// Check for existing key.
	mkey, err := room.MasterKey(c.ctx)
	if err != nil {
		return err
	}
	if mkey != nil && !*force {
		return fmt.Errorf("room already locked; use --force to relock and invalidate all previous grants")
	}

	_, err = room.GenerateMasterKey(c.ctx, c.kms)
	if err != nil {
		return err
	}

	return nil
}
