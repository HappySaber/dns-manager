package server

import (
	"context"
	dnsv1 "dnsmanager/gen/dns-manager"
	"dnsmanager/internal/service"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DNSService interface {
	Add(ctx context.Context, addr string) error
	Remove(ctx context.Context, addr string) error
	List(ctx context.Context) ([]string, error)
}

type Server struct {
	dnsv1.UnimplementedDNSServiceServer
	svc DNSService
	log *slog.Logger
}

func New(svc DNSService, log *slog.Logger) *Server {
	return &Server{
		svc: svc,
		log: log,
	}
}

func (s *Server) AddServer(ctx context.Context, req *dnsv1.AddServerRequest) (*dnsv1.AddServerResponse, error) {
	if err := s.svc.Add(ctx, req.GetAdresses()); err != nil {
		return nil, s.toStatus(err)
	}
	return &dnsv1.AddServerResponse{}, nil
}

func (s *Server) RemoveServer(ctx context.Context, req *dnsv1.RemoveServerRequest) (*dnsv1.RemoveServerResponse, error) {
	if err := s.svc.Remove(ctx, req.GetAdresses()); err != nil {
		return nil, s.toStatus(err)
	}
	return &dnsv1.RemoveServerResponse{}, nil
}

func (s *Server) ListServers(ctx context.Context, req *dnsv1.ListServersRequest) (*dnsv1.ListServersResponse, error) {
	res, err := s.svc.List(ctx)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return &dnsv1.ListServersResponse{
		Adresses: res,
	}, nil
}

func (s *Server) toStatus(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidAddress):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, service.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrLimitReached):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	default:
		s.log.Error("internal error", "err", err)
		return status.Error(codes.Internal, "internal error")
	}
}
