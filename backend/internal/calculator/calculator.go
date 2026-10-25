// Package calculator implements the arithmetic operations offered by the
// service. It is pure domain logic with no knowledge of HTTP or JSON, which
// keeps it trivial to test and reuse.
package calculator

import (
	"errors"
	"fmt"
	"math"
)

// Operation identifies a supported arithmetic operation.
type Operation string

// Supported operations.
const (
	Add        Operation = "add"
	Subtract   Operation = "subtract"
	Multiply   Operation = "multiply"
	Divide     Operation = "divide"
	Power      Operation = "power"
	Sqrt       Operation = "sqrt"
	Percentage Operation = "percentage"
)

// Errors returned by Calculate. Callers should match them with errors.Is,
// because they may be wrapped with additional context.
var (
	ErrUnsupportedOperation = errors.New("unsupported operation")
	ErrOperandCount         = errors.New("wrong number of operands")
	ErrInvalidOperand       = errors.New("operands must be finite numbers")
	ErrDivisionByZero       = errors.New("division by zero")
	ErrNegativeSqrt         = errors.New("square root of a negative number is not a real number")
	ErrNegativePercentage   = errors.New("percentage must not be negative")
	ErrUndefinedResult      = errors.New("result is not a real number")
	ErrOutOfRange           = errors.New("result is too large to represent")
)

type definition struct {
	arity int
	apply func(x []float64) (float64, error)
}

var definitions = map[Operation]definition{
	Add: {2, func(x []float64) (float64, error) {
		return x[0] + x[1], nil
	}},
	Subtract: {2, func(x []float64) (float64, error) {
		return x[0] - x[1], nil
	}},
	Multiply: {2, func(x []float64) (float64, error) {
		return x[0] * x[1], nil
	}},
	Divide: {2, func(x []float64) (float64, error) {
		if x[1] == 0 {
			return 0, ErrDivisionByZero
		}
		return x[0] / x[1], nil
	}},
	Power: {2, func(x []float64) (float64, error) {
		// 0 to a negative power is 1/0; report it as such rather than as an
		// overflow.
		if x[0] == 0 && x[1] < 0 {
			return 0, ErrDivisionByZero
		}
		return math.Pow(x[0], x[1]), nil
	}},
	Sqrt: {1, func(x []float64) (float64, error) {
		if x[0] < 0 {
			return 0, ErrNegativeSqrt
		}
		return math.Sqrt(x[0]), nil
	}},
	// Percentage is "a percent of b". The percent must not be negative; the
	// base may be (10% of -50 is -5).
	Percentage: {2, func(x []float64) (float64, error) {
		if x[0] < 0 {
			return 0, ErrNegativePercentage
		}
		return x[0] / 100 * x[1], nil
	}},
}

// Calculate applies op to operands. Operands must be finite and their count
// must match the operation's arity. A result that is not a finite number is
// reported as an error instead of being returned as NaN or ±Inf, which JSON
// cannot represent.
func Calculate(op Operation, operands []float64) (float64, error) {
	def, ok := definitions[op]
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedOperation, op)
	}
	if len(operands) != def.arity {
		return 0, fmt.Errorf("%w: %s expects %d, got %d", ErrOperandCount, op, def.arity, len(operands))
	}
	for _, v := range operands {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, ErrInvalidOperand
		}
	}

	result, err := def.apply(operands)
	if err != nil {
		return 0, err
	}
	switch {
	case math.IsNaN(result):
		return 0, ErrUndefinedResult
	case math.IsInf(result, 0):
		return 0, ErrOutOfRange
	case result == 0:
		// Normalise -0 so clients never see "-0".
		return 0, nil
	}
	return result, nil
}
