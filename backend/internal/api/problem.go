package api

import "net/http"

// Problem codes: a stable, machine-readable reason for every error. They are
// part of the API contract, so clients branch on them, never on detail text.
const (
	codeInvalidRequest       = "INVALID_REQUEST"
	codeUnsupportedMediaType = "UNSUPPORTED_MEDIA_TYPE"
	codePayloadTooLarge      = "PAYLOAD_TOO_LARGE"
	codeUnsupportedOperation = "UNSUPPORTED_OPERATION"
	codeInvalidOperandCount  = "INVALID_OPERAND_COUNT"
	codeInvalidOperand       = "INVALID_OPERAND"
	codeDivisionByZero       = "DIVISION_BY_ZERO"
	codeNegativeSqrt         = "NEGATIVE_SQUARE_ROOT"
	codeNegativePercentage   = "NEGATIVE_PERCENTAGE"
	codeUndefinedResult      = "UNDEFINED_RESULT"
	codeOutOfRange           = "RESULT_OUT_OF_RANGE"
	codeNotFound             = "NOT_FOUND"
	codeMethodNotAllowed     = "METHOD_NOT_ALLOWED"
	codeInternal             = "INTERNAL_ERROR"
)

const problemContentType = "application/problem+json"

// problem is an error response body, based on RFC 9457 Problem Details and
// trimmed to what clients need.
type problem struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	writeBody(w, problemContentType, status, problem{Code: code, Detail: detail})
}
