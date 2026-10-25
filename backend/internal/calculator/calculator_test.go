package calculator

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

// Reference points of the float64 range. Expected values in these tests are
// exact: either representable as written, or the single correctly rounded
// result of one IEEE 754 operation, so results are compared with ==.
var (
	negZero = math.Copysign(0, -1)
	maxF    = math.MaxFloat64
	tiny    = math.SmallestNonzeroFloat64 // 2^-1074, the smallest subnormal
	minNorm = 0x1p-1022                   // the smallest normal number
)

// allOperations must list every supported operation; the loop-based tests
// and the fuzz test iterate over it.
var allOperations = []Operation{Add, Subtract, Multiply, Divide, Power, Sqrt, Percentage}

var allErrors = []error{
	ErrUnsupportedOperation, ErrOperandCount, ErrInvalidOperand, ErrDivisionByZero,
	ErrNegativeSqrt, ErrNegativePercentage, ErrUndefinedResult, ErrOutOfRange,
}

func TestEveryOperationIsCovered(t *testing.T) {
	if len(allOperations) != len(definitions) {
		t.Fatalf("allOperations lists %d operations, definitions has %d", len(allOperations), len(definitions))
	}
	for _, op := range allOperations {
		if _, ok := definitions[op]; !ok {
			t.Errorf("allOperations lists %q, which is not defined", op)
		}
	}
}

type successCase struct {
	name     string
	op       Operation
	operands []float64
	want     float64
}

