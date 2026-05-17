package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rguziy/ndrop/internal/config"
)

func TestLoadServerAPIKeyConfigFromFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PORT", "")
	t.Setenv("MAX_SIZE_MB", "")
	t.Setenv("TTL_HOURS", "")
	t.Setenv("ALLOW_ANY_API_KEY", "")
	t.Setenv("ALLOWED_API_KEYS", "")

	cfgPath := config.DefaultServerConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "port = \"9090\"\nmax_size_mb = 20\nttl_hours = 2\nallow_any_api_key = false\nallowed_api_keys = [\" key-a \", \"key-b\", \"key-a\"]\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.LoadServer()

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.MaxSizeMB != 20 {
		t.Fatalf("MaxSizeMB = %d, want 20", cfg.MaxSizeMB)
	}
	if cfg.TTL != 2*time.Hour {
		t.Fatalf("TTL = %s, want 2h", cfg.TTL)
	}
	if cfg.AllowAnyAPIKey {
		t.Fatal("AllowAnyAPIKey = true, want false")
	}
	want := []string{"key-a", "key-b"}
	if len(cfg.AllowedAPIKeys) != len(want) {
		t.Fatalf("AllowedAPIKeys = %#v, want %#v", cfg.AllowedAPIKeys, want)
	}
	for i := range want {
		if cfg.AllowedAPIKeys[i] != want[i] {
			t.Fatalf("AllowedAPIKeys = %#v, want %#v", cfg.AllowedAPIKeys, want)
		}
	}
}

func TestLoadServerAPIKeyConfigFromEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("ALLOW_ANY_API_KEY", "false")
	t.Setenv("ALLOWED_API_KEYS", "key-a, key-b,,key-a")

	cfg := config.LoadServer()

	if cfg.AllowAnyAPIKey {
		t.Fatal("AllowAnyAPIKey = true, want false")
	}
	want := []string{"key-a", "key-b"}
	if len(cfg.AllowedAPIKeys) != len(want) {
		t.Fatalf("AllowedAPIKeys = %#v, want %#v", cfg.AllowedAPIKeys, want)
	}
	for i := range want {
		if cfg.AllowedAPIKeys[i] != want[i] {
			t.Fatalf("AllowedAPIKeys = %#v, want %#v", cfg.AllowedAPIKeys, want)
		}
	}
}
