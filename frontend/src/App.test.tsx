import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import App from './App';
import { instantBackend } from './test/fakes';

describe('App', () => {
  it('wires the calculator to the injected API', async () => {
    const calculate = instantBackend();
    const user = userEvent.setup();
    render(<App api={{ calculate }} />);

    expect(screen.getByRole('heading', { name: 'Calculator' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'API docs' })).toHaveAttribute('href', '/docs/');
    for (const name of ['3', 'Add', '4', 'Equals']) {
      await user.click(screen.getByRole('button', { name }));
    }

    expect(screen.getByLabelText('Display')).toHaveTextContent('7');
    expect(calculate).toHaveBeenCalledOnce();
  });
});
