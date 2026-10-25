import { describe, expect, it } from 'vitest';
import type { CalculationRequest } from '../api/types';
import {
  INITIAL_STATE,
  MAX_DIGITS,
  displayText,
  failed,
  isBlockedByError,
  press,
  type CalculatorState,
  type Digit,
  type Key,
} from './engine';
import { displayNumber } from './format';

/** Stands in for the backend. */
function evaluate({ operation, operands: [a = NaN, b = NaN] }: CalculationRequest): number {
  switch (operation) {
    case 'add':
      return a + b;
    case 'subtract':
      return a - b;
    case 'multiply':
      return a * b;
    case 'divide':
      return a / b;
    case 'power':
      return a ** b;
    case 'sqrt':
      return Math.sqrt(a);
    case 'percentage':
      return (a / 100) * b;
  }
}

const TOKENS: Record<string, Key> = {
  '.': { type: 'decimal' },
  '+': { type: 'operator', operator: 'add' },
  '-': { type: 'operator', operator: 'subtract' },
  '*': { type: 'operator', operator: 'multiply' },
  '/': { type: 'operator', operator: 'divide' },
  '^': { type: 'operator', operator: 'power' },
  '=': { type: 'equals' },
  '%': { type: 'operator', operator: 'percentage' },
  sqrt: { type: 'sqrt' },
  sqr: { type: 'square' },
  neg: { type: 'negate' },
  bs: { type: 'backspace' },
  ce: { type: 'clearEntry' },
  c: { type: 'clear' },
};

function toKeys(tokens: string): Key[] {
  return tokens.split(' ').flatMap((token): Key[] => {
    if (/^\d+$/.test(token)) {
      return (token.match(/\d/g) ?? []).map((digit) => ({ type: 'digit', digit: digit as Digit }));
    }
    const key = TOKENS[token];
    if (!key) throw new Error(`unknown token ${token}`);
    return [key];
  });
}

interface Run {
  state: CalculatorState;
  requests: CalculationRequest[];
  recorded: string[];
}

/** Presses keys like a user, answering computations with `evaluate`. */
function run(tokens: string, start: CalculatorState = INITIAL_STATE): Run {
  const result: Run = { state: start, requests: [], recorded: [] };
  for (const key of toKeys(tokens)) {
    const transition = press(result.state, key);
    result.state = transition.state;
    if (transition.compute) {
      const { operation, operands, complete, historyExpression } = transition.compute;
      const value = evaluate({ operation, operands });
      result.requests.push({ operation, operands });
      result.state = complete(value);
      result.recorded.push(`${historyExpression} ${displayNumber(value)}`);
    }
  }
  return result;
}

function screen(tokens: string) {
  const { state } = run(tokens);
  return { display: displayText(state), expression: state.expression };
}

describe('typing numbers', () => {
  it.each([
    ['12 . 5', '12.5'],
    ['0 0 7', '7'],
    ['1 . . 2', '1.2'],
    ['.', '0.'],
    ['1234567', '1,234,567'],
    ['5 neg', '-5'],
    ['5 neg neg', '5'],
    ['0 neg', '0'],
    ['neg 5', '5'],
  ])('%s shows %s', (tokens, display) => {
    expect(screen(tokens).display).toBe(display);
  });

  it(`stops at ${MAX_DIGITS} digits`, () => {
    expect(screen('1234567890123456789').display).toBe('1,234,567,890,123,456');
  });

  it('does not count the sign or decimal point towards the limit', () => {
    expect(screen('123456789012345 . 6 7').display).toBe('123,456,789,012,345.6');
  });
});

describe('editing', () => {
  it.each([
    ['123 bs', '12'],
    ['5 bs', '0'],
    ['5 neg bs', '0'],
    ['1 . bs', '1'],
    ['5 + 3 ce', '0'],
    ['5 + 3 c', '0'],
  ])('%s shows %s', (tokens, display) => {
    expect(screen(tokens).display).toBe(display);
  });

  it('CE keeps the pending operation', () => {
    expect(screen('5 + 3 ce')).toEqual({ display: '0', expression: '5 +' });
    expect(screen('5 + 3 ce 2 =').display).toBe('7');
  });

  it('CE after a result starts over', () => {
    expect(screen('2 + 3 = ce')).toEqual({ display: '0', expression: '' });
  });

  it('C clears everything', () => {
    expect(run('5 + 3 c').state).toEqual(INITIAL_STATE);
  });

  it('backspace after a result only clears the expression', () => {
    expect(screen('2 + 3 = bs')).toEqual({ display: '5', expression: '' });
  });

  it('backspace does not edit a computed value', () => {
    expect(screen('9 sqrt bs').display).toBe('3');
  });
});

