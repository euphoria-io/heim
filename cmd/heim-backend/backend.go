package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/backend/console"
	"euphoria.io/heim/backend/psql"
	_ "euphoria.io/heim/cmd" // for -newflags
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
)

var (
	addr   = flag.String("http", ":8080", "address to serve http on")
	config = flag.String("config", "", "path to local config (default: use config stored in etcd)")
	static = flag.String("static", "", "path to static files")

	consoleAddr = flag.String("console", "", "")

	version string
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	if backend.Config.Cluster.ServerID == "" {
		id, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("hostname error: %s", err)
		}
		backend.Config.Cluster.ServerID = id
	}

	era, err := snowflake.New()
	if err != nil {
		return fmt.Errorf("era error: %s", err)
	}

	backend.Config.Cluster.Era = era.String()
	backend.Config.Cluster.Version = version

	c, err := backend.Config.Cluster.EtcdCluster()
	if err != nil {
		return fmt.Errorf("cluster error: %s", err)
	}

	if *config == "" {
		if err := backend.Config.LoadFromEtcd(c); err != nil {
			return fmt.Errorf("config: %s", err)
		}
	} else {
		if err := backend.Config.LoadFromFile(*config); err != nil {
			return fmt.Errorf("config: %s", err)
		}
	}

	fmt.Printf("active config:\n\n%s\n", backend.Config.String())

	kms, err := backend.Config.KMS.Get()
	if err != nil {
		return fmt.Errorf("kms error: %s", err)
	}

	serverDesc := backend.Config.Cluster.DescribeSelf()
	b, err := psql.NewBackend(backend.Config.DB.DSN, c, serverDesc)
	if err != nil {
		return fmt.Errorf("backend error: %s", err)
	}
	defer b.Close()

	if err := controller(b, kms); err != nil {
		return fmt.Errorf("controller error: %s", err)
	}

	server, err := backend.NewServer(b, c, kms, serverDesc.ID, serverDesc.Era, *static)
	if err != nil {
		return fmt.Errorf("server error: %s", err)
	}

	fmt.Printf("serving era %s on %s\n", serverDesc.Era, *addr)
	http.ListenAndServe(*addr, newVersioningHandler(server))
	return nil
}

func controller(b proto.Backend, kms security.KMS) error {
	if *consoleAddr != "" {
		ctrl, err := console.NewController(*consoleAddr, b, kms)
		if err != nil {
			return err
		}

		if backend.Config.Console.HostKey != "" {
			if err := ctrl.AddHostKey(backend.Config.Console.HostKey); err != nil {
				return err
			}
		}

		for _, authKey := range backend.Config.Console.AuthKeys {
			if err := ctrl.AddAuthorizedKeys(authKey); err != nil {
				return err
			}
		}

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
		version: version,
		handler: handler,
	}
}

func (vh *versioningHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if vh.version != "" {
		w.Header().Set("X-Heim-Version", vh.version)
	}
	vh.handler.ServeHTTP(w, r)
}
