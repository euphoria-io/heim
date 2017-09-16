package scope

import (
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestContext(t *testing.T) {
	Convey("Context data", t, func() {
		Convey("Local Get/GetOK/Set", func() {
			ctx := New()
			So(ctx.Get("test"), ShouldBeNil)
			val, ok := ctx.GetOK("test")
			So(ok, ShouldBeFalse)
			So(val, ShouldBeNil)
			ctx.Set("test", "value")
			So(ctx.Get("test"), ShouldEqual, "value")
			val, ok = ctx.GetOK("test")
			So(ok, ShouldBeTrue)
			So(val, ShouldEqual, "value")
		})

		Convey("Copy-on-write", func() {
			root := New()
			child := root.Fork()
			gchild := child.Fork()

			root.Set("a", 1)
			root.Set("b", 2)
			root.Set("c", 3)

			So(gchild.Get("a"), ShouldEqual, 1)

			child.Set("b", 4)
			root.Set("a", 5)
			So(child.Get("a"), ShouldEqual, 1)
			So(child.Get("b"), ShouldEqual, 4)

			// gchild should still refer to root's data
			So(gchild.Get("a"), ShouldEqual, 5)
			So(gchild.Get("b"), ShouldEqual, 2)
		})
	})

	Convey("Context lifecycle", t, func() {
		Convey("Local Cancel/Terminate/Err/Done", func() {
			ctx := New()
			So(ctx.Err(), ShouldBeNil)
			select {
			case <-ctx.Done():
				t.Error("ctx.Done() should not be ready")
			default:
			}
			ctx.Cancel()
			<-ctx.Done()
			So(ctx.Err(), ShouldEqual, Cancelled)
		})

		Convey("Termination propagates down tree", func() {
			root := New()
			child := root.Fork()
			gchild := child.Fork()
			sib := root.Fork()

			err := fmt.Errorf("test error")
			child.Terminate(err)
			<-gchild.Done()
			So(gchild.Err(), ShouldEqual, err)
			<-child.Done()
			So(child.Err(), ShouldEqual, err)
			So(root.Err(), ShouldBeNil)
			So(sib.Err(), ShouldBeNil)

			root.Cancel()
			So(sib.Err(), ShouldEqual, Cancelled)
			So(root.Err(), ShouldEqual, Cancelled)
			So(child.Err(), ShouldEqual, err)
			So(gchild.Err(), ShouldEqual, err)
		})
	})

	Convey("Shared wait group", t, func() {
		root := New()
		child := root.Fork()
		gchild := child.Fork()
		sib := root.Fork()

		for _, ctx := range []Context{child, gchild, sib} {
			So(ctx.WaitGroup(), ShouldEqual, root.WaitGroup())
		}
	})

	Convey("Breakpoints", t, func() {
		ctx := New()
		f := func() error { return ctx.Check("test") }

		Convey("Unlatched", func() {
			So(f(), ShouldBeNil)
		})

		Convey("Latched", func() {
			ctrl := ctx.Breakpoint("test")
			ch := make(chan error)
			go func() { ch <- f() }()
			testErr := fmt.Errorf("test error")
			<-ctrl
			ctrl <- testErr
			So(<-ch, ShouldEqual, testErr)
		})

		Convey("Cancellation", func() {
			Convey("Before synchronization", func() {
				root := New()
				root.Breakpoint("test")
				child := root.Fork()
				ready := make(chan bool)
				ch := make(chan error)
				go func() {
					ready <- true
					ch <- child.Check("test")
				}()
				<-ready
				root.Cancel()
				So(<-ch, ShouldEqual, Cancelled)
			})

			Convey("After synchronization", func() {
				root := New()
				ctrl := root.Breakpoint("test")
				child := root.Fork()
				ch := make(chan error)
				go func() { ch <- child.Check("test") }()
				<-ctrl
				root.Cancel()
				So(<-ch, ShouldEqual, Cancelled)
			})
		})

		Convey("Timeout", func() {
			Convey("Timeout expires", func() {
				start := time.Now()
				ctx := New().ForkWithTimeout(10 * time.Millisecond)
				<-ctx.Done()
				So(time.Now().Sub(start), ShouldBeGreaterThanOrEqualTo, 10*time.Millisecond)
				So(ctx.Err(), ShouldEqual, TimedOut)
			})

			Convey("Context terminates before expiration", func() {
				ctx := New().ForkWithTimeout(10 * time.Millisecond)
				time.Sleep(5 * time.Millisecond)
				ctx.Terminate(nil)
				time.Sleep(10 * time.Millisecond)
				So(ctx.Err(), ShouldBeNil)
			})
		})
	})
}
