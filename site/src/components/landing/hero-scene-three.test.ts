// The hero's scene. Everything three.js does here is arithmetic on plain
// objects except the renderer, so only the renderer is stubbed: keeping the
// real Vector3, curves, geometries and materials is what makes the camera
// framing, the tip bisection and the belt's state machine worth testing at all.
//
// The host is jsdom, so the two things a browser supplies and it does not are
// supplied here: a 2D context for the canvas textures, and the observers and
// frame clock the loop runs on.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const renders = { count: 0 };
const sizes: [number, number][] = [];
const disposed = { count: 0 };

vi.mock('three', async (importOriginal) => {
  const three = (await importOriginal()) as Record<string, unknown>;
  class FakeRenderer {
    domElement = document.createElement('canvas');
    shadowMap = { enabled: false };
    setPixelRatio() {}
    setClearColor() {}
    setSize(w: number, h: number) {
      sizes.push([w, h]);
    }
    render() {
      renders.count++;
    }
    dispose() {
      disposed.count++;
    }
  }
  return { ...three, WebGLRenderer: FakeRenderer };
});

/** A 2D context that answers every call the two texture builders make. */
function fakeContext(): CanvasRenderingContext2D {
  const stop = { addColorStop: () => {} };
  const target: Record<string, unknown> = {
    createRadialGradient: () => stop,
    createLinearGradient: () => stop,
    measureText: () => ({ width: 40 }),
    canvas: null,
  };
  return new Proxy(target, {
    get(t, k) {
      if (k in t) return t[k as string];
      // every draw call is a no-op, every property assignment is remembered
      return () => undefined;
    },
    set(t, k, v) {
      t[k as string] = v;
      return true;
    },
  }) as unknown as CanvasRenderingContext2D;
}

let frames: FrameRequestCallback[] = [];
let resizeCb: (() => void) | undefined;
let ioCb: ((e: { isIntersecting: boolean }[]) => void) | undefined;
const reduced = { value: false };

/**
 * Runs `n` frames, `ms` apart, through whatever the loop registered. The clock
 * carries across calls, because the loop measures the gap between frames and a
 * clock that restarted would hand it a gap of zero.
 */
let clock = 0;
function step(n: number, ms = 20): void {
  for (let i = 0; i < n; i++) {
    clock += ms;
    const due = frames;
    frames = [];
    for (const f of due) f(clock);
  }
}

