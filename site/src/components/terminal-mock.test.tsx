import { describe, expect, it } from 'vitest';
import { render, screen } from '@/test/render';
import { TerminalMock } from './terminal-mock';

describe('TerminalMock', () => {
  it('frames its children with window chrome and a default title', () => {
    const { container } = render(
      <TerminalMock>
        <p>rows</p>
      </TerminalMock>,
    );
    expect(screen.getByText('rows')).toBeInTheDocument();
    expect(screen.getByText(/Aria2t/)).toBeInTheDocument();
    // The three traffic-light dots are decorative.
    expect(container.querySelectorAll('[aria-hidden]')).toHaveLength(3);
    // Always LTR: a terminal's own layout does not mirror.
    expect(container.firstElementChild).toHaveAttribute('dir', 'ltr');
  });

  it('takes a title and a className', () => {
    const { container } = render(
      <TerminalMock title="custom title" className="max-w-md">
        <p>rows</p>
      </TerminalMock>,
    );
    expect(screen.getByText('custom title')).toBeInTheDocument();
    expect(container.firstElementChild).toHaveClass('max-w-md');
  });
});
