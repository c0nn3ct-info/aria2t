// One queue, seen from two places. The store is what makes the popup and the
// terminal list agree, so these pin the rules that keep it honest: progress
// only ever climbs, a finished torrent seeds and a finished file stops, and the
// slot it frees goes to whatever was waiting.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@/test/render';
import { ITEMS, WAVE_MAX, fmtSize, useQueueScene } from './queue-scene';

const MiB = 1048576;
const GiB = 1073741824;

const webdriver = { value: false };
const hidden = { value: false };

beforeEach(() => {
  Object.defineProperty(navigator, 'webdriver', { configurable: true, get: () => webdriver.value });
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden.value });
  webdriver.value = false;
  hidden.value = false;
});

afterEach(() => {
  vi.useRealTimers();
});

describe('fmtSize', () => {
  it('is FmtBytes and fmtBytes, to the digit', () => {
    // tui/internal/ui/format.go and extension/src/lib/aria2/format.ts: bytes
    // whole, one decimal below ten, none from ten up.
    expect(fmtSize(0)).toBe('0 B');
    expect(fmtSize(-5)).toBe('0 B');
    expect(fmtSize(512)).toBe('512 B');
    expect(fmtSize(84 * 1024)).toBe('84 KiB');
    expect(fmtSize(680 * MiB)).toBe('680 MiB');
    expect(fmtSize(1.4 * GiB)).toBe('1.4 GiB');
    expect(fmtSize(12.6 * GiB)).toBe('13 GiB');
    // and it does not run out of units
    expect(fmtSize(3 * 1024 * GiB)).toBe('3.0 TiB');
    expect(fmtSize(4 * 1024 * 1024 * GiB)).toBe('4.0 PiB');
  });
});

describe('the queue', () => {
  it('opens settled, with a full minute of wave under its own ceiling', () => {
    const { result } = renderHook(() => useQueueScene());
    const s = result.current;
    expect(s.live).toHaveLength(ITEMS.length);
    expect(s.wave).toHaveLength(60);
    for (const v of s.wave) expect(v).toBeLessThan(WAVE_MAX);
    // five transferring, and the headline counts those and nothing else
    expect(s.downloading).toBe(5);
    expect(s.live.filter((l) => l.kind === 'active')).toHaveLength(5);
  });

  it('counts a seeding torrent as upload, never as download', () => {
    const { result } = renderHook(() => useQueueScene());
    const s = result.current;
    const active = s.live.filter((l) => l.kind === 'active').reduce((a, l) => a + l.speed, 0);
    const seeding = s.live.filter((l) => l.kind === 'seeding').reduce((a, l) => a + l.speed, 0);
    expect(s.down).toBeCloseTo(active, 5);
    expect(s.up).toBeCloseTo(seeding, 5);
    expect(seeding).toBeGreaterThan(0);
  });

  it('climbs, finishes into seeding or done, and starts the next one', () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useQueueScene());
    const first = result.current.live.map((l) => l.pct);

    // A minute of walking: nothing may go backwards.
    let prev = first;
    for (let i = 0; i < 60; i++) {
      act(() => void vi.advanceTimersByTime(1000));
      const now = result.current.live.map((l) => l.pct);
      now.forEach((p, j) => expect(p).toBeGreaterThanOrEqual(prev[j]));
      prev = now;
    }
    expect(prev[0]).toBeGreaterThan(first[0]);

    // Long enough for the head of the queue to finish, and for the torrent
    // four rows down: 2.0 GiB left at ~9.8 MiB/s is about 210s, and 5.0 GiB at
    // ~6.8 MiB/s about 750s. The .iso stops when it finishes; the torrent
    // starts seeding.
    act(() => void vi.advanceTimersByTime(1500 * 1000));
    const s = result.current;
    expect(s.live[0]).toMatchObject({ kind: 'done', pct: 100, speed: 0 });
    expect(s.live[3].kind).toBe('seeding');
    expect(s.live[3].pct).toBe(100);
    // The freed slots went to what was waiting: by now the one queued download
    // has started, and finished too (380 MiB at ~5 MiB/s).
    expect(s.live[8].kind).not.toBe('waiting');
    expect(s.live.filter((l) => l.kind === 'waiting')).toHaveLength(0);
  });

  it('holds still while the tab is hidden', () => {
    vi.useFakeTimers();
    hidden.value = true;
    const { result } = renderHook(() => useQueueScene());
    const before = result.current;
    act(() => void vi.advanceTimersByTime(10_000));
    expect(result.current).toBe(before);
  });

  it('never starts a clock the prerenderer would capture mid-walk', () => {
    vi.useFakeTimers();
    webdriver.value = true;
    const { result } = renderHook(() => useQueueScene());
    const before = result.current;
    act(() => void vi.advanceTimersByTime(10_000));
    expect(result.current).toBe(before);
  });

  it('runs one clock for however many readers, and stops with the last', () => {
    vi.useFakeTimers();
    const a = renderHook(() => useQueueScene());
    const b = renderHook(() => useQueueScene());
    act(() => void vi.advanceTimersByTime(1000));
    expect(b.result.current).toBe(a.result.current);

    a.unmount();
    const mid = b.result.current;
    act(() => void vi.advanceTimersByTime(1000));
    expect(b.result.current).not.toBe(mid);

    b.unmount();
    const last = b.result.current;
    act(() => void vi.advanceTimersByTime(5000));
    expect(b.result.current).toBe(last);
  });
});

