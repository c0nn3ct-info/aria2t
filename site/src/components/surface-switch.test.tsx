import { describe, expect, it, vi } from 'vitest';
import { render, screen, userEvent } from '@/test/render';
import { SurfaceSwitch } from './surface-switch';

describe('SurfaceSwitch', () => {
  it('is a group of toggle buttons, not a tablist', async () => {
    // Nothing it renders controls a tabpanel, so aria-pressed is the right
    // contract (extension parity). Query by pressed state, not by tab role.
    render(<SurfaceSwitch value="extension" onChange={() => {}} />);
    expect(screen.getByRole('group')).toBeInTheDocument();
    expect(screen.queryByRole('tab')).toBeNull();

    const buttons = screen.getAllByRole('button');
    expect(buttons).toHaveLength(2);
    expect(buttons.filter((b) => b.getAttribute('aria-pressed') === 'true')).toHaveLength(1);
  });

  it('marks the chosen surface and reports a change', async () => {
    const onChange = vi.fn();
    render(<SurfaceSwitch value="extension" onChange={onChange} />);

    const terminal = screen.getByRole('button', { name: /terminal/i });
    expect(terminal).toHaveAttribute('aria-pressed', 'false');
    await userEvent.click(terminal);
    expect(onChange).toHaveBeenCalledWith('terminal');
  });

  it('follows the value it is given', () => {
    const { rerender } = render(<SurfaceSwitch value="terminal" onChange={() => {}} />);
    expect(screen.getByRole('button', { name: /terminal/i })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    rerender(<SurfaceSwitch value="extension" onChange={() => {}} />);
    expect(screen.getByRole('button', { name: /terminal/i })).toHaveAttribute(
      'aria-pressed',
      'false',
    );
  });
});
