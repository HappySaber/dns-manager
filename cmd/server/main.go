package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"dnsmanager/internal/app"
	"dnsmanager/internal/config"
)

func main() {
	os.Exit(run())
}

func run() int {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load config failed", "err", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a, err := app.New(cfg, log)
	if err != nil {
		log.Error("init failed", "err", err)
		return 1
	}
	if err := a.Run(ctx); err != nil {
		log.Error("run failed", "err", err)
		return 1
	}
	return 0
}
