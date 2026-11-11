package service

import (
	"context"
	"dnsmanager/internal/model"
	"fmt"
	"net/netip"
	"slices"
)

func (s *Service) State() model.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.state
	st.Servers = slices.Clone(st.Servers)
	return st
}

func (s *Service) ApplyRemote(ctx context.Context, st model.State) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	addrs, err := parseAll(st.Servers)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !st.NewerThan(s.state) {
		return nil
	}
	f, err := s.rc.Read()
	if err != nil {
		return fmt.Errorf("read resolv.conf: %w", err)
	}
	f.SetNameservers(addrs)
	if err := s.rc.Write(f); err != nil {
		return fmt.Errorf("write resolv.conf: %w", err)
	}
	s.known, s.knownSet = slices.Clone(addrs), true
	s.state = model.State{Revision: st.Revision, Origin: st.Origin, Servers: toStrings(addrs)}
	s.log.Info("applied state from peer", "revision", st.Revision, "origin", st.Origin,
		"servers", s.state.Servers)

	if err := s.bu.Save(s.state); err != nil {
		return fmt.Errorf("resolv.conf updated, but backup failed: %w", err)
	}
	return nil
}

func parseAll(raw []string) ([]netip.Addr, error) {
	if len(raw) > MaxNameservers {
		return nil, fmt.Errorf("%w: got %d, limit is %d", ErrLimitReached, len(raw), MaxNameservers)
	}
	addrs := make([]netip.Addr, 0, len(raw))
	for _, r := range raw {
		a, err := parseAddr(r)
		if err != nil {
			return nil, err
		}
		if slices.Contains(addrs, a) {
			return nil, fmt.Errorf("%w: duplicate %s", ErrInvalidAddress, a)
		}
		addrs = append(addrs, a)
	}
	return addrs, nil
}
