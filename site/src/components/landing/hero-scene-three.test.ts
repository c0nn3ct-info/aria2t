// The hero's scene. Everything three.js does here is arithmetic on plain
// objects except the renderer, so only the renderer is stubbed: keeping the
// real Vector3, curves, geometries and materials is what makes the camera
// framing, the tip bisection and the belt's state machine worth testing at all.
//
// The host is jsdom, so the two things a browser supplies and it does not are
// supplied here: a 2D context for the canvas textures, and the observers and
// frame clock the loop runs on.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AdditiveBlending, Box3, NormalBlending, Vector3 } from 'three';
import type {
  BufferGeometry,
  Mesh,
  MeshBasicMaterial,
  MeshStandardMaterial,
  Object3D,
  PerspectiveCamera,
  Scene,
} from 'three';

const renders = { count: 0 };
/** What one `render()` costs, in ms of the fake clock the suite installs. */
const renderCost = { ms: 0 };
const sizes: [number, number][] = [];
const disposed = { count: 0 };
let lastScene: Scene | undefined;
let lastCamera: PerspectiveCamera | undefined;

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
    render(scene: Scene, camera: PerspectiveCamera) {
      lastScene = scene;
      lastCamera = camera;
      renders.count++;
      fakeNow += renderCost.ms;
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

/** The clock `performance.now` reads, so a frame's cost is what the fake
 *  renderer says it is rather than what this machine happens to take. */
let fakeNow = 0;
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
  lastScene = undefined;
  sizes.length = 0;
  frames = [];
  clock = 0;
  fakeNow = 0;
  renderCost.ms = 0;
  vi.spyOn(performance, 'now').mockImplementation(() => fakeNow);
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
  opts: {
    alignTipsNdc?: () => number | null;
    copyEdgeNdc?: () => number | null;
    copyBottomNdc?: () => number | null;
  } = {},
) {
  const { bootHeroScene } = await import('./hero-scene-three');
  const el = host(w, h);
  const canvas = document.createElement('canvas');
  el.append(canvas);
  return { handle: bootHeroScene(el, canvas, opts), el };
}

/** The standard material a named part wears. */
function mat(name: string): MeshStandardMaterial {
  return (lastScene!.getObjectByName(name) as Mesh).material as MeshStandardMaterial;
}

/** The first part whose name matches — for the generated names (a crate's
 * pictogram is `crate_icon_<kind>`, and which kind is up first is the PRNG's
 * business, not this test's). */
function named(match: (o: Mesh) => boolean): Mesh {
  let found: Mesh | undefined;
  lastScene!.traverse((o) => {
    if (!found && (o as Mesh).isMesh && o.name && match(o as Mesh)) found = o as Mesh;
  });
  return found!;
}

/** How every glow mesh in the scene blends. All of them go through
 * `glowMat`, so this is the whole of the stage's glow behaviour. */
function glowBlends(): number[] {
  const out: number[] = [];
  lastScene!.traverse((o) => {
    const m = (o as Mesh).material as MeshBasicMaterial | undefined;
    // `depthWrite: false` is `glowMat`'s own signature, which is what makes
    // this the whole glow layer and nothing else: the backdrop halo is a
    // texture rather than a flat colour, and the blueprint's brackets and
    // lines are plain overlays that never blended additively on either stage.
    if (m?.isMeshBasicMaterial && !m.map && !m.depthWrite) out.push(m.blending);
  });
  return out;
}

/** The flight pool's shared geometry, found by the morph target that is
 * unique to it. */
function morphing(): Mesh {
  return pooled()[0];
}

/** Every mesh in the flight pool — the ones carrying the ball-to-cube morph. */
function pooled(): Mesh[] {
  const out: Mesh[] = [];
  lastScene!.traverse((o) => {
    const m = o as Mesh;
    if (m.isMesh && m.geometry?.morphAttributes?.position?.length) out.push(m);
  });
  return out;
}

/** Every glow mesh's alpha, in traversal order. */
function glowAlphas(): number[] {
  const out: number[] = [];
  lastScene!.traverse((o) => {
    const m = (o as Mesh).material as MeshBasicMaterial | undefined;
    if (m?.isMeshBasicMaterial && !m.map && !m.depthWrite) out.push(m.opacity);
  });
  return out;
}

