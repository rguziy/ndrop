package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

// ClientConfig holds all client-side configuration.
type ClientConfig struct {
	Server         ServerSection `toml:"server"`
	Pull           PullSection   `toml:"pull"`
	TimeoutSeconds int           `toml:"timeout_seconds"`
}

type ServerSection struct {
	URL    string `toml:"url"`
	APIKey string `toml:"api_key"`
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
// flagURL and flagAPIKey are non-empty only when explicitly set via CLI flag.
func LoadClient(path, flagURL, flagAPIKey string, flagTimeout int) (ClientConfig, error) {
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
	if v := os.Getenv("NDROP_API_KEY"); v != "" {
		cfg.Server.APIKey = v
	}

	if v := os.Getenv("NDROP_TIMEOUT_SECONDS"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			cfg.TimeoutSeconds = secs
		}
	}

	// CLI flag overrides (highest priority).
	if flagURL != "" {
		cfg.Server.URL = flagURL
	}
	if flagAPIKey != "" {
		cfg.Server.APIKey = flagAPIKey
	}
	if flagTimeout > 0 {
		cfg.TimeoutSeconds = flagTimeout
	}

	return cfg, nil
}

// Validate returns an error if required fields are missing.
func (c ClientConfig) Validate() error {
	if c.Server.URL == "" {
		return errors.New("server URL is required (set in config or --server flag or NDROP_URL env)")
	}
	if c.Server.APIKey == "" {
		return errors.New("API key is required (set in config or --api-key flag or NDROP_API_KEY env)")
	}
	return nil
}
