package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rguziy/ndrop/internal/config"
	"github.com/rguziy/ndrop/internal/server"
	"github.com/rguziy/ndrop/internal/version"
)

func main() {
	if len(os.Args) <= 1 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "version", "-v", "--version":
		printVersion()
	case "init":
		initFlags := flag.NewFlagSet("init", flag.ExitOnError)
		force := initFlags.Bool("force", false, "overwrite existing server config")
		initFlags.Parse(os.Args[2:])
		if err := initServerConfig(*force); err != nil {
			log.Fatalf("init config: %v", err)
		}
	case "start":
		runServer()
	case "stop":
		if err := stopServer(); err != nil {
			log.Fatalf("stop server: %v", err)
		}
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		printHelp()
		os.Exit(2)
	}
}

func runServer() {
	cfg := config.LoadServer()

	pidPath := serverPIDPath()
	if err := writePIDFile(pidPath); err != nil {
		log.Fatalf("write pid file: %v", err)
	}
	defer removePIDFile(pidPath)

	maxBytes := cfg.MaxSizeMB << 20 // MB → bytes

	store := server.NewStore(cfg.TTL)
	handler := server.NewHandler(store, maxBytes, server.AuthConfig{
		AllowAnyAPIKey: cfg.AllowAnyAPIKey,
		AllowedAPIKeys: cfg.AllowedAPIKeys,
	})

	versionMiddleware := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Server-Version", version.Version)
		handler.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           versionMiddleware,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("ndropd %s listening on :%s  max_size=%dMB  ttl=%s  allow_any_api_key=%t  allowed_api_keys=%d",
			version.Version, cfg.Port, cfg.MaxSizeMB, cfg.TTL, cfg.AllowAnyAPIKey, len(cfg.AllowedAPIKeys))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}

	log.Println("stopped")
}

func printVersion() {
	fmt.Fprintf(os.Stdout, "ndropd %s\n", version.Version)
}

func printHelp() {
	fmt.Fprintf(os.Stdout, `ndropd %s

Self-hosted ndrop HTTP server for encrypted text, file, and folder drops.

The server stores encrypted payloads in memory, keyed by a bucket derived
from the client API key. Use HTTPS or a reverse proxy when exposing ndropd
over a network.

Usage:
  ndropd start   # run the server in foreground
  ndropd stop    # stop the running server if started with start
  ndropd init    # create ~/.config/ndrop/ndropd.toml
  ndropd version # show version
  ndropd help    # show this help message

Config file:
  %s

Environment overrides:
  PORT
  MAX_SIZE_MB
  TTL_HOURS
  ALLOW_ANY_API_KEY
  ALLOWED_API_KEYS

Example:
  ALLOW_ANY_API_KEY=false ALLOWED_API_KEYS=laptop-key,phone-key ndropd start

For long-running Linux deployments, see deploy/systemd/ndropd.service.
`, version.Version, config.DefaultServerConfigPath())
}

func stopServer() error {
	pidPath := serverPIDPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return fmt.Errorf("read pid file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("parse pid: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal process: %w", err)
	}

	return os.Remove(pidPath)
}

func serverPIDPath() string {
	cfgPath := config.DefaultServerConfigPath()
	return filepath.Join(filepath.Dir(cfgPath), "ndropd.pid")
}

func writePIDFile(path string) error {
	if data, err := os.ReadFile(path); err == nil {
		pid, perr := strconv.Atoi(strings.TrimSpace(string(data)))
		if perr == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				if err := proc.Signal(syscall.Signal(0)); err == nil {
					return fmt.Errorf("server already running with pid %d", pid)
				}
			}
		}
		_ = os.Remove(path)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644)
}

func removePIDFile(path string) {
	_ = os.Remove(path)
}

func initServerConfig(force bool) error {
	cfgPath := config.DefaultServerConfigPath()
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if _, err := os.Stat(cfgPath); err == nil && !force {
		fmt.Fprintf(os.Stderr, "skipping %s (exists)\n", cfgPath)
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat config: %w", err)
	}

	content := "port = \"8080\"\nmax_size_mb = 10\nttl_hours = 1\nallow_any_api_key = true\nallowed_api_keys = []\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "wrote %s\n", cfgPath)
	return nil
}
