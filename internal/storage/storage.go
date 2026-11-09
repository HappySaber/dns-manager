package storage

import (
	"dnsmanager/internal/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	fileName = "dns-servers.json"
)

type Store struct {
	path string
	now  func() time.Time
}

type snapshot struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Servers   []string  `json:"servers"`
}

// New creates directory, if cant than returns  error
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}
	return &Store{path: filepath.Join(dir, fileName), now: time.Now}, nil
}

func (s *Store) Save(servers []string) error {
	snap := snapshot{UpdatedAt: s.now().UTC(), Servers: servers}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup: %w", err)
	}
	if err := utils.WriteFileAtomic(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	return nil
}
