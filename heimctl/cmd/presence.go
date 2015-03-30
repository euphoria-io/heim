package cmd

import (
	"flag"
	"fmt"
	"time"

	"euphoria.io/heim/heimctl/presence"
	"euphoria.io/scope"
)

func init() {
	register("presence-exporter", &presenceCmd{})
}

type presenceCmd struct {
	addr     string
	interval time.Duration
}

func (presenceCmd) desc() string {
	return "start up the monitoring/cleanup service for the presence table"
}

func (presenceCmd) usage() string {
	return "presence-exporter [--http=IFACE:PORT] [--interval=DURATION]"
}

func (presenceCmd) longdesc() string {
	return `
	Start the presence-exporter server. This is a service that continually
	monitors heim's presence table. This table provides a snapshot of live
	(and recently terminated) sessions to chat rooms. The exporter polls
	this table, collecting metrics about usage and cleaning up dead entries.
`[1:]
}

func (cmd *presenceCmd) flags() *flag.FlagSet {
	flags := flag.NewFlagSet("presence-exporter", flag.ExitOnError)
	flags.StringVar(&cmd.addr, "http", ":8080", "address to serve metrics on")
	flags.DurationVar(
		&cmd.interval, "interval", 60*time.Second, "sleep interval between presence table scans")
	return flags
}

func (cmd *presenceCmd) run(ctx scope.Context, args []string) error {
	c, err := getCluster(ctx)
	if err != nil {
		return err
	}

	b, err := getBackend(ctx, c)
	if err != nil {
		return fmt.Errorf("backend error: %s", err)
	}
	defer b.Close()

	defer func() {
		ctx.Cancel()
		ctx.WaitGroup().Wait()
	}()

	// Start metrics server.
	ctx.WaitGroup().Add(1)
	go presence.Serve(ctx, cmd.addr)

	// Start scanner.
	presence.ScanLoop(ctx, c, b, cmd.interval)

	return nil
}
