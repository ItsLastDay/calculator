package main

import (
	"fmt"
	"os"
	"time"
)

// config is read from the environment, following twelve-factor practice.
type config struct {
	// Addr is the listen address (ADDR, default ":8080").
	Addr string
	// StaticDir, if set, is a directory of frontend files to serve (STATIC_DIR).
	StaticDir string
	// ShutdownTimeout bounds graceful shutdown (SHUTDOWN_TIMEOUT, default 10s).
	ShutdownTimeout time.Duration
}

// loadConfig takes getenv as a parameter so tests need not touch the real
// environment. It fails fast on invalid values instead of starting a server
// that cannot do its job.
func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{
		Addr:            ":8080",
		StaticDir:       getenv("STATIC_DIR"),
		ShutdownTimeout: 10 * time.Second,
	}

	if v := getenv("ADDR"); v != "" {
		cfg.Addr = v
	}

	if v := getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration such as 10s, got %q", v)
		}
		cfg.ShutdownTimeout = d
	}

	if cfg.StaticDir != "" {
		info, err := os.Stat(cfg.StaticDir)
		if err != nil {
			return config{}, fmt.Errorf("STATIC_DIR: %w", err)
		}
		if !info.IsDir() {
			return config{}, fmt.Errorf("STATIC_DIR %q is not a directory", cfg.StaticDir)
		}
	}

	return cfg, nil
}
