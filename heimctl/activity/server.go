package activity

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"euphoria.io/heim/backend"
	"euphoria.io/scope"
	"github.com/prometheus/client_golang/prometheus"
)

func Serve(ctx scope.Context, addr string) {
	http.Handle("/metrics", prometheus.Handler())

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		ctx.Terminate(err)
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

	backend.Logger(ctx).Printf("serving /metrics on %s", addr)
	if err := http.Serve(listener, nil); err != nil {
		fmt.Printf("http[%s]: %s\n", addr, err)
		ctx.Terminate(err)
	}

	closeListener()
	ctx.WaitGroup().Done()
}
