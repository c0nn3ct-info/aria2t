import { describe, expect, it } from 'vitest';
import { render } from '@/test/render';
import { AmbientWave } from './ambient-wave';

function pathsOf(container: HTMLElement): SVGPathElement[] {
  return [...container.querySelectorAll('path')];
}

describe('AmbientWave', () => {
  it('draws nothing without two samples to draw between', () => {
    expect(render(<AmbientWave points={[]} max={1} />).container.innerHTML).toBe('');
    expect(render(<AmbientWave points={[5]} max={5} />).container.innerHTML).toBe('');
  });

  it('draws a filled area under a smoothed line', () => {
    const { container } = render(<AmbientWave points={[1, 4, 2, 8]} max={8} />);
    const [area, line] = pathsOf(container);
    // the area closes back along the baseline; the line does not
    expect(area.getAttribute('d')).toMatch(/Z$/);
    expect(area.getAttribute('fill')).toMatch(/^url\(#/);
    expect(line.getAttribute('d')).toContain('C');
    expect(line.getAttribute('fill')).toBe('none');
    expect(line.getAttribute('stroke')).toBe('currentColor');
  });

  it('is decorative, and fades its oldest edge out', () => {
    const { container } = render(<AmbientWave points={[1, 2]} max={2} />);
    const svg = container.querySelector('svg')!;
    expect(svg).toHaveAttribute('aria-hidden');
    expect(svg.getAttribute('class')).toContain('mask-image');
  });

  it('scales against the given ceiling, so a taller ceiling sits the wave lower', () => {
    const near = pathsOf(render(<AmbientWave points={[4, 4]} max={4} />).container)[1];
    const far = pathsOf(render(<AmbientWave points={[4, 4]} max={400} />).container)[1];
    const y = (d: string) => Number(d.split(' ')[1]);
    expect(y(far.getAttribute('d')!)).toBeGreaterThan(y(near.getAttribute('d')!));
  });

  it('never divides by a zero ceiling', () => {
    const line = pathsOf(render(<AmbientWave points={[0, 0]} max={0} />).container)[1];
    expect(line.getAttribute('d')).not.toContain('NaN');
  });

  it('smooths a long run point to point', () => {
    const { container } = render(<AmbientWave points={[1, 5, 2, 9, 3, 7]} max={9} />);
    // one cubic segment per gap between samples
    expect((pathsOf(container)[1].getAttribute('d')!.match(/C/g) ?? []).length).toBe(5);
  });
});
