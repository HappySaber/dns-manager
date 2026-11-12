package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr         string        `yaml:"listen_addr"`
	ResolvConfPath     string        `yaml:"resolv_conf_path"`
	BackupDir          string        `yaml:"backup_dir"`
	ShutdownTimeout    time.Duration `yaml:"shutdown_timeout"`
	WatchDebounce      time.Duration `yaml:"watch_debounce"`
	NodeID             string        `yaml:"node_id"`
	PeerAddr           string        `yaml:"peer_addr"`
	SyncInterval       time.Duration `yaml:"sync_interval"`
	ReplicationTimeout time.Duration `yaml:"replication_timeout"`
}

func Default() Config {
	return Config{
		ListenAddr:         ":50051",
		ShutdownTimeout:    10 * time.Second,
		WatchDebounce:      100 * time.Millisecond,
		SyncInterval:       5 * time.Second,
		ReplicationTimeout: 2 * time.Second,
	}
}

func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var errs []error
	if c.ListenAddr == "" {
		errs = append(errs, errors.New("listen_addr is required"))
	}
	if c.ResolvConfPath == "" {
		errs = append(errs, errors.New("resolv_conf_path is required"))
	}
	if c.BackupDir == "" {
		errs = append(errs, errors.New("backup_dir is required"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("shutdown_timeout must be positive"))
	}
	if c.SyncInterval <= 0 {
		errs = append(errs, errors.New("sync_interval must be positive"))
	}
	if c.ReplicationTimeout <= 0 {
		errs = append(errs, errors.New("replication_timeout must be positive"))
	}
	if c.PeerAddr != "" && c.NodeID == "" {
		errs = append(errs, errors.New("node_id is required when peer_addr is set"))
	}
	return errors.Join(errs...)
}
