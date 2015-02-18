package console

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type testHandler struct{}

func (testHandler) run(c *console, args []string) error {
	if len(args) != 1 {
		return usageError("invalid number of arguments: %d", len(args))
	}
	fmt.Fprintln(c, "ok")
	return nil
}

type testHandlerWithUsage struct {
	testHandler
}

func (testHandlerWithUsage) usage() string { return "usage" }

func TestRunHandler(t *testing.T) {
	ctrl := &Controller{}

	Convey("Successfully runs", t, func() {
		term := &bytes.Buffer{}
		runHandler(testHandler{}, cmdConsole(ctrl, "test", term), []string{"test"})
		So(term.String(), ShouldEqual, "ok\r\n")
	})

	Convey("Usage error", t, func() {
		Convey("Handler serves usage", func() {
			term := &bytes.Buffer{}
			runHandler(testHandlerWithUsage{}, cmdConsole(ctrl, "test", term), nil)
			So(term.String(), ShouldEqual,
				"error: invalid number of arguments: 0\r\nusage\r\n\r\nOPTIONS:\r\n")
		})

		Convey("Handler doesn't serve usage", func() {
			term := &bytes.Buffer{}
			runHandler(testHandler{}, cmdConsole(ctrl, "test", term), nil)
			So(term.String(), ShouldEqual, "error: invalid number of arguments: 0\r\n")
		})
	})
}

func TestRunCommand(t *testing.T) {
	Convey("Unregistered command prints error", t, func() {
		term := &bytes.Buffer{}
		runCommand(nil, "asdf", term, nil)
		So(term.String(), ShouldEqual, "invalid command: asdf\r\n")
	})

	Convey("Registered command is invoked", t, func() {
		save := handlers
		defer func() { handlers = save }()
		handlers = map[string]handler{}
		register("test", testHandler{})
		term := &bytes.Buffer{}
		runCommand(&Controller{}, "test", term, []string{"arg"})
		So(term.String(), ShouldEqual, "ok\r\n")
	})
}
