// Package api is the HTTP boundary of the calculator: it is the only package
// that knows about HTTP and JSON, and it implements the contract documented
// in backend/api/openapi.yaml.
//
//	POST /api/v1/calculations  evaluate a calculation
//	GET  /api/v1/openapi.yaml  the contract itself (rendered by the /docs/ page)
//	GET  /healthz              liveness probe
//
// Errors use RFC 9457 Problem Details (application/problem+json).
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"reflect"
	"strings"

	contract "github.com/example/calculator/backend/api"
	"github.com/example/calculator/backend/internal/calculator"
)

// maxBodyBytes bounds request bodies; a calculation request is tiny.
const maxBodyBytes = 1 << 16

// calculationRequest is the body of POST /api/v1/calculations.
//
// Operands are pointers so that a JSON null can be told apart from 0.
type calculationRequest struct {
	Operation string     `json:"operation"`
	Operands  []*float64 `json:"operands"`
}

// calculationResponse is the success body of POST /api/v1/calculations. The
// client already knows what it asked for, so only the result is returned.
type calculationResponse struct {
	Result float64 `json:"result"`
}

// domainErrors maps calculator errors to HTTP statuses and problem codes.
// Malformed input is a 400; well-formed input that is mathematically
// invalid is a 422.
var domainErrors = []struct {
	err    error
	status int
	code   string
}{
	{calculator.ErrUnsupportedOperation, http.StatusBadRequest, codeUnsupportedOperation},
	{calculator.ErrOperandCount, http.StatusBadRequest, codeInvalidOperandCount},
	{calculator.ErrInvalidOperand, http.StatusBadRequest, codeInvalidOperand},
	{calculator.ErrDivisionByZero, http.StatusUnprocessableEntity, codeDivisionByZero},
	{calculator.ErrNegativeSqrt, http.StatusUnprocessableEntity, codeNegativeSqrt},
	{calculator.ErrNegativePercentage, http.StatusUnprocessableEntity, codeNegativePercentage},
	{calculator.ErrUndefinedResult, http.StatusUnprocessableEntity, codeUndefinedResult},
	{calculator.ErrOutOfRange, http.StatusUnprocessableEntity, codeOutOfRange},
}

// NewRouter wires the routes and middleware. When static is non-nil it
// serves every path outside /api/ and /healthz, which is how the production
// container ships the frontend and backend as a single process.
func NewRouter(logger *slog.Logger, static http.Handler) http.Handler {
	mux := http.NewServeMux()

	route(mux, http.MethodPost, "/api/v1/calculations", createCalculation)
	route(mux, http.MethodGet, "/api/v1/openapi.yaml", openAPI)
	route(mux, http.MethodGet, "/healthz", health)
	mux.HandleFunc("/api/", notFound)
	if static != nil {
		mux.Handle("/", static)
	}

	return chain(mux,
		requestID,
		logRequests(logger),
		recoverPanics(logger),
		securityHeaders,
	)
}

// route registers h for method on path, and a Problem Details 405 for every
// other method on the same path (the default ServeMux 405 is plain text).
func route(mux *http.ServeMux, method, path string, h http.HandlerFunc) {
	allow := method
	if method == http.MethodGet {
		allow = "GET, HEAD"
	}
	mux.HandleFunc(method+" "+path, h)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", allow)
		writeProblem(w, http.StatusMethodNotAllowed, codeMethodNotAllowed,
			fmt.Sprintf("Method %s is not allowed on this resource; allowed: %s.", r.Method, allow))
	})
}

