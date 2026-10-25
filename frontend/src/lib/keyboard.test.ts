import { describe, expect, it } from 'vitest';
import { keyFromKeyboard } from './keyboard';

const event = (
  key: string,
  modifiers: Partial<Record<'ctrlKey' | 'metaKey' | 'altKey', boolean>> = {},
) => ({
  key,
  ctrlKey: false,
  metaKey: false,
  altKey: false,
  ...modifiers,
});

describe('keyFromKeyboard', () => {
  it.each([
    ['7', { type: 'digit', digit: '7' }],
    ['.', { type: 'decimal' }],
    [',', { type: 'decimal' }],
    ['+', { type: 'operator', operator: 'add' }],
    ['-', { type: 'operator', operator: 'subtract' }],
    ['*', { type: 'operator', operator: 'multiply' }],
    ['/', { type: 'operator', operator: 'divide' }],
    ['^', { type: 'operator', operator: 'power' }],
    ['Enter', { type: 'equals' }],
    ['=', { type: 'equals' }],
    ['%', { type: 'operator', operator: 'percentage' }],
    ['@', { type: 'sqrt' }],
    ['q', { type: 'square' }],
    ['F9', { type: 'negate' }],
    ['Backspace', { type: 'backspace' }],
    ['Delete', { type: 'clearEntry' }],
    ['Escape', { type: 'clear' }],
  ])('maps %s', (key, expected) => {
    expect(keyFromKeyboard(event(key))).toEqual(expected);
  });

  it.each(['a', 'Tab', ' ', 'ArrowUp', '10'])('ignores %j', (key) => {
    expect(keyFromKeyboard(event(key))).toBeNull();
  });

  it('ignores shortcuts with modifiers, such as Ctrl+C', () => {
    expect(keyFromKeyboard(event('c', { ctrlKey: true }))).toBeNull();
    expect(keyFromKeyboard(event('1', { metaKey: true }))).toBeNull();
    expect(keyFromKeyboard(event('1', { altKey: true }))).toBeNull();
  });
});
