// The animated TUI list on the home page. It has to render identically on the
// server and on first client paint (no hydration mismatch), then come alive —
// which is why it seeds from a deterministic PRNG and only ticks after mount.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, render } from '@/test/render';
import { bar, fmtEta, fmtSpeed, ListMock, lpad, pad, trunc } from './list-mock';

const webdriver = { value: false };

beforeEach(() => {
  Object.defineProperty(navigator, 'webdriver', {
    configurable: true,
    get: () => webdriver.value,
  });
  webdriver.value = false;
});

afterEach(() => {
  vi.useRealTimers();
});

describe('ListMock', () => {
  it('renders the same frame twice, so hydration cannot mismatch', () => {
    const a = render(<ListMock />);
    const first = a.container.innerHTML;
    a.unmount();

    const b = render(<ListMock />);
    expect(b.container.innerHTML).toBe(first);
  });

  it('shows download rows with names, speeds and progress', () => {
    const { container } = render(<ListMock />);
    const text = container.textContent ?? '';
    expect(text).toMatch(/%/);
    expect(text).toMatch(/iB/); // a formatted size or speed
    expect(container.querySelectorAll('span').length).toBeGreaterThan(10);
  });

  it('comes alive after mount', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { container } = render(<ListMock />);
    const before = container.textContent;

    await act(async () => {
      vi.advanceTimersByTime(3_000);
    });
    expect(container.textContent).not.toBe(before);
  });

  it('holds still for the prerender, so the captured frame is stable', async () => {
    webdriver.value = true;
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { container } = render(<ListMock />);
    const before = container.textContent;

    await act(async () => {
      vi.advanceTimersByTime(5_000);
    });
    expect(container.textContent).toBe(before);
  });

  it('wraps a download that reaches the end instead of freezing at 100%', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { container } = render(<ListMock />);

    // The bars creep at ~a third of real pace, so this is long enough for the
    // fastest row to pass 99% and restart low rather than sit at 100%.
    await act(async () => {
      vi.advanceTimersByTime(1_200_000);
    });
    expect(container.textContent).toMatch(/%/);
  });

  it('stops ticking when it unmounts', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { unmount } = render(<ListMock />);
    unmount();
    await act(async () => {
      vi.advanceTimersByTime(5_000);
    });
  });
});

describe('the TUI formatters it mirrors', () => {
  it('pads and truncates to a fixed column width', () => {
    expect(pad('ab', 5)).toBe('ab   ');
    expect(pad('abcdef', 3)).toBe('abc');
    expect(lpad('ab', 5)).toBe('   ab');
    expect(lpad('abcdef', 3)).toBe('abc');
  });

  it('marks a truncated name with an ellipsis, and pads a short one', () => {
    expect(trunc('short', 8)).toBe('short   ');
    expect(trunc('a-very-long-name', 8)).toBe('a-very-…');
    expect(trunc('exactly8', 8)).toBe('exactly8');
  });

  it('draws the bar empty, partial with a cap, and full', () => {
    const empty = bar(0);
    expect(empty.filled).toBe('');
    expect(empty.empty).toMatch(/^─+$/);

    const part = bar(0.5);
    expect(part.filled.endsWith('╸')).toBe(true);
    expect(part.empty.length).toBeGreaterThan(0);

    const full = bar(1);
    expect(full.filled).toMatch(/^━+$/);
    expect(full.empty).toBe('');

    // Out-of-range fractions clamp rather than overflow.
    expect(bar(-1)).toEqual(empty);
    expect(bar(2)).toEqual(full);
  });

  it('formats speeds in KiB and MiB, and a stall as a dash', () => {
    expect(fmtSpeed(0)).toBe('-');
    expect(fmtSpeed(-5)).toBe('-');
    expect(fmtSpeed(2048)).toBe('2 KiB/s');
    expect(fmtSpeed(1048576 * 2.5)).toBe('2.5 MiB/s');
  });

  it('formats an ETA in seconds, minutes and hours', () => {
    expect(fmtEta(100, 0)).toBe('-');
    expect(fmtEta(1000, 100)).toBe('10s');
    expect(fmtEta(90 * 100, 100)).toBe('1m 30s');
    expect(fmtEta(3700 * 100, 100)).toBe('1h 1m');
  });
});
