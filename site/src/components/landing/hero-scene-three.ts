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
  Color,
  CylinderGeometry,
  DirectionalLight,
  EdgesGeometry,
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
  SphereGeometry,
  TorusGeometry,
  Vector3,
  WebGLRenderer,
} from 'three';
import { mulberry32 } from '@/lib/mock-motion';

// The page around this canvas is the comp's dark stage (`data-theme="dark"`
// plus `data-accent="blue"`), so the scene is lit for that ground and not for
// the site's default neutral one. BG has to equal what the page paints — the fog and the pylons' sinking supports fade into it, and a canvas
// that fades to a slightly different black leaves a visible rectangle.
const BG = 0x0b0b0f;
const PRIMARY = 0xa8c7fa;
const VIOLET = 0xc6b2ff;
const CYAN = 0x7dcfee;
const MAGENTA = 0xbb9af7;

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
function typeTexture(type: FileType, mirrorText: boolean): CanvasTexture {
  const c = document.createElement('canvas');
  c.width = c.height = 256;
  // A fresh canvas always yields a 2D context in a browser that got this far
  // (the wrapper only boots when WebGL exists).
  const ctx = c.getContext('2d')!;
  ctx.fillStyle = '#0f1726';
  ctx.fillRect(0, 0, 256, 256);
  ctx.strokeStyle = 'rgba(125,207,238,0.45)';
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
  ctx.fillStyle = '#e8f4ff';
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
  const belt = new Mesh(
    new BoxGeometry(27, 0.1, 1.2),
    new MeshStandardMaterial({ color: 0x191a2c, roughness: 0.66, metalness: 0.26 }),
  );
  belt.position.set(5, BELT_Y - 0.05, 0);
  world.add(belt);
  for (const z of [-0.62, 0.62]) {
    const rail = new Mesh(
      new BoxGeometry(27, 0.045, 0.055),
      new MeshStandardMaterial({ color: VIOLET, emissive: VIOLET, emissiveIntensity: 1.25, roughness: 0.3, metalness: 0.4 }),
    );
    rail.position.set(5, BELT_Y + 0.012, z);
    world.add(rail);
    const rg = new Mesh(new BoxGeometry(27, 0.18, 0.2), glowMat(VIOLET, 0.13));
    rg.position.copy(rail.position);
    world.add(rg);
  }
  const rungs: Mesh<BoxGeometry, MeshBasicMaterial>[] = [];
  const rungGeo = new BoxGeometry(0.055, 0.02, 1.05);
  for (let i = 0; i < 54; i++) {
    const r = new Mesh(rungGeo, new MeshBasicMaterial({ color: VIOLET, transparent: true, opacity: 0.4 }));
    r.position.set(i * (SLOT / 2) - 8, BELT_Y + 0.012, 0);
    world.add(r);
    rungs.push(r);
  }

  // ---------- finished cubes (opaque)
  const texes = TYPES.map((type) => typeTexture(type, mirrorText));
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

  // ---------- pylons: octagonal segmented towers, no spikes, opaque
  const shellMat = new MeshStandardMaterial({ color: 0x342c4e, roughness: 0.44, metalness: 0.4 });
  const shellDark = new MeshStandardMaterial({ color: 0x241e39, roughness: 0.5, metalness: 0.36 });
  const traceMag = new MeshStandardMaterial({ color: MAGENTA, emissive: MAGENTA, emissiveIntensity: 1.25, roughness: 0.25, metalness: 0.3 });
  const traceViolet = new MeshStandardMaterial({ color: VIOLET, emissive: VIOLET, emissiveIntensity: 1.2, roughness: 0.25, metalness: 0.3 });
  const traceCyan = new MeshStandardMaterial({ color: CYAN, emissive: CYAN, emissiveIntensity: 1.05, roughness: 0.25, metalness: 0.3 });
  const outlineViolet = new LineBasicMaterial({ color: VIOLET, transparent: true, opacity: 0.38 });

  const buildPylon = (): { group: Group; emitter: Pylon['emitter']; emGlow: Pylon['emGlow']; tipY: number } => {
    const p = new Group();
    const traceOf = (i: number) => (i % 2 ? traceViolet : traceMag);
    const TIER = 0.66;
    const CAP = 0.07;

    // base: plate, two neon rings, four anchors
    const plate = new Mesh(new CylinderGeometry(1.06, 1.14, 0.07, 56), shellDark);
    plate.position.y = 0.035;
    p.add(plate);
    for (const [r, alt] of [[0.99, 0], [0.79, 1]]) {
      const mat = alt ? traceMag : traceViolet;
      const ring = new Mesh(new TorusGeometry(r, 0.018, 10, 72), mat);
      ring.rotation.x = Math.PI / 2;
      ring.position.y = 0.085 + alt * 0.008;
      p.add(ring);
      const rg = new Mesh(new TorusGeometry(r, 0.075, 10, 72), glowMat(alt ? MAGENTA : VIOLET, 0.17));
      rg.rotation.x = Math.PI / 2;
      rg.position.y = ring.position.y;
      p.add(rg);
    }
    for (let i = 0; i < 4; i++) {
      const ang = (Math.PI * 2 * i) / 4 + Math.PI / 4;
      const anchor = new Mesh(new BoxGeometry(0.3, 0.14, 0.18), shellMat);
      anchor.position.set(Math.cos(ang) * 0.93, 0.12, Math.sin(ang) * 0.93);
      anchor.rotation.y = -ang;
      p.add(anchor);
      const al = new Mesh(new BoxGeometry(0.22, 0.032, 0.032), traceViolet);
      al.position.set(Math.cos(ang) * 0.93, 0.2, Math.sin(ang) * 0.93);
      al.rotation.y = -ang;
      p.add(al);
    }

    // four stacked tiers, each with circuit strips on all faces
    for (let t = 0; t < 4; t++) {
      const w = 0.68 - t * 0.05;
      const y = 0.24 + TIER / 2 + t * (TIER + CAP);
      const body = new Mesh(new BoxGeometry(w, TIER, w), shellMat);
      body.position.y = y;
      p.add(body);
      const bodyEdge = new LineSegments(new EdgesGeometry(new BoxGeometry(w, TIER, w)), outlineViolet);
      bodyEdge.position.y = y;
      p.add(bodyEdge);
      const cap = new Mesh(new BoxGeometry(w + 0.09, CAP, w + 0.09), shellDark);
      cap.position.y = y + TIER / 2 + CAP / 2;
      p.add(cap);
      const capEdge = new LineSegments(new EdgesGeometry(new BoxGeometry(w + 0.09, CAP, w + 0.09)), outlineViolet);
      capEdge.position.y = cap.position.y;
      p.add(capEdge);
      [[1, 0], [-1, 0], [0, 1], [0, -1]].forEach(([sx, sz], fi) => {
        const mat = traceOf(t + fi);
        const strip = new Mesh(new BoxGeometry(sx ? 0.016 : 0.032, TIER * 0.68, sz ? 0.016 : 0.032), mat);
        strip.position.set(sx ? sx * (w / 2 + 0.009) : 0, y, sz ? sz * (w / 2 + 0.009) : 0);
        p.add(strip);
        const sg = new Mesh(
          new BoxGeometry(sx ? 0.06 : 0.1, TIER * 0.72, sz ? 0.06 : 0.1),
          glowMat(mat === traceMag ? MAGENTA : VIOLET, 0.1),
        );
        sg.position.copy(strip.position);
        p.add(sg);
        const elbow = new Mesh(new BoxGeometry(sx ? 0.014 : 0.11, 0.02, sz ? 0.014 : 0.11), mat);
        elbow.position.set(strip.position.x, y + TIER * 0.34, strip.position.z);
        if (sx) elbow.scale.z = 7;
        p.add(elbow);
      });
    }

    // crown: four prongs and the emitter the parts come from
    const topY = 0.24 + 4 * (TIER + CAP);
    for (let i = 0; i < 4; i++) {
      const ang = (Math.PI * 2 * i) / 4 + Math.PI / 4;
      const prong = new Mesh(new BoxGeometry(0.12, 0.36, 0.12), shellMat);
      prong.position.set(Math.cos(ang) * 0.18, topY + 0.18, Math.sin(ang) * 0.18);
      prong.rotation.y = -ang;
      p.add(prong);
      const pl = new Mesh(new BoxGeometry(0.034, 0.3, 0.034), i % 2 ? traceViolet : traceMag);
      pl.position.set(Math.cos(ang) * 0.235, topY + 0.18, Math.sin(ang) * 0.235);
      p.add(pl);
      const plg = new Mesh(new BoxGeometry(0.1, 0.32, 0.1), glowMat(i % 2 ? VIOLET : MAGENTA, 0.12));
      plg.position.copy(pl.position);
      p.add(plg);
    }
    const emitter = new Mesh(new SphereGeometry(0.16, 20, 16), traceMag);
    emitter.position.y = topY + 0.44;
    p.add(emitter);
    const emGlow = new Mesh(new SphereGeometry(0.36, 20, 16), glowMat(MAGENTA, 0.2));
    emGlow.position.y = emitter.position.y;
    p.add(emGlow);

    // support: segments that darken as they sink out of frame, with faint
    // energy lines carried over from the tiers
    for (let s = 0; s < 7; s++) {
      const f = s / 6;
      const h = 0.55;
      const y = -0.12 - s * h;
      // the support sinks into the page background and is gone by the last segment
      const col = new Color(0x241e39).lerp(new Color(BG), Math.min(1, f * 1.5));
      const segMesh = new Mesh(
        new CylinderGeometry(0.55 + s * 0.03, 0.58 + s * 0.03, h, 32),
        new MeshStandardMaterial({
          color: col,
          roughness: 0.55,
          metalness: 0.3,
          transparent: true,
          opacity: Math.max(0, 1 - Math.pow(f, 0.85) * 1.15),
        }),
      );
      segMesh.position.y = y;
      p.add(segMesh);
      if (s < 5) {
        for (let i = 0; i < 4; i++) {
          const ang = (Math.PI * 2 * i) / 4 + Math.PI / 4;
          const r = 0.57 + s * 0.03;
          const fade = Math.pow(Math.max(0, 1 - f * 1.25), 1.6);
          const seam = new Mesh(
            new BoxGeometry(0.03, h * 0.7, 0.03),
            new MeshBasicMaterial({ color: i % 2 ? VIOLET : MAGENTA, transparent: true, opacity: 0.75 * fade }),
          );
          seam.position.set(Math.cos(ang) * r, y, Math.sin(ang) * r);
          p.add(seam);
          const sg = new Mesh(new BoxGeometry(0.1, h * 0.72, 0.1), glowMat(i % 2 ? VIOLET : MAGENTA, 0.1 * fade));
          sg.position.copy(seam.position);
          p.add(sg);
        }
      }
    }

    return { group: p, emitter, emGlow, tipY: emitter.position.y };
  };

  const pylons: Pylon[] = [];
  // Two identical machines, so both are built at true scale and the camera
  // decides how big each one looks.
  //
  // The comp shrank the second to 0.86 "for depth", which the geometry does not
  // support: after the world transform their distances from the camera are
  // 16.9 and 17.3 units, 2% apart. A 14% reduction on a 2% difference does not
  // read as distance, it reads as a smaller machine.
  //
  // Bases must still clear the belt band. The plate radius is 1.14, the band is
  // |z| < 0.62, and at |z| of 2.15 and 2.0 both leave a gap (1.01 and 0.86).
  for (const [x, z] of [[4.5, -2.15], [7.9, 2.0]]) {
    const p = buildPylon();
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
      rungs.forEach((r, i) => void (r.position.x = i * (SLOT / 2) - 8 - SLOT * e));
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
        rungs.forEach((r, i) => void (r.position.x = i * (SLOT / 2) - 8));
        state.phase = 'idle';
        state.clock = 0;
      }
    } else if (state.clock >= IDLE) {
      state.landed = 0;
      state.phase = 'fill';
      state.clock = 0;
    }

    // Rungs fade with distance; cubes keep full presence and ride out of frame.
    for (const r of rungs) {
      const f = Math.max(0, 1 - Math.max(0, r.position.x) / 22);
      r.material.opacity = 0.05 + 0.26 * f;
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
