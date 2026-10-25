/**
 * The calculator's key-by-key behaviour, modelled on the Windows Standard
 * calculator (without memory), except that % is a two-number operator:
 * "15 % 200 =" is 15% of 200. It is a pure state machine: `press` never does
 * I/O. When a key needs arithmetic it returns a `Computation` for the
 * backend, plus how to continue once the result is known.
 */
import type { Operation } from '../api/types';
import { displayNumber, groupDigits } from './format';

export type BinaryOperator = Extract<
  Operation,
  'add' | 'subtract' | 'multiply' | 'divide' | 'power' | 'percentage'
>;

export type Digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9';

export type Key =
  | { type: 'digit'; digit: Digit }
  | { type: 'decimal' }
  | { type: 'operator'; operator: BinaryOperator }
  | { type: 'equals' }
  | { type: 'sqrt' }
  | { type: 'square' }
  | { type: 'negate' }
  | { type: 'backspace' }
  | { type: 'clearEntry' }
  | { type: 'clear' }
  | { type: 'recall'; expression: string; value: number };

export interface CalculatorState {
  /** Text being typed, or null while a computed value is shown. */
  entry: string | null;
  /** The number on the display, at full precision. */
  value: number;
  /** The line above the display, e.g. "12 +" or "12 + √(9) =". */
  expression: string;
  /** Left operand (and how it is written) and operator, waiting for a right operand. */
  pending: { left: number; leftText: string; operator: BinaryOperator } | null;
  /**
   * operand: the display holds the operand being built (typed or computed).
   * awaitingOperand: an operator was just pressed; typing starts the right operand.
   * result: "=" finished a calculation; typing starts a new one.
   */
  phase: 'operand' | 'awaitingOperand' | 'result';
  /** How the operand is written in expressions, e.g. "√(9)"; null means its value. */
  operandText: string | null;
  /** The last operation, re-applied by pressing "=" again. */
  repeat: { operator: BinaryOperator; right: number } | null;
  /** Message shown instead of a number; most keys are disabled until it is cleared. */
  error: string | null;
}

export interface Computation {
  operation: Operation;
  operands: number[];
  /** Expression to show if the backend rejects the computation. */
  expression: string;
  /** Builds the next state from the backend's result. */
  complete: (result: number) => CalculatorState;
  /**
   * How the calculation reads in the history, e.g. "√(9) =" or "2 + 3 =".
   * Every successful computation is recorded, including √x, x² and the
   * intermediate results of chained operators.
   */
  historyExpression: string;
}

export interface Transition {
  state: CalculatorState;
  compute?: Computation;
}

/** Windows caps input at 16 significant digits. */
export const MAX_DIGITS = 16;

const SYMBOLS: Record<BinaryOperator, string> = {
  add: '+',
  subtract: '−',
  multiply: '×',
  divide: '÷',
  power: '^',
  percentage: '% of',
};

/** Writes a left operand and its operator: "12 +", or "15% of". */
function withOperator(leftText: string, operator: BinaryOperator): string {
  const separator = operator === 'percentage' ? '' : ' ';
  return `${leftText}${separator}${SYMBOLS[operator]}`;
}

export const INITIAL_STATE: CalculatorState = {
  entry: '0',
  value: 0,
  expression: '',
  pending: null,
  phase: 'operand',
  operandText: null,
  repeat: null,
  error: null,
};

/** The text for the main display. */
export function displayText(state: CalculatorState): string {
  if (state.error !== null) return state.error;
  return state.entry !== null ? groupDigits(state.entry) : displayNumber(state.value);
}

/** The state after the backend rejected a computation. */
export function failed(expression: string, message: string): CalculatorState {
  return { ...INITIAL_STATE, expression, error: message };
}

/** Keys that do nothing while an error is shown (the UI disables them). */
export function isBlockedByError(key: Key): boolean {
  return !['digit', 'decimal', 'backspace', 'clearEntry', 'clear', 'recall'].includes(key.type);
}

export function press(state: CalculatorState, key: Key): Transition {
  if (state.error !== null) {
    // After an error, allowed keys start over from a clean state.
    return isBlockedByError(key) ? { state } : press(INITIAL_STATE, key);
  }

  switch (key.type) {
    case 'digit':
      return { state: typeDigit(state, key.digit) };
    case 'decimal':
      return { state: typeDecimal(state) };
    case 'backspace':
      return { state: backspace(state) };
    case 'clearEntry':
      return { state: startEntry(state, '0') };
    case 'clear':
      return { state: INITIAL_STATE };
    case 'negate':
      return { state: negate(state) };
    case 'operator':
      return pressOperator(state, key.operator);
    case 'equals':
      return pressEquals(state);
    case 'sqrt':
      return applyUnary(state, 'sqrt', [state.value], `√(${operandText(state)})`);
    case 'square':
      return applyUnary(state, 'power', [state.value, 2], `sqr(${operandText(state)})`);
    case 'recall':
      return { state: showResult(key.value, key.expression, null) };
  }
}

function operandText(state: CalculatorState): string {
  return state.operandText ?? displayNumber(state.value);
}

/** The pending part of the expression, e.g. "12 + ". */
function pendingPrefix(state: CalculatorState): string {
  if (!state.pending) return '';
  return `${withOperator(state.pending.leftText, state.pending.operator)} `;
}

