// Command server runs the calculator REST API and, optionally, serves the
// built frontend from STATIC_DIR.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/calculator/backend/internal/api"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// "healthcheck" lets the container probe itself; the distroless runtime
	// image has no shell or curl.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := healthcheck(os.Getenv); err != nil {
			logger.Error("healthcheck failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		return err
	}

	var static http.Handler
	if cfg.StaticDir != "" {
		static = http.FileServerFS(os.DirFS(cfg.StaticDir))
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewRouter(logger, static),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr, "static_dir", cfg.StaticDir)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
