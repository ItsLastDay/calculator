package main

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// healthcheck calls the local /healthz endpoint and reports whether it
// answered 204 No Content.
func healthcheck(getenv func(string) string) error {
	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}
	url, err := healthURL(cfg.Addr)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // URL is built from our own listen address.
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return nil
}

// healthURL maps a listen address such as ":8080" or "0.0.0.0:8080" to a URL
// reachable from inside the same container.
func healthURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid ADDR %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/healthz", nil
}
