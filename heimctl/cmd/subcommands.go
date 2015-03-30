package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"text/template"
	"time"

	"euphoria.io/scope"
)

var (
	globalTemplate *template.Template
	out            io.Writer
)

type subcommand interface {
	desc() string
	longdesc() string
	usage() string
	flags() *flag.FlagSet
	run(scope.Context, []string) error
}

var subcommands = map[string]subcommand{}

func register(name string, cmd subcommand) { subcommands[name] = cmd }

func Run(args []string) {
	out = tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	if len(args) == 0 {
		generalHelp()
		return
	}

	exe := filepath.Base(os.Args[0])
	cmd, ok := subcommands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "%s: invalid command: %s\n", exe, args[0])
		fmt.Fprintf(os.Stderr, "Run '%s help' for usage.\n", exe)
		os.Exit(2)
	}

	flags := cmd.flags()
	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", exe, args[0], err)
		os.Exit(2)
	}

	ctx := scope.New()
	if err := cmd.run(ctx, flags.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	timeout := time.After(10 * time.Second)
	completed := make(chan struct{})
	go func() {
		ctx.WaitGroup().Wait()
		close(completed)
	}()

	fmt.Println("waiting for graceful shutdown...")
	select {
	case <-timeout:
		fmt.Println("timed out")
		os.Exit(1)
	case <-completed:
		fmt.Println("ok")
		os.Exit(0)
	}
}

func generalHelp() {
	out := tabwriter.NewWriter(os.Stderr, 0, 8, 1, '\t', 0)
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(out, "USAGE:\n\t%s [global options] <command> [command options] [arguments...]\n\n", exe)
	fmt.Fprintf(out, "VERSION:\n\t%s\n\n", Version)

	fmt.Fprintf(out, "COMMANDS:\n")
	names := sort.StringSlice{}
	for name, _ := range subcommands {
		names = append(names, name)
	}
	names.Sort()
	for _, name := range names {
		cmd := subcommands[name]
		fmt.Fprintf(out, "\t%s\t%s\n", name, cmd.desc())
	}
	fmt.Fprintln(out)

	fmt.Fprintf(out, "GLOBAL OPTIONS:\n")
	flag.VisitAll(func(f *flag.Flag) {
		prefix := "-"
		if len(f.Name) > 1 {
			prefix = "--"
		}
		fmt.Fprintf(out, "\t%s%s=%s\t%s\n", prefix, f.Name, f.DefValue, f.Usage)
	})

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Run \"%s help <command>\" for more details about a command.\n", exe)
}
