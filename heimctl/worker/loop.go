package worker

import (
	"fmt"

	"euphoria.io/heim/proto"
	"euphoria.io/scope"
)

func Loop(ctx scope.Context, heim *proto.Heim, workerName, queueName string) error {
	fmt.Printf("Loop\n")
	ctrl, err := NewController(ctx, heim, workerName, queueName)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return err
	}

	ctx.WaitGroup().Add(1)
	go ctrl.background(ctx)
	ctx.WaitGroup().Wait()
	return ctx.Err()
}
