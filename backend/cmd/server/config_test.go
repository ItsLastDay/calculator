package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := loadConfig(env(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := config{Addr: ":8080", ShutdownTimeout: 10 * time.Second}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	dir := t.TempDir()

	cfg, err := loadConfig(env(map[string]string{
		"ADDR":             "127.0.0.1:9000",
		"STATIC_DIR":       dir,
		"SHUTDOWN_TIMEOUT": "3s",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := config{Addr: "127.0.0.1:9000", StaticDir: dir, ShutdownTimeout: 3 * time.Second}
	if cfg != want {
		t.Errorf("config = %+v, want %+v", cfg, want)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		vars map[string]string
	}{
		{"invalid timeout", map[string]string{"SHUTDOWN_TIMEOUT": "soon"}},
		{"negative timeout", map[string]string{"SHUTDOWN_TIMEOUT": "-1s"}},
		{"missing static dir", map[string]string{"STATIC_DIR": filepath.Join(dir, "missing")}},
		{"static dir is a file", map[string]string{"STATIC_DIR": file}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := loadConfig(env(tt.vars)); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}
