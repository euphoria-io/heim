package main

import (
	"flag"
	"fmt"
	"net/http"

	"heim/backend"
)

var addr = flag.String("http", ":8080", "")
var static = flag.String("static", "", "")

func main() {
	flag.Parse()

	server := backend.NewServer(*static)
	fmt.Printf("serving on %s\n", *addr)
	http.ListenAndServe(*addr, server)
}
