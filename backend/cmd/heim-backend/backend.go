package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"heim/backend"
	"heim/backend/persist"
)

var (
	addr   = flag.String("http", ":8080", "")
	psql   = flag.String("psql", "psql", "")
	static = flag.String("static", "", "")
)

var version string

func main() {
	flag.Parse()

	b, err := persist.NewBackend(*psql)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	server := backend.NewServer(b, *static)
	fmt.Printf("serving on %s\n", *addr)
	http.ListenAndServe(*addr, newVersioningHandler(server))
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