function withEntry(state: CalculatorState, entry: string): CalculatorState {
  return { ...state, entry, value: Number(entry) };
}

/** Replaces whatever the display shows with a freshly typed operand. */
function startEntry(state: CalculatorState, entry: string): CalculatorState {
  const base = state.phase === 'result' ? INITIAL_STATE : state;
  return {
    ...withEntry(base, entry),
    phase: 'operand',
    operandText: null,
    expression: pendingPrefix(base).trimEnd(),
  };
}

function countDigits(text: string): number {
  return text.replace(/\D/g, '').length;
}

function typeDigit(state: CalculatorState, digit: Digit): CalculatorState {
  const { entry } = state;
  if (entry === null) return startEntry(state, digit);
  if (entry === '0' || entry === '-0') return withEntry(state, entry.replace('0', digit));
  if (countDigits(entry) >= MAX_DIGITS) return state;
  return withEntry(state, entry + digit);
}

function typeDecimal(state: CalculatorState): CalculatorState {
  const { entry } = state;
  if (entry === null) return startEntry(state, '0.');
  return entry.includes('.') ? state : withEntry(state, `${entry}.`);
}

function backspace(state: CalculatorState): CalculatorState {
  const { entry } = state;
  if (entry === null) {
    // Like Windows, backspace after "=" only clears the expression line.
    return state.phase === 'result' ? { ...state, expression: '' } : state;
  }
  const next = entry.slice(0, -1);
  return withEntry(state, next === '' || next === '-' ? '0' : next);
}

/** Sign change is input editing, not arithmetic, so it stays local. */
function negate(state: CalculatorState): CalculatorState {
  const { entry } = state;
  if (entry !== null) {
    if (state.value === 0) return state;
    return withEntry(state, entry.startsWith('-') ? entry.slice(1) : `-${entry}`);
  }
  const base = state.phase === 'result' ? INITIAL_STATE : state;
  const text = `negate(${operandText(state)})`;
  return {
    ...base,
    entry: null,
    value: state.value === 0 ? 0 : -state.value,
    phase: 'operand',
    operandText: text,
    expression: pendingPrefix(base) + text,
  };
}

function awaitOperand(
  left: number,
  operator: BinaryOperator,
  leftText = displayNumber(left),
): CalculatorState {
  return {
    ...INITIAL_STATE,
    entry: null,
    value: left,
    pending: { left, leftText, operator },
    phase: 'awaitingOperand',
    expression: withOperator(leftText, operator),
  };
}

function showResult(
  value: number,
  expression: string,
  repeat: CalculatorState['repeat'],
): CalculatorState {
  return { ...INITIAL_STATE, entry: null, value, expression, phase: 'result', repeat };
}

/**
 * Immediate execution, as in Windows Standard mode: pressing an operator
 * evaluates the pending operation first, so 2 + 3 × 4 = 20.
 */
function pressOperator(state: CalculatorState, operator: BinaryOperator): Transition {
  const { pending } = state;
  if (!pending) {
    return { state: awaitOperand(state.value, operator, operandText(state)) };
  }
  if (state.phase === 'awaitingOperand') {
    // Pressing another operator straight away replaces the previous one.
    return { state: awaitOperand(pending.left, operator, pending.leftText) };
  }
  const calculation = pendingPrefix(state) + operandText(state);
  return {
    state,
    compute: {
      operation: pending.operator,
      operands: [pending.left, state.value],
      expression: withOperator(calculation, operator),
      complete: (result) => awaitOperand(result, operator),
      historyExpression: `${calculation} =`,
    },
  };
}

function pressEquals(state: CalculatorState): Transition {
  const { pending, repeat } = state;
  if (pending) {
    // With no right operand typed yet, the left one is reused: 5 + = → 10.
    const right = state.value;
    const expression = `${pendingPrefix(state)}${operandText(state)} =`;
    return {
      state,
      compute: {
        operation: pending.operator,
        operands: [pending.left, right],
        expression,
        complete: (result) => showResult(result, expression, { operator: pending.operator, right }),
        historyExpression: expression,
      },
    };
  }
  if (state.phase === 'result' && repeat) {
    const expression = `${withOperator(displayNumber(state.value), repeat.operator)} ${displayNumber(repeat.right)} =`;
    return {
      state,
      compute: {
        operation: repeat.operator,
        operands: [state.value, repeat.right],
        expression,
        complete: (result) => showResult(result, expression, repeat),
        historyExpression: expression,
      },
    };
  }
  // Nothing left to calculate: a bare "5 =", or "√(9) =" whose square root
  // was already calculated (and recorded) when √ was pressed.
  return { state: showResult(state.value, `${operandText(state)} =`, null) };
}

function applyUnary(
  state: CalculatorState,
  operation: Operation,
  operands: number[],
  text: string,
): Transition {
  const base = state.phase === 'result' ? INITIAL_STATE : state;
  const expression = pendingPrefix(base) + text;
  return {
    state,
    compute: {
      operation,
      operands,
      expression,
      complete: (result) => ({
        ...base,
        entry: null,
        value: result,
        phase: 'operand',
        operandText: text,
        expression,
      }),
      // Only the unary part, not any pending "12 +" in front of it.
      historyExpression: `${text} =`,
    },
  };
}
