package service

import (
	"dnsmanager/internal/model"
	"slices"
)

func (s *Service) State() model.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	st.Servers = slices.Clone(st.Servers)
	return st
}
