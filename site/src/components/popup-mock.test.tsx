// The animated extension popup on the home page. Same contract as the TUI list
// mock beside it: identical on the server and on first client paint, then live.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, render } from '@/test/render';
import { fmtSpeed, PopupMock } from './popup-mock';
import { setLocale } from '../i18n';

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

describe('PopupMock', () => {
  it('renders the same frame twice, so hydration cannot mismatch', () => {
    const a = render(<PopupMock />);
    const first = a.container.innerHTML;
    a.unmount();
    const b = render(<PopupMock />);
    expect(b.container.innerHTML).toBe(first);
  });

  it('shows the queue, the statuses and the way in', () => {
    const { container } = render(<PopupMock />);
    const text = container.textContent ?? '';
    expect(text).toContain('downloading');
    expect(text).toContain('ubuntu-24.04.2-desktop-amd64.iso');
    // seeding is its own status, distinct from active — the whole point of the
    // extension's status colours
    expect(text).toContain('seeding');
    expect(text).toContain('paused');
    expect(text).toContain('Add');
    expect(text).toMatch(/iB\/s/);
  });

  it('comes alive after mount', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { container } = render(<PopupMock />);
    const before = container.textContent;
    await act(async () => {
      vi.advanceTimersByTime(3_000);
    });
    expect(container.textContent).not.toBe(before);
  });

  it('holds still for the prerender, so the captured frame is stable', async () => {
    webdriver.value = true;
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { container } = render(<PopupMock />);
    const before = container.textContent;
    await act(async () => {
      vi.advanceTimersByTime(5_000);
    });
    expect(container.textContent).toBe(before);
  });

  it('wraps a download that reaches the end instead of freezing at 100%', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { container } = render(<PopupMock />);
    await act(async () => {
      vi.advanceTimersByTime(600_000);
    });
    // the seeding row stays whole, and no row ever prints past 100%
    const pcts = [...(container.textContent ?? '').matchAll(/(\d+)%/g)].map((m) => Number(m[1]));
    expect(pcts.length).toBeGreaterThan(0);
    expect(Math.max(...pcts)).toBeLessThanOrEqual(100);
  });

  it('stops ticking when it unmounts', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const { unmount } = render(<PopupMock />);
    unmount();
    await act(async () => {
      vi.advanceTimersByTime(5_000);
    });
    // no act() warning and no throw is the assertion here
    expect(true).toBe(true);
  });
});

describe('speaking the page language', () => {
  afterEach(() => setLocale('en'));

  // The strings are the extension's own, copied into the site dictionary, so
  // the mock says what the app says in that language rather than a paraphrase.
  it('follows the site locale, plural included', () => {
    setLocale('ru');
    const { container } = render(<PopupMock />);
    const text = container.textContent ?? '';
    // uppercased by CSS, so the DOM still holds the sentence case
    expect(text).toContain('Передача данных');
    // three downloading rows -> the `few` plural, not the bare form
    expect(text).toContain('Загружаются 3');
    expect(text).toContain('Загрузки');
    expect(text).toContain('активна');
    expect(text).toContain('Добавить');
  });

  it('mirrors for a right-to-left locale without pinning its own direction', () => {
    setLocale('ar');
    const { container } = render(<PopupMock />);
    const text = container.textContent ?? '';
    expect(text).toContain('حالة النقل');
    expect(text).toContain('3 قيد التنزيل');
    // the shell must not force LTR, or the popup would not mirror with the page
    expect(container.firstElementChild?.getAttribute('dir')).toBeNull();
    // a measurement still reads left-to-right inside a mirrored line
    const ltr = [...container.querySelectorAll('[dir="ltr"]')].map((e) => e.textContent);
    expect(ltr.some((x) => x?.includes('MiB/s'))).toBe(true);
  });
});

describe('the extension formatter it mirrors', () => {
  it('formats a speed in B, KiB and MiB, and nothing as zero', () => {
    expect(fmtSpeed(0)).toBe('0 B/s');
    expect(fmtSpeed(-1)).toBe('0 B/s');
    expect(fmtSpeed(512)).toBe('512 B/s');
    expect(fmtSpeed(2048)).toBe('2 KiB/s');
    expect(fmtSpeed(1048576 * 3.25)).toBe('3.3 MiB/s');
  });
});
