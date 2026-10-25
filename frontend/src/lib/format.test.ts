import { describe, expect, it } from 'vitest';
import { displayNumber, formatNumber, groupDigits } from './format';

describe('formatNumber', () => {
  it.each([
    [5, '5'],
    [-12, '-12'],
    [0, '0'],
    [2.5, '2.5'],
    [0.1 + 0.2, '0.3'],
    [1 / 3, '0.333333333333333'],
    [1234567890123, '1234567890123'],
    [1e25, '1e+25'],
    [1.5e-10, '1.5e-10'],
  ])('formats %d as %s', (value, expected) => {
    expect(formatNumber(value)).toBe(expected);
  });
});

describe('groupDigits', () => {
  it.each([
    ['0', '0'],
    ['999', '999'],
    ['1000', '1,000'],
    ['1234567', '1,234,567'],
    ['-1234567.891', '-1,234,567.891'],
    ['1234.', '1,234.'],
    ['0.12345', '0.12345'],
    ['-0', '-0'],
    ['1e+25', '1e+25'],
  ])('groups %s as %s', (input, expected) => {
    expect(groupDigits(input)).toBe(expected);
  });
});

describe('displayNumber', () => {
  it('formats and groups', () => {
    expect(displayNumber(1234567.25)).toBe('1,234,567.25');
  });
});
