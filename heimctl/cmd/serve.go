package cmd

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/console"
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	register("serve", &serveCmd{})
	register("serve-embed", &serveEmbedCmd{})
}

type serveCmd struct {
	addr        string
	static      string
	consoleAddr string
}

func (serveCmd) desc() string { return "start up a heim backend server" }

func (serveCmd) usage() string {
	return "serve [--http=<interface:port>] [--console=<interface:port>] [--static=<path>]"
}

func (serveCmd) longdesc() string {
	return `
	Start a heim backend server. The server will listen for HTTP requests
	at the address given by -http (defaults to port 8080 on any interface).
	The server will run until killed or instructed to shut down via console
	command.

	An optional ssh console is available. Use the -console flag to specify
	the address to listen on.
`[1:]
}

func (cmd *serveCmd) flags() *flag.FlagSet {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	flags.StringVar(&cmd.addr, "http", ":8080", "address to serve http on")
	flags.StringVar(&cmd.static, "static", "", "path to static files")
	flags.StringVar(&cmd.consoleAddr, "console", "", "")
	return flags
}

func (cmd *serveCmd) run(ctx scope.Context, args []string) error {
	listener, err := net.Listen("tcp", cmd.addr)
	if err != nil {
		return err
	}

	m := sync.Mutex{}
	closed := false
	closeListener := func() {
		m.Lock()
		if !closed {
			closed = true
			listener.Close()
		}
		m.Unlock()
	}
	defer closeListener()

	c, err := getCluster(ctx)
	if err != nil {
		return err
	}

	kms, err := backend.Config.KMS.Get()
	if err != nil {
		return fmt.Errorf("kms error: %s", err)
	}

	b, err := getBackend(ctx, c)
	if err != nil {
		return fmt.Errorf("backend error: %s", err)
	}
	defer b.Close()

	if err := controller(ctx, cmd.consoleAddr, b, kms, c); err != nil {
		return fmt.Errorf("controller error: %s", err)
	}

	serverDesc := backend.Config.Cluster.DescribeSelf()
	server, err := backend.NewServer(ctx, b, c, kms, serverDesc.ID, serverDesc.Era, cmd.static)
	if err != nil {
		return fmt.Errorf("server error: %s", err)
	}

	server.AllowRoomCreation(backend.Config.AllowRoomCreation)
	server.NewAccountMinAgentAge(backend.Config.NewAccountMinAgentAge)

	// Spin off goroutine to watch ctx and close listener if shutdown requested.
	go func() {
		<-ctx.Done()
		closeListener()
	}()

	fmt.Printf("serving era %s on %s\n", serverDesc.Era, cmd.addr)
	if err := http.Serve(listener, newVersioningHandler(server)); err != nil {
		if strings.HasSuffix(err.Error(), "use of closed network connection") {
			return nil
		}
		return err
	}

	return nil
}

func controller(
	ctx scope.Context, addr string, b proto.Backend, kms security.KMS, c cluster.Cluster) error {

	if addr != "" {
		ctrl, err := console.NewController(ctx, addr, b, kms, c)
		if err != nil {
			return err
		}

		if backend.Config.Console.HostKey != "" {
			if err := ctrl.AddHostKey(backend.Config.Console.HostKey); err != nil {
				return err
			}
		} else {
			if err := ctrl.AddHostKeyFromCluster(backend.Config.Cluster.ServerID); err != nil {
				return err
			}
		}

		for _, authKey := range backend.Config.Console.AuthKeys {
			if authKey == "" {
				continue
			}
			if err := ctrl.AddAuthorizedKeys(authKey); err != nil {
				return err
			}
		}

		ctx.WaitGroup().Add(1)
		go ctrl.Serve()
	}
	return nil
}

type versioningHandler struct {
	version string
	handler http.Handler
}

func newVersioningHandler(handler http.Handler) http.Handler {
	return &versioningHandler{
		version: Version,
		handler: handler,
	}
}

func (vh *versioningHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if vh.version != "" {
		w.Header().Set("X-Heim-Version", vh.version)
	}
	vh.handler.ServeHTTP(w, r)
}

type serveEmbedCmd struct {
	addr   string
	static string
	domain string
}

func (serveEmbedCmd) desc() string  { return "start up the embed.space static server" }
func (serveEmbedCmd) usage() string { return "serve [--http=<interface:port>] [--static=<path>]" }

func (serveEmbedCmd) longdesc() string {
	return `
	Start the embed.space static server. The server will listen for HTTP
	requests at the address given by -http (defaults to port 8081 on any
	interface).
`[1:]
}

func (cmd *serveEmbedCmd) flags() *flag.FlagSet {
	flags := flag.NewFlagSet("serve-embed", flag.ExitOnError)
	flags.StringVar(&cmd.addr, "http", ":8080", "address to serve http on")
	flags.StringVar(&cmd.static, "static", "", "path to static files")
	flags.StringVar(&cmd.domain, "domain", "", "require a Host header matching this domain, if given")
	return flags
}

func (cmd *serveEmbedCmd) run(ctx scope.Context, args []string) error {
	listener, err := net.Listen("tcp", cmd.addr)
	if err != nil {
		return err
	}

	closed := false
	m := sync.Mutex{}
	closeListener := func() {
		m.Lock()
		if !closed {
			listener.Close()
			closed = true
		}
		m.Unlock()
	}

	// Spin off goroutine to watch ctx and close listener if shutdown requested.
	go func() {
		<-ctx.Done()
		closeListener()
	}()

	if err := http.Serve(listener, cmd); err != nil {
		fmt.Printf("http[%s]: %s\n", cmd.addr, err)
		return err
	}

	closeListener()
	ctx.WaitGroup().Done()
	return ctx.Err()
}

func (cmd *serveEmbedCmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if cmd.domain != "" && r.Host != cmd.domain {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	if r.URL.Path == "/" {
		r.URL.Path = "/embed.html"
	}
	if r.URL.Path == "/metrics" {
		prometheus.Handler().ServeHTTP(w, r)
		return
	}
	http.FileServer(http.Dir(cmd.static)).ServeHTTP(w, r)
}
