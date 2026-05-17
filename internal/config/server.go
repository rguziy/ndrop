package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// ServerConfig holds all server-side configuration loaded from environment variables and ndropd.toml.
type ServerConfig struct {
	Port           string
	MaxSizeMB      int64
	TTL            time.Duration
	AllowAnyAPIKey bool
	AllowedAPIKeys []string
}

func LoadServer() ServerConfig {
	cfg := ServerConfig{
		Port:           "8080",
		MaxSizeMB:      10,
		TTL:            1 * time.Hour,
		AllowAnyAPIKey: true,
	}

	path := DefaultServerConfigPath()
	if _, err := os.Stat(path); err == nil {
		var fileCfg struct {
			Port           string   `toml:"port"`
			MaxSizeMB      int64    `toml:"max_size_mb"`
			TTLHours       float64  `toml:"ttl_hours"`
			AllowAnyAPIKey *bool    `toml:"allow_any_api_key"`
			AllowedAPIKeys []string `toml:"allowed_api_keys"`
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
			if fileCfg.AllowAnyAPIKey != nil {
				cfg.AllowAnyAPIKey = *fileCfg.AllowAnyAPIKey
			}
			if len(fileCfg.AllowedAPIKeys) > 0 {
				cfg.AllowedAPIKeys = normalizeAPIKeys(fileCfg.AllowedAPIKeys)
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

	if v := os.Getenv("ALLOW_ANY_API_KEY"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.AllowAnyAPIKey = b
		}
	}

	if v := os.Getenv("ALLOWED_API_KEYS"); v != "" {
		cfg.AllowedAPIKeys = splitAPIKeys(v)
	}

	return cfg
}

func splitAPIKeys(value string) []string {
	return normalizeAPIKeys(strings.Split(value, ","))
}

func normalizeAPIKeys(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	keys := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

// DefaultServerConfigPath returns ~/.config/ndrop/ndropd.toml.
func DefaultServerConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "ndropd.toml"
	}
	return filepath.Join(home, ".config", "ndrop", "ndropd.toml")
}
