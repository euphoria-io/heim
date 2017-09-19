package main

import (
	"flag"

	"euphoria.io/heim/heimctl/cmd"
)

var Version string

func main() {
	if Version != "" {
		cmd.Version = Version
	}
	flag.Parse()
	cmd.Run(flag.Args())
}
