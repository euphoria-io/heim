/*
Package scope provides context objects for the sharing of scope across
goroutines. This context object provides a number of utilities for
coordinating concurrent work, in addition to sharing data.

Lifecycle

Contexts are nodes in a tree. A context is born either by forking from
an existing context (becoming a child of that node in the tree), or a
new tree is started by calling New().

A context can be terminated at any time. This is usually done by calling
the Terminate() or Cancel() method. Termination is associated with an
error value (which may be nil if one wants to indicate success). When
a node in the tree is terminated, that termination is propagated down
to all its unterminated descendents.

For example, here is how one might fan out a search:

	// Fan out queries.
	for _, q := range queries {
		go func() {
			a, err := q.Run(ctx.Fork())
			if err != nil {
				answers <- nil
			} else {
				answers <- a
			}
		}()
	}
	// Receive answers (or failures).
	for answer := range answers {
		if answer != nil {
			ctx.Cancel() // tell outstanding queries to give up
			return answer, nil
		}
	}
	return nil, fmt.Errorf("all queries failed")

Contexts can be terminated at any time. You can even fork a context
with a deadline:

	ctx := scope.New()
	result, err := Search(ctx.ForkWithTimeout(5 * time.Second), queries)
	if err == scope.TimedOut {
		// one or more backends timed out, have the caller back off
	}

There is a termination channel, Done(), available if you want to interrupt
your work when a context is terminated:

	// Wait for 10 seconds or termination incurred from another goroutine,
	// whichever occurs first.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.After(10*time.Second):
		return nil
	}

You can also spot-check for termination with a call to the Alive() method.

	for ctx.Alive() {
		readChunk()
	}

Data Sharing

Contexts provide a data store for key value pairs, shared across the entire
scope. When a context is forked, the child context shares the same data map
as its parent.

This data store maps blank interfaces to blank interfaces, in the exact
same manner as http://www.gorillatoolkit.org/pkg/context. This means you
must use type assertions at runtime. To keep this reasonably safe, it's
recommended to define and use your own unexported type for all keys maintained
by your package.

	type myKey int
	const (
		loggerKey myKey = iota
		dbKey
		// etc.
	)

	func SetLogger(ctx scope.Context, logger *log.Logger) {
		ctx.Set(loggerKey, logger)
	}

	func GetLogger(ctx scope.Context) logger *log.Logger) {
		return ctx.Get(loggerKey).(*log.Logger)
	}

The shared data store is managed in a copy-on-write fashion as the tree
branches. When a context is forked, the child maintains a pointer to the
parent's data map. When Set() is called on the child, the original map
is duplicated for the child, and the update is only applied to the child's
map.

Common WaitGroup

Each context provides a WaitGroup() method, which returns the same pointer
across the entire tree. You can use this to spin off background tasks and
then wait for them before you completely shut down the scope.

	ctx.WaitGroup().Add(1)
	go func() {
		doSomeThing(ctx)
		ctx.WaitGroup().Done()
	}()
	ctx.WaitGroup().Wait()

Breakpoints

Contexts provide an optional feature to facilitate unit testing, called
breakpoints. A breakpoint is identified by a list of hashable values.
Production code can pass this list to the Check() method to synchronize
and allow for an error to be injected. Test code can register a breakpoint
with Breakpoint(), which returns a channel of errors. The test can
receive from this channel to synchronize with the entry of the corresponding
Check() call, and then write back an error to synchronize with the exit.

	func Get(ctx scope.Context, url string) (*http.Response, error) {
		if err := ctx.Check("http.Get", url); err != nil {
			return nil, err
		}
		return http.Get(url)
	}

	func TestGetError(t *testing.T) {
		ctx := scope.New()
		ctrl := ctx.Breakpoint("http.Get", "http://google.com")
		testErr := fmt.Errorf("test error")
		go func() {
			<-ctrl
			ctrl <- testErr
		}()
		if err := Get(ctx, "http://google.com"); err != testErr {
			t.Fail()
		}
	}
*/
package scope