func runSuccessCases(t *testing.T, tests []successCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.op, tt.operands)
			if err != nil {
				t.Fatalf("Calculate(%s, %v) returned error: %v", tt.op, tt.operands, err)
			}
			if got != tt.want {
				t.Errorf("Calculate(%s, %v) = %v (%b), want %v (%b)", tt.op, tt.operands, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestCalculate(t *testing.T) {
	runSuccessCases(t, []successCase{
		{"add integers", Add, []float64{2, 3}, 5},
		{"add negatives", Add, []float64{-2.5, -0.5}, -3},
		{"subtract", Subtract, []float64{10, 4}, 6},
		{"subtract to negative", Subtract, []float64{4, 10}, -6},
		{"multiply", Multiply, []float64{6, 7}, 42},
		{"multiply by zero", Multiply, []float64{123, 0}, 0},
		{"multiply negatives", Multiply, []float64{-4, -2.5}, 10},
		{"divide", Divide, []float64{10, 4}, 2.5},
		{"divide negative", Divide, []float64{-9, 3}, -3},
		{"divide zero numerator", Divide, []float64{0, 5}, 0},
		{"power", Power, []float64{2, 10}, 1024},
		{"power negative exponent", Power, []float64{2, -2}, 0.25},
		{"power fractional exponent", Power, []float64{9, 0.5}, 3},
		{"power zero to zero", Power, []float64{0, 0}, 1},
		{"power negative base integer exponent", Power, []float64{-2, 3}, -8},
		{"sqrt", Sqrt, []float64{16}, 4},
		{"sqrt zero", Sqrt, []float64{0}, 0},
		{"sqrt fraction", Sqrt, []float64{0.25}, 0.5},
		{"percentage", Percentage, []float64{15, 200}, 30},
		{"percentage over 100", Percentage, []float64{150, 40}, 60},
		{"percentage of a negative base", Percentage, []float64{10, -50}, -5},
		{"zero percent", Percentage, []float64{0, 50}, 0},
	})
}

// Decimal fractions are not exact in binary; results follow IEEE 754, and
// clients format them for display.
func TestCalculateDecimals(t *testing.T) {
	runSuccessCases(t, []successCase{
		{"add", Add, []float64{0.1, 0.2}, 0.30000000000000004},
		{"subtract", Subtract, []float64{0.3, 0.1}, 0.19999999999999998},
		{"multiply", Multiply, []float64{0.1, 3}, 0.30000000000000004},
		{"divide", Divide, []float64{0.3, 0.1}, 2.9999999999999996},
		{"one third", Divide, []float64{1, 3}, 1.0 / 3},
		{"two thirds", Divide, []float64{2, 3}, 2.0 / 3},
		{"exact binary fractions", Add, []float64{0.5, 0.25}, 0.75},
		{"fractional percentage", Percentage, []float64{12.5, 80}, 10},
	})
}

func TestCalculateLargeNumbers(t *testing.T) {
	runSuccessCases(t, []successCase{
		{"add up to the largest float", Add, []float64{maxF / 2, maxF / 2}, maxF},
		{"add opposite extremes", Add, []float64{maxF, -maxF}, 0},
		{"add numbers of very different scale", Add, []float64{1e300, 1e-300}, 1e300},
		{"subtract the largest float from itself", Subtract, []float64{maxF, maxF}, 0},
		{"subtract a negligible amount", Subtract, []float64{1e20, 1}, 1e20},
		{"multiply powers of two", Multiply, []float64{0x1p511, 0x1p511}, 0x1p1022},
		{"multiply trillions", Multiply, []float64{1e12, 1e12}, 1e24},
		{"halve the largest float", Multiply, []float64{maxF, 0.5}, maxF / 2},
		{"multiply large integers (rounded)", Multiply, []float64{123456789, 987654321}, 121932631112635269},
		{"divide the largest float by itself", Divide, []float64{maxF, maxF}, 1},
		{"divide large powers of two", Divide, []float64{0x1p1000, 0x1p500}, 0x1p500},
		{"power of two at the top of the range", Power, []float64{2, 1023}, 0x1p1023},
		{"negative base, odd exponent, top of range", Power, []float64{-2, 1023}, -0x1p1023},
		{"largest exact power of ten", Power, []float64{10, 22}, 1e22},
		{"one to the largest power", Power, []float64{1, maxF}, 1},
		{"minus one to a huge even power", Power, []float64{-1, 1e18}, 1},
		{"largest float to the zeroth power", Power, []float64{maxF, 0}, 1},
		{"largest float to the first power", Power, []float64{maxF, 1}, maxF},
		{"sqrt of a large power of two", Sqrt, []float64{0x1p1022}, 0x1p511},
		{"sqrt of the largest float", Sqrt, []float64{maxF}, 1.3407807929942596e154},
		{"sqrt of a large perfect square", Sqrt, []float64{1e30}, 1e15},
		{"100 percent of the largest float", Percentage, []float64{100, maxF}, maxF},
		{"a thousand percent", Percentage, []float64{1000, 3}, 30},
	})
}

// Integers are exact up to 2^53; beyond that, results round to even.
func TestCalculateIntegerPrecision(t *testing.T) {
	runSuccessCases(t, []successCase{
		{"largest exact integer", Add, []float64{4503599627370496, 4503599627370495}, 9007199254740991},
		{"reach 2^53", Add, []float64{9007199254740991, 1}, 9007199254740992},
		{"2^53 + 1 rounds to even", Add, []float64{9007199254740992, 1}, 9007199254740992},
		{"2^53 + 2 is exact", Add, []float64{9007199254740992, 2}, 9007199254740994},
		{"sqrt of 2^52", Sqrt, []float64{4503599627370496}, 67108864},
	})
}

// Results smaller than the smallest subnormal round to zero, as in IEEE 754;
// that is a valid result, not an error.
func TestCalculateSmallNumbers(t *testing.T) {
	runSuccessCases(t, []successCase{
		{"add the smallest subnormals", Add, []float64{tiny, tiny}, 2 * tiny},
		{"subtract down to the smallest subnormal", Subtract, []float64{2 * tiny, tiny}, tiny},
		{"tiny additions to one are lost", Add, []float64{1, 1e-20}, 1},
		{"multiply small powers of two", Multiply, []float64{0x1p-500, 0x1p-500}, 0x1p-1000},
		{"multiply into the subnormal range", Multiply, []float64{0x1p-1000, 0x1p-70}, 0x1p-1070},
		{"multiply below the smallest subnormal", Multiply, []float64{tiny, 0.5}, 0},
		{"divide the smallest normal by two", Divide, []float64{minNorm, 2}, 0x1p-1023},
		{"divide equal small numbers", Divide, []float64{0x1p-1000, 0x1p-1000}, 1},
		{"divide small by large underflows to zero", Divide, []float64{0x1p-1000, 0x1p100}, 0},
		{"power of two at the bottom of the range", Power, []float64{2, -1074}, tiny},
		{"power below the bottom of the range", Power, []float64{2, -1075}, 0},
		{"square a tiny number", Power, []float64{1e-200, 2}, 0},
		{"sqrt of the smallest subnormal", Sqrt, []float64{tiny}, 0x1p-537},
		{"sqrt of two", Sqrt, []float64{2}, math.Sqrt2},
		{"half of a tiny number", Percentage, []float64{50, 2 * tiny}, tiny},
		{"quarter of a small power of two", Percentage, []float64{25, 0x1p-1000}, 0x1p-1002},
		{"percent of zero", Percentage, []float64{37, 0}, 0},
	})
}

func TestCalculateErrors(t *testing.T) {
	tests := []struct {
		name     string
		op       Operation
		operands []float64
		wantErr  error
	}{
		{"unknown operation", Operation("modulo"), []float64{1, 2}, ErrUnsupportedOperation},
		{"empty operation", Operation(""), []float64{1, 2}, ErrUnsupportedOperation},
		{"operation names are case-sensitive", Operation("ADD"), []float64{1, 2}, ErrUnsupportedOperation},

		{"NaN operand", Add, []float64{math.NaN(), 1}, ErrInvalidOperand},
		{"NaN second operand", Multiply, []float64{1, math.NaN()}, ErrInvalidOperand},
		{"infinite operand", Add, []float64{math.Inf(1), 1}, ErrInvalidOperand},
		{"negative infinite operand", Subtract, []float64{1, math.Inf(-1)}, ErrInvalidOperand},
		{"infinite sqrt operand", Sqrt, []float64{math.Inf(1)}, ErrInvalidOperand},
		{"invalid operand is checked before division by zero", Divide, []float64{math.NaN(), 0}, ErrInvalidOperand},

		{"division by zero", Divide, []float64{1, 0}, ErrDivisionByZero},
		{"zero divided by zero", Divide, []float64{0, 0}, ErrDivisionByZero},
		{"division by negative zero", Divide, []float64{1, negZero}, ErrDivisionByZero},
		{"tiny divided by zero", Divide, []float64{tiny, 0}, ErrDivisionByZero},
		{"zero to a negative power", Power, []float64{0, -1}, ErrDivisionByZero},
		{"zero to a negative fractional power", Power, []float64{0, -0.5}, ErrDivisionByZero},
		{"negative zero to a negative power", Power, []float64{negZero, -1}, ErrDivisionByZero},

		{"negative sqrt", Sqrt, []float64{-1}, ErrNegativeSqrt},
		{"tiniest negative sqrt", Sqrt, []float64{-tiny}, ErrNegativeSqrt},
		{"largest negative sqrt", Sqrt, []float64{-maxF}, ErrNegativeSqrt},

		{"negative percentage", Percentage, []float64{-10, 50}, ErrNegativePercentage},
		{"negative fractional percentage", Percentage, []float64{-0.5, 50}, ErrNegativePercentage},
		{"tiniest negative percentage", Percentage, []float64{-tiny, 1}, ErrNegativePercentage},

		{"negative base, fractional exponent", Power, []float64{-8, 1.0 / 3}, ErrUndefinedResult},
		{"minus one to the half", Power, []float64{-1, 0.5}, ErrUndefinedResult},

		{"add overflow", Add, []float64{maxF, maxF}, ErrOutOfRange},
		{"subtract overflow", Subtract, []float64{maxF, -maxF}, ErrOutOfRange},
		{"subtract overflow, negative", Subtract, []float64{-maxF, maxF}, ErrOutOfRange},
		{"multiply overflow", Multiply, []float64{maxF, 2}, ErrOutOfRange},
		{"multiply overflow, negative", Multiply, []float64{-maxF, 2}, ErrOutOfRange},
		{"multiply large numbers", Multiply, []float64{1e200, 1e200}, ErrOutOfRange},
		{"divide overflow", Divide, []float64{maxF, 0.1}, ErrOutOfRange},
		{"divide by the smallest subnormal", Divide, []float64{1, tiny}, ErrOutOfRange},
		{"power overflow", Power, []float64{10, 400}, ErrOutOfRange},
		{"power just past the top of the range", Power, []float64{2, 1024}, ErrOutOfRange},
		{"power overflow, negative", Power, []float64{-2, 1025}, ErrOutOfRange},
		{"power of a tiny number to a negative power", Power, []float64{tiny, -2}, ErrOutOfRange},
		{"percentage overflow", Percentage, []float64{200, maxF}, ErrOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.op, tt.operands)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Calculate(%s, %v) error = %v, want %v", tt.op, tt.operands, err, tt.wantErr)
			}
			if got != 0 {
				t.Errorf("Calculate(%s, %v) = %v on error, want 0", tt.op, tt.operands, got)
			}
		})
	}
}

