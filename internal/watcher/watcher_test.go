package watcher

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

const debounce = 50 * time.Millisecond

func start(t *testing.T, path string) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	w := New(path, debounce, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx, func(context.Context) { calls.Add(1) }) }()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run returned %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Run did not stop after cancel")
		}
	})
	time.Sleep(100 * time.Millisecond)
	return &calls
}

func waitCalls(t *testing.T, calls *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if calls.Load() >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("calls = %d, want at least %d", calls.Load(), want)
}

func replaceViaRename(t *testing.T, path, content string) {
	t.Helper()
	tmp := path + ".new"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
}

func newFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "resolv.conf")
	if err := os.WriteFile(path, []byte("nameserver 1.1.1.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWatcher_DirectWrite(t *testing.T) {
	path := newFile(t)
	calls := start(t, path)

	if err := os.WriteFile(path, []byte("nameserver 2.2.2.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitCalls(t, calls, 1)
}

func TestWatcher_RenameTwice(t *testing.T) {
	path := newFile(t)
	calls := start(t, path)

	replaceViaRename(t, path, "nameserver 2.2.2.2\n")
	waitCalls(t, calls, 1)

	replaceViaRename(t, path, "nameserver 3.3.3.3\n")
	waitCalls(t, calls, 2)
}

func TestWatcher_FileCreatedAfterStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resolv.conf")
	calls := start(t, path)

	if err := os.WriteFile(path, []byte("nameserver 1.1.1.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitCalls(t, calls, 1)
}

func TestWatcher_DebounceMergesBurst(t *testing.T) {
	path := newFile(t)
	calls := start(t, path)

	for i := 0; i < 10; i++ {
		if err := os.WriteFile(path, []byte("nameserver 2.2.2.2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	waitCalls(t, calls, 1)
	time.Sleep(4 * debounce)
	if n := calls.Load(); n != 1 {
		t.Fatalf("calls = %d, want 1 (burst must be merged)", n)
	}
}

func TestWatcher_IgnoresOtherFiles(t *testing.T) {
	path := newFile(t)
	calls := start(t, path)

	other := filepath.Join(filepath.Dir(path), "other.txt")
	if err := os.WriteFile(other, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * debounce)
	if n := calls.Load(); n != 0 {
		t.Fatalf("calls = %d, want 0", n)
	}
}

func TestWatcher_StopsOnCancel(t *testing.T) {
	path := newFile(t)
	w := New(path, debounce, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx, func(context.Context) {}) }()
	time.Sleep(50 * time.Millisecond)

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not stop")
	}
}

func TestWatcher_MissingDirFails(t *testing.T) {
	w := New(filepath.Join(t.TempDir(), "nope", "resolv.conf"), debounce,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := w.Run(context.Background(), func(context.Context) {}); err == nil {
		t.Fatal("expected error for missing directory")
	}
}
