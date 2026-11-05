package service

import "context"

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
