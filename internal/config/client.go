package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ClientConfig holds all client-side configuration.
type ClientConfig struct {
	Server ServerSection `toml:"server"`
	Pull   PullSection   `toml:"pull"`
}

type ServerSection struct {
	URL   string `toml:"url"`
	Token string `toml:"token"`
}

type PullSection struct {
	DefaultSaveDir string `toml:"default_save_dir"`
}

// DefaultClientConfig returns a config with sensible defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Pull: PullSection{
			DefaultSaveDir: "~/Downloads",
		},
	}
}

// DefaultConfigPath returns ~/.config/ndrop/ndrop.toml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "ndrop.toml"
	}
	return filepath.Join(home, ".config", "ndrop", "ndrop.toml")
}

// LoadClient loads config from path, applies env overrides, then flag overrides.
// flagURL and flagToken are non-empty only when explicitly set via CLI flag.
func LoadClient(path, flagURL, flagToken string) (ClientConfig, error) {
	cfg := DefaultClientConfig()

	// Load from file if it exists.
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return cfg, err
		}
	}

	// Env overrides.
	if v := os.Getenv("NDROP_URL"); v != "" {
		cfg.Server.URL = v
	}
	if v := os.Getenv("NDROP_TOKEN"); v != "" {
		cfg.Server.Token = v
	}

	// CLI flag overrides (highest priority).
	if flagURL != "" {
		cfg.Server.URL = flagURL
	}
	if flagToken != "" {
		cfg.Server.Token = flagToken
	}

	return cfg, nil
}

// Validate returns an error if required fields are missing.
func (c ClientConfig) Validate() error {
	if c.Server.URL == "" {
		return errors.New("server URL is required (set in config or --server flag or NDROP_URL env)")
	}
	if c.Server.Token == "" {
		return errors.New("token is required (set in config or --token flag or NDROP_TOKEN env)")
	}
	return nil
}
