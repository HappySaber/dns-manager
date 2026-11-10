package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	dnsv1 "dnsmanager/gen/dns-manager"
	"dnsmanager/internal/config"
	"dnsmanager/internal/resolvconf"
	"dnsmanager/internal/server"
	"dnsmanager/internal/service"
	"dnsmanager/internal/storage"
	"dnsmanager/internal/watcher"
)

type App struct {
	log             *slog.Logger
	lis             net.Listener
	grpcServer      *grpc.Server
	svc             *service.Service
	watcher         *watcher.Watcher
	shutdownTimeout time.Duration
}

func New(cfg config.Config, log *slog.Logger) (*App, error) {
	bk, err := storage.New(cfg.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("init backup storage: %w", err)
	}
	lis, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", cfg.ListenAddr, err)
	}

	rc := resolvconf.NewStore(cfg.ResolvConfPath)
	svc := service.New(rc, bk, log)
	wtchr := watcher.New(cfg.ResolvConfPath, cfg.WatchDebounce, log)
	grpcServer := grpc.NewServer()
	dnsv1.RegisterDNSServiceServer(grpcServer, server.New(svc, log))

	return &App{
		log:             log,
		lis:             lis,
		grpcServer:      grpcServer,
		watcher:         wtchr,
		svc:             svc,
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
}

func (a *App) Addr() net.Addr { return a.lis.Addr() }

func (a *App) Run(ctx context.Context) error {
	if err := a.svc.Restore(ctx); err != nil {
		a.log.Warn("restore failed, continuing without it", "err", err)
	}
	a.svc.Sync(ctx)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return a.watcher.Run(ctx, a.svc.Sync) })
	g.Go(func() error { return a.serve(ctx) })
	return g.Wait()
}

func (a *App) serve(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("grpc server started", "addr", a.lis.Addr().String())
		errCh <- a.grpcServer.Serve(a.lis)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	a.log.Info("shutting down")
	stopped := make(chan struct{})
	go func() {
		a.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(a.shutdownTimeout):
		a.log.Warn("graceful stop timed out, forcing")
		a.grpcServer.Stop()
	}
	return nil
}
