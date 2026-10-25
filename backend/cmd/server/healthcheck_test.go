package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthURL(t *testing.T) {
	tests := []struct {
		addr, want string
	}{
		{":8080", "http://127.0.0.1:8080/healthz"},
		{"0.0.0.0:9000", "http://127.0.0.1:9000/healthz"},
		{"[::]:9000", "http://127.0.0.1:9000/healthz"},
		{"localhost:8080", "http://localhost:8080/healthz"},
		{"[::1]:8080", "http://[::1]:8080/healthz"},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got, err := healthURL(tt.addr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("healthURL(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}

	if _, err := healthURL("no-port"); err == nil {
		t.Error("healthURL(no-port) succeeded, want error")
	}
}

func TestHealthcheck(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer healthy.Close()

	unhealthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()

	addr := func(s *httptest.Server) string { return strings.TrimPrefix(s.URL, "http://") }

	if err := healthcheck(env(map[string]string{"ADDR": addr(healthy)})); err != nil {
		t.Errorf("healthy server: unexpected error: %v", err)
	}
	if err := healthcheck(env(map[string]string{"ADDR": addr(unhealthy)})); err == nil {
		t.Error("unhealthy server: expected an error")
	}
	if err := healthcheck(env(map[string]string{"ADDR": "bad"})); err == nil {
		t.Error("invalid ADDR: expected an error")
	}
}
