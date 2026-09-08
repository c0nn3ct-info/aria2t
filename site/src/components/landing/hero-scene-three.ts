// The hero's assembly line: two neon pylons feed particles into a transparent
// blueprint volume; when the blueprint is full the particles compact into one
// solid file cube (icon + extension), the cube drops onto the belt and the line
// steps toward the viewer. Nothing rotates. Deterministic: a fixed 1/120 step,
// with the order and the labels drawn from mulberry32 — the same PRNG the two
// live mocks seed from (src/lib/mock-motion.ts).
//
// This module is loaded on demand by `hero-scene.tsx`, so three.js stays out
// of the page's main chunk; the wrapper also decides whether to boot at all
// (WebGL present, not the prerender).
import {
  AdditiveBlending,
  BoxGeometry,
  BufferGeometry,
  CanvasTexture,
  CapsuleGeometry,
  Color,
  CylinderGeometry,
  DirectionalLight,
  EdgesGeometry,
  ExtrudeGeometry,
  Fog,
  Group,
  HemisphereLight,
  LineBasicMaterial,
  LineSegments,
  Mesh,
  MeshBasicMaterial,
  MeshStandardMaterial,
  PerspectiveCamera,
  PlaneGeometry,
  PointLight,
  QuadraticBezierCurve3,
  Scene,
  Shape,
  SphereGeometry,
  TorusGeometry,
  Vector3,
  WebGLRenderer,
} from 'three';
import { mulberry32 } from '@/lib/mock-motion';

// The scene used to assume the page around this canvas was always the comp's
// dark stage. It now reads which theme is actually active and builds one of
// two palettes: dark is the comp's original fixed set below; light derives
// its background/primary/tertiary from the same `[data-accent='blue']` CSS
// tokens the surrounding bands use (`globals.css`), so the WebGL ground and
// the page's own background can never drift the way two hand-picked hex
// constants could. `cyan`/`magenta`/the card-face colors have no CSS token —
// they are scene-only accents — and are tuned by eye for each stage.
interface Palette {
  bg: number;
  primary: number;
  tertiary: number;
  cyan: number;
  magenta: number;
  /** The face `typeTexture` paints each file-type icon onto. */
  cardBg: string;
  /** The extension label drawn on that face. */
  label: string;
}

const DARK: Palette = {
  bg: 0x0b0b0f,
  primary: 0xa8c7fa,
  tertiary: 0xc6b2ff,
  cyan: 0x7dcfee,
  magenta: 0xbb9af7,
  cardBg: '#0f1726',
  label: '#e8f4ff',
};

/**
 * Parses a `"H S% L%"` custom property (the format every token in
 * `globals.css` is written in) into the fractions `Color.setHSL` wants.
 * Returns null when the property is unset or unparseable — the jsdom test
 * host has no stylesheet loaded by default, and `isDark`/`currentPalette`
 * both fall back to the dark palette in that case, which is what every
 * pre-existing test in this file already boots against.
 */
function readHSL(name: string): [h: number, s: number, l: number] | null {
  const raw = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  const m = /^([\d.]+)\s+([\d.]+)%\s+([\d.]+)%$/.exec(raw);
  if (!m) return null;
  return [parseFloat(m[1]), parseFloat(m[2]) / 100, parseFloat(m[3]) / 100];
}

function hslHex([h, s, l]: [number, number, number]): number {
  return new Color().setHSL(h / 360, s, l).getHex();
}

/** Whether the page is currently resolved dark — the one thing both the
 * scene and `hero-scene.tsx`'s reboot-on-flip watcher need to agree on. */
export function isDark(): boolean {
  const bg = readHSL('--background');
  return !bg || bg[2] < 0.5;
}

function currentPalette(): Palette {
  if (isDark()) return DARK;
  const primary = readHSL('--primary');
  const tertiary = readHSL('--tertiary');
  return {
    bg: hslHex(readHSL('--background')!),
    primary: primary ? hslHex(primary) : DARK.primary,
    tertiary: tertiary ? hslHex(tertiary) : DARK.tertiary,
    cyan: 0x1f8fae,
    magenta: 0x7a4fc9,
    cardBg: '#e9f0fb',
    label: '#16223a',
  };
}

/** `rgba()` string for a canvas 2D stroke/fill from a three.js hex color. */
function hexRgba(hex: number, alpha: number): string {
  return `rgba(${(hex >> 16) & 255},${(hex >> 8) & 255},${hex & 255},${alpha})`;
}

const CUBE = 0.78;
const LX = 4;
const LY = 3;
const LZ = 4;
const PARTS = LX * LY * LZ;
const SX = CUBE / LX;
const SY = CUBE / LY;
const SZ = CUBE / LZ;
const BUILD_Y = 1.85;
const BELT_Y = 0.32;
const SLOT = 0.94;
/** Opacity of a glow shell relative to the core it wraps. */
const HALO = 0.42;
const START_X = 5.6;
/** Belt slots a cube rides before it leaves the frame. */
const LINE = 13;
const CELL_STEP = 0.075;
const TRAVEL = 1.0;
const COMPACT = 0.6;
const DROP = 0.45;
const ADVANCE = 0.65;
const IDLE = 0.45;
const STEP = 1 / 120;
/** A frame that costs this long, in ms, is a machine drawing in software. */
const SLOW_FRAME = 80;
/** How many of those in a row before the loop leaves the scene as a still. */
const SLOW_STREAK = 5;

const DEG = Math.PI / 180;

// Framing.
//
// A perspective camera fixes the *vertical* field, so the same scene in a
// taller or wider canvas shows a different amount of world. What has to stay
// put instead is the horizontal half-extent H, since a point's horizontal
// screen position is viewX / (depth * H). `framingFor` converts H back into the
// vertical field three.js wants, through H = tan(fov / 2) * aspect.
//
// The composition is anchored at two shapes and interpolated between them, so
// there is no breakpoint where the hero jumps:
//
//   portrait (aspect 0.95 and below) - a phone or a narrow window, where the
//     copy runs the full width. The machines stand just above the middle of
//     the frame, right of centre, with the belt crossing under the copy: the
//     same reading the desktop has, rotated. Aiming lower than the machines
//     tilts the camera down and lifts them in frame, which is why the target
//     sits above them.
//   wide (aspect 1.90 and above) - the machines right of centre with the copy
//     over the empty left. At 1440x720 this resolves to exactly the 30 degree
//     field and the aim point the composition was drawn at.
//
// Past ZOOM_STOP an ultra-wide window stops narrowing the field, or the
// vertical would squeeze until the pylons clipped.
//
// Solved, not guessed: projecting the pylons, the blueprint and the belt
// through this maths over every hero shape from 320x620 to 2560x800 holds the
// pylon tips on the heading line at every shape, nothing clipped closer than
// 0.23 of a half-frame, and the belt leaving the band between 35% and 96% of
// its height depending on how tall the copy makes it. Below the belt the band
// is empty on purpose: that is what the closing fade is for.
interface Framing {
  /** Horizontal half-extent. */
  h: number;
  target: { x: number; y: number; z: number };
}
const PORTRAIT: Framing & { aspect: number } = {
  aspect: 0.97,
  h: 0.3,
  target: { x: 5.75, y: 1.9, z: -2.5 },
};
const WIDE: Framing & { aspect: number } = {
  aspect: 1.9,
  h: Math.tan(15 * DEG) * 2.0,
  target: { x: 3.1, y: 1.05, z: 0.8 },
};
const ZOOM_STOP = 2.6;

