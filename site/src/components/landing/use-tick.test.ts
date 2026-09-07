// One clock for every live figure on the page. Nothing may start it where a
// prerender would capture the walk, or where the reader asked for less motion,
// and nothing may advance it on a tab nobody is looking at.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@/test/render';
import { TICK_MS, useTick } from './use-tick';

const webdriver = { value: false };
const hidden = { value: false };
const reduce = { value: false };

beforeEach(() => {
  vi.useFakeTimers();
  Object.defineProperty(navigator, 'webdriver', { configurable: true, get: () => webdriver.value });
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden.value });
  webdriver.value = false;
  hidden.value = false;
  reduce.value = false;
  vi.spyOn(window, 'matchMedia').mockImplementation(
    (q: string) => ({ matches: q.includes('reduce') && reduce.value, media: q }) as MediaQueryList,
  );
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('useTick', () => {
  it('starts at zero and counts on the cadence the design asks for', () => {
    const { result } = renderHook(() => useTick());
    expect(result.current).toBe(0);
    act(() => void vi.advanceTimersByTime(TICK_MS * 3));
    expect(result.current).toBe(3);
  });

  it('takes an interval of its own', () => {
    const { result } = renderHook(() => useTick(1000));
    act(() => void vi.advanceTimersByTime(3000));
    expect(result.current).toBe(3);
  });

  it('holds at zero for the prerenderer', () => {
    webdriver.value = true;
    const { result } = renderHook(() => useTick());
    act(() => void vi.advanceTimersByTime(TICK_MS * 5));
    expect(result.current).toBe(0);
  });

  it('holds at zero when the reader asked for less motion', () => {
    reduce.value = true;
    const { result } = renderHook(() => useTick());
    act(() => void vi.advanceTimersByTime(TICK_MS * 5));
    expect(result.current).toBe(0);
  });

  it('stops counting while the tab is hidden, and picks up after', () => {
    const { result } = renderHook(() => useTick());
    hidden.value = true;
    act(() => void vi.advanceTimersByTime(TICK_MS * 4));
    expect(result.current).toBe(0);
    hidden.value = false;
    act(() => void vi.advanceTimersByTime(TICK_MS * 2));
    expect(result.current).toBe(2);
  });

  it('clears its interval when the figure goes', () => {
    const spy = vi.spyOn(globalThis, 'clearInterval');
    renderHook(() => useTick()).unmount();
    expect(spy).toHaveBeenCalled();
  });
});
