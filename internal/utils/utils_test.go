package utils

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var res []string
	for _, e := range entries {
		res = append(res, e.Name())
	}
	return res
}

func TestWriteFileAtomic_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	if err := WriteFileAtomic(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "hello" {
		t.Fatalf("data = %q, err = %v", data, err)
	}
	if got := names(t, dir); !slices.Equal(got, []string{"f.txt"}) {
		t.Fatalf("unexpected files: %v", got)
	}
}

func TestWriteFileAtomic_Overwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	for _, content := range []string{"one", "two-longer", "3"} {
		if err := WriteFileAtomic(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if string(data) != content {
			t.Fatalf("data = %q, want %q", data, content)
		}
	}
	if got := names(t, dir); !slices.Equal(got, []string{"f.txt"}) {
		t.Fatalf("temp files left behind: %v", got)
	}
}

func TestWriteFileAtomic_Permissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	if err := WriteFileAtomic(path, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("perm = %v, want 0640", info.Mode().Perm())
	}
}

func TestWriteFileAtomic_EmptyData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.txt")
	if err := WriteFileAtomic(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != 0 {
		t.Fatalf("info = %v, err = %v", info, err)
	}
}

func TestWriteFileAtomic_MissingDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "f.txt")
	if err := WriteFileAtomic(path, []byte("x"), 0o644); err == nil {
		t.Fatal("expected error for missing directory")
	}
}

// Если rename не удался, временный файл должен быть удалён.
func TestWriteFileAtomic_CleansUpOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil { // rename файла поверх каталога невозможен
		t.Fatal(err)
	}
	if err := WriteFileAtomic(target, []byte("x"), 0o644); err == nil {
		t.Fatal("expected error")
	}
	if got := names(t, dir); !slices.Equal(got, []string{"target"}) {
		t.Fatalf("temp file left behind: %v", got)
	}
}
