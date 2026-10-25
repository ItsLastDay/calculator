import { act, fireEvent, render, screen } from '@testing-library/react';
import userEvent, { type UserEvent } from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { ApiError, NETWORK_ERROR, type CalculateFn } from '../api/client';
import { REQUEST_TIMEOUT_MS } from '../hooks/useCalculator';
import { instantBackend, manualBackend } from '../test/fakes';
import { Calculator } from './Calculator';

function setup(calculate: CalculateFn = instantBackend()) {
  const user = userEvent.setup();
  const view = render(<Calculator calculate={calculate} />);
  return { user, calculate, ...view };
}

const display = () => screen.getByLabelText('Display');
const expression = () => screen.getByTestId('expression');
const key = (name: string) => screen.getByRole('button', { name });

async function click(user: UserEvent, ...names: string[]) {
  for (const name of names) await user.click(key(name));
}

/** Settles a fake response and lets React and the key queue catch up. */
async function settle(action: () => void) {
  await act(async () => {
    action();
    for (let i = 0; i < 20; i++) await Promise.resolve();
  });
}

describe('Calculator', () => {
  it('computes through the backend and shows the expression', async () => {
    const { user, calculate } = setup();

    await click(user, '1', '2', 'Add', '3', 'Equals');

    expect(display()).toHaveTextContent('15');
    expect(expression()).toHaveTextContent('12 + 3 =');
    expect(calculate).toHaveBeenCalledWith(
      { operation: 'add', operands: [12, 3] },
      expect.any(AbortSignal),
    );
  });

  it('evaluates left to right like the Windows calculator', async () => {
    const { user } = setup();

    await click(user, '2', 'Add', '3', 'Multiply', '4', 'Equals');

    expect(display()).toHaveTextContent('20');
  });

  it('computes a percent of b from two numbers', async () => {
    const { user, calculate } = setup();

    await click(user, '1', '5', 'Percent of');
    expect(expression()).toHaveTextContent('15% of');

    await click(user, '2', '0', '0', 'Equals');
    expect(display()).toHaveTextContent('30');
    expect(expression()).toHaveTextContent('15% of 200 =');
    expect(calculate).toHaveBeenCalledWith(
      { operation: 'percentage', operands: [15, 200] },
      expect.any(AbortSignal),
    );
  });

  it('computes percent of from the keyboard', async () => {
    const { user } = setup();

    await user.keyboard('15%200{Enter}');

    expect(display()).toHaveTextContent('30');
  });

  it('explains when the percent is negative', async () => {
    const calculate = vi
      .fn<CalculateFn>()
      .mockRejectedValue(
        new ApiError('Percentage must not be negative.', 'NEGATIVE_PERCENTAGE', 422),
      );
    const { user } = setup(calculate);

    await click(user, '1', '0', 'Negate', 'Percent of', '5', '0', 'Equals');

    expect(display()).toHaveTextContent('Percentage must not be negative.');
    expect(expression()).toHaveTextContent('-10% of 50 =');
  });

  it('shows the backend explanation for a negative square root', async () => {
    const calculate = vi
      .fn<CalculateFn>()
      .mockRejectedValue(
        new ApiError(
          'Square root of a negative number is not a real number.',
          'NEGATIVE_SQUARE_ROOT',
          422,
        ),
      );
    const { user } = setup(calculate);

    await click(user, '5', 'Negate', 'Square root');

    expect(display()).toHaveTextContent('Square root of a negative number is not a real number.');
    expect(expression()).toHaveTextContent('√(-5)');
    expect(calculate).toHaveBeenCalledWith(
      { operation: 'sqrt', operands: [-5] },
      expect.any(AbortSignal),
    );
  });

  it('supports unary keys', async () => {
    const { user } = setup();

    await click(user, '2', '3', '0', 'Square root');

    expect(display()).toHaveTextContent('15.1657508881031');
    expect(expression()).toHaveTextContent('√(230)');
  });

  it('records each repeated square root in the history', async () => {
    const { user } = setup();

    await click(user, '2', '5', '6', 'Square root', 'Square root', 'Square root');

    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(3);
    expect(items[0]).toHaveTextContent('√(√(√(256))) =2');
    expect(items[1]).toHaveTextContent('√(√(256)) =4');
    expect(items[2]).toHaveTextContent('√(256) =16');
  });

  it('records squares and intermediate results without "="', async () => {
    const { user } = setup();

    await click(user, '3', 'Square', 'Add', '1', 'Multiply');

    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent('sqr(3) + 1 =10');
    expect(items[1]).toHaveTextContent('sqr(3) =9');
  });

  it('works from the keyboard', async () => {
    const { user } = setup();

    await user.keyboard('12*3{Enter}');
    expect(display()).toHaveTextContent('36');

    await user.keyboard('{Escape}');
    expect(display()).toHaveTextContent('0');

    await user.keyboard('45{Backspace}');
    expect(display()).toHaveTextContent('4');
  });

  it('treats Enter as "=" after clicking keys, since keys do not take focus', async () => {
    const { user } = setup();

    await click(user, '2', 'Add', '3');
    expect(key('3')).not.toHaveFocus();
    await user.keyboard('{Enter}');

    expect(display()).toHaveTextContent('5');
  });

  it('shows friendly errors and disables keys that cannot apply', async () => {
    const calculate = vi
      .fn<CalculateFn>()
      .mockRejectedValue(new ApiError('Division by zero.', 'DIVISION_BY_ZERO', 422));
    const { user } = setup(calculate);

    await click(user, '1', 'Divide', '0', 'Equals');

    expect(display()).toHaveTextContent('Division by zero.');
    expect(expression()).toHaveTextContent('1 ÷ 0 =');
    for (const name of ['Add', 'Equals', 'Square root', 'Percent of', 'Negate']) {
      expect(key(name)).toBeDisabled();
    }
    expect(key('5')).toBeEnabled();

    await click(user, '5');
    expect(display()).toHaveTextContent('5');
    expect(key('Add')).toBeEnabled();
  });

  it('shows the API message for other failures', async () => {
    const calculate = vi
      .fn<CalculateFn>()
      .mockRejectedValue(new ApiError('Could not reach the calculator service.', NETWORK_ERROR, 0));
    const { user } = setup(calculate);

    await click(user, '1', 'Add', '1', 'Equals');

    expect(display()).toHaveTextContent('Could not reach the calculator service.');
  });

  it('shows a generic message for unexpected errors', async () => {
    const calculate = vi.fn<CalculateFn>().mockRejectedValue(new Error('boom'));
    const { user } = setup(calculate);

    await click(user, '1', 'Add', '1', 'Equals');

    expect(display()).toHaveTextContent('Something went wrong');
  });

  it('records history, recalls entries, and clears it', async () => {
    const { user } = setup();
    expect(screen.getByText("There's no history yet.")).toBeInTheDocument();

    await click(user, '2', 'Add', '3', 'Equals', '9', 'Multiply', '9', 'Equals');

    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent('9 × 9 =81');
    expect(items[1]).toHaveTextContent('2 + 3 =5');

    await user.click(screen.getByRole('button', { name: '2 + 3 = 5' }));
    expect(display()).toHaveTextContent('5');
    expect(expression()).toHaveTextContent('2 + 3 =');

    await user.click(key('Clear history'));
    expect(screen.queryAllByRole('listitem')).toHaveLength(0);
  });

  it('lets Enter activate a focused button instead of meaning "="', async () => {
    const { user, calculate } = setup();
    await click(user, '2', 'Add', '3', 'Equals');

    key('Clear history').focus();
    await user.keyboard('{Enter}');

    expect(screen.queryAllByRole('listitem')).toHaveLength(0);
    expect(calculate).toHaveBeenCalledOnce();
  });

  it('shows progress while the backend works', async () => {
    const { calculate, calls } = manualBackend();
    const { user } = setup(calculate);

    expect(screen.getByTestId('busy')).not.toBeVisible();
    await click(user, '1', 'Add', '2', 'Equals');
    expect(screen.getByTestId('busy')).toBeVisible();

    await settle(() => {
      calls[0]!.resolve(3);
    });
    expect(screen.getByTestId('busy')).not.toBeVisible();
    expect(display()).toHaveTextContent('3');
  });
});

