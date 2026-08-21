import { describe, expect, it } from 'vitest';
import { render } from '@/test/render';
import { ConnectionVisual } from './connection-visual';

// The visual's whole job is a per-state animation, so the states are the test.
const STATES = ['idle', 'connecting', 'connected', 'error'] as const;

describe('ConnectionVisual', () => {
  it('animates each state differently', () => {
    const shots = STATES.map((state) => {
      const { container, unmount } = render(<ConnectionVisual state={state} />);
      const html = container.innerHTML;
      unmount();
      return html;
    });
    expect(new Set(shots).size).toBe(STATES.length);
  });

  it('staggers three pulsing rings while there is traffic', () => {
    const { container } = render(<ConnectionVisual state="connected" />);
    const rings = [...container.querySelectorAll('span')].filter((s) =>
      s.className.includes('rounded-full'),
    );
    expect(rings.length).toBeGreaterThanOrEqual(3);
    const pulsing = rings.filter((r) => r.className.includes('animate-pulse-ring'));
    expect(pulsing).toHaveLength(3);
    // Each ring starts a third of a cycle after the last.
    const delays = pulsing.map((r) => r.style.animationDelay);
    expect(new Set(delays).size).toBe(3);
  });

  it('breathes just the middle ring when idle-ish, and leaves the rest dim', () => {
    for (const state of STATES) {
      const { container, unmount } = render(<ConnectionVisual state={state} />);
      const rings = [...container.querySelectorAll('span')].filter((s) =>
        s.className.includes('rounded-full'),
      );
      const breathing = rings.filter((r) => r.className.includes('animate-breathe'));
      const dim = rings.filter((r) => r.className.includes('opacity-25'));
      // A state either pulses all three, breathes exactly one, or dims them.
      expect(breathing.length, state).toBeLessThanOrEqual(1);
      if (breathing.length === 1) expect(dim.length, state).toBe(2);
      unmount();
    }
  });

  it('scales with its size and merges a className', () => {
    const { container } = render(<ConnectionVisual state="connected" size={20} className="shrink-0" />);
    const root = container.firstElementChild as HTMLElement;
    expect(root).toHaveClass('shrink-0');
    expect(root.style.width).toBe('20px');
    expect(root.style.getPropertyValue('--pulse-dur')).toMatch(/s$/);
  });
});
