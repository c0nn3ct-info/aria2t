// The animated TUI list on the home page. It has to render identically on the
// server and on first client paint (no hydration mismatch), then come alive —
// which is why it seeds from a deterministic PRNG and only ticks after mount.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, render } from '@/test/render';
import { bar, fitHints, fmtEta, fmtSpeed, LIST_HINTS, ListMock, lpad, pad, Row, trunc } from './list-mock';

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
    expect(fmtSpeed(512)).toBe('512 B/s');
    expect(fmtSpeed(2048)).toBe('2 KiB/s');
    expect(fmtSpeed(1048576 * 2.5)).toBe('2.5 MiB/s');
  });

  // The bar is width-adaptive in the real TUI and drops the lowest-priority
  // hints from the right, so the mock must not show a hint the real screen
  // would have dropped at the same width (hintbarEx, tui/internal/ui/app.go).
  it('keeps the leftmost hints and drops the tail that would not fit', () => {
    const wide = fitHints(LIST_HINTS, 200, '1/12');
    expect(wide).toHaveLength(LIST_HINTS.length);

    const narrow = fitHints(LIST_HINTS, 100, '1/12');
    expect(narrow.map(([k, l]) => `${k} ${l}`)).toEqual([
      'a add', 'space pause', '↵ details', 'd remove',
      'l limit', '/ filter', 'y copy url', 'g stats', 's servers',
    ]);
    expect(narrow.some(([, l]) => l === 'help')).toBe(false);
  });

  it('keeps the first hint even when nothing fits, and needs no trailer', () => {
    expect(fitHints(LIST_HINTS, 4, '1/12')).toEqual([['a', 'add']]);
    // No trailer means the whole width is the budget: 'a add' then 'space
    // pause' would end at column 18, past 12, so only the first survives.
    expect(fitHints(LIST_HINTS, 12, '').map(([, l]) => l)).toEqual(['add']);
    expect(fitHints(LIST_HINTS, 24, '').map(([, l]) => l)).toEqual(['add', 'pause']);
  });

  it('draws a red bar over the part a failed download did fetch', () => {
    // list.go renders "error" as Bar() in red over the faint remainder, so a
    // download that failed at 40% keeps the part it got. At 0% the bar is empty
    // and the row is indistinguishable from waiting, which is all the mock's
    // own data ever shows.
    const { container } = render(
      <Row name="x.iso" status="error" pct={40} size="1 GiB" speed="-" conn="-" eta="-" />,
    );
    // The STATUS word is red too, so pick the red span that holds bar glyphs.
    const reds = [...container.querySelectorAll<HTMLElement>('span[style*="--tui-red"]')];
    const filled = reds.find((el) => el.textContent?.includes('━'));
    expect(filled, 'no red span holding bar glyphs').toBeDefined();
    expect(filled?.textContent).toBe(bar(0.4).filled);
    expect(filled?.textContent).toContain('╸');
    // and the unfetched remainder stays faint, not red
    expect(reds.some((el) => el.textContent?.includes('─'))).toBe(false);
  });

  it('formats an ETA in seconds, minutes and hours', () => {
    expect(fmtEta(100, 0)).toBe('-');
    expect(fmtEta(1000, 100)).toBe('10s');
    expect(fmtEta(90 * 100, 100)).toBe('1m 30s');
    expect(fmtEta(3700 * 100, 100)).toBe('1h 1m');
  });
});