describe('binary operations', () => {
  it.each([
    ['2 + 3 =', '5', '2 + 3 ='],
    ['9 - 12 =', '-3', '9 − 12 ='],
    ['6 * 7 =', '42', '6 × 7 ='],
    ['7 / 2 =', '3.5', '7 ÷ 2 ='],
    ['2 ^ 10 =', '1,024', '2 ^ 10 ='],
  ])('%s → %s', (tokens, display, expression) => {
    expect(screen(tokens)).toEqual({ display, expression });
  });

  it('sends exactly the operands on screen to the backend', () => {
    expect(run('12 + 3 =').requests).toEqual([{ operation: 'add', operands: [12, 3] }]);
  });

  it('evaluates immediately when the next operator is pressed (2 + 3 × 4 = 20)', () => {
    const { state, requests } = run('2 + 3 * 4 =');
    expect(displayText(state)).toBe('20');
    expect(requests).toEqual([
      { operation: 'add', operands: [2, 3] },
      { operation: 'multiply', operands: [5, 4] },
    ]);
  });

  it('shows the intermediate result while chaining', () => {
    expect(screen('2 + 3 *')).toEqual({ display: '5', expression: '5 ×' });
  });

  it('replaces the operator when two are pressed in a row', () => {
    const { state, requests } = run('2 + - 3 =');
    expect(displayText(state)).toBe('-1');
    expect(requests).toHaveLength(1);
  });

  it('reuses the left operand when "=" follows an operator', () => {
    expect(screen('5 + =')).toEqual({ display: '10', expression: '5 + 5 =' });
  });

  it('repeats the last operation on repeated "="', () => {
    expect(screen('2 + 3 = =')).toEqual({ display: '8', expression: '5 + 3 =' });
    expect(screen('2 + 3 = = =').display).toBe('11');
  });

  it('starts a new calculation when typing after a result', () => {
    expect(screen('2 + 3 = 7')).toEqual({ display: '7', expression: '' });
    expect(screen('2 + 3 = 7 =').display).toBe('7');
  });

  it('continues from the result when an operator follows "="', () => {
    expect(screen('2 + 3 = * 2 =')).toEqual({ display: '10', expression: '5 × 2 =' });
  });

  it('keeps full precision between steps', () => {
    expect(screen('1 / 3 = * 3 =').display).toBe('1');
  });

  it('uses typed decimals as numbers', () => {
    expect(run('1 . 5 + 1 . =').requests).toEqual([{ operation: 'add', operands: [1.5, 1] }]);
  });
});

describe('unary operations', () => {
  it.each([
    ['9 sqrt', '3', '√(9)'],
    ['16 sqrt sqrt', '2', '√(√(16))'],
    ['3 sqr', '9', 'sqr(3)'],
    ['12 + 9 sqrt', '3', '12 + √(9)'],
    ['12 + 9 sqrt =', '15', '12 + √(9) ='],
    ['5 + neg', '-5', '5 + negate(5)'],
    ['2 + 3 = neg', '-5', 'negate(5)'],
    ['2 + 3 = sqr', '25', 'sqr(5)'],
    ['9 sqrt =', '3', '√(9) ='],
    ['9 sqrt 4', '4', ''],
    ['3 sqr + 1 =', '10', 'sqr(3) + 1 ='],
  ])('%s → %s with expression "%s"', (tokens, display, expression) => {
    expect(screen(tokens)).toEqual({ display, expression });
  });

  it('squares via the power operation', () => {
    expect(run('3 sqr').requests).toEqual([{ operation: 'power', operands: [3, 2] }]);
  });
});

describe('percent of', () => {
  it('computes a percent of b from two numbers', () => {
    expect(screen('15 % 200 =')).toEqual({ display: '30', expression: '15% of 200 =' });
    expect(run('15 % 200 =').requests).toEqual([{ operation: 'percentage', operands: [15, 200] }]);
  });

  it('waits for the second number after %', () => {
    expect(screen('15 %')).toEqual({ display: '15', expression: '15% of' });
    expect(run('15 %').requests).toEqual([]);
  });

  it('chains like any other operator', () => {
    expect(screen('200 + 15 %')).toEqual({ display: '215', expression: '215% of' });
    expect(screen('200 + 15 % 2 =').display).toBe('4.3');
    expect(screen('15 % 200 = + 5 =').display).toBe('35');
  });

  it('reuses the first number when "=" follows directly', () => {
    expect(screen('50 % =')).toEqual({ display: '25', expression: '50% of 50 =' });
  });

  it('repeats on repeated "="', () => {
    expect(screen('10 % 200 = =')).toEqual({ display: '40', expression: '20% of 200 =' });
  });

  it('can replace, or be replaced by, another operator', () => {
    expect(screen('15 + %')).toEqual({ display: '15', expression: '15% of' });
    expect(screen('15 % +')).toEqual({ display: '15', expression: '15 +' });
  });

  it('writes computed operands in the expression', () => {
    expect(screen('9 sqrt % 200 =')).toEqual({ display: '6', expression: '√(9)% of 200 =' });
  });

  it('leaves the negative-percent rule to the backend', () => {
    expect(run('10 neg % 50 =').requests).toEqual([
      { operation: 'percentage', operands: [-10, 50] },
    ]);
  });
});

