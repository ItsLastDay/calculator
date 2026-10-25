import type { Calculation, CalculationRequest, Problem } from './types';

/** Error codes produced by the client itself rather than the server. */
export const NETWORK_ERROR = 'NETWORK_ERROR';
export const BAD_RESPONSE = 'BAD_RESPONSE';

/**
 * A failed API call. `code` is the server's problem code (e.g.
 * DIVISION_BY_ZERO) or one of the client-side codes above; the message is
 * safe to show to users.
 */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly code: string,
    readonly status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/** Evaluates a calculation and resolves to its result. */
export type CalculateFn = (request: CalculationRequest, signal?: AbortSignal) => Promise<number>;

export interface CalculatorApi {
  calculate: CalculateFn;
}

interface ApiOptions {
  /** Prefix for API paths; empty means same origin. */
  baseUrl?: string;
  /** Injected for tests. */
  fetch?: typeof fetch;
}

/**
 * The frontend's only gateway to the backend: it owns the URL, the wire
 * format and error translation, so the rest of the app deals in numbers and
 * ApiErrors.
 */
export function createCalculatorApi({
  baseUrl = '',
  fetch: fetchFn = (...args) => fetch(...args),
}: ApiOptions = {}): CalculatorApi {
  async function calculate(request: CalculationRequest, signal?: AbortSignal): Promise<number> {
    let response: Response;
    try {
      response = await fetchFn(`${baseUrl}/api/v1/calculations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json, application/problem+json',
        },
        body: JSON.stringify(request),
        signal,
      });
    } catch (err) {
      // Let callers distinguish a deliberate cancellation from a failure.
      if (isAbortError(err)) throw err;
      throw new ApiError(
        'Could not reach the calculator service. Please check that it is running.',
        NETWORK_ERROR,
        0,
      );
    }

    const body: unknown = await response.json().catch(() => undefined);

    if (!response.ok) {
      if (isProblem(body)) {
        throw new ApiError(body.detail, body.code, response.status);
      }
      throw new ApiError(
        `The calculator service failed with status ${response.status}.`,
        BAD_RESPONSE,
        response.status,
      );
    }

    if (!isCalculation(body)) {
      throw new ApiError(
        'The calculator service returned an unexpected response.',
        BAD_RESPONSE,
        response.status,
      );
    }
    return body.result;
  }

  return { calculate };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isAbortError(value: unknown): boolean {
  return isRecord(value) && value.name === 'AbortError';
}

function isProblem(value: unknown): value is Problem {
  return isRecord(value) && typeof value.code === 'string' && typeof value.detail === 'string';
}

function isCalculation(value: unknown): value is Calculation {
  return isRecord(value) && typeof value.result === 'number';
}

export const calculatorApi = createCalculatorApi();
