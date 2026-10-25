package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type middleware func(http.Handler) http.Handler

// chain applies middleware so that the first one listed is the outermost.
func chain(h http.Handler, mw ...middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

type ctxKey struct{}

const requestIDHeader = "X-Request-ID"

// validRequestID limits accepted client IDs so they are safe to log.
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// requestID propagates a caller-supplied X-Request-ID or generates one, and
// echoes it on the response so client and server logs can be correlated.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID.MatchString(id) {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error.
	return hex.EncodeToString(b)
}

// requestIDFrom returns the request ID stored by the requestID middleware.
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// statusRecorder captures the status code written by a handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// logRequests logs one structured line per request.
func logRequests(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.Info("request",
				"request_id", requestIDFrom(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration", time.Since(start),
			)
		})
	}
}

// recoverPanics turns a handler panic into a 500 Problem response instead of
// a dropped connection, and logs it with the request ID.
func recoverPanics(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				v := recover()
				if v == nil {
					return
				}
				// ErrAbortHandler is net/http's sanctioned way to abort a
				// response; let the server handle it as designed.
				if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(v)
				}
				logger.Error("panic serving request",
					"request_id", requestIDFrom(r.Context()),
					"panic", v,
				)
				writeProblem(w, http.StatusInternalServerError, codeInternal, "An unexpected error occurred.")
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// securityHeaders sets conservative defaults for every response. API
// responses are computed per request, so they must not be cached.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		// data: images are for the icons embedded in Swagger UI's stylesheet.
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
