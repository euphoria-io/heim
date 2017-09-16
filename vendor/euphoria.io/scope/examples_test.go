package scope_test

import (
	"fmt"
	"time"

	"euphoria.io/scope"
)

func ExampleBreakpointer() {
	root := scope.New()

	// A function that returns an error, which we want to simulate.
	output := func(arg string) error {
		_, err := fmt.Println(arg)
		return err
	}

	// A function that we want to test the error handling of.
	verifyOutput := func(ctx scope.Context, arg string) error {
		if err := ctx.Check("output()", arg); err != nil {
			return err
		}
		return output(arg)
	}

	// Set a breakpoint on a particular invocation of output.
	ctrl := root.Breakpoint("output()", "fail")

	// Other invocations should proceed as normal.
	err := verifyOutput(root, "normal behavior")
	fmt.Println("verifyOutput returned", err)

	// Our breakpoint should allow us to inject an error. To control it
	// we must spin off a goroutine.
	go func() {
		<-ctrl // synchronize at beginning of verifyOutput
		ctrl <- fmt.Errorf("test error")
	}()

	err = verifyOutput(root, "fail")
	fmt.Println("verifyOutput returned", err)

	// We can also inject an error by terminating the context.
	go func() {
		<-ctrl
		root.Cancel()
	}()

	err = verifyOutput(root, "fail")
	fmt.Println("verifyOutput returned", err)

	// Output:
	// normal behavior
	// verifyOutput returned <nil>
	// verifyOutput returned test error
	// verifyOutput returned context cancelled
}

func ExampleContext_cancellation() {
	ctx := scope.New()

	go func() {
		time.Sleep(50 * time.Millisecond)
		ctx.Cancel()
	}()

loop:
	for {
		t := time.After(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			break loop
		case <-t:
			fmt.Println("tick")
		}
	}
	fmt.Println("finished with", ctx.Err())
	// Output:
	// tick
	// tick
	// tick
	// tick
	// finished with context cancelled
}
