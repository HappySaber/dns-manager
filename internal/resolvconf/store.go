package resolvconf

import (
	"dnsmanager/internal/utils"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{
		path: path,
	}
}

// Read returns the current content of resolv.conf. A missing file is not an error:
// it returns an empty File, because this is the normal state after a system disk failure.
func (s *Store) Read() (*File, error) {
	data, err := os.ReadFile(s.path)

	if errors.Is(err, fs.ErrNotExist) {
		return &File{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	}
	return Parse(data), nil
}

// Write atomically replaces resolv.conf with the content of f.
// It creates the parent directory if needed (for example on a freshly replaced disk).

func (s *Store) Write(f *File) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", s.path, err)
	}
	return utils.WriteFileAtomic(s.path, f.Render(), 0o644)
}
