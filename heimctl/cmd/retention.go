package cmd

import (
	"flag"
	"fmt"
	"time"

	"euphoria.io/heim/heimctl/retention"
	"euphoria.io/scope"
)

func init() {
	register("log-retention", &retentionCmd{})
}

type retentionCmd struct {
	addr     string
	interval time.Duration
}

func (retentionCmd) desc() string {
	return "start up the service to delete expired messages."
}

func (retentionCmd) usage() string {
	return "log-retention [--http=<interface:port>] [--interval=DURATION]"
}

func (retentionCmd) longdesc() string {
	return `
	Start the service that deletes expired messages. This is a service that 
	polls the postgres db for messages sent longer than the per-room retention
	duration and deletes them.
`[1:]
}

func (cmd *retentionCmd) flags() *flag.FlagSet {
	flags := flag.NewFlagSet("log-retention", flag.ExitOnError)
	flags.StringVar(&cmd.addr, "http", ":8080", "address to serve metrics on")
	flags.DurationVar(&cmd.interval, "interval", 60*time.Second, "sleep interval between presence table scans")
	return flags
}

func (cmd *retentionCmd) run(ctx scope.Context, args []string) error {
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

	// start metrics server
	ctx.WaitGroup().Add(1)
	go retention.Serve(ctx, cmd.addr)

	// start metrics scanner
	ctx.WaitGroup().Add(1)
	retention.ExpiredScanLoop(ctx, c, b, cmd.interval)

	return nil
}
