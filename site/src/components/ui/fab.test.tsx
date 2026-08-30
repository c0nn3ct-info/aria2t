import { describe, expect, it, vi } from 'vitest';
import { createRef } from 'react';
import { render, screen, userEvent } from '@/test/render';
import { Fab, fabVariants } from './fab';

const COLORS = ['primary', 'surface', 'secondary', 'tertiary', 'success', 'error'] as const;
const SIZES = ['small', 'regular', 'large'] as const;

describe('Fab', () => {
  it('renders a button and fires its click', async () => {
    const onClick = vi.fn();
    render(<Fab aria-label="pause all" onClick={onClick} />);
    await userEvent.click(screen.getByRole('button', { name: 'pause all' }));
    expect(onClick).toHaveBeenCalled();
  });

  it('renders as its child when asChild is set', () => {
    render(
      <Fab asChild>
        <a href="https://example.com" aria-label="open" />
      </Fab>,
    );
    expect(screen.getByRole('link', { name: 'open' }).className).toContain('inline-flex');
  });

  it('gives every colour and size its own classes', () => {
    expect(new Set(COLORS.map((color) => fabVariants({ color }))).size).toBe(COLORS.length);
    expect(new Set(SIZES.map((size) => fabVariants({ size }))).size).toBe(SIZES.length);
  });

  it('defaults to the regular primary FAB', () => {
    const base = fabVariants();
    expect(base).toContain('bg-primary-container');
    expect(base).toContain('h-14 w-14');
  });

  it('forwards ref and className', () => {
    const ref = createRef<HTMLButtonElement>();
    render(<Fab ref={ref} className="ms-1" aria-label="x" />);
    expect(ref.current).toHaveClass('ms-1');
  });
});
