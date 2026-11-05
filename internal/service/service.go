package service

import (
	"context"
	"errors"
)

var (
	ErrInvalidAddress = errors.New("invalid address")
	ErrAlreadyExists  = errors.New("server already exists")
	ErrNotFound       = errors.New("server not found")
)

type Service struct {
	servers []string
}

func New() *Service {
	return &Service{}
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