func TestOperandCountForEveryOperation(t *testing.T) {
	for _, op := range allOperations {
		arity := definitions[op].arity
		for _, n := range []int{0, 1, 2, 3} {
			if n == arity {
				continue
			}
			t.Run(fmt.Sprintf("%s with %d operands", op, n), func(t *testing.T) {
				operands := make([]float64, n)
				for i := range operands {
					operands[i] = 1
				}
				if _, err := Calculate(op, operands); !errors.Is(err, ErrOperandCount) {
					t.Errorf("error = %v, want %v", err, ErrOperandCount)
				}
			})
		}
	}
}

func TestCalculateNeverReturnsNegativeZero(t *testing.T) {
	tests := []struct {
		name     string
		op       Operation
		operands []float64
	}{
		{"add negative zeros", Add, []float64{negZero, negZero}},
		{"subtract zero from negative zero", Subtract, []float64{negZero, 0}},
		{"multiply negative by zero", Multiply, []float64{-1, 0}},
		{"multiply underflowing to negative zero", Multiply, []float64{tiny, -0.5}},
		{"divide negative zero", Divide, []float64{negZero, 5}},
		{"divide zero by a negative", Divide, []float64{0, -5}},
		{"divide underflowing to negative zero", Divide, []float64{-tiny, 0x1p100}},
		{"negative zero cubed", Power, []float64{negZero, 3}},
		{"sqrt of negative zero", Sqrt, []float64{negZero}},
		{"zero percent of a negative", Percentage, []float64{0, -5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.op, tt.operands)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != 0 || math.Signbit(got) {
				t.Errorf("Calculate(%s, %v) = %v, want +0", tt.op, tt.operands, got)
			}
		})
	}
}

