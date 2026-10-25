import type { HistoryEntry } from '../hooks/useCalculator';
import { displayNumber } from '../lib/format';

interface HistoryProps {
  entries: readonly HistoryEntry[];
  onRecall: (entry: HistoryEntry) => void;
  onClear: () => void;
}

export function History({ entries, onRecall, onClear }: HistoryProps) {
  return (
    <section className="history" aria-labelledby="history-heading">
      <div className="history-header">
        <h2 id="history-heading">History</h2>
        {entries.length > 0 && (
          <button
            type="button"
            className="history-clear"
            aria-label="Clear history"
            onClick={onClear}
          >
            <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
              <path
                d="M6 2h4M2.5 4h11M4 4l.7 9.1a1 1 0 0 0 1 .9h4.6a1 1 0 0 0 1-.9L12 4M6.5 6.5v5M9.5 6.5v5"
                fill="none"
                stroke="currentColor"
                strokeLinecap="round"
                strokeWidth="1.2"
              />
            </svg>
          </button>
        )}
      </div>
      {entries.length === 0 ? (
        <p className="history-empty">There&apos;s no history yet.</p>
      ) : (
        <ol className="history-list">
          {entries.map((entry) => (
            <li key={entry.id}>
              <button
                type="button"
                className="history-item"
                onClick={() => {
                  onRecall(entry);
                }}
              >
                <span className="history-expression">{entry.expression}</span>
                <span className="history-value">{displayNumber(entry.value)}</span>
              </button>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
