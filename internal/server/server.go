package server

import (
	"context"
	dnsv1 "dnsmanager/gen/dns-manager"
	"errors"
)

type DNSService interface {
	Add(ctx context.Context, addr string) error
	Remove(ctx context.Context, addr string) error
	List(ctx context.Context) ([]string, error)
}

type Server struct {
	dnsv1.UnimplementedDNSServiceServer
	svc DNSService
}

func New(svc DNSService) *Server {
	return &Server{
		svc: svc,
	}
}

func (s *Server) AddServer(ctx context.Context, req *dnsv1.AddServerRequest) (*dnsv1.AddServerResponse, error) {
	if err := s.svc.Add(ctx, req.GetAdresses()); err != nil {
		return nil, errors.New("err")
	}
	return &dnsv1.AddServerResponse{}, nil
}
