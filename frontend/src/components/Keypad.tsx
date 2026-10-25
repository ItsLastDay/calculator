import type { MouseEvent } from 'react';
import { isBlockedByError, type Digit, type Key } from '../lib/engine';

interface KeyDefinition {
  label: string;
  /** Accessible name. */
  name: string;
  key: Key;
  kind: 'number' | 'function' | 'operator' | 'equals';
}

const digit = (d: Digit): KeyDefinition => ({
  label: d,
  name: d,
  key: { type: 'digit', digit: d },
  kind: 'number',
});

// The Windows Standard layout without the memory row; xʸ takes the place of 1/x.
const KEYS: readonly KeyDefinition[] = [
  {
    label: '%',
    name: 'Percent of',
    key: { type: 'operator', operator: 'percentage' },
    kind: 'function',
  },
  { label: 'CE', name: 'Clear entry', key: { type: 'clearEntry' }, kind: 'function' },
  { label: 'C', name: 'Clear', key: { type: 'clear' }, kind: 'function' },
  { label: '⌫', name: 'Backspace', key: { type: 'backspace' }, kind: 'function' },
  { label: 'xʸ', name: 'Power', key: { type: 'operator', operator: 'power' }, kind: 'function' },
  { label: 'x²', name: 'Square', key: { type: 'square' }, kind: 'function' },
  { label: '√x', name: 'Square root', key: { type: 'sqrt' }, kind: 'function' },
  { label: '÷', name: 'Divide', key: { type: 'operator', operator: 'divide' }, kind: 'operator' },
  digit('7'),
  digit('8'),
  digit('9'),
  {
    label: '×',
    name: 'Multiply',
    key: { type: 'operator', operator: 'multiply' },
    kind: 'operator',
  },
  digit('4'),
  digit('5'),
  digit('6'),
  {
    label: '−',
    name: 'Subtract',
    key: { type: 'operator', operator: 'subtract' },
    kind: 'operator',
  },
  digit('1'),
  digit('2'),
  digit('3'),
  { label: '+', name: 'Add', key: { type: 'operator', operator: 'add' }, kind: 'operator' },
  { label: '+/−', name: 'Negate', key: { type: 'negate' }, kind: 'number' },
  digit('0'),
  { label: '.', name: 'Decimal point', key: { type: 'decimal' }, kind: 'number' },
  { label: '=', name: 'Equals', key: { type: 'equals' }, kind: 'equals' },
];

// Clicking a key must not move focus to it; otherwise a later Enter would
// press that key again instead of meaning "=".
function keepFocus(event: MouseEvent) {
  event.preventDefault();
}

interface KeypadProps {
  onPress: (key: Key) => void;
  hasError: boolean;
}

export function Keypad({ onPress, hasError }: KeypadProps) {
  return (
    <div className="keypad" role="group" aria-label="Keypad">
      {KEYS.map(({ label, name, key, kind }) => (
        <button
          key={name}
          type="button"
          className={`key key-${kind}`}
          aria-label={name}
          disabled={hasError && isBlockedByError(key)}
          onMouseDown={keepFocus}
          onClick={() => {
            onPress(key);
          }}
        >
          {label}
        </button>
      ))}
    </div>
  );
}
