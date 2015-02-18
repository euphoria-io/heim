package console

import (
	"bytes"
	"flag"
	"fmt"
	"io"

	"heim/proto"
	"heim/proto/security"

	"golang.org/x/net/context"
)

type ioterm interface {
	io.Writer
}

func cmdConsole(ctrl *Controller, cmd string, term ioterm) *console {
	c := &console{
		ctx:     context.Background(),
		backend: ctrl.backend,
		kms:     ctrl.kms,
		ioterm:  term,
		FlagSet: flag.NewFlagSet(cmd, flag.ContinueOnError),
	}
	c.SetOutput(c)
	return c
}

type console struct {
	ioterm
	*flag.FlagSet

	ctx     context.Context
	backend proto.Backend
	kms     security.KMS
}

func (c *console) Print(args ...interface{})                 { fmt.Fprint(c, args...) }
func (c *console) Println(args ...interface{})               { fmt.Fprintln(c, args...) }
func (c *console) Printf(format string, args ...interface{}) { fmt.Fprintf(c, format, args...) }

func (c *console) Write(data []byte) (int, error) {
	data = bytes.Replace(data, []byte("\n"), []byte("\r\n"), -1)
	return c.ioterm.Write(data)
}

type handler interface {
	run(c *console, args []string) error
}

var handlers = map[string]handler{}

func register(name string, h handler) { handlers[name] = h }

type usager interface {
	usage() string
}

func usageError(format string, args ...interface{}) error { return uerror(fmt.Sprintf(format, args...)) }

type uerror string

func (e uerror) Error() string { return string(e) }

func runHandler(h handler, c *console, args []string) {
	if err := h.run(c, args); err != nil {
		u, uok := h.(usager)
		if err != flag.ErrHelp {
			c.Printf("error: %s\n", err.Error())
		}
		_, ok := err.(uerror)
		if ok || err == flag.ErrHelp {
			if uok {
				c.Println(u.usage())
				c.Printf("\nOPTIONS:\n")
			}
			c.PrintDefaults()
		}
	}
}

func runCommand(ctrl *Controller, cmd string, term ioterm, args []string) {
	if handler, ok := handlers[cmd]; ok {
		runHandler(handler, cmdConsole(ctrl, cmd, term), args)
	} else {
		fmt.Fprintf(term, "invalid command: %s\r\n", cmd)
	}
}
