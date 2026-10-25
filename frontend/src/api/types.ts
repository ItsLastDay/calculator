// Wire types for the calculator REST API. The contract is defined in
// backend/api/openapi.yaml; keep these in sync with it.

export type Operation =
  'add' | 'subtract' | 'multiply' | 'divide' | 'power' | 'sqrt' | 'percentage';

/** Body of POST /api/v1/calculations. */
export interface CalculationRequest {
  operation: Operation;
  operands: number[];
}

/** Successful response of POST /api/v1/calculations. */
export interface Calculation {
  result: number;
}

/** RFC 9457 Problem Details, returned for every error response. */
export interface Problem {
  code: string;
  detail: string;
}
