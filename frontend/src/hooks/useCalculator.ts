import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiError, type CalculateFn } from '../api/client';
import {
  INITIAL_STATE,
  displayText,
  failed,
  press,
  type CalculatorState,
  type Key,
} from '../lib/engine';

export interface HistoryEntry {
  id: number;
  expression: string;
  value: number;
}

// The backend answers in well under a millisecond, so a second is generous;
// anything slower means it is down or unreachable.
export const REQUEST_TIMEOUT_MS = 1_000;
const MAX_HISTORY = 50;

/**
 * The backend's problem detail is written for end users, so it is shown as
 * is; the backend stays the single source of error wording.
 */
function describeError(err: unknown, timedOut: boolean): string {
  if (timedOut) return 'The calculator service did not respond. Please try again.';
  if (err instanceof ApiError) return err.message;
  return 'Something went wrong. Please try again.';
}

/**
 * Connects the pure calculator engine to the backend.
 *
 * Keys are processed strictly one at a time, in the order they were pressed.
 * A key that needs the backend holds the queue until its response arrives,
 * so every key sees the state left by the one before it and responses can
 * never be applied out of order or on top of newer input.
 *
 * "C" is the escape hatch: it bypasses the queue, drops queued keys and
 * invalidates the in-flight request (whose late response is then ignored), so
 * a slow backend never traps the user. Requests also time out.
 */
export function useCalculator(calculate: CalculateFn) {
  const [state, setState] = useState(INITIAL_STATE);
  const [busy, setBusy] = useState(false);
  const [history, setHistory] = useState<HistoryEntry[]>([]);

  // The queue runs outside React's render cycle, so it reads and writes the
  // authoritative state through a ref and mirrors it into React state.
  const stateRef = useRef(INITIAL_STATE);
  const queue = useRef<Promise<void>>(Promise.resolve());
  const generation = useRef(0);
  const inFlight = useRef<AbortController | null>(null);
  const nextHistoryId = useRef(1);

  const commit = useCallback((next: CalculatorState) => {
    stateRef.current = next;
    setState(next);
  }, []);

  const record = useCallback((expression: string, value: number) => {
    const entry = { id: nextHistoryId.current++, expression, value };
    setHistory((h) => [entry, ...h].slice(0, MAX_HISTORY));
  }, []);

  const process = useCallback(
    async (key: Key, ticket: number) => {
      if (ticket !== generation.current) return;

      const transition = press(stateRef.current, key);
      commit(transition.state);
      const { compute } = transition;
      if (!compute) return;

      const controller = new AbortController();
      inFlight.current = controller;
      let timedOut = false;
      let timer: ReturnType<typeof setTimeout> | undefined;
      // Racing a timer (rather than relying on the abort alone) guarantees
      // the queue is released even if the transport ignores the signal.
      const timeout = new Promise<never>((_, reject) => {
        timer = setTimeout(() => {
          timedOut = true;
          controller.abort();
          reject(new Error('Request timed out'));
        }, REQUEST_TIMEOUT_MS);
      });
      setBusy(true);

      try {
        const result = await Promise.race([
          calculate(
            { operation: compute.operation, operands: compute.operands },
            controller.signal,
          ),
          timeout,
        ]);
        if (ticket !== generation.current) return;
        commit(compute.complete(result));
        record(compute.historyExpression, result);
      } catch (err) {
        if (ticket !== generation.current) return;
        commit(failed(compute.expression, describeError(err, timedOut)));
      } finally {
        clearTimeout(timer);
        if (ticket === generation.current) {
          inFlight.current = null;
          setBusy(false);
        }
      }
    },
    [calculate, commit, record],
  );

  /** Makes queued keys and the in-flight request obsolete. */
  const invalidate = useCallback(() => {
    generation.current++;
    inFlight.current?.abort();
    inFlight.current = null;
    queue.current = Promise.resolve();
  }, []);

  const pressKey = useCallback(
    (key: Key) => {
      if (key.type === 'clear') {
        invalidate();
        setBusy(false);
        commit(INITIAL_STATE);
        return;
      }
      const ticket = generation.current;
      queue.current = queue.current
        .then(() => process(key, ticket))
        .catch((err: unknown) => {
          // process() handles every expected failure; this only keeps the
          // queue alive if something unexpected throws.
          console.error('Calculator key failed', err);
        });
    },
    [commit, invalidate, process],
  );

  const clearHistory = useCallback(() => {
    setHistory([]);
  }, []);

  useEffect(() => invalidate, [invalidate]);

  return {
    display: displayText(state),
    expression: state.expression,
    hasError: state.error !== null,
    busy,
    history,
    press: pressKey,
    clearHistory,
  };
}