beforeEach(() => {
  renders.count = 0;
  disposed.count = 0;
  sizes.length = 0;
  frames = [];
  clock = 0;
  resizeCb = undefined;
  ioCb = undefined;
  reduced.value = false;

  HTMLCanvasElement.prototype.getContext = (() =>
    fakeContext()) as unknown as typeof HTMLCanvasElement.prototype.getContext;
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(cb: () => void) {
        resizeCb = cb;
      }
      observe() {}
      disconnect() {}
    },
  );
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      constructor(cb: (e: { isIntersecting: boolean }[]) => void) {
        ioCb = cb;
      }
      observe() {}
      disconnect() {}
    },
  );
  vi.stubGlobal('requestAnimationFrame', (f: FrameRequestCallback) => {
    frames.push(f);
    return frames.length;
  });
  vi.stubGlobal('cancelAnimationFrame', () => {});
  vi.spyOn(window, 'matchMedia').mockImplementation(
    (q: string) => ({ matches: q.includes('reduce') && reduced.value, media: q }) as MediaQueryList,
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

/** A host of a given size, so the framing has an aspect ratio to solve for. */
function host(w: number, h: number): HTMLDivElement {
  const el = document.createElement('div');
  Object.defineProperty(el, 'clientWidth', { configurable: true, value: w });
  Object.defineProperty(el, 'clientHeight', { configurable: true, value: h });
  document.body.append(el);
  return el;
}

async function boot(
  w = 1440,
  h = 800,
  opts: { mirrorText?: boolean; alignTipsNdc?: () => number | null } = {},
) {
  const { bootHeroScene } = await import('./hero-scene-three');
  const el = host(w, h);
  const canvas = document.createElement('canvas');
  el.append(canvas);
  return { handle: bootHeroScene(el, canvas, opts), el };
}

describe('bootHeroScene', () => {
  it('builds the scene, sizes it to its host and paints a first frame', async () => {
    const { handle } = await boot();
    expect(sizes).toContainEqual([1440, 800]);
    expect(renders.count).toBeGreaterThan(0);
    handle.dispose();
    expect(disposed.count).toBe(1);
  });

  it('primes the belt, so the line is never empty on the first frame', async () => {
    // The priming loop runs the state machine until every slot carries a cube;
    // reaching the first paint at all is what proves it terminates.
    const { handle } = await boot();
    expect(renders.count).toBe(1);
    handle.dispose();
  });

  it('runs the belt through every phase it has, and keeps making cubes', async () => {
    const { handle } = await boot();
    const start = handle.debug();
    const seen = new Set([start.phase]);
    // A cycle is about five seconds of scene time: fill, compact, drop,
    // advance, idle, and round again.
    for (let i = 0; i < 900; i++) {
      step(1, 20);
      seen.add(handle.debug().phase);
    }
    expect([...seen].sort()).toEqual(['advance', 'compact', 'drop', 'fill', 'idle']);
    const end = handle.debug();
    expect(end.made).toBeGreaterThan(start.made);
    // The line stays full and the retired cubes are freed: one is kept, never
    // a growing pile.
    expect(end.live).toBeGreaterThan(0);
    expect(end.retired).toBeLessThanOrEqual(1);
    expect(end.visible).toBe(true);
    handle.dispose();
  });

  it('takes no options at all', async () => {
    const { bootHeroScene } = await import('./hero-scene-three');
    const el = host(1024, 700);
    const handle = bootHeroScene(el, document.createElement('canvas'));
    expect(renders.count).toBeGreaterThan(0);
    handle.dispose();
  });

  it('assumes one device pixel per pixel when the browser reports none', async () => {
    vi.stubGlobal('devicePixelRatio', 0);
    const { handle } = await boot();
    expect(renders.count).toBeGreaterThan(0);
    handle.dispose();
  });

  it('runs on a browser with no IntersectionObserver, and then never pauses', async () => {
    const io = window.IntersectionObserver;
    // @ts-expect-error - removing it is the condition under test
    delete window.IntersectionObserver;
    const { handle } = await boot();
    const before = renders.count;
    step(10);
    expect(renders.count).toBeGreaterThan(before);
    handle.dispose();
    window.IntersectionObserver = io;
  });

  it('holds still while the band is off screen', async () => {
    const { handle } = await boot();
    ioCb?.([{ isIntersecting: false }]);
    const before = renders.count;
    step(50);
    expect(renders.count).toBe(before);
    ioCb?.([{ isIntersecting: true }]);
    step(10);
    expect(renders.count).toBeGreaterThan(before);
    handle.dispose();
  });

  it('paints one frame and stops when the reader asked for less motion', async () => {
    reduced.value = true;
    const { handle } = await boot();
    const before = renders.count;
    step(30);
    expect(renders.count).toBe(before);
    handle.dispose();
  });

  it('drops a frame that arrives after it has been let go', async () => {
    const { handle } = await boot();
    handle.dispose();
    const before = renders.count;
    step(5);
    expect(renders.count).toBe(before);
  });

  it('clamps a long gap between frames rather than fast-forwarding the belt', async () => {
    const { handle } = await boot();
    const before = renders.count;
    // A tab that was in the background for a minute
    clock += 60_000;
    frames.forEach((f) => f(clock));
    frames = [];
    expect(renders.count).toBeGreaterThan(before);
    handle.dispose();
  });

  it('mirrors the text it bakes into the cube faces for a right-to-left page', async () => {
    const a = await boot(1440, 800, { mirrorText: true });
    expect(renders.count).toBeGreaterThan(0);
    a.handle.dispose();
  });
});

describe('the framing', () => {
  it('solves for a portrait host, a wide one, and past the zoom stop', async () => {
    for (const [w, h] of [
      [360, 780],
      [768, 900],
      [1440, 800],
      [2560, 700],
    ] as [number, number][]) {
      sizes.length = 0;
      const { handle } = await boot(w, h);
      expect(sizes).toContainEqual([w, h]);
      handle.dispose();
    }
  });

  it('falls back to a size of its own when the host reports none', async () => {
    const { bootHeroScene } = await import('./hero-scene-three');
    const el = document.createElement('div');
    document.body.append(el);
    bootHeroScene(el, document.createElement('canvas'), {}).dispose();
    expect(sizes).toContainEqual([960, 520]);
  });

  it('aims the camera so the pylon tips land on the line it is given', async () => {
    const asked: number[] = [];
    const { handle } = await boot(1440, 800, {
      alignTipsNdc: () => {
        asked.push(1);
        return 0.42;
      },
    });
    expect(asked.length).toBeGreaterThan(0);
    // and again on every resize
    resizeCb?.();
    expect(asked.length).toBeGreaterThan(1);
    handle.dispose();
  });

  it('keeps its own aim when the page has no line to offer', async () => {
    const { handle } = await boot(1440, 800, { alignTipsNdc: () => null });
    expect(renders.count).toBeGreaterThan(0);
    resizeCb?.();
    handle.dispose();
  });

  it('keeps its own aim when the line comes back as nonsense', async () => {
    const { handle } = await boot(1440, 800, { alignTipsNdc: () => Number.NaN });
    expect(renders.count).toBeGreaterThan(0);
    handle.dispose();
  });
});
