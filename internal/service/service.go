package service

import (
	"context"
	"dnsmanager/internal/resolvconf"
	"errors"
)

var (
	ErrInvalidAddress = errors.New("invalid address")
	ErrAlreadyExists  = errors.New("server already exists")
	ErrNotFound       = errors.New("server not found")
)

type Service struct {
	rc ResolvConf
}

type ResolvConf interface {
	Read() (*resolvconf.File, error)
	Write(*resolvconf.File) error
}

func New(rc ResolvConf) *Service {
	return &Service{
		rc: rc,
	}
}

func (s *Service) List(ctx context.Context) ([]string, error) {
	//impl
	return []string{}, nil
}

func (s *Service) Add(ctx context.Context, addr string) error {
	//impl
	return nil
}

func (s *Service) Remove(ctx context.Context, addr string) error {
	//impl
	return nil
}
