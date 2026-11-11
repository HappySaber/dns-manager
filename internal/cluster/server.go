package cluster

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dnsv1 "dnsmanager/gen/dns-manager"
	"dnsmanager/internal/model"
	"dnsmanager/internal/service"
)

type Applier interface {
	State() model.State
	ApplyRemote(ctx context.Context, st model.State) error
}

type Server struct {
	dnsv1.UnimplementedClusterServiceServer
	svc Applier
	log *slog.Logger
}

func NewServer(svc Applier, log *slog.Logger) *Server { return &Server{svc: svc, log: log} }

func (s *Server) GetState(context.Context, *dnsv1.GetStateRequest) (*dnsv1.GetStateResponse, error) {
	return &dnsv1.GetStateResponse{State: toProto(s.svc.State())}, nil
}

func (s *Server) Replicate(ctx context.Context, req *dnsv1.ReplicateRequest) (*dnsv1.ReplicateResponse, error) {
	if req.GetState() == nil {
		return nil, status.Error(codes.InvalidArgument, "state is required")
	}
	err := s.svc.ApplyRemote(ctx, fromProto(req.GetState()))
	switch {
	case err == nil:
		return &dnsv1.ReplicateResponse{}, nil
	case errors.Is(err, service.ErrInvalidAddress), errors.Is(err, service.ErrLimitReached):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, context.Canceled):
		return nil, status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return nil, status.Error(codes.DeadlineExceeded, "deadline exceeded")
	default:
		s.log.Error("apply replicated state failed", "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
}
