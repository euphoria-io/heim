package cmd

import (
	"flag"
	"fmt"

	"euphoria.io/heim/heimctl/worker"
	"euphoria.io/scope"
)

func init() {
	register("worker", &workerCmd{})
}

type workerCmd struct {
	addr   string
	worker string
}

func (workerCmd) desc() string {
	return "run a worker for processing a job queue"
}

func (workerCmd) usage() string {
	return "worker [--http=<interface:port>] [--worker=ID] QUEUE"
}

func (workerCmd) longdesc() string {
	return `
	Run a worker for processing job items from QUEUE. The worker will idle
	until it can claim a job.
`[1:]
}

func (cmd *workerCmd) flags() *flag.FlagSet {
	flags := flag.NewFlagSet("worker", flag.ExitOnError)
	flags.StringVar(&cmd.addr, "http", ":8080", "address to serve metrics on")
	flags.StringVar(&cmd.worker, "worker", "worker", "prefix for handler IDs")
	return flags
}

func (cmd *workerCmd) run(ctx scope.Context, args []string) error {
	if len(args) < 1 {
		fmt.Printf("Usage: %s\r\n", cmd.usage())
		// TODO: list queues
		return nil
	}

	fmt.Printf("getting config\n")
	cfg, err := getConfig(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("getting heim\n")
	heim, err := cfg.Heim(ctx)
	if err != nil {
		fmt.Printf("heim error: %s\n", err)
		return err
	}

	defer func() {
		ctx.Cancel()
		ctx.WaitGroup().Wait()
	}()

	// Start metrics server.
	fmt.Printf("starting server\n")
	ctx.WaitGroup().Add(1)
	go worker.Serve(ctx, cmd.addr)

	// Start scanner.
	return worker.Loop(ctx, heim, cmd.worker, args[0])
}
