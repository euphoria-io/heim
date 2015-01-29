package main

import (
	"flag"
	"fmt"
	"os"

	"heim/backend/psql"
)

var psqlDSN = flag.String("psql", "psql", "")

func main() {
	flag.Parse()

	b, err := psql.NewBackend(*psqlDSN, "upgradedb")
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}

	if err := b.UpgradeDB(); err != nil {
		fmt.Print("error: %s\n", err)
		os.Exit(1)
	}
}
