package main

import (
	"flag"

	"euphoria.io/heim/heimctl/cmd"
)

var version string

func main() {
	if version != "" {
		cmd.Version = version
	}
	flag.Parse()
	cmd.Run(flag.Args())
}
