package main

import (
	"context"
	dnsv1 "dnsmanager/gen/dns-manager"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

type server struct {
	dnsv1.UnimplementedDNSServiceServer
}

func (s *server) ListServers(_ context.Context, _ *dnsv1.ListServersRequest) (*dnsv1.ListServersResponse, error) {
	return &dnsv1.ListServersResponse{Adresses: []string{"8.8.8.8"}}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("listen failed", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	dnsv1.RegisterDNSServiceServer(grpcServer, &server{})

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		grpcServer.GracefulStop()
	}()

	slog.Info("listening", "addr", lis.Addr().String())
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("serve failed", "err", err)
		os.Exit(1)
	}
}
