package scope

import (
	"errors"
	"sync"
	"time"
)

var (
	Cancelled = errors.New("context cancelled")
	Canceled  = Cancelled

	TimedOut = errors.New("context timed out")
)

// A Context is a handle on a node within a shared scope. This shared scope
// takes the form of a tree of such nodes, for sharing state across
// coordinating goroutines.
type Context interface {
	// Alive returns true if the context has not completed.
	Alive() bool

	// Done returns a receive-only channel that will be closed when this
	// context (or any of its ancestors) terminates.
	Done() <-chan struct{}

	// Err returns the error this context was terminated with.
	Err() error

	// Cancel terminates this context (and all its descendents) with the
	// Cancelled error.
	Cancel()

	// Terminate marks this context and all descendents as terminated.
	// This sets the error returned by Err(), closed channels returned by
	// Done(), and injects the given error into any pending breakpoint
	// checks.
	Terminate(error)

	// Fork creates and returns a new context as a child of this one.
	Fork() Context

	// ForkWithTimeout creates and returns a new context as a child of this
	// one. It also spins off a timer which will cancel the context after
	// the given duration (unless the context terminates first).
	ForkWithTimeout(time.Duration) Context

	// Get returns the value associated with the given key. If this context
	// has had no values set, then the lookup is made on the nearest ancestor
	// with data. If no value is found, an unboxed nil value is returned.
	Get(key interface{}) interface{}

	// GetOK returns the value associated with the given key, along with a
	// bool value indicating successful lookup. See Get for details.
	GetOK(key interface{}) (interface{}, bool)

	// Set associates the given key and value in this context's data.
	Set(key, val interface{})

	// WaitGroup returns a wait group pointer common to the entire context
	// tree.
	WaitGroup() *sync.WaitGroup

	// Breakpointer provides a harness for injecting errors and coordinating
	// goroutines when unit testing.
	Breakpointer
}

type builtinKey int

const (
	bpmapKey builtinKey = iota
)

type kvmap map[interface{}]interface{}

// New returns an empty Context with no ancestor. This serves as the root
// of a shared scope.
func New() Context {
	ctx := &ContextTree{
		wg:       &sync.WaitGroup{},
		done:     make(chan struct{}),
		data:     kvmap{},
		children: map[*ContextTree]struct{}{},
	}
	ctx.Set(bpmapKey, bpmap{})
	return ctx
}

// ContextTree is the default implementation of Context.
type ContextTree struct {
	wg       *sync.WaitGroup
	m        sync.RWMutex
	termed   bool
	done     chan struct{}
	err      error
	data     kvmap
	aliased  *ContextTree
	children map[*ContextTree]struct{}
}

// WaitGroup returns a wait group pointer common to the entire context
// tree.
func (ctx *ContextTree) WaitGroup() *sync.WaitGroup { return ctx.wg }

// Alive returns true if the context has not completed.
func (ctx *ContextTree) Alive() bool { return !ctx.termed }

// Done returns a receive-only channel that will be closed when this
// context (or any of its ancestors) is terminated.
func (ctx *ContextTree) Done() <-chan struct{} { return ctx.done }

// Err returns the error this context was terminated with.
func (ctx *ContextTree) Err() error { return ctx.err }

// Cancel terminates this context (and all its descendents) with the
// Cancelled error.
func (ctx *ContextTree) Cancel() { ctx.Terminate(Cancelled) }

// Terminate marks this context and all descendents as terminated.
// This sets the error returned by Err(), closed channels returned by
// Done(), and injects the given error into any pending breakpoint
// checks.
func (ctx *ContextTree) Terminate(err error) {
	ctx.m.Lock()
	ctx.terminate(err)
	ctx.m.Unlock()
}

func (ctx *ContextTree) terminate(err error) {
	if ctx.Alive() {
		ctx.termed = true
		ctx.err = err
		for child := range ctx.children {
			child.m.Lock()
			child.terminate(err)
			child.m.Unlock()
		}
		close(ctx.done)
	}
}

// Fork creates and returns a new context as a child of this one.
func (ctx *ContextTree) Fork() Context {
	ctx.m.Lock()
	defer ctx.m.Unlock()

	child := &ContextTree{
		wg:       ctx.wg,
		done:     make(chan struct{}),
		children: map[*ContextTree]struct{}{},
	}
	if ctx.aliased == nil {
		child.aliased = ctx
	} else {
		child.aliased = ctx.aliased
	}
	ctx.children[child] = struct{}{}
	return child
}

// ForkWithTimeout creates and returns a new context as a child of this
// one. It also spins off a timer which will cancel the context after
// the given duration (unless the context terminates first).
func (ctx *ContextTree) ForkWithTimeout(dur time.Duration) Context {
	timer := time.NewTimer(dur)
	child := ctx.Fork()
	go func() {
		select {
		case <-child.Done():
			timer.Stop()
		case <-timer.C:
			child.Terminate(TimedOut)
		}
	}()
	return child
}

// Get returns the value associated with the given key. If this context
// has had no values set, then the lookup is made on the nearest ancestor
// with data. If no value is found, an unboxed nil value is returned.
func (ctx *ContextTree) Get(key interface{}) interface{} {
	val, _ := ctx.GetOK(key)
	return val
}

// GetOK returns the value associated with the given key, along with a
// bool value indicating successful lookup. See Get for details.
func (ctx *ContextTree) GetOK(key interface{}) (interface{}, bool) {
	ctx.m.RLock()
	defer ctx.m.RUnlock()

	if ctx.aliased != nil {
		return ctx.aliased.GetOK(key)
	}

	val, ok := ctx.data[key]
	return val, ok
}

// Set associates the given key and value in this context's data.
func (ctx *ContextTree) Set(key, val interface{}) {
	ctx.m.Lock()
	defer ctx.m.Unlock()

	if ctx.aliased != nil {
		ctx.data = kvmap{}
		ctx.aliased.m.RLock()
		for k, v := range ctx.aliased.data {
			ctx.data[k] = v
		}
		ctx.aliased.m.RUnlock()
		ctx.aliased = nil
	}
	ctx.data[key] = val
}

// Breakpoint returns an error channel that can be used to synchronize
// with a call to Check with the exact same parameters from another
// goroutine. The call to Check will send a nil value across this
// channel, and then receive a value to return to its caller.
func (ctx *ContextTree) Breakpoint(scope ...interface{}) chan error {
	return ctx.Get(bpmapKey).(bpmap).get(true, scope...)
}

// Check synchronizes with a registered breakpoint to obtain an error
// value to return, or immediately returns nil if no breakpoint is
// registered.
func (ctx *ContextTree) Check(scope ...interface{}) error {
	ch := ctx.Get(bpmapKey).(bpmap).get(false, scope...)
	if ch == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- nil:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}