const mix = (a: number, b: number, k: number) => a + (b - a) * k;

function framingFor(aspect: number): { fov: number; target: { x: number; y: number; z: number } } {
  const k = Math.max(0, Math.min(1, (aspect - PORTRAIT.aspect) / (WIDE.aspect - PORTRAIT.aspect)));
  const h = mix(PORTRAIT.h, WIDE.h, k);
  return {
    fov: (2 * Math.atan(h / Math.min(aspect, ZOOM_STOP))) / DEG,
    target: {
      x: mix(PORTRAIT.target.x, WIDE.target.x, k),
      y: mix(PORTRAIT.target.y, WIDE.target.y, k),
      z: mix(PORTRAIT.target.z, WIDE.target.z, k),
    },
  };
}

interface FileType {
  ext: string;
  kind: 'image' | 'video' | 'disc' | 'archive' | 'doc';
  color: string;
}

const TYPES: readonly FileType[] = [
  { ext: '.png', kind: 'image', color: '#7dcfee' },
  { ext: '.mp4', kind: 'video', color: '#bb9af7' },
  { ext: '.iso', kind: 'disc', color: '#a8c7fa' },
  { ext: '.zip', kind: 'archive', color: '#c6b2ff' },
  { ext: '.pdf', kind: 'doc', color: '#9ece6a' },
];

interface Cell {
  x: number;
  y: number;
  z: number;
}

/** Deterministic but broken up: layer by layer, shuffled inside each layer. */
export function fillOrder(): Cell[] {
  const rnd = mulberry32(0x61726961);
  const order: Cell[] = [];
  for (let y = 0; y < LY; y++) {
    const layer: Cell[] = [];
    for (let x = 0; x < LX; x++) for (let z = 0; z < LZ; z++) layer.push({ x, y, z });
    for (let i = layer.length - 1; i > 0; i--) {
      const j = Math.floor(rnd() * (i + 1));
      [layer[i], layer[j]] = [layer[j], layer[i]];
    }
    order.push(...layer);
  }
  return order;
}

const easeInOut = (t: number) => (t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2);
const easeIn = (t: number) => t * t;

/**
 * Paints a file-type icon and its extension onto a 256x256 canvas texture.
 *
 * `mirrorText` pre-flips the extension label. A right-to-left page mirrors the
 * whole canvas so the belt runs the other way, and that mirror would otherwise
 * reverse the `.iso` and `.mp4` baked into these faces. The icons survive a
 * mirror; Latin text does not.
 */
function typeTexture(
  type: FileType,
  mirrorText: boolean,
  cardBg: string,
  cyanBorder: number,
  label: string,
): CanvasTexture {
  const c = document.createElement('canvas');
  c.width = c.height = 256;
  // A fresh canvas always yields a 2D context in a browser that got this far
  // (the wrapper only boots when WebGL exists).
  const ctx = c.getContext('2d')!;
  ctx.fillStyle = cardBg;
  ctx.fillRect(0, 0, 256, 256);
  ctx.strokeStyle = hexRgba(cyanBorder, 0.45);
  ctx.lineWidth = 5;
  ctx.strokeRect(9, 9, 238, 238);
  ctx.strokeStyle = type.color;
  ctx.fillStyle = type.color;
  ctx.lineWidth = 7;
  ctx.lineJoin = 'round';
  switch (type.kind) {
    case 'image':
      ctx.strokeRect(58, 62, 140, 106);
      ctx.beginPath();
      ctx.arc(96, 96, 13, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.moveTo(70, 156);
      ctx.lineTo(112, 116);
      ctx.lineTo(146, 156);
      ctx.lineTo(168, 132);
      ctx.lineTo(190, 156);
      ctx.closePath();
      ctx.fill();
      break;
    case 'video':
      ctx.strokeRect(58, 66, 140, 100);
      ctx.beginPath();
      ctx.moveTo(112, 92);
      ctx.lineTo(112, 140);
      ctx.lineTo(154, 116);
      ctx.closePath();
      ctx.fill();
      break;
    case 'disc':
      ctx.beginPath();
      ctx.arc(128, 114, 54, 0, Math.PI * 2);
      ctx.stroke();
      ctx.beginPath();
      ctx.arc(128, 114, 16, 0, Math.PI * 2);
      ctx.stroke();
      break;
    case 'archive':
      ctx.strokeRect(64, 60, 128, 110);
      ctx.beginPath();
      ctx.moveTo(128, 60);
      ctx.lineTo(128, 170);
      ctx.stroke();
      ctx.lineWidth = 5;
      for (let y = 74; y < 166; y += 18) {
        ctx.beginPath();
        ctx.moveTo(116, y);
        ctx.lineTo(140, y);
        ctx.stroke();
      }
      break;
    default:
      ctx.beginPath();
      ctx.moveTo(76, 54);
      ctx.lineTo(150, 54);
      ctx.lineTo(184, 88);
      ctx.lineTo(184, 174);
      ctx.lineTo(76, 174);
      ctx.closePath();
      ctx.stroke();
      ctx.beginPath();
      ctx.moveTo(150, 54);
      ctx.lineTo(150, 88);
      ctx.lineTo(184, 88);
      ctx.stroke();
      ctx.lineWidth = 5;
      for (const y of [116, 136, 156]) {
        ctx.beginPath();
        ctx.moveTo(96, y);
        ctx.lineTo(164, y);
        ctx.stroke();
      }
  }
  ctx.fillStyle = label;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.font = '500 34px ui-monospace, SFMono-Regular, Menlo, monospace';
  ctx.save();
  if (mirrorText) {
    ctx.translate(256, 0);
    ctx.scale(-1, 1);
  }
  ctx.fillText(type.ext, 128, 210);
  ctx.restore();
  const t = new CanvasTexture(c);
  t.anisotropy = 4;
  return t;
}

function haloTexture(): CanvasTexture {
  const c = document.createElement('canvas');
  c.width = c.height = 256;
  const ctx = c.getContext('2d')!;
  const g = ctx.createRadialGradient(128, 128, 0, 128, 128, 128);
  g.addColorStop(0, 'rgba(125,207,238,0.4)');
  g.addColorStop(0.42, 'rgba(187,154,247,0.14)');
  g.addColorStop(1, 'rgba(0,0,0,0)');
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, 256, 256);
  return new CanvasTexture(c);
}

