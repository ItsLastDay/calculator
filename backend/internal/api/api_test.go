package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	contract "github.com/example/calculator/backend/api"
)

const calculationsPath = "/api/v1/calculations"

func newTestServer(t *testing.T, static http.Handler) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(logger, static)
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeExact decodes a JSON body and fails if it has any member other than
// wantKeys, so redundant fields cannot creep into responses unnoticed.
func decodeExact(t *testing.T, rec *httptest.ResponseRecorder, wantContentType string, wantKeys ...string) map[string]any {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != wantContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, wantContentType)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	slices.Sort(wantKeys)
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("body members = %v, want exactly %v", keys, wantKeys)
	}
	return body
}

func decodeResult(t *testing.T, rec *httptest.ResponseRecorder) float64 {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	result, ok := decodeExact(t, rec, "application/json", "result")["result"].(float64)
	if !ok {
		t.Fatal("result is not a number")
	}
	return result
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) problem {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, wantStatus, rec.Body)
	}
	body := decodeExact(t, rec, problemContentType, "code", "detail")
	p := problem{}
	p.Code, _ = body["code"].(string)
	p.Detail, _ = body["detail"].(string)
	if p.Code != wantCode {
		t.Errorf("code = %q, want %q", p.Code, wantCode)
	}
	if p.Detail == "" {
		t.Error("detail is empty")
	}
	return p
}

func TestCreateCalculation(t *testing.T) {
	h := newTestServer(t, nil)

	tests := []struct {
		name string
		body string
		want float64
	}{
		{"add", `{"operation":"add","operands":[2,3]}`, 5},
		{"subtract", `{"operation":"subtract","operands":[2,3]}`, -1},
		{"multiply", `{"operation":"multiply","operands":[2.5,4]}`, 10},
		{"divide", `{"operation":"divide","operands":[7,2]}`, 3.5},
		{"power", `{"operation":"power","operands":[2,8]}`, 256},
		{"sqrt", `{"operation":"sqrt","operands":[81]}`, 9},
		{"percentage", `{"operation":"percentage","operands":[20,50]}`, 10},
		{"exponent notation", `{"operation":"multiply","operands":[1e3,2E-3]}`, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, calculationsPath, tt.body)
			if got := decodeResult(t, rec); got != tt.want {
				t.Errorf("result = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateCalculationErrors(t *testing.T) {
	h := newTestServer(t, nil)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
		wantDetail string
	}{
		{"malformed JSON", `{"operation":`, http.StatusBadRequest, codeInvalidRequest, "Request body is not valid JSON: unexpected end of input."},
		{"syntax error", `{"operation" "add"}`, http.StatusBadRequest, codeInvalidRequest, "Request body is not valid JSON: syntax error at offset 14."},
		{"not an object", `[1,2]`, http.StatusBadRequest, codeInvalidRequest, "Request body must be a JSON object."},
		{"unknown field", `{"operation":"add","operands":[1,2],"extra":true}`, http.StatusBadRequest, codeInvalidRequest, `Request body contains unknown field "extra".`},
		{"trailing data", `{"operation":"add","operands":[1,2]} {}`, http.StatusBadRequest, codeInvalidRequest, "Request body must contain a single JSON object."},
		{"trailing brace", `{"operation":"add","operands":[1,2]}}`, http.StatusBadRequest, codeInvalidRequest, "Request body must contain a single JSON object."},
		{"missing operation", `{"operands":[1,2]}`, http.StatusBadRequest, codeInvalidRequest, `Field "operation" is required.`},
		{"missing operands", `{"operation":"add"}`, http.StatusBadRequest, codeInvalidRequest, `Field "operands" is required.`},
		{"null operands", `{"operation":"add","operands":null}`, http.StatusBadRequest, codeInvalidRequest, `Field "operands" is required.`},
		{"operation not a string", `{"operation":1,"operands":[1,2]}`, http.StatusBadRequest, codeInvalidRequest, `Field "operation" must be a string, got number.`},
		{"operands not an array", `{"operation":"add","operands":5}`, http.StatusBadRequest, codeInvalidRequest, `Field "operands" must be an array, got number.`},
		{"string operand", `{"operation":"add","operands":["1",2]}`, http.StatusBadRequest, codeInvalidRequest, `Field "operands" must be a finite number, got string.`},
		{"operand too large", `{"operation":"add","operands":[1e400,2]}`, http.StatusBadRequest, codeInvalidRequest, `Field "operands" must be a finite number, got number 1e400.`},
		{"null operand", `{"operation":"add","operands":[1,null]}`, http.StatusBadRequest, codeInvalidOperand, "Every operand must be a number, but operands[1] is null."},
		{"unsupported operation", `{"operation":"modulo","operands":[1,2]}`, http.StatusBadRequest, codeUnsupportedOperation, `Unsupported operation: "modulo".`},
		{"too few operands", `{"operation":"add","operands":[1]}`, http.StatusBadRequest, codeInvalidOperandCount, "Wrong number of operands: add expects 2, got 1."},
		{"empty operands", `{"operation":"add","operands":[]}`, http.StatusBadRequest, codeInvalidOperandCount, "Wrong number of operands: add expects 2, got 0."},
		{"division by zero", `{"operation":"divide","operands":[1,0]}`, http.StatusUnprocessableEntity, codeDivisionByZero, "Division by zero."},
		{"negative sqrt", `{"operation":"sqrt","operands":[-4]}`, http.StatusUnprocessableEntity, codeNegativeSqrt, "Square root of a negative number is not a real number."},
		{"negative percentage", `{"operation":"percentage","operands":[-10,50]}`, http.StatusUnprocessableEntity, codeNegativePercentage, "Percentage must not be negative."},
		{"undefined result", `{"operation":"power","operands":[-8,0.5]}`, http.StatusUnprocessableEntity, codeUndefinedResult, "Result is not a real number."},
		{"overflow", `{"operation":"multiply","operands":[1e308,10]}`, http.StatusUnprocessableEntity, codeOutOfRange, "Result is too large to represent."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPost, calculationsPath, tt.body)
			p := decodeProblem(t, rec, tt.wantStatus, tt.wantCode)
			if p.Detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", p.Detail, tt.wantDetail)
			}
		})
	}
}

