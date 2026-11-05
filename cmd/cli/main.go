package main

import (
	"context"
	"dnsmanager/internal/cli"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.NewRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", cli.Message(err))
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}
