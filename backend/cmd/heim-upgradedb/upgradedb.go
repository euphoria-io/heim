package main

import (
	"flag"
	"fmt"
	"os"

	"heim/backend/persist"
)

var psql = flag.String("psql", "psql", "")

func main() {
	flag.Parse()

	b, err := persist.NewBackend(*psql)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	if err := b.UpgradeDB(); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}
