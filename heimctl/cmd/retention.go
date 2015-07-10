package cmd

import (
	"flag"
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
	heim, b, err := getHeimWithPsqlBackend(ctx)
	if err != nil {
		return err
	}

	defer func() {
		ctx.Cancel()
		ctx.WaitGroup().Wait()
		heim.Backend.Close()
	}()

	// start metrics server
	ctx.WaitGroup().Add(1)
	go retention.Serve(ctx, cmd.addr)

	// start metrics scanner
	ctx.WaitGroup().Add(1)
	go retention.ExpiredScanLoop(ctx, heim.Cluster, b, cmd.interval)

	// start delete scanner
	ctx.WaitGroup().Add(1)
	retention.DeleteScanLoop(ctx, heim.Cluster, b, cmd.interval)

	return nil
}
