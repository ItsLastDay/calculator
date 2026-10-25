// Significant digits shown for non-integers. Fifteen is the most a double
// represents exactly in decimal, and it hides binary floating point noise
// (0.1 + 0.2 = 0.30000000000000004).
const DISPLAY_PRECISION = 15;

/** Formats a number without floating point noise, e.g. 0.30000000000000004 → "0.3". */
export function formatNumber(value: number): string {
  if (Number.isInteger(value) && Math.abs(value) < 1e21) {
    return value.toString();
  }
  return Number(value.toPrecision(DISPLAY_PRECISION)).toString();
}

/** Adds thousands separators to a plain decimal string: "-1234567.5" → "-1,234,567.5". */
export function groupDigits(text: string): string {
  if (/e/i.test(text)) return text;
  const sign = text.startsWith('-') ? '-' : '';
  const [integer = '', fraction] = text.slice(sign.length).split('.');
  const grouped = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return sign + grouped + (fraction === undefined ? '' : `.${fraction}`);
}

/** Formats a number for display: noise-free and digit-grouped. */
export function displayNumber(value: number): string {
  return groupDigits(formatNumber(value));
}