function rig(): { points: number[]; key: number } {
  const points: number[] = [];
  let key = 0;
  lastScene!.traverse((o) => {
    const l = o as unknown as { isPointLight?: boolean; isDirectionalLight?: boolean; intensity: number };
    if (l.isPointLight) points.push(l.intensity);
    if (l.isDirectionalLight) key = l.intensity;
  });
  return { points, key };
}

/** Every self-lit material in the scene, in traversal order. */
function emissives(): number[] {
  const out: number[] = [];
  lastScene!.traverse((o) => {
    const m = (o as Mesh).material as MeshStandardMaterial | undefined;
    if (m?.isMeshStandardMaterial && m.emissiveIntensity > 0 && m.emissive.getHex() > 0) {
      out.push(m.emissiveIntensity);
    }
  });
  return out;
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

  it('builds a closed slatted belt without rollers and brackets it with the pylons', async () => {
    const { handle } = await boot();
    const belt = lastScene!.getObjectByName('belt_body');
    const tread = lastScene!.getObjectByName('belt_tread_0');
    const back = lastScene!.getObjectByName('belt_return');
    const far = lastScene!.getObjectByName('pylon_far');
    const near = lastScene!.getObjectByName('pylon_near');
    const names: string[] = [];
    lastScene!.traverse((object) => names.push(object.name));

    expect(belt).toBeDefined();
    // Slat, light-line, halo. The lower run is its own strip, since it travels
    // the other way, and the strip is long enough that an ultra-wide frame -
    // which sees a long way down the line - never reaches its end.
    expect(tread?.children).toHaveLength(3);
    expect(back?.children).toHaveLength(72);
    // The two machines stand on one line across the conveyor: same station,
    // mirrored across the belt, so the line runs between them.
    expect(far?.position.z).toBeLessThan(0);
    expect(near?.position.z).toBeGreaterThan(0);
    expect(far?.position.z).toBeCloseTo(-near!.position.z, 6);
    expect(far?.position.x).toBeCloseTo(near!.position.x, 6);
    expect(names.some((name) => /roller|drum|idler/.test(name))).toBe(false);
    handle.dispose();
  });

  it('holds both machines in frame and clear of the copy at every hero shape', async () => {
    // The shapes the band is actually built at, against what the hero hands
    // over at each - measured off the rendered page, not invented here.
    //
    // Stacked, it states the line just under its copy's last row and no column
    // at all - the band carries a strip below that line for the machinery, so
    // the line lands between a tenth and a third below the band's middle,
    // wherever the copy's wrap and the strip's own vh cap put it. In its
    // column layout it states the heading's first line, which lands at +0.47
    // whatever the width, and the column's own far edge - which barely moves
    // above 1240px, where the column stops growing and centres, and swings
    // right as the band narrows towards it.
    //
    // The heights are the band's, not the window's: below `lg` it is the copy
    // plus that strip, so it runs taller than the window on a short one.
    const stacked = [-0.11, -0.24, -0.36];
    const column = [0.47];
    // The fourth column is the copy column's far edge, the fifth the line its
    // last row ends on: stacked shapes state the second and no column at all,
    // column ones the other way about. The two stacked values bracket the
    // choice the scene makes with them - a row close under the crowns leaves
    // the volume behind it and the composition drops below the row, a row
    // further up and it stands in the gap above it.
    const shapes: [number, number, number[], number | null, number | null][] = [
      [320, 849, stacked, null, -0.5], [390, 841, stacked, null, -0.5],
      [640, 772, stacked, null, -0.9], [768, 832, stacked, null, -0.5],
      [900, 725, stacked, null, -0.9], [1023, 769, stacked, null, -0.5],
      [1024, 720, column, 0.25, null], [1180, 720, column, 0.1, null],
      [1280, 720, column, 0.09, null], [1440, 720, column, 0.03, null],
      [1920, 720, column, 0.02, null], [2560, 720, column, 0.02, null],
      [3440, 720, column, 0.01, null],
    ];
    let worstEdge = 1;
    let worstGap = 1;
    let worstCrate = 1;
    for (const [w, h, aims, edge, below] of shapes) {
      for (const aim of aims) {
        const { handle } = await boot(w, h, {
          alignTipsNdc: () => aim,
          copyEdgeNdc: () => edge,
          copyBottomNdc: () => below,
        });
        const camera = lastCamera!;
        camera.updateMatrixWorld(true);
        let left = Infinity;
        for (const name of ['pylon_far', 'pylon_near']) {
          const box = new Box3().setFromObject(lastScene!.getObjectByName(name)!);
          for (let corner = 0; corner < 8; corner++) {
            const ndc = new Vector3(
              corner & 1 ? box.max.x : box.min.x,
              corner & 2 ? box.max.y : box.min.y,
              corner & 4 ? box.max.z : box.min.z,
            ).project(camera);
            worstEdge = Math.min(worstEdge, 1 - Math.abs(ndc.x), 1 - ndc.y);
            left = Math.min(left, ndc.x);
          }
        }
        if (edge !== null) worstGap = Math.min(worstGap, left - edge);

        // And the crate at the station - the newest one on the belt, the one
        // the volume above it just compacted into.
        const crates: Object3D[] = [];
        lastScene!.traverse((object) => {
          if (/^crate_/.test(object.name) && object.parent?.name === '') crates.push(object);
        });
        const station = crates.reduce((a, b) => (a.position.x > b.position.x ? a : b));
        const box = new Box3().setFromObject(station);
        for (let corner = 0; corner < 4; corner++) {
          const ndc = new Vector3(
            corner & 1 ? box.max.x : box.min.x,
            box.min.y,
            corner & 2 ? box.max.z : box.min.z,
          ).project(camera);
          worstCrate = Math.min(worstCrate, ndc.y);
        }
        handle.dispose();
      }
    }

    // Both towers keep a twentieth of the half-frame clear of the sides and
    // the top, at every shape, anywhere in the range of lines its layout can
    // ask for - which is `EDGE`, since the near one is what the field eases
    // off to hold. The bottom is deliberately not held: a plinth cropped by
    // the band's edge reads as a machine carrying on past the screen, and the
    // closing fade dissolves it rather than cutting it.
    expect(worstEdge).toBeGreaterThan(0.05);
    // What must not be cropped is the crate at the station, which is what the
    // whole line is about: `CRATE_FLOOR` holds its centre at -0.62, so even
    // its lowest corner stays out of the closing fade's stronger half.
    expect(worstCrate).toBeGreaterThan(-0.8);
    // And stand clear of the copy column wherever there is one, which is the
    // constraint the field eases off to satisfy on a narrow one.
    expect(worstGap).toBeGreaterThan(0.02);
  });

  it('runs the tread on rather than snapping it back after each advance', async () => {
    const { handle } = await boot();
    // One slat pitch, and the strip of 72 of them a slat is recycled through.
    const PITCH = 0.94 / 2;
    const STRIP = 72 * PITCH;
    // A slat far enough along the strip that it cannot reach the wrap here.
    const tread = lastScene!.getObjectByName('belt_tread_40')!;
    const back = lastScene!.getObjectByName('belt_return')!;
    let x = tread.position.x;
    let backX = back.position.x;
    let travelled = 0;
    let backTravel = 0;
    const forward: number[] = [];
    const backwards: number[] = [];
    // Two full cycles of the line, so at least one advance ends inside the run.
    for (let i = 0; i < 700; i++) {
      step(1);
      const dx = tread.position.x - x;
      travelled -= dx;
      if (dx > 1e-9) forward.push(dx);
      x = tread.position.x;
      // The lower run travels the other way and is allowed exactly one kind of
      // discontinuity: the pitch it wraps by, which is invisible on a strip of
      // identical slats. Undo that and it has to be the mirror of the run
      // above it, frame for frame.
      const db = back.position.x - backX;
      const step0 = db < -PITCH / 2 ? db + PITCH : db;
      if (step0 < -1e-9) backwards.push(step0);
      backTravel += step0;
      backX = back.position.x;
    }

    expect(forward).toEqual([]);
    expect(backwards).toEqual([]);
    expect(travelled).toBeGreaterThan(0.94);
    expect(backTravel).toBeCloseTo(travelled, 6);
    expect(back.position.x).toBeGreaterThanOrEqual(0);
    expect(back.position.x).toBeLessThan(PITCH);
    expect(tread.position.x).toBeGreaterThan(-8 - STRIP);
    handle.dispose();
  });

  it('makes every file crate a single type on all four faces and its lid', async () => {
    const { handle } = await boot();
    const kinds = ['image', 'video', 'audio', 'doc'];
    const accentColors = new Set<number>();

    for (const kind of kinds) {
      const roots: Object3D[] = [];
      lastScene!.traverse((object) => {
        if (object.name === `crate_${kind}`) roots.push(object);
      });
      expect(roots.length).toBeGreaterThan(0);

      const pictograms = new Set<BufferGeometry>();
      for (const root of roots) {
        const iconNames: string[] = [];
        const faces: string[] = [];
        root.traverse((object) => {
          if (!/^crate_icon_(image|video|audio|doc)$/.test(object.name)) {
            if (object.name.startsWith('crate_face_')) faces.push(object.name);
            return;
          }
          iconNames.push(object.name);
          pictograms.add((object as Mesh<BufferGeometry, MeshStandardMaterial>).geometry);
        });
        expect(iconNames).toEqual(Array(5).fill(`crate_icon_${kind}`));
        expect(faces).toHaveLength(4);
      }
      // One extruded pictogram per type for the whole belt: the artwork never
      // differs between crates, so nothing rebuilds it as the line runs.
      expect(pictograms.size).toBe(1);

      const trim = roots[0].getObjectByName('crate_plinth_trim') as Mesh<BufferGeometry, MeshStandardMaterial>;
      accentColors.add(trim.material.color.getHex());

      // The model's own proportions, at the scale that maps its 0.856-wide
      // body onto the 0.78 blueprint cube: 0.877 across its widest part, the
      // lit plinth flange, and 0.79 tall — still a clear gap inside one
      // 0.94-unit belt slot, glow shell included.
      const sample = roots[0].clone(true);
      sample.visible = true;
      sample.position.set(0, 0, 0);
      sample.rotation.set(0, 0, 0);
      const size = new Box3().setFromObject(sample).getSize(new Vector3());
      expect(size.x).toBeCloseTo(0.877, 2);
      expect(size.y).toBeCloseTo(0.791, 2);
      expect(size.z).toBeCloseTo(0.877, 2);
    }
    expect(accentColors.size).toBe(4);
    handle.dispose();
  });

  it('sends its parts out as balls and hardens them into cubes in flight', async () => {
    const { handle } = await boot();
    // One geometry for the whole pool, carrying the cube as a morph target,
    // so each mesh only has to slide its own influence between the two.
    const fly = morphing();
    expect(fly.geometry.morphAttributes.position).toHaveLength(1);
    expect(fly.geometry.morphAttributes.normal).toHaveLength(1);
    // A launch resets the shape: the pool hands meshes back round-robin and
    // the last flight landed as a cube.
    const seen = new Set<number>();
    for (let i = 0; i < 400; i++) {
      step(1);
      for (const m of pooled()) if (m.visible) seen.add(Math.round(m.morphTargetInfluences![0] * 10));
    }
    // Caught mid-morph at both ends and in between, rather than only ever as
    // one shape or the other.
    expect(seen.has(0)).toBe(true);
    expect(seen.has(10)).toBe(true);
    expect([...seen].some((v) => v > 0 && v < 10)).toBe(true);
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

  it('leaves the scene as a still on a machine drawing in software', async () => {
    // Five frames costing more than the frame they draw: the loop stops rather
    // than pinning the main thread, which is what kept a GPU-less runner from
    // ever reaching network idle.
    const { handle } = await boot();
    step(6);
    const fast = renders.count;
    expect(fast).toBeGreaterThan(0);

    renderCost.ms = 200;
    step(6);
    const slow = renders.count;
    expect(slow).toBe(fast + 5);

    renderCost.ms = 0;
    step(6);
    expect(renders.count).toBe(slow);
    handle.dispose();
  });

  it('forgives a slow frame that does not become a streak', async () => {
    const { handle } = await boot();
    for (let i = 0; i < 15; i++) {
      renderCost.ms = i % 3 === 0 ? 200 : 0;
      step(1);
    }
    const before = renders.count;
    renderCost.ms = 0;
    step(3);
    expect(renders.count).toBe(before + 3);
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

describe('isDark', () => {
  afterEach(() => {
    document.documentElement.style.removeProperty('--background');
  });

  it('defaults to dark when the background token is unset (this suite loads no stylesheet)', async () => {
    const { isDark } = await import('./hero-scene-three');
    expect(isDark()).toBe(true);
  });

  it('reads light off a bright --background token', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { isDark } = await import('./hero-scene-three');
    expect(isDark()).toBe(false);
  });

  it('reads dark off a dim --background token', async () => {
    document.documentElement.style.setProperty('--background', '240 15% 5%');
    const { isDark } = await import('./hero-scene-three');
    expect(isDark()).toBe(true);
  });
});

describe('bootHeroScene in light mode', () => {
  afterEach(() => {
    document.documentElement.style.removeProperty('--background');
    document.documentElement.style.removeProperty('--primary');
    document.documentElement.style.removeProperty('--tertiary');
  });

  it('boots fine off the light tokens, with no dark-only fallback needed', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    document.documentElement.style.setProperty('--primary', '217 60% 42%');
    const { handle } = await boot();
    expect(renders.count).toBeGreaterThan(0);
    handle.dispose();
  });

  it('boots fine in light mode even when --primary is unset', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { handle } = await boot();
    expect(renders.count).toBeGreaterThan(0);
    handle.dispose();
  });

  // The stage does not merely retint the dark machines: `.claude/3d/light`
  // holds a separately authored material set, and these are the parts of it a
  // render would show and a boot-succeeded assertion would not.
  it('dresses the machines in the light models’ own materials', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { handle } = await boot();
    // The slats invert to near-white...
    expect((named((o) => o.name === 'belt_return_0').material as MeshStandardMaterial).color.getHex()).toBe(0xc7cee0);
    // ...and the pictograms, which were the one light thing on the dark
    // stage, invert the other way.
    const icon = named((o) => o.name.startsWith('crate_icon_'));
    expect((icon.material as MeshStandardMaterial).color.getHex()).toBe(0x2b3446);
    handle.dispose();
  });

  it('hangs the slats on a glass chassis, where the dark stage’s near-black is a wedge across the page', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { handle } = await boot();
    const chassis = mat('belt_body');
    expect(chassis.transparent).toBe(true);
    expect(chassis.opacity).toBeLessThan(0.5);
    // Its inset strip is shell, so it draws a pale line down the frame.
    expect(mat('belt_side').color.getHex()).toBe(0xc7cee0);
    handle.dispose();
  });

  it('keeps the chassis opaque and near-black on the dark stage, which is what the belt reads by there', async () => {
    const { handle } = await boot();
    expect(mat('belt_body').color.getHex()).toBe(0x151a26);
    expect(mat('belt_body').transparent).toBe(false);
    handle.dispose();
  });



  it('opens its trim up and takes its bloom out, which is the whole of the glow split', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { handle } = await boot();
    const lit = glowAlphas();
    handle.dispose();
    document.documentElement.style.removeProperty('--background');
    const { handle: h2 } = await boot();
    const dark = glowAlphas();
    h2.dispose();
    // The scene is deterministic, so the two traversals line up index for
    // index and the same glow can be compared across stages.
    expect(lit.length).toBe(dark.length);
    // Flat trim — a mark painted on the machine — has to fight a white ground
    // and goes up. A bloom is escaped light with nothing to escape into and
    // goes nearly to zero. Both directions, or the split is not happening.
    expect(lit.some((a, i) => a > dark[i])).toBe(true);
    expect(lit.some((a, i) => a < dark[i])).toBe(true);
  });

  it('draws its glow normally, since adding to near-white is a no-op', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { handle } = await boot();
    const blends = new Set(glowBlends());
    expect(blends).toEqual(new Set([NormalBlending]));
    handle.dispose();
  });

  it('still glows additively on the dark stage', async () => {
    const { handle } = await boot();
    const blends = new Set(glowBlends());
    expect(blends).toEqual(new Set([AdditiveBlending]));
    handle.dispose();
  });

  it('pulls the rig back, since near-white shells clip under the dark stage’s rims', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { handle } = await boot();
    const lit = rig();
    handle.dispose();
    document.documentElement.style.removeProperty('--background');
    const { handle: h2 } = await boot();
    const dark = rig();
    h2.dispose();
    expect(lit.points.every((p, i) => p < dark.points[i])).toBe(true);
    expect(lit.key).toBeLessThan(dark.key);
  });

  it('dims everything self-lit, the way the light models drop their emissive strength', async () => {
    document.documentElement.style.setProperty('--background', '217 30% 99%');
    const { handle } = await boot();
    const lit = emissives();
    handle.dispose();
    document.documentElement.style.removeProperty('--background');
    const { handle: h2 } = await boot();
    const dark = emissives();
    h2.dispose();
    expect(lit.length).toBe(dark.length);
    expect(lit.every((e, i) => e < dark[i])).toBe(true);
  });
});
