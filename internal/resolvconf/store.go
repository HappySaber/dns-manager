package resolvconf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