// magnitudes spans the float64 range, from the smallest subnormal to the
// largest finite value, in both signs.
var magnitudes = func() []float64 {
	positive := []float64{tiny, minNorm, 1e-300, 1e-10, 0.1, 1, 3, 1e10, 1e300, maxF}
	values := make([]float64, 0, 2*len(positive))
	for _, v := range positive {
		values = append(values, v, -v)
	}
	return values
}()

func TestIdentitiesAcrossMagnitudes(t *testing.T) {
	identities := []struct {
		name     string
		op       Operation
		operands func(v float64) []float64
		want     func(v float64) float64
	}{
		{"v + 0 = v", Add, func(v float64) []float64 { return []float64{v, 0} }, func(v float64) float64 { return v }},
		{"v + (-v) = 0", Add, func(v float64) []float64 { return []float64{v, -v} }, func(float64) float64 { return 0 }},
		{"v - v = 0", Subtract, func(v float64) []float64 { return []float64{v, v} }, func(float64) float64 { return 0 }},
		{"v - 0 = v", Subtract, func(v float64) []float64 { return []float64{v, 0} }, func(v float64) float64 { return v }},
		{"v × 1 = v", Multiply, func(v float64) []float64 { return []float64{v, 1} }, func(v float64) float64 { return v }},
		{"v × 0 = 0", Multiply, func(v float64) []float64 { return []float64{v, 0} }, func(float64) float64 { return 0 }},
		{"v × -1 = -v", Multiply, func(v float64) []float64 { return []float64{v, -1} }, func(v float64) float64 { return -v }},
		{"v ÷ 1 = v", Divide, func(v float64) []float64 { return []float64{v, 1} }, func(v float64) float64 { return v }},
		{"v ÷ v = 1", Divide, func(v float64) []float64 { return []float64{v, v} }, func(float64) float64 { return 1 }},
		{"v ^ 1 = v", Power, func(v float64) []float64 { return []float64{v, 1} }, func(v float64) float64 { return v }},
		{"v ^ 0 = 1", Power, func(v float64) []float64 { return []float64{v, 0} }, func(float64) float64 { return 1 }},
		{"100% of v = v", Percentage, func(v float64) []float64 { return []float64{100, v} }, func(v float64) float64 { return v }},
		{"0% of v = 0", Percentage, func(v float64) []float64 { return []float64{0, v} }, func(float64) float64 { return 0 }},
	}

	for _, id := range identities {
		for _, v := range magnitudes {
			t.Run(fmt.Sprintf("%s for v=%g", id.name, v), func(t *testing.T) {
				got, err := Calculate(id.op, id.operands(v))
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if want := id.want(v); got != want || math.Signbit(got) != math.Signbit(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			})
		}
	}
}

func TestSqrtAcrossMagnitudes(t *testing.T) {
	for _, v := range magnitudes {
		if v < 0 {
			continue
		}
		got, err := Calculate(Sqrt, []float64{v})
		if err != nil {
			t.Fatalf("sqrt(%g): unexpected error: %v", v, err)
		}
		if got <= 0 || got != math.Sqrt(v) {
			t.Errorf("sqrt(%g) = %g, want %g", v, got, math.Sqrt(v))
		}
	}
}

// randomValue returns finite values spread over the whole float64 range.
func randomValue(r *rand.Rand) float64 {
	v := (r.Float64() + 0.5) * math.Pow(10, float64(r.IntN(617)-308))
	if r.IntN(2) == 0 {
		v = -v
	}
	return v
}

func TestCommutativity(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2)) // fixed seed: failures are reproducible
	for i := 0; i < 2000; i++ {
		a, b := randomValue(r), randomValue(r)
		for _, op := range []Operation{Add, Multiply} {
			ab, errAB := Calculate(op, []float64{a, b})
			ba, errBA := Calculate(op, []float64{b, a})
			if ab != ba || !errors.Is(errAB, errBA) {
				t.Fatalf("%s is not commutative for %g and %g: %v (%v) vs %v (%v)", op, a, b, ab, errAB, ba, errBA)
			}
		}
	}
}

