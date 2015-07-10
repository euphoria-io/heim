package cmd

import (
	"flag"
	"fmt"
	"time"

	"github.com/lib/pq"

	"euphoria.io/heim/heimctl/activity"
	"euphoria.io/scope"
)

func init() {
	register("activity-exporter", &activityCmd{})
}

type activityCmd struct {
	addr string
}

func (activityCmd) desc() string {
	return "start up the monitoring service for the activity table"
}

func (activityCmd) usage() string {
	return "activity-exporter [--http=<interface:port>]"
}

func (activityCmd) longdesc() string {
	return `
	Start the activity-exporter server. This is a service that listens to
	the postgres firehose and collects per-room metrics.
`[1:]
}

func (cmd *activityCmd) flags() *flag.FlagSet {
	flags := flag.NewFlagSet("activity-exporter", flag.ExitOnError)
	flags.StringVar(&cmd.addr, "http", ":8080", "address to serve metrics on")
	return flags
}

func (cmd *activityCmd) run(ctx scope.Context, args []string) error {
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}

	listener := pq.NewListener(cfg.DB.DSN, 200*time.Millisecond, 5*time.Second, nil)
	if err := listener.Listen("broadcast"); err != nil {
		return fmt.Errorf("pq listen error: %s", err)
	}

	defer func() {
		ctx.Cancel()
		ctx.WaitGroup().Wait()
	}()

	// Start metrics server.
	ctx.WaitGroup().Add(1)
	go activity.Serve(ctx, cmd.addr)

	// Start scanner.
	ctx.WaitGroup().Add(1)
	activity.ScanLoop(ctx, listener)

	return nil
}
