package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

// ServerConfig holds all server-side configuration loaded from environment variables and ndropd.toml.
type ServerConfig struct {
	Port      string
	MaxSizeMB int64
	TTL       time.Duration
}

func LoadServer() ServerConfig {
	cfg := ServerConfig{
		Port:      "8080",
		MaxSizeMB: 10,
		TTL:       1 * time.Hour,
	}

	path := DefaultServerConfigPath()
	if _, err := os.Stat(path); err == nil {
		var fileCfg struct {
			Port      string  `toml:"port"`
			MaxSizeMB int64   `toml:"max_size_mb"`
			TTLHours  float64 `toml:"ttl_hours"`
		}
		if _, err := toml.DecodeFile(path, &fileCfg); err == nil {
			if fileCfg.Port != "" {
				cfg.Port = fileCfg.Port
			}
			if fileCfg.MaxSizeMB > 0 {
				cfg.MaxSizeMB = fileCfg.MaxSizeMB
			}
			if fileCfg.TTLHours > 0 {
				cfg.TTL = time.Duration(fileCfg.TTLHours * float64(time.Hour))
			}
		}
	}

	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}

	if v := os.Getenv("MAX_SIZE_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.MaxSizeMB = n
		}
	}

	if v := os.Getenv("TTL_HOURS"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			cfg.TTL = time.Duration(n * float64(time.Hour))
		}
	}

	return cfg
}

// DefaultServerConfigPath returns ~/.config/ndrop/ndropd.toml.
func DefaultServerConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "ndropd.toml"
	}
	return filepath.Join(home, ".config", "ndrop", "ndropd.toml")
}
