import { describe, expect, it } from 'vitest';
import { render } from '@/test/render';
import { Aria2tLogo } from './aria2t-logo';

describe('Aria2tLogo', () => {
  it('renders a decorative currentColor mark and merges a className', () => {
    const { container } = render(<Aria2tLogo className="h-8 w-8" />);
    const svg = container.querySelector('svg')!;
    expect(svg).toHaveAttribute('aria-hidden', 'true');
    expect(svg).toHaveAttribute('fill', 'currentColor');
    expect(svg).toHaveClass('h-8', 'w-8');
  });
});
