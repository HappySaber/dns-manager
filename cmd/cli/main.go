package main

import (
	"dnsmanager/internal/cli"
	"fmt"
	"os"
)

func run() int {
	if err := cli.NewRootCmd(); err != nil {
		fmt.Fprintln(os.Stderr, "Error in cmd initial")
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
