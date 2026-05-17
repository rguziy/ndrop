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
)

func main() {
	if len(os.Args) <= 1 {
		printUsage()
		return
	}

	switch os.Args[1] {
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
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		printUsage()
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

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("ndropd listening on :%s  max_size=%dMB  ttl=%s  allow_any_api_key=%t  allowed_api_keys=%d",
			cfg.Port, cfg.MaxSizeMB, cfg.TTL, cfg.AllowAnyAPIKey, len(cfg.AllowedAPIKeys))
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

func printUsage() {
	fmt.Fprintf(os.Stdout, `Usage:
  ndropd start   # run the server in foreground
  ndropd stop    # stop the running server if started with start
  ndropd init    # create ~/.config/ndrop/ndropd.toml
  ndropd help    # show this help message
`)
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
