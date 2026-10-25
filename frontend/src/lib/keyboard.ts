import type { BinaryOperator, Digit, Key } from './engine';

const operator = (op: BinaryOperator): Key => ({ type: 'operator', operator: op });

/** Shortcuts follow the Windows calculator where it has one. */
const SHORTCUTS: Readonly<Record<string, Key>> = {
  '.': { type: 'decimal' },
  ',': { type: 'decimal' },
  '+': operator('add'),
  '-': operator('subtract'),
  '*': operator('multiply'),
  '/': operator('divide'),
  '^': operator('power'),
  '=': { type: 'equals' },
  Enter: { type: 'equals' },
  '%': operator('percentage'),
  '@': { type: 'sqrt' },
  q: { type: 'square' },
  F9: { type: 'negate' },
  Backspace: { type: 'backspace' },
  Delete: { type: 'clearEntry' },
  Escape: { type: 'clear' },
};

type KeyEventLike = Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'metaKey' | 'altKey'>;

/** Maps a keyboard event to a calculator key, or null if it is not a shortcut. */
export function keyFromKeyboard(event: KeyEventLike): Key | null {
  if (event.ctrlKey || event.metaKey || event.altKey) return null;
  if (/^\d$/.test(event.key)) return { type: 'digit', digit: event.key as Digit };
  return SHORTCUTS[event.key] ?? null;
}
