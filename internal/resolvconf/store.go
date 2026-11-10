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

func (s *Store) Write(f *File) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", s.path, err)
	}
	return utils.WriteFileAtomic(s.path, f.Render(), 0o644)
}
