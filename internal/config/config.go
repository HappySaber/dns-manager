package config

type Config struct {
	ListenAddress int `yaml:listen_addr`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	return cfg, nil
}
