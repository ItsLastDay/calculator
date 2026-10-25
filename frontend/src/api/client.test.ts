import { describe, expect, it, vi } from 'vitest';
import { ApiError, BAD_RESPONSE, NETWORK_ERROR, createCalculatorApi } from './client';

function jsonResponse(body: unknown, status = 200, contentType = 'application/json'): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': contentType },
  });
}

function problem(status: number, code: string, detail: string): Response {
  return jsonResponse({ code, detail }, status, 'application/problem+json');
}

function setup(outcome: { resolve: Response } | { reject: unknown }) {
  const fetch = vi.fn<typeof globalThis.fetch>();
  if ('resolve' in outcome) fetch.mockResolvedValue(outcome.resolve);
  else fetch.mockRejectedValue(outcome.reject);
  return { fetch, api: createCalculatorApi({ baseUrl: 'http://api.test', fetch }) };
}

async function captureError(promise: Promise<unknown>): Promise<ApiError> {
  try {
    await promise;
  } catch (err) {
    expect(err).toBeInstanceOf(ApiError);
    return err as ApiError;
  }
  throw new Error('expected promise to reject');
}

describe('calculatorApi.calculate', () => {
  it('POSTs the request to the calculations resource and resolves to the result', async () => {
    const { fetch, api } = setup({ resolve: jsonResponse({ result: 5 }) });
    const signal = new AbortController().signal;

    const result = await api.calculate({ operation: 'add', operands: [2, 3] }, signal);

    expect(result).toBe(5);
    expect(fetch).toHaveBeenCalledOnce();
    const [url, init] = fetch.mock.calls[0]!;
    expect(url).toBe('http://api.test/api/v1/calculations');
    expect(init?.method).toBe('POST');
    expect(init?.headers).toMatchObject({ 'Content-Type': 'application/json' });
    expect(JSON.parse(init?.body as string)).toEqual({ operation: 'add', operands: [2, 3] });
    expect(init?.signal).toBe(signal);
  });

  it('surfaces the problem code and detail from the server', async () => {
    const { api } = setup({ resolve: problem(422, 'DIVISION_BY_ZERO', 'Division by zero.') });

    const err = await captureError(api.calculate({ operation: 'divide', operands: [1, 0] }));

    expect(err.message).toBe('Division by zero.');
    expect(err.code).toBe('DIVISION_BY_ZERO');
    expect(err.status).toBe(422);
  });

  it('handles error responses without a problem body', async () => {
    const { api } = setup({ resolve: new Response('Bad Gateway', { status: 502 }) });

    const err = await captureError(api.calculate({ operation: 'add', operands: [1, 2] }));

    expect(err.code).toBe(BAD_RESPONSE);
    expect(err.status).toBe(502);
    expect(err.message).toContain('502');
  });

  it('rejects a success response without a numeric result', async () => {
    const { api } = setup({ resolve: jsonResponse({ result: '5' }) });

    const err = await captureError(api.calculate({ operation: 'add', operands: [1, 2] }));

    expect(err.code).toBe(BAD_RESPONSE);
  });

  it('turns network failures into a friendly error', async () => {
    const { api } = setup({ reject: new TypeError('Failed to fetch') });

    const err = await captureError(api.calculate({ operation: 'add', operands: [1, 2] }));

    expect(err.code).toBe(NETWORK_ERROR);
    expect(err.status).toBe(0);
    expect(err.message).toMatch(/could not reach/i);
  });

  it('rethrows aborts unchanged so callers can ignore them', async () => {
    const abort = new DOMException('The operation was aborted.', 'AbortError');
    const { api } = setup({ reject: abort });

    await expect(api.calculate({ operation: 'add', operands: [1, 2] })).rejects.toBe(abort);
  });

  it('uses same-origin paths by default', async () => {
    const fetch = vi.fn<typeof globalThis.fetch>().mockResolvedValue(jsonResponse({ result: 1 }));
    const api = createCalculatorApi({ fetch });

    await api.calculate({ operation: 'sqrt', operands: [1] });

    expect(fetch.mock.calls[0]![0]).toBe('/api/v1/calculations');
  });
});
