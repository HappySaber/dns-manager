package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	dnsv1 "dnsmanager/gen/dns-manager"
	"dnsmanager/internal/service"
)

type fakeSvc struct {
	mu      sync.Mutex
	servers []string
	err     error
	added   []string
	removed []string
}

func (f *fakeSvc) List(context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.servers), f.err
}

func (f *fakeSvc) Add(_ context.Context, addr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.added = append(f.added, addr)
	return f.err
}

func (f *fakeSvc) Remove(_ context.Context, addr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, addr)
	return f.err
}

func newClient(t *testing.T, svc DNSService) dnsv1.DNSServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	dnsv1.RegisterDNSServiceServer(srv, New(svc, slog.New(slog.NewTextHandler(io.Discard, nil))))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return dnsv1.NewDNSServiceClient(conn)
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	return c
}

func TestListServers(t *testing.T) {
	svc := &fakeSvc{servers: []string{"1.1.1.1", "8.8.8.8"}}
	resp, err := newClient(t, svc).ListServers(ctx(t), &dnsv1.ListServersRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.GetAdresses(); !slices.Equal(got, svc.servers) {
		t.Fatalf("got %v, want %v", got, svc.servers)
	}
}

func TestListServers_Empty(t *testing.T) {
	resp, err := newClient(t, &fakeSvc{}).ListServers(ctx(t), &dnsv1.ListServersRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetAdresses()) != 0 {
		t.Fatalf("got %v, want empty", resp.GetAdresses())
	}
}

func TestAddServer_PassesAddress(t *testing.T) {
	svc := &fakeSvc{}
	if _, err := newClient(t, svc).AddServer(ctx(t), &dnsv1.AddServerRequest{Adresses: "8.8.8.8"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(svc.added, []string{"8.8.8.8"}) {
		t.Fatalf("service got %v", svc.added)
	}
}

func TestRemoveServer_PassesAddress(t *testing.T) {
	svc := &fakeSvc{}
	if _, err := newClient(t, svc).RemoveServer(ctx(t), &dnsv1.RemoveServerRequest{Adresses: "8.8.8.8"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(svc.removed, []string{"8.8.8.8"}) {
		t.Fatalf("service got %v", svc.removed)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"invalid address", service.ErrInvalidAddress, codes.InvalidArgument},
		{"already exists wrapped", fmt.Errorf("add 8.8.8.8: %w", service.ErrAlreadyExists), codes.AlreadyExists},
		{"not found", service.ErrNotFound, codes.NotFound},
		{"limit", service.ErrLimitReached, codes.FailedPrecondition},
		{"canceled", context.Canceled, codes.Canceled},
		{"deadline", context.DeadlineExceeded, codes.DeadlineExceeded},
		{"unknown", errors.New("boom"), codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newClient(t, &fakeSvc{err: tt.err})
			calls := map[string]func() error{
				"List": func() error {
					_, err := client.ListServers(ctx(t), &dnsv1.ListServersRequest{})
					return err
				},
				"Add": func() error {
					_, err := client.AddServer(ctx(t), &dnsv1.AddServerRequest{Adresses: "8.8.8.8"})
					return err
				},
				"Remove": func() error {
					_, err := client.RemoveServer(ctx(t), &dnsv1.RemoveServerRequest{Adresses: "8.8.8.8"})
					return err
				},
			}
			for rpcName, call := range calls {
				if got := status.Code(call()); got != tt.want {
					t.Errorf("%s: code = %v, want %v", rpcName, got, tt.want)
				}
			}
		})
	}
}

func TestInternalErrorDetailsAreNotLeaked(t *testing.T) {
	svc := &fakeSvc{err: errors.New("open /etc/secret/path: permission denied")}
	_, err := newClient(t, svc).AddServer(ctx(t), &dnsv1.AddServerRequest{Adresses: "8.8.8.8"})

	st := status.Convert(err)
	if st.Code() != codes.Internal {
		t.Fatalf("code = %v, want Internal", st.Code())
	}
	if strings.Contains(st.Message(), "secret") || strings.Contains(st.Message(), "permission") {
		t.Fatalf("internal details leaked to client: %q", st.Message())
	}
}

func TestUserErrorsKeepMessage(t *testing.T) {
	svc := &fakeSvc{err: fmt.Errorf("%w: %s", service.ErrAlreadyExists, "8.8.8.8")}
	_, err := newClient(t, svc).AddServer(ctx(t), &dnsv1.AddServerRequest{Adresses: "8.8.8.8"})
	if msg := status.Convert(err).Message(); !strings.Contains(msg, "8.8.8.8") {
		t.Fatalf("message %q should mention the address", msg)
	}
}
