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
	"time"
)

// Max nimber of servers, another won't work in file
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

type Peer interface {
	GetState(ctx context.Context) (model.State, error)
	Replicate(ctx context.Context, st model.State) error
}

type Service struct {
	bu       Backup
	rc       ResolvConf
	known    []netip.Addr
	knownSet bool
	mu       sync.Mutex
	log      *slog.Logger

	state model.State

	nodeID      string
	peer        Peer
	replTimeout time.Duration
}

type Option func(*Service)

func WithCluster(nodeID string, peer Peer, timeout time.Duration) Option {
	return func(s *Service) {
		s.nodeID, s.peer, s.replTimeout = nodeID, peer, timeout
	}
}

func New(rc ResolvConf, bu Backup, log *slog.Logger, opts ...Option) *Service {
	s := &Service{rc: rc, bu: bu, log: log, replTimeout: 2 * time.Second}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Service) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := s.rc.Read()
	if err != nil {
		return nil, fmt.Errorf("read resolv.conf: %w", err)
	}
	return toStrings(f.Nameservers()), nil
}

func (s *Service) Add(ctx context.Context, addr string) error {
	addrParsed, err := parseAddr(addr)
	if err != nil {
		return err
	}
	st, err := s.mutate(ctx, func(cur []netip.Addr) ([]netip.Addr, error) {
		if slices.Contains(cur, addrParsed) {
			return nil, fmt.Errorf("%w: %s", ErrAlreadyExists, addrParsed)
		}
		if len(cur) >= MaxNameservers {
			return nil, fmt.Errorf("%w: limit is %d", ErrLimitReached, MaxNameservers)
		}
		return append(cur, addrParsed), nil
	})
	if err != nil {
		return err
	}
	s.log.Info("dns server added", "addr", addrParsed.String(), "revision", st.Revision)
	s.replicate(ctx, st)
	return nil
}

func (s *Service) Remove(ctx context.Context, addr string) error {
	addrParsed, err := parseAddr(addr)
	if err != nil {
		return err
	}
	st, err := s.mutate(ctx, func(cur []netip.Addr) ([]netip.Addr, error) {
		i := slices.Index(cur, addrParsed)
		if i < 0 {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, addr)
		}
		return slices.Delete(cur, i, i+1), nil
	})

	if err != nil {
		return err
	}

	s.log.Info("dns server removed", "addr", addrParsed.String(), "revision", st.Revision)
	s.replicate(ctx, st)
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

	saved, err := s.bu.Load()
	if errors.Is(err, storage.ErrNotFound) {
		s.log.Info("no backup found, nothing to restore")
		return nil
	}
	if err != nil {
		return fmt.Errorf("load backup: %w", err)
	}
	s.state = model.State{Revision: saved.Revision, Origin: saved.Origin, Servers: slices.Clone(saved.Servers)}

	f, err := s.rc.Read()
	if err != nil {
		return fmt.Errorf("read resolv.conf: %w", err)
	}
	if len(f.Nameservers()) > 0 {
		s.log.Info("resolv.conf already has nameservers, restore skipped")
		return nil
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
	s.known, s.knownSet = slices.Clone(addrs), true
	s.log.Info("dns servers restored from backup", "count", len(addrs), "revision", saved.Revision)
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

func (s *Service) apply(f *resolvconf.File, addrs []netip.Addr) (model.State, error) {
	f.SetNameservers(addrs)
	if err := s.rc.Write(f); err != nil {
		return model.State{}, fmt.Errorf("write resolv.conf: %w", err)
	}
	s.known, s.knownSet = slices.Clone(addrs), true

	st := model.State{Revision: s.state.Revision + 1, Origin: s.nodeID, Servers: toStrings(addrs)}
	s.state = st
	if err := s.bu.Save(st); err != nil {
		return model.State{}, fmt.Errorf("resolv.conf updated, but backup failed: %w", err)
	}
	return st, nil
}

func toStrings(addrs []netip.Addr) []string {
	res := make([]string, len(addrs))
	for i, a := range addrs {
		res[i] = a.String()
	}
	return res
}

func (s *Service) mutate(ctx context.Context, change func(cur []netip.Addr) ([]netip.Addr, error)) (model.State, error) {
	if err := ctx.Err(); err != nil {
		return model.State{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := s.rc.Read()
	if err != nil {
		return model.State{}, fmt.Errorf("read resolv.conf: %w", err)
	}
	next, err := change(f.Nameservers())
	if err != nil {
		return model.State{}, err
	}
	return s.apply(f, next)
}

func (s *Service) replicate(ctx context.Context, st model.State) {
	if s.peer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.replTimeout)
	defer cancel()
	if err := s.peer.Replicate(ctx, st); err != nil {
		s.log.Warn("replication to peer failed, will retry in background",
			"revision", st.Revision, "err", err)
	}
}
