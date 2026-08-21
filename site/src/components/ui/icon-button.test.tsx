import { describe, expect, it, vi } from 'vitest';
import { createRef } from 'react';
import { render, screen, userEvent } from '@/test/render';
import { IconButton, iconButtonVariants } from './icon-button';

const VARIANTS = ['filled', 'filled-tonal', 'outlined', 'standard'] as const;
const SIZES = ['xs', 's', 'm', 'l', 'xl'] as const;

describe('IconButton', () => {
  it('renders a button and fires its click', async () => {
    const onClick = vi.fn();
    render(<IconButton aria-label="pause" onClick={onClick} />);
    await userEvent.click(screen.getByRole('button', { name: 'pause' }));
    expect(onClick).toHaveBeenCalled();
  });

  it('renders as its child when asChild is set', () => {
    render(
      <IconButton asChild>
        <a href="https://example.com" aria-label="open" />
      </IconButton>,
    );
    expect(screen.getByRole('link', { name: 'open' }).className).toContain('inline-flex');
  });

  it('gives every variant and size its own classes', () => {
    expect(new Set(VARIANTS.map((variant) => iconButtonVariants({ variant }))).size).toBe(VARIANTS.length);
    expect(new Set(SIZES.map((size) => iconButtonVariants({ size }))).size).toBe(SIZES.length);
  });

  it('morphs from pill to a per-size square', () => {
    expect(iconButtonVariants({ shape: 'round' })).toContain('rounded-pill');
    for (const size of SIZES) {
      expect(iconButtonVariants({ shape: 'square', size })).toMatch(/rounded-(md|lg|2xl|3xl)/);
    }
  });

  it('forwards ref and className', () => {
    const ref = createRef<HTMLButtonElement>();
    render(<IconButton ref={ref} className="ms-1" aria-label="x" />);
    expect(ref.current).toHaveClass('ms-1');
  });
});
