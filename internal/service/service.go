package service

import (
	"context"
	"dnsmanager/internal/model"
	"dnsmanager/internal/resolvconf"
	"dnsmanager/internal/storage"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"
	"sync"
)

const MaxNameservers = 3

var (
	ErrInvalidAddress = errors.New("invalid address")
	ErrAlreadyExists  = errors.New("server already exists")
	ErrNotFound       = errors.New("server not found")
	ErrLimitReached   = errors.New("too many DNS servers")
)

type Backup interface {
	Save(model.State) error
	Load() (model.State, error)
}

type ResolvConf interface {
	Read() (*resolvconf.File, error)
	Write(*resolvconf.File) error
}

type Service struct {
	bu       Backup
	rc       ResolvConf
	known    []netip.Addr
	knownSet bool
	mu       sync.Mutex
	log      *slog.Logger
}

func New(rc ResolvConf, bu Backup, log *slog.Logger) *Service {
	return &Service{
		rc:  rc,
		bu:  bu,
		log: log,
	}
}

func (s *Service) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f, err := s.rc.Read()
	if err != nil {
		return nil, fmt.Errorf("read resolv.conf: %w", err)
	}

	addrs := f.Nameservers()
	res := make([]string, len(addrs))

	for i, val := range addrs {
		res[i] = val.String()
	}
	return res, nil
}

func (s *Service) Add(ctx context.Context, addr string) error {
	addrParsed, err := parseAddr(addr)
	if err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.rc.Read()
	if err != nil {
		return fmt.Errorf("read resolv.conf: %w", err)
	}

	current := f.Nameservers()
	if slices.Contains(current, addrParsed) {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, addr)
	}
	if len(current) >= MaxNameservers {
		return fmt.Errorf("%w: limit is %d", ErrLimitReached, MaxNameservers)
	}

	if err := s.apply(f, append(current, addrParsed)); err != nil {
		return err
	}
	s.log.Info("dns server added", "addr", addrParsed.String())
	return nil
}

func (s *Service) Remove(ctx context.Context, addr string) error {
	addrParsed, err := parseAddr(addr)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.rc.Read()
	if err != nil {
		return fmt.Errorf("read resolv.conf: %w", err)
	}
	current := f.Nameservers()
	i := slices.Index(current, addrParsed)
	if i < 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, addrParsed)
	}

	f.SetNameservers(slices.Delete(current, i, i+1))
	if err := s.rc.Write(f); err != nil {
		return fmt.Errorf("write resolv.conf: %w", err)
	}
	s.log.Info("dns server removed", "addr", addrParsed.String())
	return nil
}

func parseAddr(addr string) (netip.Addr, error) {
	addrParsed, err := netip.ParseAddr(addr)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %q", ErrInvalidAddress, addr)
	}
	if addrParsed.IsUnspecified() || addrParsed.IsMulticast() {
		return netip.Addr{}, fmt.Errorf("%w: %s", ErrInvalidAddress, addrParsed)
	}
	return addrParsed, nil
}

func (s *Service) Restore(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.rc.Read()
	if err != nil {
		return fmt.Errorf("read resolv.conf: %w", err)
	}
	if len(f.Nameservers()) > 0 {
		s.log.Info("resolv.conf already has nameservers, restore skipped")
		return nil
	}

	saved, err := s.bu.Load()
	if errors.Is(err, storage.ErrNotFound) {
		s.log.Info("no backup found, nothing to restore")
		return nil
	}
	if err != nil {
		return fmt.Errorf("load backup: %w", err)
	}

	var addrs []netip.Addr
	for _, raw := range saved.Servers {
		a, err := parseAddr(raw)
		if err != nil {
			s.log.Warn("skipping invalid address from backup", "addr", raw)
			continue
		}
		if !slices.Contains(addrs, a) && len(addrs) < MaxNameservers {
			addrs = append(addrs, a)
		}
	}
	if len(addrs) == 0 {
		return nil
	}

	f.SetNameservers(addrs)
	if err := s.rc.Write(f); err != nil {
		return fmt.Errorf("write resolv.conf: %w", err)
	}
	s.log.Info("dns servers restored from backup", "count", len(addrs))
	return nil
}

func (s *Service) Sync(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.rc.Read()
	if err != nil {
		s.log.Warn("sync: read resolv.conf failed", "err", err)
		return
	}
	cur := f.Nameservers()
	if !s.knownSet {
		s.known, s.knownSet = cur, true
		return
	}
	if slices.Equal(cur, s.known) {
		return
	}
	s.log.Warn("resolv.conf changed externally",
		"before", toStrings(s.known), "after", toStrings(cur))
	s.known = cur
}

func (s *Service) apply(f *resolvconf.File, addrs []netip.Addr) error {
	f.SetNameservers(addrs)
	if err := s.rc.Write(f); err != nil {
		return fmt.Errorf("write resolv.conf: %w", err)
	}
	s.known, s.knownSet = slices.Clone(addrs), true
	if err := s.bu.Save(model.State{Servers: toStrings(addrs)}); err != nil {
		return fmt.Errorf("resolv.conf updated, but backup failed: %w", err)
	}
	return nil
}

func toStrings(addrs []netip.Addr) []string {
	res := make([]string, len(addrs))
	for i, a := range addrs {
		res[i] = a.String()
	}
	return res
}