// Last, because it resets the module registry: every test above shares one
// module-level store (the whole point of it), and none of them advance the
// clock past a few seconds without immediately asserting only relative
// motion — but toggleItem/toggleAll need the *exact* pcts the seed declares
// (63, 100, 23, ...), which only holds before any beat() has run. A fresh
// import, never ticked, is the only way to get that.
describe('toggleItem and toggleAll', () => {
  it('pauses a moving row in place and resumes it back to what it was', async () => {
    vi.resetModules();
    const { useQueueScene: fresh, toggleItem } = await import('./queue-scene');
    const { result } = renderHook(() => fresh());

    const before = result.current.down;
    act(() => toggleItem(0));
    expect(result.current.live[0]).toMatchObject({ kind: 'paused', pct: 63, speed: 0 });
    expect(result.current.down).toBeLessThan(before);

    act(() => toggleItem(0));
    // Resumed rather than warmed up, so the speed is exactly the target
    // rather than wherever a random walk left it.
    expect(result.current.live[0]).toMatchObject({ kind: 'active', pct: 63, speed: ITEMS[0].target });
  });

  it('resumes a paused torrent into seeding when its bar was already full', async () => {
    vi.resetModules();
    const { useQueueScene: fresh, toggleItem } = await import('./queue-scene');
    const { result } = renderHook(() => fresh());

    act(() => toggleItem(1)); // seeding -> paused
    expect(result.current.live[1]).toMatchObject({ kind: 'paused', pct: 100, speed: 0 });
    act(() => toggleItem(1)); // paused -> seeding, not active: its bar never left 100
    expect(result.current.live[1]).toMatchObject({
      kind: 'seeding',
      pct: 100,
      speed: ITEMS[1].target * 0.52,
    });
  });

  it('leaves a waiting, errored, or finished row untouched', async () => {
    vi.resetModules();
    const { useQueueScene: fresh, toggleItem } = await import('./queue-scene');
    const { result } = renderHook(() => fresh());
    const before = result.current;

    act(() => {
      toggleItem(8); // waiting
      toggleItem(10); // error
      toggleItem(11); // done
    });
    // No listener fired, so useSyncExternalStore never handed back a new
    // snapshot - the strongest way to show these three did nothing at all.
    expect(result.current).toBe(before);
  });

  it('pauses everything moving, then resumes only the rows it paused', async () => {
    vi.resetModules();
    const { useQueueScene: fresh, toggleAll } = await import('./queue-scene');
    const { result } = renderHook(() => fresh());

    act(() => toggleAll());
    const s = result.current;
    // Every row the seed declared active or seeding.
    for (const i of [0, 2, 3, 5, 6]) expect(s.live[i].kind).toBe('paused');
    for (const i of [1, 7]) expect(s.live[i].kind).toBe('paused');
    // Rows the seed already had paused are undisturbed.
    expect(s.live[4]).toMatchObject({ kind: 'paused', pct: 18, speed: 0 });
    expect(s.live[9]).toMatchObject({ kind: 'paused', pct: 36, speed: 0 });
    expect(s.down).toBe(0);
    expect(s.up).toBe(0);

    act(() => toggleAll());
    const r = result.current;
    for (const i of [0, 2, 3, 5, 6]) {
      expect(r.live[i]).toMatchObject({ kind: 'active', speed: ITEMS[i].target });
    }
    for (const i of [1, 7]) {
      expect(r.live[i]).toMatchObject({ kind: 'seeding', speed: ITEMS[i].target * 0.52 });
    }
    // Still exactly how the seed left them - toggleAll never touched these.
    expect(r.live[4]).toMatchObject({ kind: 'paused', pct: 18, speed: 0 });
    expect(r.live[9]).toMatchObject({ kind: 'paused', pct: 36, speed: 0 });
  });

  it('skips a row already resumed by hand before the second click', async () => {
    vi.resetModules();
    const { useQueueScene: fresh, toggleAll, toggleItem } = await import('./queue-scene');
    const { result } = renderHook(() => fresh());

    act(() => toggleAll()); // pauses 0, 1, 2, 3, 5, 6, 7
    act(() => toggleItem(0)); // resumed by its own row button, ahead of the rest
    expect(result.current.live[0].kind).toBe('active');

    act(() => toggleAll()); // resumes what it paused and is still paused
    // Untouched: toggleItem already moved it, so the guard on an
    // already-non-paused row left it exactly where the row button put it.
    expect(result.current.live[0]).toMatchObject({ kind: 'active', speed: ITEMS[0].target });
    for (const i of [2, 3, 5, 6]) {
      expect(result.current.live[i]).toMatchObject({ kind: 'active', speed: ITEMS[i].target });
    }
    for (const i of [1, 7]) {
      expect(result.current.live[i]).toMatchObject({ kind: 'seeding', speed: ITEMS[i].target * 0.52 });
    }
  });
});
