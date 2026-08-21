import { describe, expect, it, vi } from 'vitest';
import { createRef } from 'react';
import { render, screen, userEvent } from '@/test/render';
import { Button, buttonVariants } from './button';

const VARIANTS = ['filled', 'filled-tonal', 'outlined', 'text', 'elevated', 'destructive', 'ghost'] as const;
const SIZES = ['xs', 's', 'm', 'l', 'xl'] as const;

describe('Button', () => {
  it('renders a button and fires its click', async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Add</Button>);
    await userEvent.click(screen.getByRole('button', { name: 'Add' }));
    expect(onClick).toHaveBeenCalled();
  });

  it('renders as its child when asChild is set', () => {
    render(
      <Button asChild>
        <a href="https://example.com">open</a>
      </Button>,
    );
    const link = screen.getByRole('link', { name: 'open' });
    expect(link.tagName).toBe('A');
    expect(link.className).toContain('inline-flex'); // took the button classes
    expect(screen.queryByRole('button')).toBeNull();
  });

  it('gives every variant and size its own classes', () => {
    expect(new Set(VARIANTS.map((variant) => buttonVariants({ variant }))).size).toBe(VARIANTS.length);
    expect(new Set(SIZES.map((size) => buttonVariants({ size }))).size).toBe(SIZES.length);
  });

  it('tightens padding for text buttons at the small sizes', () => {
    expect(buttonVariants({ variant: 'text', size: 'xs' })).toContain('px-3');
    expect(buttonVariants({ variant: 'text', size: 's' })).toContain('px-4');
    expect(buttonVariants({ variant: 'text', size: 'm' })).toContain('px-6');
    // Only text does this — a filled button keeps its own padding.
    expect(buttonVariants({ variant: 'filled', size: 'xs' })).not.toContain('px-3');
  });

  it('squares the corners per size when shape is square', () => {
    for (const size of SIZES) {
      expect(buttonVariants({ shape: 'square', size })).toMatch(/!rounded-/);
    }
    expect(buttonVariants({ shape: 'round', size: 's' })).toContain('rounded-pill');
  });

  it('forwards ref, className and disabled', () => {
    const ref = createRef<HTMLButtonElement>();
    render(
      <Button ref={ref} className="w-full" disabled>
        x
      </Button>,
    );
    expect(ref.current).toBeInstanceOf(HTMLButtonElement);
    expect(ref.current).toHaveClass('w-full');
    expect(screen.getByRole('button')).toBeDisabled();
  });
});
