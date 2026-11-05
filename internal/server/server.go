package server

import (
	"context"
	dnsv1 "dnsmanager/gen/dns-manager"
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