func createCalculation(w http.ResponseWriter, r *http.Request) {
	if !isJSON(r.Header.Get("Content-Type")) {
		writeProblem(w, http.StatusUnsupportedMediaType, codeUnsupportedMediaType,
			"Content-Type must be application/json.")
		return
	}

	req, err := decodeCalculationRequest(w, r)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeProblem(w, http.StatusRequestEntityTooLarge, codePayloadTooLarge,
				fmt.Sprintf("Request body must not exceed %d bytes.", maxBodyBytes))
			return
		}
		writeProblem(w, http.StatusBadRequest, codeInvalidRequest, sentence(err.Error()))
		return
	}

	operands, err := derefOperands(req.Operands)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, codeInvalidOperand, sentence(err.Error()))
		return
	}

	result, err := calculator.Calculate(calculator.Operation(req.Operation), operands)
	if err != nil {
		writeCalculationError(w, err)
		return
	}

	// 200 rather than 201: the calculation is evaluated, not stored, so
	// there is no new resource to point a Location header at.
	writeJSON(w, http.StatusOK, calculationResponse{Result: result})
}

// openAPI publishes the contract (application/yaml per RFC 9512).
func openAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contract.OpenAPI)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, http.StatusNotFound, codeNotFound, fmt.Sprintf("No resource at %s.", r.URL.Path))
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && (mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"))
}

// decodeCalculationRequest strictly decodes the body: unknown fields,
// trailing data and a missing operation or operand list are all rejected.
func decodeCalculationRequest(w http.ResponseWriter, r *http.Request) (calculationRequest, error) {
	var req calculationRequest

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, describeDecodeError(err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, errors.New("request body must contain a single JSON object")
	}

	if req.Operation == "" {
		return req, errors.New(`field "operation" is required`)
	}
	if req.Operands == nil {
		return req, errors.New(`field "operands" is required`)
	}
	return req, nil
}

// describeDecodeError rewrites encoding/json errors, which mention Go types,
// into messages that make sense to an API client.
func describeDecodeError(err error) error {
	var (
		tooLarge  *http.MaxBytesError
		syntaxErr *json.SyntaxError
		typeErr   *json.UnmarshalTypeError
	)
	switch {
	case errors.As(err, &tooLarge):
		return err
	case errors.Is(err, io.EOF):
		return errors.New("request body is empty")
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("request body is not valid JSON: unexpected end of input")
	case errors.As(err, &syntaxErr):
		return fmt.Errorf("request body is not valid JSON: syntax error at offset %d", syntaxErr.Offset)
	case errors.As(err, &typeErr):
		if typeErr.Field == "" {
			return errors.New("request body must be a JSON object")
		}
		return fmt.Errorf("field %q must be %s, got %s", typeErr.Field, describeKind(typeErr.Type.Kind()), typeErr.Value)
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		return fmt.Errorf("request body contains unknown field %s", strings.TrimPrefix(err.Error(), "json: unknown field "))
	default:
		return errors.New("request body is not valid JSON")
	}
}

func describeKind(k reflect.Kind) string {
	switch k {
	case reflect.Float64:
		return "a finite number"
	case reflect.String:
		return "a string"
	case reflect.Slice:
		return "an array"
	default:
		return "a valid value"
	}
}

func derefOperands(in []*float64) ([]float64, error) {
	out := make([]float64, len(in))
	for i, v := range in {
		if v == nil {
			return nil, fmt.Errorf("every operand must be a number, but operands[%d] is null", i)
		}
		out[i] = *v
	}
	return out, nil
}

func writeCalculationError(w http.ResponseWriter, err error) {
	for _, m := range domainErrors {
		if errors.Is(err, m.err) {
			writeProblem(w, m.status, m.code, sentence(err.Error()))
			return
		}
	}
	writeProblem(w, http.StatusInternalServerError, codeInternal, "An unexpected error occurred.")
}

// sentence capitalises the first letter and adds a full stop, turning a Go
// error string into user-facing text.
func sentence(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	writeBody(w, "application/json", status, body)
}

func writeBody(w http.ResponseWriter, contentType string, status int, body any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	// The status line is already sent, so an encoding error can only be
	// a broken connection; there is nothing useful left to do with it.
	_ = json.NewEncoder(w).Encode(body)
}
