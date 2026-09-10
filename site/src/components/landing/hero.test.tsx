// The hero and the thin React side of its WebGL backdrop. The scene itself is
// mocked here: what matters at this level is that it boots once, on a canvas in
// the tree, only where WebGL exists, and that it is handed the heading line to
// aim at.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, render, screen } from '@/test/render';
import type { HeroSceneHandle } from './hero-scene-three';

const boot = vi.fn();
const dispose = vi.fn();
const debug = vi.fn(() => ({}) as ReturnType<HeroSceneHandle['debug']>);
const dark = { value: true };

vi.mock('./hero-scene-three', () => ({
  bootHeroScene: (...args: unknown[]) => {
    boot(...args);
    return { dispose, debug } satisfies HeroSceneHandle;
  },
  isDark: () => dark.value,
}));

/** The chunk loads in a microtask; let it. */
const settle = () => act(async () => void (await Promise.resolve()));

let mutationCb: MutationCallback | undefined;

beforeEach(() => {
  boot.mockClear();
  dispose.mockClear();
  dark.value = true;
  mutationCb = undefined;
  vi.stubGlobal('WebGLRenderingContext', class {});
  vi.stubGlobal(
    'MutationObserver',
    class {
      constructor(cb: MutationCallback) {
        mutationCb = cb;
      }
      observe() {}
      disconnect() {}
    },
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  document.documentElement.dir = '';
});

describe('canRunScene', () => {
  it('asks only whether the browser has WebGL at all', async () => {
    const { canRunScene } = await import('./hero-scene');
    expect(canRunScene()).toBe(true);
    vi.unstubAllGlobals();
    expect(canRunScene()).toBe(false);
  });
});

describe('HeroScene', () => {
  it('boots the scene onto its canvas', async () => {
    const { HeroScene } = await import('./hero-scene');
    const { container } = render(<HeroScene aria-label="scene" />);
    await settle();
    expect(container.querySelector('canvas')).not.toBeNull();
    expect(screen.getByRole('img', { name: 'scene' })).toBeInTheDocument();
    expect(boot).toHaveBeenCalledTimes(1);
    expect(boot.mock.calls[0][2]).toEqual({ alignTipsNdc: undefined, copyEdgeNdc: undefined });
  });

  it('does not boot where there is no WebGL', async () => {
    vi.unstubAllGlobals();
    const { HeroScene } = await import('./hero-scene');
    render(<HeroScene aria-label="scene" />);
    await settle();
    expect(boot).not.toHaveBeenCalled();
  });

  it('never boots into a node that has already gone', async () => {
    const { HeroScene } = await import('./hero-scene');
    const { unmount } = render(<HeroScene aria-label="scene" />);
    unmount();
    await settle();
    expect(boot).not.toHaveBeenCalled();
  });

  it('lets the scene go when the hero does', async () => {
    const { HeroScene } = await import('./hero-scene');
    const { unmount } = render(<HeroScene aria-label="scene" />);
    await settle();
    unmount();
    expect(dispose).toHaveBeenCalledTimes(1);
  });

  it('reboots the scene when the resolved theme actually flips', async () => {
    const { HeroScene } = await import('./hero-scene');
    render(<HeroScene aria-label="scene" />);
    await settle();
    expect(boot).toHaveBeenCalledTimes(1);

    dark.value = false;
    mutationCb!([], {} as MutationObserver);
    expect(dispose).toHaveBeenCalledTimes(1);
    expect(boot).toHaveBeenCalledTimes(2);
  });

  it('ignores a class mutation that does not change the resolved theme', async () => {
    const { HeroScene } = await import('./hero-scene');
    render(<HeroScene aria-label="scene" />);
    await settle();

    mutationCb!([], {} as MutationObserver);
    expect(boot).toHaveBeenCalledTimes(1);
    expect(dispose).not.toHaveBeenCalled();
  });
});

describe('LandingHero', () => {
  it('states which line the scene should aim at, per layout', async () => {
    const { LandingHero } = await import('./hero');
    const { container } = render(<LandingHero />);
    await settle();
    const aim = boot.mock.calls[0][2] as { alignTipsNdc: () => number | null };

    // jsdom has no `Range.getClientRects`, so the layout the hero measures has
    // to be supplied. With no rects there is no line to aim at.
    let rects: DOMRect[] = [];
    Range.prototype.getClientRects = (() => rects) as unknown as typeof Range.prototype.getClientRects;
    const section = container.querySelector('section')!;
    const band = (height: number) =>
      vi.spyOn(section, 'getBoundingClientRect').mockReturnValue({ top: 0, height } as DOMRect);

    band(1000);
    expect(aim.alignTipsNdc()).toBeNull();

    // A band 1000 tall whose heading's first line is centred on 250: a quarter
    // of the way down is +0.5 in normalised device coordinates.
    rects = [{ top: 200, bottom: 300 } as DOMRect];
    expect(aim.alignTipsNdc()).toBeCloseTo(0.5, 5);

    // Stacked - anything below the `lg` breakpoint - it states no line at all:
    // the copy's own edges are the whole of what the scene needs there, and
    // where the crowns land between them is the scene's business.
    window.innerWidth = 500;
    expect(aim.alignTipsNdc()).toBeNull();
    window.innerWidth = 1024;

    // A band with no height cannot place anything either.
    band(0);
    expect(aim.alignTipsNdc()).toBeNull();
  });

  it('states the two edges of the copy the scene has to work around', async () => {
    const { LandingHero } = await import('./hero');
    const { container } = render(<LandingHero />);
    await settle();
    const aim = boot.mock.calls[0][2] as {
      copyEdgeNdc: () => number | null;
      copyBottomNdc: () => number | null;
    };
    const section = container.querySelector('section')!;
    vi.spyOn(section, 'getBoundingClientRect')
      .mockReturnValue({ left: 0, top: 0, width: 1000, height: 1000 } as DOMRect);
    // The rows it measures: the heading's own line boxes, supplied here since
    // jsdom has none, and every chip in the row of them.
    Range.prototype.getClientRects = (() =>
      [{ left: 40, right: 500 }] as unknown as DOMRectList) as typeof Range.prototype.getClientRects;
    const chips = container.querySelector('ul')!;
    for (const chip of chips.children) {
      vi.spyOn(chip, 'getBoundingClientRect').mockReturnValue({ left: 80, right: 640 } as DOMRect);
    }
    vi.spyOn(chips, 'getBoundingClientRect').mockReturnValue({ bottom: 674 } as DOMRect);

    // A 1000-wide band whose widest row reaches 640: a little past the middle.
    // And its last row ends 26 pixels above -0.4.
    expect(aim.copyEdgeNdc()).toBeCloseTo(0.28, 5);
    expect(aim.copyBottomNdc()).toBeCloseTo(-0.4, 5);

    // Right to left the canvas is mirrored, so a row's near edge in page
    // coordinates is its far edge in the scene's.
    document.documentElement.dir = 'rtl';
    expect(aim.copyEdgeNdc()).toBeCloseTo(0.92, 5);
    document.documentElement.dir = 'ltr';

    // A band with no size cannot place anything.
    vi.spyOn(section, 'getBoundingClientRect')
      .mockReturnValue({ left: 0, top: 0, width: 0, height: 0 } as DOMRect);
    expect(aim.copyEdgeNdc()).toBeNull();
    expect(aim.copyBottomNdc()).toBeNull();
  });

  it('names every source aria2 accepts, and both ways to get the app', async () => {
    const { LandingHero, SOURCES } = await import('./hero');
    render(<LandingHero />);
    await settle();
    for (const s of SOURCES) expect(screen.getByText(s)).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument();
    // the store button cannot be pressed until the listing is live
    expect(screen.getByRole('button', { name: /chrome web store/i })).toBeDisabled();
  });
});

// Last, because it swaps the module registry out from under the mock above.
describe('a chunk that will not load', () => {
  it('costs the animation and not the page', async () => {
    vi.resetModules();
    vi.doMock('./hero-scene-three', () => {
      throw new Error('offline');
    });
    const { HeroScene } = await import('./hero-scene');
    const { container } = render(<HeroScene aria-label="scene" />);
    await settle();
    // the backdrop's gradients are the hero's own; the canvas stays blank
    expect(container.querySelector('canvas')).not.toBeNull();
  });
});
