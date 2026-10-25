import { vi } from 'vitest';
import type { CalculateFn } from '../api/client';
import type { CalculationRequest } from '../api/types';

/** A backend that answers immediately, computing locally. */
export function instantBackend() {
  return vi.fn<CalculateFn>(({ operation, operands: [a = NaN, b = NaN] }) => {
    const results: Record<CalculationRequest['operation'], number> = {
      add: a + b,
      subtract: a - b,
      multiply: a * b,
      divide: a / b,
      power: a ** b,
      sqrt: Math.sqrt(a),
      percentage: (a / 100) * b,
    };
    return Promise.resolve(results[operation]);
  });
}

interface PendingCall {
  request: CalculationRequest;
  signal?: AbortSignal;
  resolve: (result: number) => void;
  reject: (error: unknown) => void;
}

/**
 * A backend whose responses the test settles by hand, in any order. Like a
 * misbehaving network, it ignores the abort signal, which proves ordering is
 * enforced by the app itself.
 */
export function manualBackend() {
  const calls: PendingCall[] = [];
  const calculate = vi.fn<CalculateFn>(
    (request, signal) =>
      new Promise<number>((resolve, reject) => {
        calls.push({ request, signal, resolve, reject });
      }),
  );
  return { calculate, calls };
}
