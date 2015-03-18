package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"euphoria.io/scope"
)

func init() { register("help", &helpCmd{}) }

type helpCmd struct{}

func (helpCmd) flags() *flag.FlagSet { return flag.NewFlagSet("help", flag.ExitOnError) }

func (helpCmd) desc() string  { return "list all commands, or show detailed help for a specific command" }
func (helpCmd) usage() string { return "help [command]" }

func (helpCmd) longdesc() string {
	return `
	Run help with no arguments to see a complete list of commands, or pass
	in a command name to get detailed help on that command.
`[1:]
}

func (helpCmd) run(ctx scope.Context, args []string) error {
	if len(args) == 0 {
		generalHelp()
		return nil
	}

	cmd, ok := subcommands[args[0]]
	if !ok {
		return fmt.Errorf("invalid command: %s", args[0])
	}

	fmt.Fprintf(out, "NAME:\n\t%s - %s\n\n", args[0], cmd.desc())

	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(out, "USAGE:\n\t%s %s\n\n", exe, cmd.usage())

	fmt.Fprintf(out, "DESCRIPTION:\n%s\n\n", cmd.longdesc())

	fmt.Fprintln(out, "OPTIONS:")
	cmd.flags().VisitAll(func(f *flag.Flag) {
		prefix := "-"
		if len(f.Name) > 1 {
			prefix = "--"
		}
		fmt.Fprintf(out, "\t%s%s=%s\t%s\n", prefix, f.Name, f.DefValue, f.Usage)
	})
	fmt.Fprintln(out)

	return nil
}