// The fake backend here ignores abort signals, so these tests prove that
// ordering is enforced by the app itself rather than by fetch.
describe('Calculator key ordering', () => {
  it('applies keys pressed during a request, in order, after it completes', async () => {
    const { calculate, calls } = manualBackend();
    const { user } = setup(calculate);

    await click(user, '2', 'Add', '3', 'Multiply', '4', 'Equals');

    // Only the first request is out; "4" and "=" wait their turn.
    expect(calls).toHaveLength(1);
    expect(calls[0]!.request).toEqual({ operation: 'add', operands: [2, 3] });

    await settle(() => {
      calls[0]!.resolve(5);
    });
    expect(calls).toHaveLength(2);
    expect(calls[1]!.request).toEqual({ operation: 'multiply', operands: [5, 4] });

    await settle(() => {
      calls[1]!.resolve(20);
    });
    expect(display()).toHaveTextContent('20');
    expect(expression()).toHaveTextContent('5 × 4 =');
  });

  it('C cancels the request and drops keys queued behind it', async () => {
    const { calculate, calls } = manualBackend();
    const { user } = setup(calculate);

    await click(user, '2', 'Add', '3', 'Equals', '7', 'Clear');

    expect(display()).toHaveTextContent('0');
    expect(calls[0]!.signal?.aborted).toBe(true);

    await settle(() => {
      calls[0]!.resolve(5);
    });
    expect(display()).toHaveTextContent('0');
    expect(expression()).toBeEmptyDOMElement();
    expect(screen.queryAllByRole('listitem')).toHaveLength(0);
  });

  it('never lets a response from before C override a newer result', async () => {
    const { calculate, calls } = manualBackend();
    const { user } = setup(calculate);

    await click(user, '2', 'Add', '3', 'Equals', 'Clear', '4', 'Add', '4', 'Equals');
    expect(calls).toHaveLength(2);

    await settle(() => {
      calls[1]!.resolve(8);
    });
    await settle(() => {
      calls[0]!.resolve(5);
    });

    expect(display()).toHaveTextContent('8');
    expect(screen.getAllByRole('listitem')).toHaveLength(1);
  });

  it('never lets an error from before C replace a newer result', async () => {
    const { calculate, calls } = manualBackend();
    const { user } = setup(calculate);

    await click(user, '1', 'Divide', '0', 'Equals', 'Clear', '4', 'Add', '4', 'Equals');
    await settle(() => {
      calls[1]!.resolve(8);
    });
    await settle(() => {
      calls[0]!.reject(new ApiError('Division by zero.', 'DIVISION_BY_ZERO', 422));
    });

    expect(display()).toHaveTextContent('8');
    expect(key('Add')).toBeEnabled();
  });

  it('aborts the request on unmount and applies nothing afterwards', async () => {
    const { calculate, calls } = manualBackend();
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
    const { user, unmount } = setup(calculate);

    await click(user, '1', 'Add', '1', 'Equals');
    unmount();

    expect(calls[0]!.signal?.aborted).toBe(true);
    await settle(() => {
      calls[0]!.resolve(2);
    });
    expect(consoleError).not.toHaveBeenCalled();
    consoleError.mockRestore();
  });

  it('times out a request that never answers and releases the queue', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    try {
      const { calculate, calls } = manualBackend();
      render(<Calculator calculate={calculate} />);

      for (const name of ['1', 'Add', '2', 'Equals', '9']) fireEvent.click(key(name));
      await settle(() => undefined);
      expect(calls).toHaveLength(1);

      await settle(() => {
        vi.advanceTimersByTime(REQUEST_TIMEOUT_MS);
      });

      expect(calls[0]!.signal?.aborted).toBe(true);
      // The queued "9" ran after the timeout and started a fresh entry.
      expect(display()).toHaveTextContent('9');

      await settle(() => {
        calls[0]!.resolve(3);
      });
      expect(display()).toHaveTextContent('9');
    } finally {
      vi.useRealTimers();
    }
  });

  it('reports the timeout when nothing else was pressed', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] });
    try {
      const { calculate } = manualBackend();
      render(<Calculator calculate={calculate} />);

      for (const name of ['1', 'Add', '2', 'Equals']) fireEvent.click(key(name));
      await settle(() => undefined);
      await settle(() => {
        vi.advanceTimersByTime(REQUEST_TIMEOUT_MS);
      });

      expect(display()).toHaveTextContent('The calculator service did not respond');
    } finally {
      vi.useRealTimers();
    }
  });
});