const glowMat = (color: number, opacity: number) =>
  new MeshBasicMaterial({ color, transparent: true, opacity, blending: AdditiveBlending, depthWrite: false });

type Phase = 'fill' | 'compact' | 'drop' | 'advance' | 'idle';

interface Cube {
  g: Group;
  mesh: Mesh<BoxGeometry, MeshStandardMaterial>;
  edges: LineSegments<EdgesGeometry, LineBasicMaterial>;
  glow: Mesh<BoxGeometry, MeshBasicMaterial>;
  slot: number;
  x: number;
}

interface Flight {
  mesh: Mesh;
  t: number;
  curve: QuadraticBezierCurve3;
  idx: number;
}

/** One belt slat: a fixed graphite crossbar plus a glowing light-line insert
 * that fades with distance the way the old bare glow rung did. */
interface Rung {
  group: Group;
  bottom: Mesh<BufferGeometry, MeshStandardMaterial>;
  accent: Mesh<BufferGeometry, MeshBasicMaterial>;
  /** Oversized additive shell around the accent — the glow's spill. */
  halo: Mesh<BufferGeometry, MeshBasicMaterial>;
  /** Peak opacity for the accent — brighter on the periodic highlighted slats. */
  base: number;
}

interface Pylon {
  /** Crown position inside the world group, where the parts fly from. */
  tip: Vector3;
  /** The same point in world space, for projecting against the camera. */
  tipWorld: Vector3;
  emitter: Mesh<SphereGeometry, MeshStandardMaterial>;
  emGlow: Mesh<SphereGeometry, MeshBasicMaterial>;
  flash: number;
}

interface Bead {
  mesh: Mesh;
  py: Pylon;
  curve: QuadraticBezierCurve3;
  dur: number;
  t: number;
}

export interface HeroSceneDebug {
  phase: Phase;
  landed: number;
  made: number;
  live: number;
  retired: number;
  flights: number;
  visible: boolean;
}

export interface HeroSceneHandle {
  dispose(): void;
  /** The simulation's counters, for tests and for poking at it in a console. */
  debug(): HeroSceneDebug;
}

/**
 * Builds the scene into `canvas`, sized to `host`, and starts animating. The
 * returned handle stops everything and frees the renderer.
 *
 * Pass `mirrorText` when the page mirrors the canvas, so the labels baked into
 * the cube faces still read forwards.
 */
export interface HeroSceneOptions {
  /** Pre-flip the labels baked into the cube faces; see `typeTexture`. */
  mirrorText?: boolean;
  /**
   * Where the pylon tips should sit, in normalised device coordinates: +1 is
   * the top edge of the canvas, -1 the bottom. The hero hands over the
   * vertical centre of its first heading line, and the aim point is solved
   * against it on every resize, which is what keeps the machines standing
   * level with that line at any size. Returning null leaves the aim at the
   * interpolated default.
   */
  alignTipsNdc?: () => number | null;
}

