package console

import (
	"fmt"

	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

func init() {
	register("set-room-passcode", setRoomPasscode{})
	register("lock-room", lockRoom{})
}

type setRoomPasscode struct{}

func (setRoomPasscode) usage() string { return "usage: set-room-passcode ROOM" }

func (setRoomPasscode) run(ctx scope.Context, c *console, args []string) error {
	if len(args) != 1 {
		return usageError("invalid command")
	}

	passcode, err := c.ReadPassword("passcode: ")
	if err != nil {
		return err
	}

	room, err := c.backend.GetRoom(ctx, args[0])
	if err != nil {
		return fmt.Errorf("room lookup error: %s", err)
	}

	rkey, err := room.MasterKey(ctx)
	if err != nil {
		return fmt.Errorf("room key error: %s", err)
	}
	if rkey == nil {
		return fmt.Errorf("room doesn't exist or isn't locked")
	}

	mkey := rkey.ManagedKey()
	if err := c.kms.DecryptKey(&mkey); err != nil {
		return fmt.Errorf("room key decryption error: %s", err)
	}

	ckey := security.KeyFromPasscode([]byte(passcode), rkey.Nonce(), security.AES128)

	capability, err := security.GrantSharedSecretCapability(ckey, rkey.Nonce(), nil, mkey.Plaintext)
	if err != nil {
		return fmt.Errorf("capability grant error: %s", err)
	}

	if err := room.SaveCapability(ctx, capability); err != nil {
		return err
	}

	c.Printf("Passcode added to %s: %s\n", args[0], capability.CapabilityID())
	return nil
}

type lockRoom struct{}

func (lockRoom) usage() string { return "usage: lock-room [OPTIONS] ROOM" }

func (lockRoom) run(ctx scope.Context, c *console, args []string) error {
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

	room, err := c.backend.GetRoom(ctx, roomName)
	if err != nil {
		return err
	}

	// Check for existing key.
	mkey, err := room.MasterKey(ctx)
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

	_, err = room.GenerateMasterKey(ctx, c.kms)
	if err != nil {
		return err
	}

	c.Printf("Room %s locked with new key.\n", roomName)
	return nil
}
