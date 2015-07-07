package cmd

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/psql"
	"euphoria.io/scope"
)

func init() {
	register("upgrade", &upgradeCmd{})
}

type upgradeCmd struct {
	yes bool
}

func (upgradeCmd) desc() string  { return "upgrade rooms to the new account system" }
func (upgradeCmd) usage() string { return "upgrade [-yes]" }

func (upgradeCmd) longdesc() string {
	return `
	Upgrade all rooms that were created before the new accounts system was
	in place. If run without -yes, only a list of rooms needed upgraded will
	be generated; no action is taken. If run with -yes, these rooms will be
	upgraded by generating new keys for them.
`[1:]
}

func (cmd *upgradeCmd) flags() *flag.FlagSet {
	flags := flag.NewFlagSet("upgrade", flag.ExitOnError)
	flags.BoolVar(&cmd.yes, "yes", false, "without this flag, only a dry run will occur")
	return flags
}

func (cmd *upgradeCmd) run(ctx scope.Context, args []string) error {
	c, err := getCluster(ctx)
	if err != nil {
		return err
	}

	kms, err := backend.Config.KMS.Get()
	if err != nil {
		return fmt.Errorf("kms error: %s", err)
	}

	b, err := getBackend(ctx, c)
	if err != nil {
		return fmt.Errorf("backend error: %s", err)
	}
	defer b.Close()

	// Scan rooms and generate an upgrade list.
	rows, err := b.DbMap.Select(psql.Room{}, "SELECT * FROM room")
	if err != nil {
		return err
	}

	upgradeNeeded := []string{}
	for _, row := range rows {
		room := row.(*psql.Room)
		if len(room.Nonce) == 0 {
			upgradeNeeded = append(upgradeNeeded, room.Name)
		}
	}

	if len(upgradeNeeded) == 0 {
		fmt.Printf("All rooms are upgraded!\n")
		return nil
	}

	sort.Strings(upgradeNeeded)
	fmt.Printf("Rooms to be upgraded (%d):\n", len(upgradeNeeded))
	for i, name := range upgradeNeeded {
		fmt.Printf("[%d] %s\n", i+1, name)
	}
	if !cmd.yes {
		fmt.Printf("(run this command again with -yes to upgrade)\n")
		return nil
	}

	for i, name := range upgradeNeeded {
		fmt.Printf("[%d/%d] %s... ", i+1, len(upgradeNeeded), name)
		os.Stdout.Sync()

		room, err := b.GetRoom(ctx, name)
		if err != nil {
			fmt.Println()
			return fmt.Errorf("room get: %s: %s", name, err)
		}

		if err := room.UpgradeRoom(ctx, kms); err != nil {
			fmt.Println()
			return fmt.Errorf("room upgrade: %s: %s", name, err)
		}

		fmt.Println("Ok")
	}

	fmt.Printf("All rooms are upgraded!\n")
	return nil
}