func TestCreateCalculationEmptyBody(t *testing.T) {
	h := newTestServer(t, nil)

	req := httptest.NewRequest(http.MethodPost, calculationsPath, http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if p := decodeProblem(t, rec, http.StatusBadRequest, codeInvalidRequest); p.Detail != "Request body is empty." {
		t.Errorf("detail = %q", p.Detail)
	}
}

func TestCreateCalculationContentType(t *testing.T) {
	h := newTestServer(t, nil)
	body := `{"operation":"add","operands":[1,2]}`

	tests := []struct {
		contentType string
		accepted    bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/merge-patch+json", true},
		{"", false},
		{"text/plain", false},
		{"application/x-www-form-urlencoded", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, calculationsPath, strings.NewReader(body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if tt.accepted {
				decodeResult(t, rec)
				return
			}
			decodeProblem(t, rec, http.StatusUnsupportedMediaType, codeUnsupportedMediaType)
		})
	}
}

func TestCreateCalculationRejectsOversizedBody(t *testing.T) {
	h := newTestServer(t, nil)

	padding := strings.Repeat(" ", maxBodyBytes)
	rec := do(t, h, http.MethodPost, calculationsPath, padding+`{"operation":"add","operands":[1,2]}`)

	decodeProblem(t, rec, http.StatusRequestEntityTooLarge, codePayloadTooLarge)
}

func TestMethodNotAllowed(t *testing.T) {
	h := newTestServer(t, nil)

	tests := []struct {
		method, path, allow string
	}{
		{http.MethodGet, calculationsPath, "POST"},
		{http.MethodPut, calculationsPath, "POST"},
		{http.MethodDelete, calculationsPath, "POST"},
		{http.MethodPost, "/healthz", "GET, HEAD"},
		{http.MethodPut, "/api/v1/openapi.yaml", "GET, HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := do(t, h, tt.method, tt.path, "")
			decodeProblem(t, rec, http.StatusMethodNotAllowed, codeMethodNotAllowed)
			if got := rec.Header().Get("Allow"); got != tt.allow {
				t.Errorf("Allow = %q, want %q", got, tt.allow)
			}
		})
	}
}

func TestUnknownAPIRoute(t *testing.T) {
	// Even with a static handler mounted, unknown API paths must return a
	// Problem 404 rather than falling through to the frontend.
	static := http.FileServerFS(fstest.MapFS{"index.html": {Data: []byte("<html></html>")}})
	h := newTestServer(t, static)

	rec := do(t, h, http.MethodGet, "/api/v1/nope", "")
	decodeProblem(t, rec, http.StatusNotFound, codeNotFound)
}

func TestHealth(t *testing.T) {
	h := newTestServer(t, nil)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		rec := do(t, h, method, "/healthz", "")
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s /healthz = %d with %d body bytes, want 204 and no body", method, rec.Code, rec.Body.Len())
		}
	}
}

func TestOpenAPIEndpoint(t *testing.T) {
	h := newTestServer(t, nil)

	rec := do(t, h, http.MethodGet, "/api/v1/openapi.yaml", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/yaml" {
		t.Errorf("Content-Type = %q, want application/yaml", ct)
	}
	if !bytes.Equal(rec.Body.Bytes(), contract.OpenAPI) {
		t.Error("served document differs from backend/api/openapi.yaml")
	}
	// The contract documents every route the server exposes.
	for _, path := range []string{"/api/v1/calculations:", "/api/v1/openapi.yaml:", "/healthz:"} {
		if !bytes.Contains(contract.OpenAPI, []byte(path)) {
			t.Errorf("openapi.yaml does not document %s", strings.TrimSuffix(path, ":"))
		}
	}
}

// Every problem code the server can emit must be documented in the contract.
func TestOpenAPIDocumentsAllProblemCodes(t *testing.T) {
	spec := contract.OpenAPI
	codes := []string{
		codeInvalidRequest, codeUnsupportedMediaType, codePayloadTooLarge,
		codeUnsupportedOperation, codeInvalidOperandCount, codeInvalidOperand,
		codeDivisionByZero, codeNegativeSqrt, codeNegativePercentage, codeUndefinedResult, codeOutOfRange,
		codeNotFound, codeMethodNotAllowed, codeInternal,
	}
	for _, code := range codes {
		if !strings.Contains(string(spec), "- "+code+"\n") {
			t.Errorf("openapi.yaml does not list problem code %s", code)
		}
	}
}

func TestStaticFiles(t *testing.T) {
	static := http.FileServerFS(fstest.MapFS{
		"index.html":    {Data: []byte("<h1>calculator</h1>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	})
	h := newTestServer(t, static)

	tests := []struct {
		path, want string
	}{
		{"/", "<h1>calculator</h1>"},
		{"/assets/app.js", "console.log(1)"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, tt.path, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if got := rec.Body.String(); got != tt.want {
				t.Errorf("body = %q, want %q", got, tt.want)
			}
			if got := rec.Header().Get("Cache-Control"); got != "" {
				t.Errorf("Cache-Control = %q on a static file, want unset", got)
			}
		})
	}
}

func TestNoStaticHandler(t *testing.T) {
	h := newTestServer(t, nil)

	rec := do(t, h, http.MethodGet, "/", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when no static handler is configured", rec.Code)
	}
}
