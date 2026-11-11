package storage

import (
	"dnsmanager/internal/model"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "backup")
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

func TestNew_CreatesNestedDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b", "backup")
	if _, err := New(dir); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("dir not created: %v", err)
	}
}

func TestNew_FailsWhenPathIsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path); err == nil {
		t.Fatal("expected error when backup path is a file")
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	s, _ := newStore(t)
	want := []string{"8.8.8.8", "1.1.1.1", "2001:db8::1"}
	if err := s.Save(model.State{Servers: want}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Servers, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSaveLoad_EmptyList(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Save(model.State{}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("empty backup must load without error, got %v", err)
	}
	if len(got.Servers) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestLoad_NotFound(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	s, _ := newStore(t)
	if err := os.WriteFile(s.path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := s.Load()
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want a parse error different from ErrNotFound", err)
	}
}

func TestLoad_UnsupportedVersion(t *testing.T) {
	for _, content := range []string{
		`{"version":99,"servers":["1.1.1.1"]}`,
		`{"servers":["1.1.1.1"]}`, // версии нет
	} {
		s, _ := newStore(t)
		if err := os.WriteFile(s.path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Load(); err == nil || errors.Is(err, ErrNotFound) {
			t.Fatalf("content %s: err = %v, want version error", content, err)
		}
	}
}

func TestSave_OverwritesAndLeavesNoTempFiles(t *testing.T) {
	s, dir := newStore(t)
	if err := s.Save(model.State{Servers: []string{"1.1.1.1", "2.2.2.2"}}); err != nil {
		t.Fatal(err)
	}

	if err := s.Save(model.State{Servers: []string{"3.3.3.3"}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil || !slices.Equal(got.Servers, []string{"3.3.3.3"}) {
		t.Fatalf("got %v, err %v", got, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != fileName {
		t.Fatalf("unexpected files: %v", entries)
	}
}

func TestSave_WritesVersionAndTimestamp(t *testing.T) {
	s, _ := newStore(t)
	fixed := time.Date(2026, 11, 10, 12, 0, 0, 0, time.FixedZone("MSK", 3*3600))
	s.now = func() time.Time { return fixed }

	if err := s.Save(model.State{Servers: []string{"8.8.8.8"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Version != schemaVersion {
		t.Fatalf("version = %d, want %d", snap.Version, schemaVersion)
	}
	if !snap.UpdatedAt.Equal(fixed) || snap.UpdatedAt.Location() != time.UTC {
		t.Fatalf("updated_at = %v, want %v in UTC", snap.UpdatedAt, fixed.UTC())
	}
}

func TestSave_FailsWhenDirRemoved(t *testing.T) {
	s, dir := newStore(t)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(model.State{Servers: []string{"8.8.8.8"}}); err == nil {
		t.Fatal("expected error when backup directory is gone")
	}
}
