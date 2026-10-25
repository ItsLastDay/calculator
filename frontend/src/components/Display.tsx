interface DisplayProps {
  expression: string;
  value: string;
  error: boolean;
  busy: boolean;
}

/** Long numbers and messages shrink to stay readable, like the Windows calculator. */
function sizeClass(text: string, error: boolean): string {
  if (error) return 'display-value-error';
  if (text.length > 18) return 'display-value-small';
  if (text.length > 12) return 'display-value-medium';
  return '';
}

export function Display({ expression, value, error, busy }: DisplayProps) {
  return (
    <div className="display" aria-busy={busy}>
      <p className="display-expression" data-testid="expression">
        {expression}
      </p>
      <output
        className={`display-value ${sizeClass(value, error)}`}
        aria-label="Display"
        aria-live="polite"
      >
        {value}
      </output>
      <div className="display-progress" hidden={!busy} data-testid="busy" />
    </div>
  );
}
