package cmd

import (
	"flag"
	"fmt"

	"euphoria.io/scope"
)

func init() {
	register("version", &versionCmd{})
}

type versionCmd struct {
}

func (versionCmd) desc() string {
	return "display heimctl version"
}

func (versionCmd) usage() string {
	return "version"
}

func (versionCmd) longdesc() string {
	return "Display the version stamped into the heimctl binary."
}

func (versionCmd) flags() *flag.FlagSet {
	return flag.NewFlagSet("version", flag.ExitOnError)
}

func (versionCmd) run(ctx scope.Context, args []string) error {
	fmt.Printf("heimctl version %s\n", Version)
	return nil
}
