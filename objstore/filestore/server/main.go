package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"heim/objstore/filestore"
)

var (
	port = flag.Int("port", 80, "port to serve http on")
	vol  = flag.String("vol", "", "path to storage volume")
)

func main() {
	flag.Parse()
	if *vol == "" {
		fmt.Printf("usage: %s [-port=PORT] -vol=PATH\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	store, err := filestore.Open(*vol, "")
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(2)
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), store); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(2)
	}
}
