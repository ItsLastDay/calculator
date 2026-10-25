import { useEffect } from 'react';
import type { Key } from '../lib/engine';
import { keyFromKeyboard } from '../lib/keyboard';

/** Sends keyboard shortcuts anywhere on the page to `onKey`. */
export function useKeyboard(onKey: (key: Key) => void) {
  useEffect(() => {
    function handle(event: KeyboardEvent) {
      // Enter on a focused button (e.g. "Clear history") activates that
      // button, as keyboard users expect, instead of meaning "=".
      if (event.key === 'Enter' && event.target instanceof HTMLButtonElement) return;
      const key = keyFromKeyboard(event);
      if (!key) return;
      event.preventDefault();
      onKey(key);
    }
    window.addEventListener('keydown', handle);
    return () => {
      window.removeEventListener('keydown', handle);
    };
  }, [onKey]);
}