export function bootHeroScene(
  host: HTMLElement,
  canvas: HTMLCanvasElement,
  { mirrorText = false, alignTipsNdc }: HeroSceneOptions = {},
): HeroSceneHandle {
  // No `preserveDrawingBuffer`: the comp set it so the design tool could
  // capture a thumbnail, and it costs a retained copy of the framebuffer
  // every frame. Nothing here reads pixels back.
  const { bg: BG, primary: PRIMARY, tertiary: VIOLET, cyan: CYAN, magenta: MAGENTA, cardBg: CARD_BG, label: LABEL } =
    currentPalette();
  const renderer = new WebGLRenderer({ canvas, alpha: true, antialias: true });
  renderer.setPixelRatio(Math.min(2, window.devicePixelRatio || 1));
  const scene = new Scene();
  scene.fog = new Fog(BG, 13, 24);
  const camera = new PerspectiveCamera(30, WIDE.aspect, 0.1, 160);
  camera.position.set(-1.6, 4.3, 11.6);
  camera.lookAt(WIDE.target.x, WIDE.target.y, WIDE.target.z);

  scene.add(new HemisphereLight(PRIMARY, 0x05050a, 0.5));
  const key = new DirectionalLight(0xffffff, 1.05);
  key.position.set(2.6, 6.4, 5.8);
  scene.add(key);
  const rimA = new PointLight(MAGENTA, 22, 20);
  rimA.position.set(3.4, 2.8, 2.4);
  scene.add(rimA);
  const rimB = new PointLight(CYAN, 16, 18);
  rimB.position.set(0.4, 1.4, 2.8);
  scene.add(rimB);

  // The belt runs along -X toward the viewer: a slight isometric tilt on Y.
  const world = new Group();
  world.position.set(3.0, -0.05, -0.3);
  world.rotation.y = 0.42;
  world.scale.setScalar(0.74);
  scene.add(world);

  const halo = new Mesh(
    new PlaneGeometry(9, 9),
    new MeshBasicMaterial({
      map: haloTexture(),
      transparent: true,
      blending: AdditiveBlending,
      depthWrite: false,
      depthTest: false,
      opacity: 0.45,
    }),
  );
  // Behind the machines in both framings. Its old spot was chosen for the
  // backdrop alone, and once the strip re-centred the camera it drifted out to
  // the left of the belt and read as a stray glow. Solved against both: it
  // lands on the machines' midpoint either way.
  halo.position.set(7.6, 1.2, -3);
  halo.renderOrder = -1;
  halo.material.fog = false;
  scene.add(halo);

  // ---------- conveyor
  //
  // The GLB exporter lost the node transforms, but kept the source geometry's
  // proportions: each tread fills about 75% of its pitch, overhangs the belt
  // body slightly and repeats on the return run. Rebuild that silhouette here
  // at the hero's much longer scale instead of replaying 47 coincident nodes.
  const carbon = new MeshStandardMaterial({ color: 0x151a26, roughness: 0.55, metalness: 0.3 });
  const graphite = new MeshStandardMaterial({ color: 0x252c3e, roughness: 0.42, metalness: 0.35 });
  const beltBody = new Mesh(new BoxGeometry(27, 0.36, 1.08), carbon);
  beltBody.name = 'belt_body';
  beltBody.position.set(5, BELT_Y - 0.2, 0);
  world.add(beltBody);

  // A shallow inset on the near and far faces gives the body the same closed
  // tread-loop read as the source model. It deliberately contains no drums,
  // axles or idlers: the user wants the belt itself, without visible rollers.
  const beltSideGeo = new BoxGeometry(27, 0.2, 0.025);
  for (const z of [-0.55, 0.55]) {
    const side = new Mesh(beltSideGeo, graphite);
    side.name = 'belt_side';
    side.position.set(5, BELT_Y - 0.2, z);
    world.add(side);
  }

  const rungs: Rung[] = [];
  const slatGeo = roundedBox(0.38, 0.075, 1.16, 0.055, 0.016);
  const accentGeo = roundedBox(0.2, 0.014, 1.02, 0.035, 0.006);
  // There is no bloom pass in this scene, so the accent glows the way the cubes
  // and the emitters do: an additive core under an oversized, fainter additive
  // shell that spills past the slat and softens its edge.
  const accentHaloGeo = roundedBox(0.46, 0.01, 1.26, 0.09, 0.004);
  for (let i = 0; i < 54; i++) {
    const group = new Group();
    group.name = `belt_tread_${i}`;
    const top = new Mesh(slatGeo, graphite);
    const bottom = new Mesh(slatGeo, graphite);
    bottom.position.y = -0.44;
    group.add(top, bottom);
    const bright = i % 3 === 0;
    const color = Math.floor(i / 3) % 2 === 0 ? VIOLET : PRIMARY;
    const base = bright ? 0.55 : 0.16;
    const accent = new Mesh(accentGeo, glowMat(color, base));
    accent.position.y = 0.075;
    const halo = new Mesh(accentHaloGeo, glowMat(color, base * HALO));
    halo.position.y = 0.079;
    group.add(accent, halo);
    group.position.set(i * (SLOT / 2) - 8, BELT_Y - 0.04, 0);
    world.add(group);
    rungs.push({ group, bottom, accent, halo, base });
  }

  // ---------- finished cubes (opaque)
  const texes = TYPES.map((type) => typeTexture(type, mirrorText, CARD_BG, CYAN, LABEL));
  const cubeGeo = new BoxGeometry(CUBE, CUBE, CUBE);
  const cubeEdgeGeo = new EdgesGeometry(cubeGeo);
  const live: Cube[] = [];
  const retired: Cube[] = [];
  const makeCube = (): Cube => {
    const g = new Group();
    const mesh = new Mesh(
      cubeGeo,
      new MeshStandardMaterial({
        color: 0xffffff,
        map: texes[0],
        emissive: PRIMARY,
        emissiveIntensity: 0.1,
        roughness: 0.45,
        metalness: 0.16,
      }),
    );
    const edges = new LineSegments(cubeEdgeGeo, new LineBasicMaterial({ color: CYAN, transparent: true, opacity: 0.65 }));
    const glow = new Mesh(new BoxGeometry(CUBE * 1.1, CUBE * 1.1, CUBE * 1.1), glowMat(CYAN, 0.08));
    g.add(mesh, edges, glow);
    world.add(g);
    return { g, mesh, edges, glow, slot: 0, x: START_X };
  };
  // Once a second cube has left the frame the older one is destroyed — nothing
  // invisible stays in memory.
  const freeCube = (s: Cube) => {
    world.remove(s.g);
    s.mesh.material.dispose();
    s.edges.material.dispose();
    s.glow.geometry.dispose();
    s.glow.material.dispose();
  };

  // ---------- blueprint volume + assembling particles
  const build = new Group();
  build.position.set(START_X, BUILD_Y, 0);
  world.add(build);

  const blueprint = new Group();
  build.add(blueprint);
  const bpEdges = new LineSegments(cubeEdgeGeo, new LineBasicMaterial({ color: CYAN, transparent: true, opacity: 0.5 }));
  blueprint.add(bpEdges);
  const bpGridPts: Vector3[] = [];
  for (let i = 1; i < LX; i++) {
    const x = -CUBE / 2 + i * SX;
    bpGridPts.push(new Vector3(x, -CUBE / 2, -CUBE / 2), new Vector3(x, -CUBE / 2, CUBE / 2));
    bpGridPts.push(new Vector3(x, CUBE / 2, -CUBE / 2), new Vector3(x, CUBE / 2, CUBE / 2));
  }
  for (let i = 1; i < LZ; i++) {
    const z = -CUBE / 2 + i * SZ;
    bpGridPts.push(new Vector3(-CUBE / 2, -CUBE / 2, z), new Vector3(CUBE / 2, -CUBE / 2, z));
    bpGridPts.push(new Vector3(-CUBE / 2, CUBE / 2, z), new Vector3(CUBE / 2, CUBE / 2, z));
  }
  const CORNERS: ReadonlyArray<[number, number]> = [[-1, -1], [1, -1], [1, 1], [-1, 1]];
  for (let i = 1; i < LY; i++) {
    const y = -CUBE / 2 + i * SY;
    CORNERS.forEach(([sx, sz], n) => {
      const [nx, nz] = CORNERS[(n + 1) % 4];
      bpGridPts.push(new Vector3((sx * CUBE) / 2, y, (sz * CUBE) / 2), new Vector3((nx * CUBE) / 2, y, (nz * CUBE) / 2));
    });
  }
  const bpGrid = new LineSegments(
    new BufferGeometry().setFromPoints(bpGridPts),
    new LineBasicMaterial({ color: CYAN, transparent: true, opacity: 0.16 }),
  );
  blueprint.add(bpGrid);
  const bracketMat = new MeshBasicMaterial({ color: CYAN, transparent: true, opacity: 0.8 });
  for (const [sx, sy, sz] of [
    [-1, -1, -1], [1, -1, -1], [1, -1, 1], [-1, -1, 1],
    [-1, 1, -1], [1, 1, -1], [1, 1, 1], [-1, 1, 1],
  ]) {
    const L = 0.14;
    const T = 0.02;
    [[L, T, T], [T, L, T], [T, T, L]].forEach(([w, h, d], i) => {
      const b = new Mesh(new BoxGeometry(w, h, d), bracketMat);
      b.position.set(
        sx * (CUBE / 2 - (i === 0 ? L / 2 : 0)),
        sy * (CUBE / 2 - (i === 1 ? L / 2 : 0)),
        sz * (CUBE / 2 - (i === 2 ? L / 2 : 0)),
      );
      blueprint.add(b);
    });
  }
  const bpFloor = new Mesh(new PlaneGeometry(CUBE * 1.7, CUBE * 1.7), glowMat(VIOLET, 0.07));
  bpFloor.rotation.x = -Math.PI / 2;
  bpFloor.position.y = -CUBE / 2 - 0.02;
  blueprint.add(bpFloor);
  const bpMats = [bpEdges.material, bpGrid.material, bracketMat, bpFloor.material];
  const bpBase = [0.68, 0.2, 0.85, 0.08];
  const setBlueprint = (k: number) => bpMats.forEach((m, i) => void (m.opacity = bpBase[i] * k));

  // Particles that make up the cube (opaque, only faintly lit).
  const matPart = new MeshStandardMaterial({ color: 0x2b3f5e, emissive: CYAN, emissiveIntensity: 0.22, roughness: 0.5, metalness: 0.25 });
  const partGeo = new BoxGeometry(SX * 0.9, SY * 0.9, SZ * 0.9);
  const partX = (p: Cell) => -CUBE / 2 + SX / 2 + p.x * SX;
  const partY = (p: Cell) => -CUBE / 2 + SY / 2 + p.y * SY;
  const partZ = (p: Cell) => -CUBE / 2 + SZ / 2 + p.z * SZ;
  const order = fillOrder();
  const parts = order.map((p) => {
    const m = new Mesh(partGeo, matPart);
    m.position.set(partX(p), partY(p), partZ(p));
    m.visible = false;
    build.add(m);
    return m;
  });

  // ---------- pylons: the "Кластер" cyberpunk-pylon body with the "Пилюля"
  // (pill) head, ported from
  // .claude/Киберпанк пилоны_ четыре варианта/pylon-cluster-model.js and
  // trimmed to exactly the one head this scene uses — this file's coverage
  // gate is 100% branches, and a parameter no call site here ever varies is
  // exactly the kind of dead branch that gate catches (see e.g. `extrudeY`,
  // which drops the reference's own default bevel since `roundedBox` always
  // forwards a concrete one, and `arcRing`, which drops the reference's own
  // `spin` param since this head never spins its rings).
  // `carbon`/`graphite` are declared above (the conveyor uses them too); one
  // more structural neutral (`bone`) and the two accent materials this design
  // uses are remapped onto the scene's own palette instead — periwinkle ->
  // primary, lilac -> the existing violet trace — so the pylon recolors with
  // everything else on a theme flip.
  const bone = new MeshStandardMaterial({ color: 0xd8e0f2, emissive: 0x9db4e6, emissiveIntensity: 0.2, roughness: 0.34, metalness: 0.14 });
  const tracePrimary = new MeshStandardMaterial({ color: PRIMARY, emissive: PRIMARY, emissiveIntensity: 1.1, roughness: 0.25, metalness: 0.3 });
  const traceMag = new MeshStandardMaterial({ color: MAGENTA, emissive: MAGENTA, emissiveIntensity: 1.25, roughness: 0.25, metalness: 0.3 });
  const traceViolet = new MeshStandardMaterial({ color: VIOLET, emissive: VIOLET, emissiveIntensity: 1.2, roughness: 0.25, metalness: 0.3 });
  const traceCyan = new MeshStandardMaterial({ color: CYAN, emissive: CYAN, emissiveIntensity: 1.05, roughness: 0.25, metalness: 0.3 });

  /** M3-style rounded-rect profile, extruded along Y and based at y = 0. */
  function roundedRectShape(w: number, d: number, r: number): Shape {
    const x = w / 2;
    const z = d / 2;
    const rr = Math.max(0.001, Math.min(r, Math.min(x, z) - 0.001));
    const shape = new Shape();
    shape.moveTo(-x + rr, -z);
    shape.lineTo(x - rr, -z);
    shape.absarc(x - rr, -z + rr, rr, -Math.PI / 2, 0, false);
    shape.lineTo(x, z - rr);
    shape.absarc(x - rr, z - rr, rr, 0, Math.PI / 2, false);
    shape.lineTo(-x + rr, z);
    shape.absarc(-x + rr, z - rr, rr, Math.PI / 2, Math.PI, false);
    shape.lineTo(-x, -z + rr);
    shape.absarc(-x + rr, -z + rr, rr, Math.PI, Math.PI * 1.5, false);
    return shape;
  }
  function baseAtZero(g: BufferGeometry): BufferGeometry {
    g.computeBoundingBox();
    g.translate(0, -g.boundingBox!.min.y, 0);
    return g;
  }
  function extrudeY(shape: Shape, height: number, bevel: number): BufferGeometry {
    const b = Math.min(bevel, height / 3);
    const g = new ExtrudeGeometry(shape, {
      depth: Math.max(0.004, height - b * 2),
      bevelEnabled: b > 0,
      bevelThickness: b,
      bevelSize: b,
      bevelSegments: 3,
      curveSegments: 10,
      steps: 1,
    });
    g.rotateX(-Math.PI / 2);
    return baseAtZero(g);
  }
  function roundedBox(w: number, h: number, d: number, r: number, bevel = 0.014): BufferGeometry {
    return extrudeY(roundedRectShape(w, d, r), h, bevel);
  }
  function arcRing(radius: number, tube: number, arc: number): TorusGeometry {
    const g = new TorusGeometry(radius, tube, 10, Math.max(10, Math.round(arc * 44)), arc);
    g.rotateX(-Math.PI / 2);
    return g;
  }
  function pylonMesh(
    name: string,
    geo: BufferGeometry,
    material: MeshStandardMaterial,
    x = 0,
    y = 0,
    z = 0,
  ): Mesh {
    const m = new Mesh(geo, material);
    m.name = name;
    m.position.set(x, y, z);
    return m;
  }
  function onFaces(parent: Group, n: number, cb: (face: Group, side: number) => void): void {
    for (let s = 0; s < n; s++) {
      const face = new Group();
      face.rotation.y = (s / n) * Math.PI * 2;
      cb(face, s);
      parent.add(face);
    }
  }
  /** The head's mounting collar — a glow ring under a plain disc. */
  function collar(g: Group, w: number, wide: number): void {
    g.add(pylonMesh('head_collar_glow', roundedBox(w * wide * 1.07, w * 0.06, w * wide * 1.07, w * 0.4), tracePrimary, 0, -w * 0.05, 0));
    g.add(pylonMesh('head_collar', roundedBox(w * wide, w * 0.14, w * wide, w * 0.36), graphite, 0, 0, 0));
  }
  /** "Пилюля" (pill) head: a capsule cap with a domed top and four light bars. */
  function headPill(g: Group, w: number): void {
    collar(g, w, 1.1);
    g.add(pylonMesh('head_pill', new CylinderGeometry(w * 0.42, w * 0.42, w * 1.08, 44), carbon, 0, w * 0.68, 0));
    g.add(
      pylonMesh('head_pill_dome', new SphereGeometry(w * 0.42, 44, 22, 0, Math.PI * 2, 0, Math.PI / 2), carbon, 0, w * 1.22, 0),
    );
    g.add(pylonMesh('head_pill_ring', arcRing(w * 0.435, w * 0.026, Math.PI * 2), traceViolet, 0, w * 0.36, 0));
    onFaces(g, 4, (face, s) => {
      face.add(
        pylonMesh(
          `head_pill_bar_${s}`,
          new CapsuleGeometry(w * 0.022, w * 0.46, 4, 14),
          s % 2 ? traceViolet : tracePrimary,
          0,
          w * 0.89,
          w * 0.432,
        ),
      );
      face.add(
        pylonMesh(`head_pill_louver_${s}`, roundedBox(w * 0.16, w * 0.045, w * 0.05, w * 0.02, w * 0.012), graphite, 0, w * 1.24, w * 0.36),
      );
    });
    g.add(pylonMesh('head_pill_dot', new CylinderGeometry(w * 0.1, w * 0.11, w * 0.05, 28), bone, 0, w * 1.55, 0));
  }

  const HEAD_W = 0.3;
  const buildPylon = (): { group: Group; emitter: Pylon['emitter']; emGlow: Pylon['emGlow']; tipY: number } => {
    const g = new Group();

    // base: plinth, four capsule feet, riser
    g.add(pylonMesh('plinth', roundedBox(1.18, 0.16, 1.18, 0.3), graphite, 0, 0, 0));
    g.add(pylonMesh('plinth_trim', roundedBox(1.26, 0.022, 1.26, 0.33), tracePrimary, 0, -0.016, 0));
    onFaces(g, 4, (face, s) => {
      face.add(pylonMesh(`foot_${s}`, new CapsuleGeometry(0.05, 0.1, 4, 14), carbon, 0.42, 0.04, 0.42));
    });
    g.add(pylonMesh('plinth_riser', roundedBox(0.72, 0.12, 0.72, 0.2), carbon, 0, 0.17, 0));

    // seven offset blocks, each spun a little further than the last, with
    // circuit strips and louvers alternating by parity
    const blocks = [[0.7, 0.3], [0.66, 0.26], [0.6, 0.3], [0.54, 0.24], [0.48, 0.28], [0.4, 0.22], [0.33, 0.26]];
    let y = 0.29;
    let spin = 0;
    blocks.forEach(([w, h], i) => {
      const block = new Group();
      block.rotation.y = spin;
      block.position.y = y;
      block.add(pylonMesh(`block_body_${i}`, roundedBox(w, h, w, w * 0.22), carbon));
      block.add(
        pylonMesh(`block_seam_${i}`, roundedBox(w * 1.03, 0.018, w * 1.03, w * 0.23), i % 2 ? tracePrimary : traceViolet, 0, h, 0),
      );
      onFaces(block, 4, (face, s) => {
        if ((s + i) % 2 === 0) {
          face.add(
            pylonMesh(`block_strip_${i}`, roundedBox(0.026, h * 0.62, 0.02, 0.012, 0.005), traceViolet, w * 0.3, h * 0.2, w * 0.5 + 0.003),
          );
        }
        face.add(
          pylonMesh(`block_louver_${i}`, roundedBox(w * 0.42, 0.016, 0.024, 0.008, 0.005), graphite, -w * 0.12, h * 0.62, w * 0.5 + 0.003),
        );
        face.add(
          pylonMesh(`block_louver2_${i}`, roundedBox(w * 0.42, 0.016, 0.024, 0.008, 0.005), graphite, -w * 0.12, h * 0.44, w * 0.5 + 0.003),
        );
      });
      g.add(block);
      y += h + 0.018;
      spin += 0.34;
    });

    // the pill head, mounted on top of the block stack
    const host = new Group();
    host.position.y = y;
    g.add(host);
    headPill(host, HEAD_W);

    // the parts that build the belt's cube fly from here, and this is the
    // point beads home in on and flash when absorbed — just above the pill's
    // own top cap (head_pill_dot, at headWidth * 1.55)
    const topY = y + HEAD_W * 1.72;
    const emitter = new Mesh(new SphereGeometry(0.1, 20, 16), tracePrimary);
    emitter.position.y = topY;
    g.add(emitter);
    const emGlow = new Mesh(new SphereGeometry(0.26, 20, 16), glowMat(PRIMARY, 0.2));
    emGlow.position.y = topY;
    g.add(emGlow);

    return { group: g, emitter, emGlow, tipY: topY };
  };

  const pylons: Pylon[] = [];
  // Two identical machines, so both are built at true scale and the camera
  // decides how big each one looks.
  //
  // The comp shrank the second to 0.86 "for depth", which the geometry does not
  // support: after the world transform their distances from the camera are
  // close but not equal, and a size cut on top of that read as a smaller
  // machine, not a farther one.
  //
  // The diagonal isometric view needs the machines to bracket the conveyor in
  // depth as well as in screen space: the first stands behind the far edge and
  // the second in front of the near edge. Their x offset keeps both silhouettes
  // clear while their equal scale lets perspective alone describe the depth.
  // The plinth trim is ~0.63 wide and the tread edge is |z| = 0.58, so these
  // positions also leave a clean physical gap on both sides.
  for (const [name, x, z] of [['pylon_far', 4.5, -2.15], ['pylon_near', 7.9, 2.0]] as const) {
    const p = buildPylon();
    p.group.name = name;
    p.group.position.set(x, -0.2, z);
    world.add(p.group);
    pylons.push({
      tip: new Vector3(x, -0.2 + p.tipY, z),
      tipWorld: new Vector3(),
      emitter: p.emitter,
      emGlow: p.emGlow,
      flash: 0,
    });
  }
  // The tips are needed in world space to project them, and they never move.
  world.updateMatrixWorld(true);
  for (const py of pylons) py.emitter.getWorldPosition(py.tipWorld);

  // ---------- beads flying in from every direction and magnetising to the crowns
  const rndBead = mulberry32(0x62656164);
  const beadGeo = new SphereGeometry(0.055, 12, 10);
  const beadPath = (py: Pylon) => {
    const ang = rndBead() * Math.PI * 2;
    const R = 6.5 + rndBead() * 5.5;
    const from = new Vector3(py.tip.x + Math.cos(ang) * R, 1.2 + rndBead() * 5, py.tip.z + Math.sin(ang) * R * 0.7);
    const ctrl = from.clone().lerp(py.tip, 0.6);
    ctrl.y += 1.1 + rndBead() * 1.2;
    return { curve: new QuadraticBezierCurve3(from, ctrl, py.tip.clone()), dur: 1.7 + rndBead() * 1.5 };
  };
  const beads: Bead[] = [];
  for (let i = 0; i < 5; i++) {
    const mat = i % 3 === 0 ? traceCyan : i % 3 === 1 ? traceViolet : traceMag;
    const mesh = new Mesh(beadGeo, mat);
    const glow = new Mesh(
      new SphereGeometry(0.16, 12, 10),
      glowMat(i % 3 === 0 ? CYAN : i % 3 === 1 ? VIOLET : MAGENTA, 0.2),
    );
    mesh.add(glow);
    world.add(mesh);
    const py = pylons[i % pylons.length];
    beads.push({ mesh, py, ...beadPath(py), t: rndBead() });
  }

  // ---------- in-flight parts
  // Flights launch every CELL_STEP and each lasts TRAVEL, so at most
  // ceil(TRAVEL / CELL_STEP) are in the air at once. A pool that size plus a
  // margin, indexed round-robin by launch number, never hands out a mesh that
  // is still flying — flight n reuses the mesh of flight n - POOL, which landed
  // (POOL - 1) * CELL_STEP > TRAVEL ago.
  const POOL = Math.ceil(TRAVEL / CELL_STEP) + 6;
  const flyGeo = new BoxGeometry(SX * 0.85, SY * 0.85, SZ * 0.85);
  const matFly = new MeshStandardMaterial({ color: 0x3a5378, emissive: CYAN, emissiveIntensity: 0.4, roughness: 0.42, metalness: 0.3 });
  const pool: Mesh[] = [];
  for (let i = 0; i < POOL; i++) {
    const m = new Mesh(flyGeo, matFly);
    m.visible = false;
    m.add(new Mesh(new BoxGeometry(SX * 1.7, SY * 1.7, SZ * 1.7), glowMat(CYAN, 0.09)));
    world.add(m);
    pool.push(m);
  }

  // ---------- the cube in transit from blueprint to belt
  const drop = new Group();
  drop.visible = false;
  world.add(drop);
  const dropMesh = new Mesh(
    cubeGeo,
    new MeshStandardMaterial({
      color: 0xffffff,
      map: texes[0],
      emissive: CYAN,
      emissiveIntensity: 0.22,
      roughness: 0.45,
      metalness: 0.18,
    }),
  );
  const dropEdges = new LineSegments(cubeEdgeGeo, new LineBasicMaterial({ color: CYAN, transparent: true, opacity: 0.85 }));
  const dropGlow = new Mesh(new BoxGeometry(CUBE * 1.16, CUBE * 1.16, CUBE * 1.16), glowMat(CYAN, 0.12));
  drop.add(dropMesh, dropEdges, dropGlow);

  const rndType = mulberry32(0x41726961);
  /**
   * The next file type: uniform over every type except the one just used, so
   * the belt never carries two identical boxes in a row and the order is still
   * unpredictable. Drawing over `TYPES.length - 1` and skipping past the
   * current index keeps every remaining type equally likely, which rerolling
   * on a collision would not.
   */
  const nextType = (current: number) => {
    const pick = Math.floor(rndType() * (TYPES.length - 1));
    return pick >= current ? pick + 1 : pick;
  };
  const state = { phase: 'fill' as Phase, clock: 0, time: 0, landed: 0, made: 0, type: 0, flights: [] as Flight[] };
  setBlueprint(1);

  const spawn = (idx: number) => {
    const mesh = pool[idx % POOL];
    const py = pylons[idx % pylons.length];
    const p = order[idx];
    const to = new Vector3(START_X + partX(p), BUILD_Y + partY(p), partZ(p));
    const ctrl = py.tip.clone().lerp(to, 0.5);
    ctrl.y += 0.7;
    mesh.visible = true;
    state.flights.push({ mesh, t: 0, curve: new QuadraticBezierCurve3(py.tip.clone(), ctrl, to), idx });
  };

  const placeCube = () => {
    const s = makeCube();
    live.push(s);
    s.g.position.set(START_X, BELT_Y + CUBE / 2 + 0.02, 0);
    s.mesh.material.map = texes[state.type];
    s.mesh.material.needsUpdate = true;
    state.type = nextType(state.type);
    dropMesh.material.map = texes[state.type];
    dropMesh.material.needsUpdate = true;
    state.made++;
  };

  const tmp = new Vector3();
  const sim = (dt: number) => {
    state.clock += dt;
    state.time += dt;

    if (state.phase === 'fill') {
      setBlueprint(Math.min(1, state.clock / 0.4));
      const want = Math.min(PARTS, Math.floor(state.clock / CELL_STEP));
      while (state.landed < want) {
        spawn(state.landed);
        state.landed++;
      }
      if (state.landed >= PARTS && state.flights.length === 0) {
        state.phase = 'compact';
        state.clock = 0;
      }
    } else if (state.phase === 'compact') {
      const k = Math.min(1, state.clock / COMPACT);
      const e = easeInOut(k);
      // Gaps close in place — particles grow to exactly fill their own slot,
      // so the finished cube can take over at the same size and position.
      const s = 0.9 + 0.1 * e;
      for (const m of parts) m.scale.setScalar(s);
      matPart.emissiveIntensity = 0.22 + Math.sin(Math.PI * k) * 0.7;
      setBlueprint(Math.max(0, 1 - e * 1.35));
      if (k >= 0.62 && !drop.visible) {
        drop.visible = true;
        drop.position.set(START_X, BUILD_Y, 0);
        drop.scale.setScalar(1);
        for (const m of parts) m.visible = false;
      }
      if (drop.visible) {
        const f = Math.min(1, (k - 0.62) / 0.38);
        dropMesh.material.emissiveIntensity = 0.95 - 0.73 * f;
        dropGlow.material.opacity = 0.3 - 0.18 * f;
      }
      if (k >= 1) {
        parts.forEach((m, i) => {
          const p = order[i];
          m.position.set(partX(p), partY(p), partZ(p));
          m.scale.setScalar(1);
        });
        matPart.emissiveIntensity = 0.22;
        setBlueprint(0);
        dropMesh.material.emissiveIntensity = 0.22;
        dropGlow.material.opacity = 0.12;
        state.phase = 'drop';
        state.clock = 0;
      }
    } else if (state.phase === 'drop') {
      const k = Math.min(1, state.clock / DROP);
      const to = BELT_Y + CUBE / 2 + 0.02;
      drop.position.y = BUILD_Y + (to - BUILD_Y) * easeIn(k);
      const squash = Math.sin(Math.PI * k);
      drop.scale.set(1 + 0.05 * squash, 1 - 0.07 * squash, 1 + 0.05 * squash);
      if (k >= 1) {
        drop.visible = false;
        placeCube();
        state.phase = 'advance';
        state.clock = 0;
      }
    } else if (state.phase === 'advance') {
      const e = easeInOut(Math.min(1, state.clock / ADVANCE));
      for (const s of live) s.g.position.x = s.x - SLOT * e;
      rungs.forEach((r, i) => {
        // The upper run carries the cubes toward -X; the return run travels in
        // the opposite direction like a real continuous tread loop.
        r.group.position.x = i * (SLOT / 2) - 8 - SLOT * e;
        r.bottom.position.x = SLOT * 2 * e;
      });
      if (state.clock >= ADVANCE) {
        for (let i = live.length - 1; i >= 0; i--) {
          const s = live[i];
          s.slot += 1;
          s.x -= SLOT;
          s.g.position.x = s.x;
          if (s.slot > LINE) {
            s.g.visible = false;
            live.splice(i, 1);
            retired.push(s);
          }
        }
        while (retired.length > 1) freeCube(retired.shift()!);
        rungs.forEach((r, i) => {
          r.group.position.x = i * (SLOT / 2) - 8;
          r.bottom.position.x = 0;
        });
        state.phase = 'idle';
        state.clock = 0;
      }
    } else if (state.clock >= IDLE) {
      state.landed = 0;
      state.phase = 'fill';
      state.clock = 0;
    }

    // Rungs fade with distance; cubes keep full presence and ride out of frame.
    // On top of that fade a slow wave runs down the line, so the lit slats read
    // as travelling light rather than paint that happens to be bright.
    for (const r of rungs) {
      const f = Math.max(0, 1 - Math.max(0, r.group.position.x) / 22);
      const lit = r.base * (0.19 + 0.81 * f) * (0.76 + 0.36 * Math.sin(state.time * 2.1 - r.group.position.x * 0.55));
      r.accent.material.opacity = lit;
      r.halo.material.opacity = lit * HALO;
    }

    for (let i = state.flights.length - 1; i >= 0; i--) {
      const f = state.flights[i];
      f.t += dt / TRAVEL;
      const k = Math.min(1, f.t);
      f.curve.getPoint(easeInOut(k), tmp);
      f.mesh.position.copy(tmp);
      f.mesh.scale.setScalar(1.1 - 0.1 * k);
      if (k >= 1) {
        parts[f.idx].visible = true;
        f.mesh.visible = false;
        f.mesh.scale.setScalar(1);
        state.flights.splice(i, 1);
      }
    }

    const pulse = 0.9 + 0.22 * Math.sin(state.time * 1.4);
    traceMag.emissiveIntensity = 1.15 * pulse;
    traceViolet.emissiveIntensity = 1.1 * (1.85 - pulse);
    traceCyan.emissiveIntensity = 1.0 * pulse;
    // Beads home in on the crowns and are absorbed there.
    for (const b of beads) {
      b.t += dt / b.dur;
      if (b.t >= 1) {
        b.py.flash = 1;
        const next = beadPath(b.py);
        b.curve = next.curve;
        b.dur = next.dur;
        b.t = 0;
      }
      const k = Math.min(1, b.t);
      b.curve.getPoint(k * k, tmp);
      b.mesh.position.copy(tmp);
      b.mesh.scale.setScalar(0.6 + 0.6 * (1 - k) + 0.5 * Math.max(0, k - 0.85) * 4);
    }
    for (const py of pylons) {
      py.flash = Math.max(0, py.flash - dt * 2.2);
      py.emGlow.material.opacity = 0.2 + py.flash * 0.45;
      py.emitter.scale.setScalar(1 + py.flash * 0.4);
    }
  };

  /**
   * Highest of the two pylon tips, in normalised device coordinates, for a
   * given aim height. Raising the aim tilts the camera up and pushes the
   * subject down, so this decreases monotonically in `y` — which is what makes
   * the bisection below sound.
   */
  const probe = new Vector3();
  const tipNdcAt = (x: number, y: number, z: number) => {
    camera.lookAt(x, y, z);
    camera.updateMatrixWorld();
    let top = -Infinity;
    for (const py of pylons) top = Math.max(top, probe.copy(py.tipWorld).project(camera).y);
    return top;
  };

  const resize = () => {
    const w = host.clientWidth || 960;
    const h = host.clientHeight || 520;
    renderer.setSize(w, h, false);
    const aspect = w / Math.max(1, h);
    camera.aspect = aspect;

    const { fov, target } = framingFor(aspect);
    camera.fov = fov;
    // Before probing: `project` reads the projection matrix.
    camera.updateProjectionMatrix();

    const want = alignTipsNdc?.() ?? null;
    let aimY = target.y;
    if (want !== null && Number.isFinite(want)) {
      // 40 halvings over a range that covers every reachable aim, so the tips
      // land on the line to well under a pixel.
      let lo = -12;
      let hi = 16;
      for (let i = 0; i < 40; i++) {
        const mid = (lo + hi) / 2;
        if (tipNdcAt(target.x, mid, target.z) > want) lo = mid;
        else hi = mid;
      }
      aimY = (lo + hi) / 2;
    }
    camera.lookAt(target.x, aimY, target.z);
    renderer.render(scene, camera);
  };
  const ro = new ResizeObserver(resize);
  ro.observe(host);
  resize();

  let visible = true;
  let io: IntersectionObserver | undefined;
  if ('IntersectionObserver' in window) {
    io = new IntersectionObserver((e) => void (visible = e[0].isIntersecting), { threshold: 0 });
    io.observe(host);
  }

  // Prime until every belt slot carries a cube, so the line is never empty.
  // Terminates by construction: each cycle places exactly one cube.
  while (state.made < LINE + 2) sim(STEP * 2.4);

  let alive = true;
  let raf = 0;
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (reduced) {
    renderer.render(scene, camera);
  } else {
    let last = performance.now();
    let acc = 0;
    // A machine with no GPU falls back to software GL, where one frame of this
    // scene costs longer than the frame it is drawing. The loop then never
    // hands the main thread back: measured in a headless Chrome on SwiftShader,
    // the page never reaches network idle at all. So it watches what a frame
    // costs and stops, leaving the last one painted - the same still image
    // reduced motion gets.
    let slow = 0;
    const frame = (now: number) => {
      if (!alive) return;
      raf = requestAnimationFrame(frame);
      const dt = Math.min(0.25, (now - last) / 1000);
      last = now;
      if (!visible) return;
      acc += dt;
      while (acc >= STEP) {
        sim(STEP);
        acc -= STEP;
      }
      const t0 = performance.now();
      renderer.render(scene, camera);
      if (performance.now() - t0 < SLOW_FRAME) {
        slow = 0;
        return;
      }
      if (++slow >= SLOW_STREAK) alive = false;
    };
    raf = requestAnimationFrame(frame);
  }

  return {
    dispose() {
      alive = false;
      cancelAnimationFrame(raf);
      ro.disconnect();
      io?.disconnect();
      renderer.dispose();
    },
    debug: () => ({
      phase: state.phase,
      landed: state.landed,
      made: state.made,
      live: live.length,
      retired: retired.length,
      flights: state.flights.length,
      visible,
    }),
  };
}