// Every result the backend calculates is recorded, whether or not "=" is pressed.
describe('history records', () => {
  it.each([
    ['2 + 3 =', ['2 + 3 = 5']],
    ['9 - 12 =', ['9 − 12 = -3']],
    ['6 * 7 =', ['6 × 7 = 42']],
    ['7 / 2 =', ['7 ÷ 2 = 3.5']],
    ['2 ^ 10 =', ['2 ^ 10 = 1,024']],
    ['15 % 200 =', ['15% of 200 = 30']],
  ])('binary operation %s', (tokens, recorded) => {
    expect(run(tokens).recorded).toEqual(recorded);
  });

  it.each([
    ['9 sqrt', ['√(9) = 3']],
    ['3 sqr', ['sqr(3) = 9']],
    ['16 sqrt sqrt', ['√(16) = 4', '√(√(16)) = 2']],
    ['2 sqr sqr sqr', ['sqr(2) = 4', 'sqr(sqr(2)) = 16', 'sqr(sqr(sqr(2))) = 256']],
    ['81 sqrt sqr', ['√(81) = 9', 'sqr(√(81)) = 81']],
  ])('unary operation %s, without "="', (tokens, recorded) => {
    expect(run(tokens).recorded).toEqual(recorded);
  });

  it('records the unary part on its own, not the pending operation around it', () => {
    expect(run('12 + 9 sqrt =').recorded).toEqual(['√(9) = 3', '12 + √(9) = 15']);
  });

  it('records intermediate results of chained operators', () => {
    expect(run('2 + 3 * 4 =').recorded).toEqual(['2 + 3 = 5', '5 × 4 = 20']);
    expect(run('200 + 15 %').recorded).toEqual(['200 + 15 = 215']);
  });

  it('records each repeated "="', () => {
    expect(run('2 + 3 = =').recorded).toEqual(['2 + 3 = 5', '5 + 3 = 8']);
  });

  it('does not record a unary result twice when "=" follows', () => {
    expect(run('9 sqrt =').recorded).toEqual(['√(9) = 3']);
  });

  it.each(['5 =', '5 neg', '5 neg =', '1 2 bs', '5 + 3 ce', '2 + -'])(
    'records nothing when nothing is calculated: %s',
    (tokens) => {
      expect(run(tokens).recorded).toEqual([]);
    },
  );
});

describe('recall', () => {
  it('shows a history entry as the current result', () => {
    const recalled = press(INITIAL_STATE, { type: 'recall', expression: '2 + 3 =', value: 5 });
    expect(displayText(recalled.state)).toBe('5');
    expect(recalled.state.expression).toBe('2 + 3 =');
    expect(run('* 2 =', recalled.state).state.value).toBe(10);
    expect(displayText(run('7', recalled.state).state)).toBe('7');
  });
});

describe('errors', () => {
  const error = failed('1 ÷ 0 =', 'Division by zero.');

  it('shows the message and the failed expression', () => {
    expect(displayText(error)).toBe('Division by zero.');
    expect(error.expression).toBe('1 ÷ 0 =');
  });

  it.each(['+', '=', 'sqrt', 'sqr', '%', 'neg'])('ignores %s', (token) => {
    const transition = press(error, toKeys(token)[0]!);
    expect(transition.state).toBe(error);
    expect(transition.compute).toBeUndefined();
  });

  it.each([
    ['7', '7'],
    ['.', '0.'],
    ['bs', '0'],
    ['ce', '0'],
    ['c', '0'],
  ])('%s starts over', (token, display) => {
    const { state } = run(token, error);
    expect(state.error).toBeNull();
    expect(displayText(state)).toBe(display);
    expect(state.expression).toBe('');
  });

  it('identifies the keys to disable', () => {
    const blocked = Object.entries(TOKENS)
      .filter(([, key]) => isBlockedByError(key))
      .map(([token]) => token);
    expect(blocked.sort()).toEqual(['%', '*', '+', '-', '/', '=', '^', 'neg', 'sqr', 'sqrt']);
  });
});
