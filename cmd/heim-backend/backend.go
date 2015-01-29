package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"heim/backend"
	"heim/backend/persist"
	"heim/server"
)

var (
	addr   = flag.String("http", ":8080", "")
	id     = flag.String("id", "singleton", "")
	psql   = flag.String("psql", "psql", "")
	static = flag.String("static", "", "")

	ctrlAddr     = flag.String("control", ":2222", "")
	ctrlHostKey  = flag.String("control-hostkey", "", "")
	ctrlAuthKeys = flag.String("control-authkeys", "", "")
)

var version string

func main() {
	flag.Parse()

	b, err := persist.NewBackend(*psql, version)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	server := backend.NewServer(b, *id, *static)
	if err := controller(server); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("serving on %s\n", *addr)
	http.ListenAndServe(*addr, newVersioningHandler(server))
}

func controller(server *backend.Server) error {
	if *ctrlAddr != "" {
		ctrl, err := control.NewController(*ctrlAddr, server)
		if err != nil {
			return err
		}

		if *ctrlHostKey != "" {
			if err := ctrl.AddHostKey(*ctrlHostKey); err != nil {
				return err
			}
		}

		if *ctrlAuthKeys != "" {
			if err := ctrl.AddAuthorizedKeys(*ctrlAuthKeys); err != nil {
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
