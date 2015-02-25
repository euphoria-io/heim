package console

import (
	"fmt"

	"heim/proto/security"
)

func init() {
	register("set-room-passcode", setRoomPasscode{})
	register("lock-room", lockRoom{})
}

type setRoomPasscode struct{}

func (setRoomPasscode) usage() string { return "usage: set-room-passcode ROOM" }

func (setRoomPasscode) run(c *console, args []string) error {
	if len(args) != 1 {
		return usageError("invalid command")
	}

	passcode, err := c.ReadPassword("passcode: ")
	if err != nil {
		return err
	}

	room, err := c.backend.GetRoom(args[0])
	if err != nil {
		return err
	}

	mkey, err := room.MasterKey(c.ctx)
	if err != nil {
		return err
	}
	if mkey == nil {
		return fmt.Errorf("room doesn't exist or isn't locked")
	}

	roomKey := mkey.ManagedKey()
	capability, err := security.GrantCapabilityOnSubjectWithPasscode(
		c.ctx, c.kms, mkey.Nonce(), &roomKey, []byte(passcode))
	if err != nil {
		return err
	}

	if err := room.SaveCapability(c.ctx, capability); err != nil {
		return err
	}

	c.Printf("Passcode added to %s: %s\n", args[0], capability.CapabilityID())
	return nil
}

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
	if mkey != nil {
		if !*force {
			return fmt.Errorf(
				"room already locked; use --force to relock and invalidate all previous grants")
		}
		c.Printf("Overwriting existing key.\n")
	}

	_, err = room.GenerateMasterKey(c.ctx, c.kms)
	if err != nil {
		return err
	}

	c.Printf("Room %s locked with new key.\n", roomName)
	return nil
}
