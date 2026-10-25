import { useCallback } from 'react';
import type { CalculateFn } from '../api/client';
import { useCalculator, type HistoryEntry } from '../hooks/useCalculator';
import { useKeyboard } from '../hooks/useKeyboard';
import { Display } from './Display';
import { History } from './History';
import { Keypad } from './Keypad';

interface CalculatorProps {
  calculate: CalculateFn;
}

export function Calculator({ calculate }: CalculatorProps) {
  const calc = useCalculator(calculate);
  const { press } = calc;
  useKeyboard(press);

  const recall = useCallback(
    ({ expression, value }: HistoryEntry) => {
      press({ type: 'recall', expression, value });
    },
    [press],
  );

  return (
    <div className="calculator">
      <section className="panel" aria-label="Standard calculator">
        <Display
          expression={calc.expression}
          value={calc.display}
          error={calc.hasError}
          busy={calc.busy}
        />
        <Keypad onPress={press} hasError={calc.hasError} />
      </section>
      <History entries={calc.history} onRecall={recall} onClear={calc.clearHistory} />
    </div>
  );
}
