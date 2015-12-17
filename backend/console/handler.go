package console

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"strings"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type ioterm interface {
	io.Writer

	ReadPassword(prompt string) (string, error)
}

func cmdConsole(ctrl *Controller, cmd string, term ioterm) *console {
	c := &console{
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

	backend proto.Backend
	kms     security.KMS
}

// Implement Session and Identity.
// TODO: log details about the client
func (c *console) Identity() proto.Identity { return (*consoleIdentity)(c) }
func (c *console) ID() string               { return "console" }
func (c *console) AgentID() string          { return "!console!" }
func (c *console) ServerID() string         { return "" }
func (c *console) SetName(name string)      {}
func (c *console) Close()                   {}
func (c *console) CheckAbandoned() error    { return nil }

func (c *console) View(level proto.PrivilegeLevel) *proto.SessionView {
	return &proto.SessionView{
		IdentityView: c.Identity().View(),
		SessionID:    "console",
	}
}

func (c *console) Send(scope.Context, proto.PacketType, interface{}) error {
	return fmt.Errorf("not implemented")
}

func (c *console) Print(args ...interface{})                 { fmt.Fprint(c, args...) }
func (c *console) Println(args ...interface{})               { fmt.Fprintln(c, args...) }
func (c *console) Printf(format string, args ...interface{}) { fmt.Fprintf(c, format, args...) }

func (c *console) Write(data []byte) (int, error) {
	data = bytes.Replace(data, []byte("\n"), []byte("\r\n"), -1)
	return c.ioterm.Write(data)
}

func (c *console) resolveAccount(ctx scope.Context, ref string) (proto.Account, error) {
	idx := strings.IndexRune(ref, ':')
	if idx < 0 {
		var accountID snowflake.Snowflake
		if err := accountID.FromString(ref); err != nil {
			return nil, err
		}
		return c.backend.AccountManager().Get(ctx, accountID)
	}
	return c.backend.AccountManager().Resolve(ctx, ref[:idx], ref[idx+1:])
}

type consoleIdentity console

func (c *consoleIdentity) ID() proto.UserID { return proto.UserID("console") }
func (c *consoleIdentity) Name() string     { return "console" }
func (c *consoleIdentity) ServerID() string { return "" }

func (c *consoleIdentity) View() *proto.IdentityView {
	return &proto.IdentityView{ID: "console", Name: "console"}
}

type handler interface {
	run(ctx scope.Context, c *console, args []string) error
}

var handlers = map[string]handler{}

func register(name string, h handler) { handlers[name] = h }

type usager interface {
	usage() string
}

func usageError(format string, args ...interface{}) error { return uerror(fmt.Sprintf(format, args...)) }

type uerror string

func (e uerror) Error() string { return string(e) }

func runHandler(ctx scope.Context, h handler, c *console, args []string) {
	if err := h.run(ctx, c, args); err != nil {
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

func runCommand(ctx scope.Context, ctrl *Controller, cmd string, term ioterm, args []string) {
	if handler, ok := handlers[cmd]; ok {
		runHandler(ctx, handler, cmdConsole(ctrl, cmd, term), args)
	} else {
		fmt.Fprintf(term, "invalid command: %s\r\n", cmd)
	}
}