// expected is an independent statement of the rules: what Calculate must
// return for any two inputs (the second is ignored by sqrt).
func expected(op Operation, a, b float64) (float64, error) {
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	if !finite(a) || (op != Sqrt && !finite(b)) {
		return 0, ErrInvalidOperand
	}

	var raw float64
	switch op {
	case Add:
		raw = a + b
	case Subtract:
		raw = a - b
	case Multiply:
		raw = a * b
	case Divide:
		if b == 0 {
			return 0, ErrDivisionByZero
		}
		raw = a / b
	case Power:
		if a == 0 && b < 0 {
			return 0, ErrDivisionByZero
		}
		raw = math.Pow(a, b)
	case Sqrt:
		if a < 0 {
			return 0, ErrNegativeSqrt
		}
		raw = math.Sqrt(a)
	case Percentage:
		if a < 0 {
			return 0, ErrNegativePercentage
		}
		raw = a / 100 * b
	}

	switch {
	case math.IsNaN(raw):
		return 0, ErrUndefinedResult
	case math.IsInf(raw, 0):
		return 0, ErrOutOfRange
	case raw == 0:
		return 0, nil
	}
	return raw, nil
}

// FuzzCalculate checks every operation against expected for arbitrary
// inputs, including NaN, ±Inf, ±0 and subnormals. `go test` runs the seed
// corpus; `go test -fuzz=FuzzCalculate ./internal/calculator` explores further.
func FuzzCalculate(f *testing.F) {
	seeds := []float64{
		0, negZero, 1, -1, 0.1, 2, -2.5, 100, 1e-300, 1e300,
		tiny, minNorm, maxF, -maxF, 9007199254740993,
		math.NaN(), math.Inf(1), math.Inf(-1),
	}
	for _, a := range seeds {
		for _, b := range seeds {
			f.Add(a, b)
		}
	}

	f.Fuzz(func(t *testing.T, a, b float64) {
		for _, op := range allOperations {
			operands := []float64{a, b}
			if definitions[op].arity == 1 {
				operands = operands[:1]
			}

			got, err := Calculate(op, operands)
			want, wantErr := expected(op, a, b)

			if !errors.Is(err, wantErr) {
				t.Fatalf("Calculate(%s, %v) error = %v, want %v", op, operands, err, wantErr)
			}
			if err != nil {
				if got != 0 {
					t.Fatalf("Calculate(%s, %v) = %v alongside error %v, want 0", op, operands, got, err)
				}
				if !isKnownError(err) {
					t.Fatalf("Calculate(%s, %v) returned an undocumented error: %v", op, operands, err)
				}
				continue
			}
			if got != want || (got == 0 && math.Signbit(got)) {
				t.Fatalf("Calculate(%s, %v) = %v, want %v", op, operands, got, want)
			}
		}
	})
}

func isKnownError(err error) bool {
	for _, known := range allErrors {
		if errors.Is(err, known) {
			return true
		}
	}
	return false
}
