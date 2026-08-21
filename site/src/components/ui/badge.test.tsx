import { describe, expect, it } from 'vitest';
import { render, screen } from '@/test/render';
import { Badge, badgeVariants } from './badge';

const VARIANTS = ['default', 'primary', 'outline', 'success', 'warning', 'info', 'destructive', 'mono'] as const;

describe('Badge', () => {
  it('renders its children in a span', () => {
    render(<Badge>done</Badge>);
    expect(screen.getByText('done').tagName).toBe('SPAN');
  });

  it('gives every variant its own classes', () => {
    const seen = new Set(VARIANTS.map((variant) => badgeVariants({ variant })));
    expect(seen.size).toBe(VARIANTS.length);
  });

  it('sizes sm and md differently', () => {
    expect(badgeVariants({ size: 'sm' })).not.toBe(badgeVariants({ size: 'md' }));
  });

  it('merges a caller className and forwards other props', () => {
    render(
      <Badge className="mt-2" variant="success" size="sm" title="tip" data-testid="b">
        ok
      </Badge>,
    );
    const el = screen.getByTestId('b');
    expect(el).toHaveClass('mt-2');
    expect(el).toHaveAttribute('title', 'tip');
  });
});
