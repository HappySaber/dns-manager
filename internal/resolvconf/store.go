package resolvconf

import (
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
		return &File{}, err
	}

	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	}
	return Parse(data), nil
}

func (s *Store) Write(f *File) error {
	return writeFileAtomic(s.path, f.Render(), 0o644)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".resolv.conf.tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	if err = tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return syncDir(dir)
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
