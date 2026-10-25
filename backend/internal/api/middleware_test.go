package api

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDIsGenerated(t *testing.T) {
	h := newTestServer(t, nil)

	first := do(t, h, http.MethodGet, "/healthz", "").Header().Get(requestIDHeader)
	second := do(t, h, http.MethodGet, "/healthz", "").Header().Get(requestIDHeader)

	if len(first) != 16 {
		t.Errorf("generated request ID %q, want 16 hex characters", first)
	}
	if first == second {
		t.Errorf("two requests got the same ID %q", first)
	}
}

func TestRequestIDIsPropagated(t *testing.T) {
	h := newTestServer(t, nil)

	tests := []struct {
		name, incoming string
		keep           bool
	}{
		{"valid ID is kept", "abc-123_X.y", true},
		{"ID with spaces is replaced", "abc 123", false},
		{"ID with newline is replaced", "abc\n123", false},
		{"overlong ID is replaced", strings.Repeat("a", 65), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.Header.Set(requestIDHeader, tt.incoming)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			got := rec.Header().Get(requestIDHeader)
			if (got == tt.incoming) != tt.keep {
				t.Errorf("response ID = %q for incoming %q, keep = %v", got, tt.incoming, tt.keep)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := newTestServer(t, nil)

	rec := do(t, h, http.MethodPost, calculationsPath, `{"operation":"add","operands":[1,2]}`)

	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "no-referrer",
		"Cache-Control":          "no-store",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("Content-Security-Policy = %q, want default-src 'self'", csp)
	}
}

func TestRecoverPanics(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := chain(panicking, requestID, recoverPanics(logger))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil))

	decodeProblem(t, rec, http.StatusInternalServerError, codeInternal)
	if !strings.Contains(logs.String(), "boom") {
		t.Errorf("panic was not logged; logs: %s", logs.String())
	}
}

func TestRecoverPanicsRepanicsOnAbort(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	aborting := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
	h := recoverPanics(logger)(aborting)

	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, http.ErrAbortHandler) {
			t.Errorf("recovered %v, want http.ErrAbortHandler", err)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	t.Error("expected ServeHTTP to panic")
}

func TestLogRequests(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := NewRouter(logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(requestIDHeader, "trace-1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	for _, want := range []string{"request_id=trace-1", "method=GET", "path=/healthz", "status=204"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log line missing %q; got: %s", want, logs.String())
		}
	}
}
