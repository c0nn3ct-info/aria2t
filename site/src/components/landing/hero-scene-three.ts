// The hero's assembly line: two neon pylons feed particles into a transparent
// blueprint volume; when the blueprint is full the particles compact into one
// solid, type-specific file crate, the crate drops onto the belt and the line
// steps toward the viewer. Nothing rotates. Deterministic: a fixed 1/120 step,
// with the order drawn from mulberry32 — the same PRNG the two
// live mocks seed from (src/lib/mock-motion.ts).
//
// This module is loaded on demand by `hero-scene.tsx`, so three.js stays out
// of the page's main chunk; the wrapper also decides whether to boot at all
// (WebGL present, not the prerender).
import type { Blending } from 'three';
import {
  AdditiveBlending,
  Box3,
  BoxGeometry,
  BufferGeometry,
  CanvasTexture,
  CapsuleGeometry,
  CylinderGeometry,
  DirectionalLight,
  EdgesGeometry,
  ExtrudeGeometry,
  Fog,
  Group,
  HemisphereLight,
  LineBasicMaterial,
  LineSegments,
  Material,
  Mesh,
  MeshBasicMaterial,
  MeshStandardMaterial,
  NormalBlending,
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
import { crateIconShapes } from './hero-crate-icons';
import { ballToCube, morphAt } from './hero-fly-morph';

// Which stage this scene stands on — and so every colour in it — lives in
// `hero-scene-palette.ts`. Re-exported here because `hero-scene.tsx` reaches
// this module through a dynamic `import()` and reads `isDark` off it to notice
// a live theme flip; importing the palette module from the wrapper directly
// would pull three.js's `Color` back into the page's main chunk, which is the
// one thing this split exists to prevent.
export { isDark } from './hero-scene-palette';
import { scenePalette } from './hero-scene-palette';

const CUBE = 0.78;
/** The crate model's body: a 0.8 cube, which its 0.028 bevel draws 0.856
 * across (see `roundedBox`). `CRATE_SCALE` maps that onto the blueprint cube,
 * which lands the whole crate — plinth base to lid plate — at `CUBE` tall. */
const CRATE_BODY = 0.8;
const CRATE_BEVEL = 0.028;
const CRATE_SCALE = CUBE / (CRATE_BODY + 2 * CRATE_BEVEL);
/** Mid-height of the crate model's solid: plinth base 0, lid plate top 0.854. */
const CRATE_MID = 0.427;
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
/** One slat pitch, and the strip of them the tread is cut into. `TREADS` is a
 * multiple of six, the period of the slats' lit pattern, so the strip can be
 * recycled through itself without the pattern jumping. */
const TREAD = SLOT / 2;
const TREADS = 90;
const STRIP = TREADS * TREAD;
/** Near end of the tread strip, where a recycled slat re-enters. Far enough
 * back that the strip's own end is never in frame: a composition standing
 * beside the copy aims further down the line than one under it, and at 844 the
 * old -8 put the belt's flat near terminus in the bottom-left corner. */
const TREAD_X0 = -16;
/** The band width from which the composition stands beside the copy rather
 * than under it. The `md` breakpoint, where the copy's measure is capped and
 * its rows stop leaving the pair a corner. */
const BESIDE_FROM = 768;
/** The belt body under it: long enough that its far end is always deeper than
 * the fog, since an ultra-wide frame sees a long way down the line. The fog is
 * what ends the line rather than the frame's edge - it dies into the page at
 * the same depth whatever the window does, where a wash at the window's edge
 * would have to know where the machines are standing not to dim them too. */
const BELT_LEN = 44;
const BELT_MID = TREAD_X0 - 0.5 + BELT_LEN / 2;
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
//     copy runs the full width and the machinery gets the strip under it. The
//     machines stand right of centre with the belt crossing out of the
//     bottom-left corner: the same reading the desktop has, rotated. Aiming
//     lower than the machines tilts the camera down and lifts them in frame,
//     which is why the target sits above them, and its x sits *past* the
//     station they flank - 6.6 against 5.6 - which is what carries the pair
//     back towards the middle of a narrow frame.
//   wide (aspect 1.90 and above) - the machines beside the copy column with
//     the belt crossing the rest. At 1440x720 this resolves to a 21.6 degree
//     field and the aim point the composition was drawn at.
//
// Both H are the composition's own, divided by 1.4: the scene was drawn at a
// 30 degree wide field and read too small on the page - a belt of crates whose
// pictograms were a few pixels each, with a third of the frame empty to the
// right of the machines. Zooming in costs the far end of the belt, which the
// fog was already taking, and nothing else.
//
// Narrower than the portrait anchor it is the *vertical* field that is held
// (see `framingFor`), and past ZOOM_STOP an ultra-wide window stops narrowing
// the field at all, or the vertical would squeeze until the pylons clipped.
//
// These are the composition's starting point, not its answer: `place` in
// `resize` stands the pair against what the page actually leaves it - beside
// the copy column, under the line the hero states, inside the frame - and
// eases the field back out until it fits. Which is why 1024, where the copy
// column takes three fifths of the band, draws the machines smaller instead of
// standing them in the sentences, and a 320px phone draws them smaller again
// instead of pushing them through the bottom edge. Checked by projecting both
// machines over every hero shape from 320x620 to 3440x1440, at every line and
// column edge the hero hands over at that shape (`holds both machines in frame
// and clear of the copy at every hero shape`). The belt is deliberately not
// held: it runs out of the frame's bottom-left corner by design, and the
// closing fade is what swallows it.
interface Framing {
  /** Horizontal half-extent. */
  h: number;
  target: { x: number; y: number; z: number };
}
const PORTRAIT: Framing & { aspect: number } = {
  aspect: 0.97,
  h: 0.21,
  target: { x: 6.6, y: 1.9, z: -2.5 },
};
const WIDE: Framing & { aspect: number } = {
  aspect: 1.9,
  h: Math.tan(10.8 * DEG) * 2.0,
  target: { x: 3.1, y: 1.05, z: 0.8 },
};
const ZOOM_STOP = 2.6;

const mix = (a: number, b: number, k: number) => a + (b - a) * k;

function framingFor(aspect: number): Framing {
  const k = Math.max(0, Math.min(1, (aspect - PORTRAIT.aspect) / (WIDE.aspect - PORTRAIT.aspect)));
  // Narrower than the portrait anchor the *vertical* field is what is held,
  // not the horizontal one: a phone band is as tall as it is wide, and a
  // fixed H there shrinks the machines by exactly as much as the frame
  // narrows. `place` eases whatever does not fit back out again.
  return {
    h: aspect < PORTRAIT.aspect ? PORTRAIT.h * (aspect / PORTRAIT.aspect) : mix(PORTRAIT.h, WIDE.h, k),
    target: {
      x: mix(PORTRAIT.target.x, WIDE.target.x, k),
      y: mix(PORTRAIT.target.y, WIDE.target.y, k),
      z: mix(PORTRAIT.target.z, WIDE.target.z, k),
    },
  };
}

interface FileType {
  kind: 'image' | 'video' | 'audio' | 'doc';
}

const TYPES: readonly FileType[] = [
  { kind: 'image' },
  { kind: 'video' },
  { kind: 'audio' },
  { kind: 'doc' },
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

function haloTexture([inner, mid]: readonly [string, string]): CanvasTexture {
  const c = document.createElement('canvas');
  c.width = c.height = 256;
  const ctx = c.getContext('2d')!;
  const g = ctx.createRadialGradient(128, 128, 0, 128, 128, 128);
  g.addColorStop(0, inner);
  g.addColorStop(0.42, mid);
  g.addColorStop(1, 'rgba(0,0,0,0)');
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, 256, 256);
  return new CanvasTexture(c);
}

/**
 * One of the scene's glow meshes. Every additive surface in here goes through
 * this, which is why the two stages cost exactly one branch each: `blending`
 * is set once at construction, so the five places that animate a glow's
 * `opacity` at runtime need to know nothing about the stage.
 *
 * The authored alphas are shared between stages deliberately. Additive over
 * near-black and normal over near-white land at comparable subtlety at the
 * low end, and at the high end — the lit slats' 0.55 — normal blending reads
 * *stronger*, which is what a daylight trace wants: saturated ink rather than
 * a bloom that has nothing left to add to.
 */
const glowMat = (color: number, opacity: number, blending: Blending) =>
  new MeshBasicMaterial({ color, transparent: true, opacity, blending, depthWrite: false });

type Phase = 'fill' | 'compact' | 'drop' | 'advance' | 'idle';

interface CrateVisual {
  g: Group;
  /** The crate's lit trim, flashed as it lands. Every other material it wears
   * is one of the scene's shared neutrals. */
  accent: MeshStandardMaterial;
  glow: MeshBasicMaterial;
}

interface Cube extends CrateVisual {
  slot: number;
  x: number;
}

interface Flight {
  mesh: Mesh;
  t: number;
  curve: QuadraticBezierCurve3;
  idx: number;
}

/** One slat of the tread's upper run: a graphite crossbar plus a glowing
 * light-line insert that fades with distance. */
interface Rung {
  group: Group;
  accent: Mesh<BufferGeometry, MeshBasicMaterial>;
  /** Oversized additive shell around the accent — the glow's spill. */
  halo: Mesh<BufferGeometry, MeshBasicMaterial>;
  /** Peak opacity for the accent — brighter on the periodic highlighted slats. */
  base: number;
  /** The spill's own peak. Not `base * HALO`: the line is trim and the spill
   * around it is a bloom, and the two convert to a light stage by different
   * factors, so each carries its already-converted peak and they share only
   * the wave that animates them. */
  haloBase: number;
}

interface Pylon {
  /** The machine itself, for measuring where it stands in the frame. */
  group: Group;
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
 * The crates use geometry-only pictograms and require no texture options.
 */
export interface HeroSceneOptions {
  /**
   * Where the pylon tips should sit, in normalised device coordinates: +1 is
   * the top edge of the canvas, -1 the bottom. The hero hands over the
   * vertical centre of its first heading line, and the aim point is solved
   * against it on every resize, which is what keeps the machines standing
   * level with that line at any size. Returning null leaves the aim at the
   * interpolated default.
   */
  alignTipsNdc?: () => number | null;
  /**
   * The right edge of the copy column, in the same coordinates, past which the
   * machines must stand. Returning null says the copy is stacked above the
   * scene instead of beside it, and the machines have the band's whole width.
   */
  copyEdgeNdc?: () => number | null;
  /**
   * The line the copy's own last row ends on, in the same coordinates. The
   * scene stands the pair under it when standing it beside the copy would come
   * out smaller, which is what a phone's full-width rows mean.
   */
  copyBottomNdc?: () => number | null;
}

export function bootHeroScene(
  host: HTMLElement,
  canvas: HTMLCanvasElement,
  { alignTipsNdc, copyEdgeNdc, copyBottomNdc }: HeroSceneOptions = {},
): HeroSceneHandle {
  // No `preserveDrawingBuffer`: the comp set it so the design tool could
  // capture a thumbnail, and it costs a retained copy of the framebuffer
  // every frame. Nothing here reads pixels back.
  const P = scenePalette();
  const { bg: BG, cyan: CYAN, magenta: MAGENTA, green: GREEN } = P;
  /** The machines' own trim colours. On the dark stage these *are* the page's
   * accents; on the light one they are the models' vivid periwinkle and
   * lilac, because a white ground gives a trace no brightness to win on and
   * saturation is all that is left. */
  const { traceA: TRACE_A, traceB: TRACE_B } = P;
  /** How hard anything self-lit burns on this stage. The light models drop
   * their accent emissive strength from 0.72 to 0.16 — glow is what a dark
   * stage has instead of daylight, and on a bright one the same value reads as
   * a blown-out smear. Every emissive number below is authored for the dark
   * stage and scaled by this, so there is one ratio to tune rather than a
   * second set of a dozen constants. */
  const EM = P.additive ? 1 : 0.22;
  /** The trim's own burn. Higher than `EM` on a light stage: the strips are
   * the only colour the machines have and have to read as lit, where a part
   * is a white shell that wants its emissive out of the way. */
  const TRIM_EM = P.trimEmissive;
  /** An authored metalness, corrected for this stage. Nothing here has an
   * environment map, so metal has nothing to reflect and only darkens; see
   * `Palette.metalScale`. */
  const mt = (m: number) => m * P.metalScale;
  const BLEND = P.additive ? AdditiveBlending : NormalBlending;
  /** An authored glow alpha, converted to this stage. Every alpha below is
   * written for the dark stage's additive blend; `ga` is what makes the same
   * number read at the same strength once it is blended normally over white
   * instead (see `Palette.glowAlpha`). Applied at the authored sites rather
   * than inside `stageGlow`, because the alphas that animate are read back
   * out of constants and would otherwise scale twice. */
  const ga = (a: number) => Math.min(1, a * P.glowAlpha);
  /** The same, for a bloom — the oversized shell around something bright,
   * which is light that escaped and so needs dark to escape into. See
   * `Palette.bloomAlpha`. */
  const ba = (a: number) => Math.min(1, a * P.bloomAlpha);
  /** The same, for a trim line's spill — the soft widening beside the line
   * that is what actually makes a painted strip read as a lit one. See
   * `Palette.spillAlpha`. */
  const sa = (a: number) => Math.min(1, a * P.spillAlpha);
  /** `glowMat` bound to this stage's blend, so the seven glow sites below read
   * the same either way. */
  const stageGlow = (color: number, opacity: number) => glowMat(color, opacity, BLEND);
  const renderer = new WebGLRenderer({ canvas, alpha: true, antialias: true });
  renderer.setPixelRatio(Math.min(2, window.devicePixelRatio || 1));
  const scene = new Scene();
  scene.fog = new Fog(BG, 13, 26);
  const camera = new PerspectiveCamera(30, WIDE.aspect, 0.1, 160);
  camera.position.set(-1.6, 4.3, 11.6);
  camera.lookAt(WIDE.target.x, WIDE.target.y, WIDE.target.z);

  // Neither authoring GLB carries a light or a camera, so the rig is the
  // scene's own on both stages — and it cannot be shared. The light models'
  // shells sit at metalness 0.06 and near-white, where the dark stage's rim
  // intensities clip them to flat paper.
  scene.add(new HemisphereLight(P.rig.sky, P.rig.ground, P.rig.hemi));
  const key = new DirectionalLight(0xffffff, P.rig.key);
  key.position.set(2.6, 6.4, 5.8);
  scene.add(key);
  const rimA = new PointLight(P.rig.rimAColor, P.rig.rimA, 20);
  rimA.position.set(3.4, 2.8, 2.4);
  scene.add(rimA);
  const rimB = new PointLight(P.rig.rimBColor, P.rig.rimB, 18);
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
      map: haloTexture(P.haloStops),
      transparent: true,
      blending: BLEND,
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
  // The structural neutrals every machine in this scene is built from, and
  // they are the authoring models' own materials: `carbon` and `graphite` are
  // the GLB's two shell materials converted out of linear light, `bone` is its
  // pictogram material, `belt` is the rubber of the tread. The belt, the
  // pylons and the crates all share them, so nothing here is per-instance —
  // but all four *do* flip with the stage, because `.claude/3d/light` is a
  // separately authored material set rather than a tint of the dark one: the
  // shells inverting to near-white is the whole difference between a machine
  // standing in daylight and a machine cut out of the page.
  const carbon = new MeshStandardMaterial(P.carbon);
  const graphite = new MeshStandardMaterial(P.graphite);
  const bone = new MeshStandardMaterial(P.bone);
  // The chassis: the closed loop the slats ride on. Near-black on the dark
  // stage, where the belt reads as light slats crossing a dark frame; pale
  // glass on the light one, where that same black is a wedge heavy enough to
  // take over the band. Transparent, so it renders after the return run and
  // the tread loop shows faintly through its housing.
  const belt = new MeshStandardMaterial(P.belt);
  const beltBody = new Mesh(new BoxGeometry(BELT_LEN, 0.36, 1.08), belt);
  beltBody.name = 'belt_body';
  beltBody.position.set(BELT_MID, BELT_Y - 0.2, 0);
  world.add(beltBody);

  // A shallow inset on the near and far faces gives the body the same closed
  // tread-loop read as the source model. It deliberately contains no drums,
  // axles or idlers: the user wants the belt itself, without visible rollers.
  const beltSideGeo = new BoxGeometry(BELT_LEN, 0.2, 0.025);
  for (const z of [-0.55, 0.55]) {
    // Shell, not chassis: a thin inset strip that keeps its closed-tread-loop
    // read on the dark stage and, on the light one, draws the pale line down
    // the black frame that the models detail their chassis with.
    const side = new Mesh(beltSideGeo, graphite);
    side.name = 'belt_side';
    side.position.set(BELT_MID, BELT_Y - 0.2, z);
    world.add(side);
  }

  const rungs: Rung[] = [];
  const slatGeo = roundedBox(0.38, 0.075, 1.16, 0.055, 0.016);
  const accentGeo = roundedBox(0.2, 0.014, 1.02, 0.035, 0.006);
  // There is no bloom pass in this scene, so the accent glows the way the cubes
  // and the emitters do: an additive core under an oversized, fainter additive
  // shell that spills past the slat and softens its edge.
  const accentHaloGeo = roundedBox(0.46, 0.01, 1.26, 0.09, 0.004);
  // The lower run is its own strip rather than a second slat parented to each
  // upper one: it travels the other way, and `layTread` can only keep both
  // runs continuous if it moves them independently.
  const returnRun = new Group();
  returnRun.name = 'belt_return';
  returnRun.position.y = BELT_Y - 0.48;
  world.add(returnRun);
  for (let i = 0; i < TREADS; i++) {
    const group = new Group();
    group.name = `belt_tread_${i}`;
    group.add(new Mesh(slatGeo, graphite));
    const bright = i % 3 === 0;
    const color = Math.floor(i / 3) % 2 === 0 ? TRACE_B : TRACE_A;
    const peak = bright ? 0.55 : 0.16;
    const base = ga(peak);
    const accent = new Mesh(accentGeo, stageGlow(color, base));
    accent.position.y = 0.075;
    const haloBase = sa(peak * HALO);
    const halo = new Mesh(accentHaloGeo, stageGlow(color, haloBase));
    halo.position.y = 0.079;
    group.add(accent, halo);
    group.position.set(i * TREAD + TREAD_X0, BELT_Y - 0.04, 0);
    world.add(group);
    rungs.push({ group, accent, halo, base, haloBase });
    returnRun.add(partMesh(`belt_return_${i}`, slatGeo, graphite, i * TREAD + TREAD_X0, 0, 0));
  }

  /**
   * Lays both runs of the tread out for a travel of `shift`, in world units
   * along -X. A slat that would run off the near end is recycled to the far
   * end instead of the strip ever being put back where it started: the belt
   * used to be re-laid at its original x after every advance, which snapped
   * the whole tread back a slot and read as the line twitching backwards.
   * `TREADS` is a multiple of the six-slat lit pattern, so a recycled slat
   * lands where an identically lit one would have been.
   */
  const layTread = (shift: number) => {
    const s = ((shift % STRIP) + STRIP) % STRIP;
    for (let i = 0; i < rungs.length; i++) {
      const x = i * TREAD + TREAD_X0 - s;
      rungs[i].group.position.x = x < TREAD_X0 ? x + STRIP : x;
    }
    // Every slat on the lower run is the same plain graphite, evenly spaced,
    // so moving that strip within a single pitch is all the return needs to
    // read as continuous travel the other way.
    returnRun.position.x = s % TREAD;
  };

  // ---------- finished file crates (opaque)
  //
  // A port of the authoring model
  // `.claude/Киберпанк пилоны_ четыре варианта/icon-crate-audio.glb`: a plinth
  // under a lit flange, a softly rounded carbon body, an inset panel with a
  // lit frame and a pictogram on each of the four sides, and a raised lid
  // plate with a pictogram of its own. The model is *universal* — it carries
  // all four pictograms at once, one per side plus four on the lid, and a
  // different accent per side — but a crate on this belt is one file, so here
  // every side and the lid carry the same pictogram and the whole shell takes
  // that type's accent.
  //
  // Every number below is the model's own, in model units, read out of the
  // GLB: heights and offsets from each node's accessor bounds through its
  // world matrix, bevels from the extrusion's y levels, corner radii from
  // where a rounded profile's straight run ends. They land on `roundedBox`
  // exactly, because the model was built by the same M3-style rounded-rect
  // extrusion this file already carries for the pylons — every part of it is
  // a ratio of the 0.8 body: the plinth 1.14 of it, its flange 1.19, the lid
  // plate 0.84, a panel 0.8, a frame rail 0.86 long and 0.02 thick.
  //
  // Widths are the *extruded* value, not the measured one: three.js bulges an
  // extrusion out by `bevelSize` between its end caps, so a part measures
  // 2 × bevel wider than the box it was asked for (the 0.8 body draws 0.856,
  // its 0.912 plinth draws 0.9347) and a radius measures one bevel larger.
  // Feeding measurements straight back in would fatten the whole crate.
  //
  // Four of the model's groups are deliberately left out, because they are
  // *inside* the shell and never reach daylight: twelve edge rails and eight
  // corner nodes that run along the body's corner-radius centres and its top
  // and bottom faces, every one of them a clear 0.014 under the surface, and a
  // lid trim whose 0.728 square is swallowed by the body's own 0.8 top face. Which is why the model reads as a dark shell with lit panels rather
  // than a wireframe cage. Its floor halo — a lilac ring 1.41 across — is left
  // out too, for a different reason: it is a display-stand element that would
  // overhang the belt and, worse, fly along with the crate on the drop. The
  // additive `crate_glow` shell this scene animates does that job instead.
  const cubeGeo = new BoxGeometry(CUBE, CUBE, CUBE);
  const cubeEdgeGeo = new EdgesGeometry(cubeGeo);
  const crateBodyGeo = roundedBox(CRATE_BODY, CRATE_BODY, CRATE_BODY, 0.12, CRATE_BEVEL);
  const cratePlinthGeo = roundedBox(0.912, 0.034, 0.912, 0.24, 0.014);
  const cratePlinthTrimGeo = roundedBox(0.952, 0.016, 0.952, 0.256, 0.014);
  const crateLidGeo = roundedBox(0.672, 0.016, 0.672, 0.104, 0.014);
  const cratePanelGeo = roundedBox(0.64, 0.02, 0.64, 0.092, 0.006);
  const crateFrameHGeo = roundedBox(0.688, 0.012, 0.016, 0.007, 0.014);
  const crateFrameVGeo = roundedBox(0.016, 0.012, 0.688, 0.007, 0.014);
  const crateGlowGeo = roundedBox(0.86, 0.868, 0.86, 0.19, 0.02);

  // One pictogram geometry per type, shared by every crate carrying it: the
  // artwork is identical between instances and only the materials differ, so
  // nothing here is built or freed as the belt runs. Extruded to the model's
  // own 0.03 of total depth — 0.023 plus a 0.0035 bevel at each end, which is
  // what widens the 0.412 shapes in `hero-crate-icons` to the model's 0.419 —
  // and re-based so the glyph's back face sits at z = 0.
  const iconGeo = new Map<FileType['kind'], BufferGeometry>();
  for (const { kind } of TYPES) {
    const geometry = new ExtrudeGeometry(crateIconShapes(kind), {
      depth: 0.023,
      bevelEnabled: true,
      bevelThickness: 0.0035,
      bevelSize: 0.0035,
      bevelSegments: 2,
    });
    geometry.computeBoundingBox();
    geometry.translate(0, 0, -geometry.boundingBox!.min.z);
    iconGeo.set(kind, geometry);
  }

  /**
   * A `roundedBox` plate stood up on a crate's side: its front face at `z`,
   * its thickness running back from there, its centre at `x`, `y`. The model's
   * panels and frames are plates extruded the same way and rotated onto the
   * sides, so this is the one transform that ports them.
   */
  const cratePlate = (
    name: string,
    geometry: BufferGeometry,
    material: Material,
    x: number,
    y: number,
    z: number,
  ): Mesh => {
    const plate = partMesh(name, geometry, material, x, y, z);
    plate.rotation.x = -Math.PI / 2;
    return plate;
  };

  // The model's own pairing, remapped onto this scene's accents: its jade
  // audio face onto the green, its lilac doc face onto the violet trace, its
  // periwinkle image face onto the cyan. Video is lilac in the model too, and
  // four types have to be told apart at a glance, so it takes the magenta.
  // `PRIMARY` is deliberately not in here — it is a pale blue a shade off
  // `bone`, and a frame in it would swallow the pictogram inside it.
  const typeColors = [CYAN, MAGENTA, GREEN, TRACE_B] as const;
  /** Resting emissive on a crate's lit trim, and how far above it the trim
   * flashes as the crate compacts out of the blueprint and lands. */
  const ACCENT_EMISSIVE = 0.55 * TRIM_EM;
  const ACCENT_FLASH = 0.7 * TRIM_EM;
  const makeCrateVisual = (typeIndex: number): CrateVisual => {
    const type = TYPES[typeIndex];
    const color = typeColors[typeIndex];
    const accent = new MeshStandardMaterial({
      color,
      emissive: color,
      emissiveIntensity: ACCENT_EMISSIVE,
      roughness: 0.3,
      metalness: 0,
    });
    // A shell wrapping the whole crate (0.86 around a 0.8 body), so it is a
    // bloom and not the floor mark its name suggests: converted up for a
    // white stage it wraps every crate in a saturated haze and the shells
    // come out pastel instead of white.
    const glow = stageGlow(color, ba(0.08));
    const g = new Group();
    g.name = `crate_${type.kind}`;
    g.scale.setScalar(CRATE_SCALE);
    // Model coordinates from here down: the shell holds the crate's own
    // mid-height on the group's origin, which is where the blueprint cube's
    // centre was, and the group's scale does the rest.
    const shell = new Group();
    shell.position.y = -CRATE_MID;
    g.add(shell);

    shell.add(
      partMesh('crate_glow', crateGlowGeo, glow, 0, -0.006, 0),
      partMesh('crate_plinth', cratePlinthGeo, graphite, 0, 0, 0),
      partMesh('crate_plinth_trim', cratePlinthTrimGeo, accent, 0, 0.026, 0),
      partMesh('crate_body', crateBodyGeo, carbon, 0, 0.046, 0),
      partMesh('crate_lid', crateLidGeo, graphite, 0, 0.838, 0),
    );

    for (let side = 0; side < 4; side++) {
      const face = new Group();
      face.name = `crate_face_${type.kind}_${side}`;
      face.rotation.y = side * (Math.PI / 2);
      // The panel sits 0.008 inside the body's face; the frame rails stand
      // 0.002 proud of it and overhang the panel on all four sides, so their
      // ends float clear of the body where its corners curve away — the lit
      // rectangle the model reads by.
      face.add(
        cratePlate('crate_panel', cratePanelGeo, graphite, 0, 0.446, 0.42),
        cratePlate('crate_frame', crateFrameHGeo, accent, 0, 0.782, 0.43),
        cratePlate('crate_frame', crateFrameHGeo, accent, 0, 0.11, 0.43),
        cratePlate('crate_frame', crateFrameVGeo, accent, -0.336, 0.446, 0.43),
        cratePlate('crate_frame', crateFrameVGeo, accent, 0.336, 0.446, 0.43),
      );
      face.add(partMesh(`crate_icon_${type.kind}`, iconGeo.get(type.kind)!, bone, 0, 0.446, 0.42));
      shell.add(face);
    }

    // Sunk into the lid until only the model's own 0.007 of relief is left: a
    // pictogram printed on the lid rather than a block sitting on it. Rotating
    // about -X (not +X) sends the glyph's own up-direction away from the
    // camera, so it reads the right way up on a lid seen from above.
    const lidIcon = partMesh(`crate_icon_${type.kind}`, iconGeo.get(type.kind)!, bone, 0, 0.831, 0);
    lidIcon.rotation.x = -Math.PI / 2;
    shell.add(lidIcon);
    return { g, accent, glow };
  };

  const live: Cube[] = [];
  const retired: Cube[] = [];
  const makeCube = (typeIndex: number): Cube => {
    const crate = makeCrateVisual(typeIndex);
    world.add(crate.g);
    return { ...crate, slot: 0, x: START_X };
  };
  // Once a second cube has left the frame the older one is destroyed — nothing
  // invisible stays in memory.
  const freeCube = (s: Cube) => {
    world.remove(s.g);
    s.accent.dispose();
    s.glow.dispose();
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
  const bpFloor = new Mesh(new PlaneGeometry(CUBE * 1.7, CUBE * 1.7), stageGlow(TRACE_B, ba(0.07)));
  bpFloor.rotation.x = -Math.PI / 2;
  bpFloor.position.y = -CUBE / 2 - 0.02;
  blueprint.add(bpFloor);
  const bpMats = [bpEdges.material, bpGrid.material, bracketMat, bpFloor.material];
  const bpBase = [0.68, 0.2, 0.85, 0.08];
  const setBlueprint = (k: number) => bpMats.forEach((m, i) => void (m.opacity = bpBase[i] * k));

  // Particles that make up the cube (opaque, only faintly lit).
  /** A part in flight, and the same material once it has landed in the
   * blueprint: `PART_EMISSIVE` is its resting burn and `PART_FLASH` the
   * overshoot it compacts through. Both are animated below, so they are
   * constants rather than literals at the write sites. */
  const PART_EMISSIVE = 0.22 * EM;
  const PART_FLASH = 0.7 * EM;
  /** The landing crate's bloom shell: the peak it flashes to as it lands, the
   * fade off that peak, and where it rests. */
  const DROP_PEAK = ba(0.3);
  const DROP_FADE = ba(0.18);
  const DROP_REST = ba(0.08);
  const matPart = new MeshStandardMaterial({ color: P.part, emissive: CYAN, emissiveIntensity: PART_EMISSIVE, roughness: 0.5, metalness: mt(0.25) });
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
  // `carbon`/`graphite`/`bone` are declared above (the conveyor and the crates
  // use them too); the two accent materials this design uses are remapped onto
  // the scene's own palette instead — periwinkle -> primary, lilac -> the
  // existing violet trace — so the pylon recolors with everything else on a
  // theme flip.
  /** The emitter bead's glow, flashed each time a part launches. */
  const EMITTER_REST = ba(0.2);
  const EMITTER_FLASH = ba(0.45);
  const tracePrimary = new MeshStandardMaterial({ color: TRACE_A, emissive: TRACE_A, emissiveIntensity: 1.1 * TRIM_EM, roughness: 0.25, metalness: mt(0.3) });
  const traceMag = new MeshStandardMaterial({ color: MAGENTA, emissive: MAGENTA, emissiveIntensity: 1.25 * TRIM_EM, roughness: 0.25, metalness: mt(0.3) });
  const traceViolet = new MeshStandardMaterial({ color: TRACE_B, emissive: TRACE_B, emissiveIntensity: 1.2 * TRIM_EM, roughness: 0.25, metalness: mt(0.3) });
  const traceCyan = new MeshStandardMaterial({ color: CYAN, emissive: CYAN, emissiveIntensity: 1.05 * TRIM_EM, roughness: 0.25, metalness: mt(0.3) });
  /** The pulse the three traces breathe on, peaks only — the shape is below. */
  const TRACE_PULSE = { mag: 1.15 * TRIM_EM, violet: 1.1 * TRIM_EM, cyan: 1.0 * TRIM_EM };

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
  /** A named mesh at a position — the whole of what every part in here is. */
  function partMesh(
    name: string,
    geo: BufferGeometry,
    material: Material,
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
    g.add(partMesh('head_collar_glow', roundedBox(w * wide * 1.07, w * 0.06, w * wide * 1.07, w * 0.4), tracePrimary, 0, -w * 0.05, 0));
    g.add(partMesh('head_collar', roundedBox(w * wide, w * 0.14, w * wide, w * 0.36), graphite, 0, 0, 0));
  }
  /** "Пилюля" (pill) head: a capsule cap with a domed top and four light bars. */
  function headPill(g: Group, w: number): void {
    collar(g, w, 1.1);
    g.add(partMesh('head_pill', new CylinderGeometry(w * 0.42, w * 0.42, w * 1.08, 44), carbon, 0, w * 0.68, 0));
    g.add(
      partMesh('head_pill_dome', new SphereGeometry(w * 0.42, 44, 22, 0, Math.PI * 2, 0, Math.PI / 2), carbon, 0, w * 1.22, 0),
    );
    g.add(partMesh('head_pill_ring', arcRing(w * 0.435, w * 0.026, Math.PI * 2), traceViolet, 0, w * 0.36, 0));
    onFaces(g, 4, (face, s) => {
      face.add(
        partMesh(
          `head_pill_bar_${s}`,
          new CapsuleGeometry(w * 0.022, w * 0.46, 4, 14),
          s % 2 ? traceViolet : tracePrimary,
          0,
          w * 0.89,
          w * 0.432,
        ),
      );
      face.add(
        partMesh(`head_pill_louver_${s}`, roundedBox(w * 0.16, w * 0.045, w * 0.05, w * 0.02, w * 0.012), graphite, 0, w * 1.24, w * 0.36),
      );
    });
    g.add(partMesh('head_pill_dot', new CylinderGeometry(w * 0.1, w * 0.11, w * 0.05, 28), bone, 0, w * 1.55, 0));
  }

  const HEAD_W = 0.3;
  const buildPylon = (): { group: Group; emitter: Pylon['emitter']; emGlow: Pylon['emGlow']; tipY: number } => {
    const g = new Group();

    // base: plinth, four capsule feet, riser
    g.add(partMesh('plinth', roundedBox(1.18, 0.16, 1.18, 0.3), graphite, 0, 0, 0));
    g.add(partMesh('plinth_trim', roundedBox(1.26, 0.022, 1.26, 0.33), tracePrimary, 0, -0.016, 0));
    onFaces(g, 4, (face, s) => {
      face.add(partMesh(`foot_${s}`, new CapsuleGeometry(0.05, 0.1, 4, 14), carbon, 0.42, 0.04, 0.42));
    });
    g.add(partMesh('plinth_riser', roundedBox(0.72, 0.12, 0.72, 0.2), carbon, 0, 0.17, 0));

    // seven offset blocks, each spun a little further than the last, with
    // circuit strips and louvers alternating by parity
    const blocks = [[0.7, 0.3], [0.66, 0.26], [0.6, 0.3], [0.54, 0.24], [0.48, 0.28], [0.4, 0.22], [0.33, 0.26]];
    let y = 0.29;
    let spin = 0;
    blocks.forEach(([w, h], i) => {
      const block = new Group();
      block.rotation.y = spin;
      block.position.y = y;
      block.add(partMesh(`block_body_${i}`, roundedBox(w, h, w, w * 0.22), carbon));
      block.add(
        partMesh(`block_seam_${i}`, roundedBox(w * 1.03, 0.018, w * 1.03, w * 0.23), i % 2 ? tracePrimary : traceViolet, 0, h, 0),
      );
      onFaces(block, 4, (face, s) => {
        if ((s + i) % 2 === 0) {
          face.add(
            partMesh(`block_strip_${i}`, roundedBox(0.026, h * 0.62, 0.02, 0.012, 0.005), traceViolet, w * 0.3, h * 0.2, w * 0.5 + 0.003),
          );
        }
        face.add(
          partMesh(`block_louver_${i}`, roundedBox(w * 0.42, 0.016, 0.024, 0.008, 0.005), graphite, -w * 0.12, h * 0.62, w * 0.5 + 0.003),
        );
        face.add(
          partMesh(`block_louver2_${i}`, roundedBox(w * 0.42, 0.016, 0.024, 0.008, 0.005), graphite, -w * 0.12, h * 0.44, w * 0.5 + 0.003),
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
    const emGlow = new Mesh(new SphereGeometry(0.26, 20, 16), stageGlow(TRACE_A, ba(0.2)));
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
  // They stand on one line across the conveyor — same x, mirrored z — facing
  // each other over the blueprint volume they feed, which is the station where
  // a crate is assembled. Staggering them along the belt instead (the comp put
  // one 3.4 further down the line) read as two unrelated machines standing on
  // the same floor rather than one gantry the line runs through.
  // The plinth trim is ~1.26 across and the tread edge is |z| = 0.58, so this
  // leaves a clean physical gap on both sides.
  for (const [name, x, z] of [['pylon_far', START_X, -2.05], ['pylon_near', START_X, 2.05]] as const) {
    const p = buildPylon();
    p.group.name = name;
    p.group.position.set(x, -0.2, z);
    world.add(p.group);
    pylons.push({
      group: p.group,
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
      stageGlow(i % 3 === 0 ? CYAN : i % 3 === 1 ? TRACE_B : MAGENTA, ba(0.2)),
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
  // A part leaves the pylon as a ball and hardens into a cube on the way, so
  // its geometry is a sphere carrying the cube as a morph target (see
  // `hero-fly-morph.ts`). The sphere's radius is the cube's smallest half
  // extent, so the ball is the one that fits inside the cube it becomes
  // rather than the other way round — it grows a little as it squares off,
  // which is the direction that reads as compaction.
  const FLY_HALF: [number, number, number] = [SX * 0.425, SY * 0.425, SZ * 0.425];
  const flyGeo = new SphereGeometry(Math.min(...FLY_HALF), 24, 16);
  const flyCube = ballToCube(flyGeo, FLY_HALF);
  flyGeo.morphAttributes.position = [flyCube.position];
  flyGeo.morphAttributes.normal = [flyCube.normal];
  const matFly = new MeshStandardMaterial({ color: P.fly, emissive: CYAN, emissiveIntensity: 0.4 * EM, roughness: 0.42, metalness: mt(0.3) });
  // The bloom shell around it is a sphere at both ends of the morph: it is a
  // diffuse glow, so it has no shape of its own to keep, and morphing it in
  // step with its parent would buy nothing anyone could see.
  const flyGlowGeo = new SphereGeometry(Math.max(...FLY_HALF) * 1.7, 16, 12);
  const pool: Mesh[] = [];
  for (let i = 0; i < POOL; i++) {
    const m = new Mesh(flyGeo, matFly);
    m.visible = false;
    m.add(new Mesh(flyGlowGeo, stageGlow(CYAN, ba(0.09))));
    world.add(m);
    pool.push(m);
  }

  // ---------- the cube in transit from blueprint to belt
  const drop = new Group();
  drop.visible = false;
  world.add(drop);
  const dropCrates = TYPES.map((_, typeIndex) => makeCrateVisual(typeIndex));
  for (const crate of dropCrates) drop.add(crate.g);
  let activeDrop = dropCrates[0];
  const setDropType = (typeIndex: number) => {
    dropCrates.forEach((crate, index) => void (crate.g.visible = index === typeIndex));
    activeDrop = dropCrates[typeIndex];
  };
  setDropType(0);

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
  const state = { phase: 'fill' as Phase, clock: 0, time: 0, belt: 0, landed: 0, made: 0, type: 0, flights: [] as Flight[] };
  setBlueprint(1);

  const spawn = (idx: number) => {
    const mesh = pool[idx % POOL];
    const py = pylons[idx % pylons.length];
    const p = order[idx];
    const to = new Vector3(START_X + partX(p), BUILD_Y + partY(p), partZ(p));
    const ctrl = py.tip.clone().lerp(to, 0.5);
    ctrl.y += 0.7;
    mesh.visible = true;
    // Every launch starts the morph over: the pool hands the same mesh back
    // round-robin, and it landed as a cube.
    mesh.morphTargetInfluences![0] = 0;
    state.flights.push({ mesh, t: 0, curve: new QuadraticBezierCurve3(py.tip.clone(), ctrl, to), idx });
  };

  const placeCube = () => {
    const s = makeCube(state.type);
    live.push(s);
    s.g.position.set(START_X, BELT_Y + CUBE / 2 + 0.02, 0);
    state.type = nextType(state.type);
    setDropType(state.type);
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
      matPart.emissiveIntensity = PART_EMISSIVE + Math.sin(Math.PI * k) * PART_FLASH;
      setBlueprint(Math.max(0, 1 - e * 1.35));
      if (k >= 0.62 && !drop.visible) {
        drop.visible = true;
        drop.position.set(START_X, BUILD_Y, 0);
        drop.scale.setScalar(1);
        for (const m of parts) m.visible = false;
      }
      if (drop.visible) {
        const f = Math.min(1, (k - 0.62) / 0.38);
        activeDrop.accent.emissiveIntensity = ACCENT_EMISSIVE + ACCENT_FLASH * (1 - f);
        activeDrop.glow.opacity = DROP_PEAK - DROP_FADE * f;
      }
      if (k >= 1) {
        parts.forEach((m, i) => {
          const p = order[i];
          m.position.set(partX(p), partY(p), partZ(p));
          m.scale.setScalar(1);
        });
        matPart.emissiveIntensity = PART_EMISSIVE;
        setBlueprint(0);
        activeDrop.accent.emissiveIntensity = ACCENT_EMISSIVE;
        activeDrop.glow.opacity = DROP_REST;
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
      // The upper run carries the cubes toward -X; the return run travels in
      // the opposite direction like a real continuous tread loop.
      layTread(state.belt + SLOT * e);
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
        // The tread keeps the slot it just travelled; the strip is periodic in
        // `STRIP`, so wrapping the total there changes nothing on screen.
        state.belt = (state.belt + SLOT) % STRIP;
        layTread(state.belt);
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
      const f = Math.max(0, 1 - Math.max(0, r.group.position.x) / 30);
      const wave = (0.19 + 0.81 * f) * (0.76 + 0.36 * Math.sin(state.time * 2.1 - r.group.position.x * 0.55));
      r.accent.material.opacity = r.base * wave;
      r.halo.material.opacity = r.haloBase * wave;
    }

    for (let i = state.flights.length - 1; i >= 0; i--) {
      const f = state.flights[i];
      f.t += dt / TRAVEL;
      const k = Math.min(1, f.t);
      f.curve.getPoint(easeInOut(k), tmp);
      f.mesh.position.copy(tmp);
      f.mesh.scale.setScalar(1.1 - 0.1 * k);
      f.mesh.morphTargetInfluences![0] = morphAt(k);
      if (k >= 1) {
        parts[f.idx].visible = true;
        f.mesh.visible = false;
        f.mesh.scale.setScalar(1);
        state.flights.splice(i, 1);
      }
    }

    const pulse = 0.9 + 0.22 * Math.sin(state.time * 1.4);
    traceMag.emissiveIntensity = TRACE_PULSE.mag * pulse;
    traceViolet.emissiveIntensity = TRACE_PULSE.violet * (1.85 - pulse);
    traceCyan.emissiveIntensity = TRACE_PULSE.cyan * pulse;
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
      py.emGlow.material.opacity = EMITTER_REST + py.flash * EMITTER_FLASH;
      py.emitter.scale.setScalar(1 + py.flash * 0.4);
    }
  };

  // Both machines' corners in world space, for measuring where they sit in
  // the frame. They never move, so this is read once.
  const machineCorners: Vector3[] = [];
  for (const py of pylons) {
    const box = new Box3().setFromObject(py.group);
    for (let corner = 0; corner < 8; corner++) {
      machineCorners.push(new Vector3(
        corner & 1 ? box.max.x : box.min.x,
        corner & 2 ? box.max.y : box.min.y,
        corner & 4 ? box.max.z : box.min.z,
      ));
    }
  }
  // And the crate at the station, the moment it lands: the lowest thing in the
  // composition that has to stay in frame. The plinths below it may be cropped
  // by the band's edge - a machine running off the bottom reads as one that
  // carries on past the screen - but a crate cut in half by the closing fade
  // is the belt's whole point gone missing.
  const stationCrate = new Vector3(START_X, BELT_Y + CUBE / 2, 0);
  world.localToWorld(stationCrate);

  // Where the machines sit in the frame, in normalised device coordinates, for
  // the frame shift currently written into `shiftX`/`shiftY`. `solveY` and
  // `solveX` walk one of those to the value that puts a given reading on a
  // given line: both push the subject down and left as they rise, so every
  // reading below falls monotonically in the one its solver moves — which is
  // what makes the bisection sound.
  /**
   * Where the frame sits over the scene, in fractions of itself: the whole of
   * how the composition is placed.
   *
   * A shift of the frame, not a move of the camera. Sliding the camera
   * sideways or tilting it down moves the vantage point, and then the belt's
   * own angle across the frame - and how much its near end looms - changes
   * with the shape of the window, which reads as the machine being rebuilt at
   * every breakpoint rather than photographed from one place. Shifting the
   * frame is what an architectural lens does: the vanishing points stay put
   * against the scene, and only what the frame holds changes. The field of
   * view still adapts (see `place`), and that is a magnification - it scales
   * the picture without touching its perspective.
   */
  let shiftX = 0;
  let shiftY = 0;
  /** The frame the shift is measured in, written by `resize`. */
  let frameW = 960;
  let frameH = 520;
  const probe = new Vector3();
  const aimAt = () => {
    // Positive `shiftY` moves the subject *down* the frame, which is the
    // direction every reading below is solved in.
    camera.setViewOffset(frameW, frameH, shiftX * frameW, -shiftY * frameH, frameW, frameH);
  };
  /** The higher of the two crowns. */
  const tipNdc = () => {
    aimAt();
    let top = -Infinity;
    for (const py of pylons) top = Math.max(top, probe.copy(py.tipWorld).project(camera).y);
    return top;
  };
  /** The crate at the station. */
  const crateNdc = () => {
    aimAt();
    return probe.copy(stationCrate).project(camera).y;
  };
  /** How far the pair reaches to each side, and how high it stands. */
  const spanNdc = () => {
    aimAt();
    let left = Infinity;
    let right = -Infinity;
    let top = -Infinity;
    for (const p of machineCorners) {
      const v = probe.copy(p).project(camera);
      left = Math.min(left, v.x);
      right = Math.max(right, v.x);
      top = Math.max(top, v.y);
    }
    return { left, right, top };
  };
  /** 24 halvings over a range that covers a frame and a half either way,
   *  which lands the reading on its line to well under a pixel. */
  const HALVINGS = 24;
  const solveY = (read: () => number, value: number) => {
    let lo = -1.6;
    let hi = 1.6;
    for (let i = 0; i < HALVINGS; i++) {
      const mid = (lo + hi) / 2;
      shiftY = mid;
      if (read() > value) lo = mid;
      else hi = mid;
    }
    shiftY = (lo + hi) / 2;
  };
  const solveX = (read: () => number, value: number) => {
    let lo = -1.6;
    let hi = 1.6;
    for (let i = 0; i < HALVINGS; i++) {
      const mid = (lo + hi) / 2;
      shiftX = mid;
      if (read() > value) lo = mid;
      else hi = mid;
    }
    shiftX = (lo + hi) / 2;
  };
  /** How far down the frame the station's crate may sit before the pair is
   *  held back up, and how far down it has to reach before the crowns are
   *  allowed to hang off the line the page states rather than dropping to meet
   *  it. Both are read on that crate rather than on the plinths: it is what
   *  has to stay in frame, and what the closing fade must not swallow - the
   *  floor is where that fade starts, so the crate lands just above it and
   *  everything nearer the camera dissolves into it. */
  const CRATE_FLOOR = -0.56;
  const CRATE_REACH = -0.12;
  /** And how much of the half-frame stays clear of the other three edges. */
  const EDGE = 0.06;
  /** The gap the pair keeps from the copy column's edge, so the two never
   *  touch at any width. */
  const COPY_GAP = 0.05;
  /** And how far it may stand *inside* the copy's widest row on a stacked
   *  layout, where that row is the chips: they carry their own grounds, and
   *  letting the far machine reach a little behind their tail is worth a
   *  quarter of the composition's size. */
  const CHIP_LAP = -0.14;
  /** How far above the line it was given the pair may still stand, before the
   *  field widens to fit it under that line properly. */
  const TIP_SLACK = 0.06;


  const resize = () => {
    const w = host.clientWidth || 960;
    const h = host.clientHeight || 520;
    renderer.setSize(w, h, false);
    const aspect = w / Math.max(1, h);
    camera.aspect = aspect;

    const { h: field, target } = framingFor(aspect);
    // One vantage point per shape, from the framing, and it does not move
    // again: `place` composes by shifting the frame over it.
    frameW = w;
    frameH = h;
    camera.lookAt(target.x, target.y, target.z);
    camera.updateMatrixWorld();
    const want = alignTipsNdc?.() ?? null;
    const clear = copyEdgeNdc?.() ?? null;
    const under = copyBottomNdc?.() ?? null;

    /**
     * Stands the pair up for a given horizontal half-extent, in one of the two
     * places the copy leaves it, and reports how much of the half-frame it has
     * left over — negative when it does not fit there.
     *
     *   `aside` — the line to stand clear of, which is the copy's own widest
     *     row and where the
     *     composition wants to be whenever that leaves it room: the copy is
     *     never over it, and the belt still runs on underneath. The crowns
     *     take the line the hero asked for if it asked for one (its heading's,
     *     on the column layout) and otherwise stand where the crate rules
     *     below put them.
     *   `aside === null` — under the copy's last row, the pair centred in the
     *     band. What
     *     a phone gets, its copy being the full width: there is nothing to
     *     stand beside.
     *
     * Then the crate rules, either way. The line the page states is where the
     * crowns go while the station's crate lands somewhere useful from it: a
     * pair the field had to ease off is too short for that, and hanging it off
     * a heading line near the top of the band would leave the whole bottom
     * half empty, so it drops until the crate reaches `CRATE_REACH`. And the
     * other way at the other end: a line low in the band would carry that
     * crate off the bottom edge, so `CRATE_FLOOR` stops it. Which is why the
     * page states lines and edges and nothing else — how tall a machine draws
     * at this shape is the scene's own business.
     */
    const place = (half: number, aside: number | null): number => {
      camera.fov = (2 * Math.atan(half / Math.min(aspect, ZOOM_STOP))) / DEG;
      shiftX = 0;
      shiftY = 0;
      const line = aside === null ? under : want;
      // Three times around, with every rule inside the loop: the vertical and
      // the horizontal are coupled through the camera's tilt - pitching it up
      // or down skews the horizontal projection - so a pass that ends on a
      // vertical solve leaves the pair off the place it was put. Which is
      // exactly what it did: on a stacked layout there is no crown line to
      // aim at, so the pass ended on the crate rules below, and they slid the
      // pair as much as four fifths of the frame back off the copy's edge.
      // Each pass shrinks that, so three of them land it.
      for (let pass = 0; pass < 3; pass++) {
        if (aside !== null) solveX(() => spanNdc().left, aside);
        else {
          // Both edges fall as the camera slides right, so their sum does too
          // - which is the reading a bisection can walk to zero.
          solveX(() => {
            const span = spanNdc();
            return span.left + span.right;
          }, 0);
        }
        if (line !== null && Number.isFinite(line)) solveY(tipNdc, line);
        if (crateNdc() > CRATE_REACH) solveY(crateNdc, CRATE_REACH);
        if (crateNdc() < CRATE_FLOOR) solveY(crateNdc, CRATE_FLOOR);
      }
      const span = spanNdc();
      // Room to the right, room above, and - when the crowns were asked for a
      // line - how far the crate rules had to push them back up off it. That
      // last one is what fits the pair to the strip the page left it: a
      // machine too tall for the space under a phone's copy reads as one
      // standing in the sentences, so the field widens until it stands under
      // them instead.
      const raised = line !== null && Number.isFinite(line) ? line - tipNdc() + TIP_SLACK : 1;
      return Math.min(1 - EDGE - span.right, 1 - EDGE - span.top, raised);
    };

    /**
     * The tightest field a placement fits at, which is the biggest the pair
     * can be drawn there. The zoom the composition was drawn at when that
     * fits; otherwise a bisection out to a fourth of it, which is what a 320px
     * band takes. Room grows with the field, so the bisection is sound.
     */
    const fit = (aside: number | null): number => {
      if (place(field, aside) >= 0) return field;
      let lo = field;
      let hi = field * 4;
      for (let i = 0; i < 12; i++) {
        const mid = (lo + hi) / 2;
        if (place(mid, aside) < 0) lo = mid;
        else hi = mid;
      }
      return hi;
    };

    // Which of the two places to use. Beside the copy from a tablet up: the
    // copy's rows stop two thirds across there and its measure is capped
    // (`md:max-w`) so they keep stopping, which is what leaves the pair a
    // corner to stand in. Narrower than that the rows run the full width, the
    // pair would be left a third of it, and standing under the copy draws it
    // two and a half times bigger.
    //
    // A width, not a measurement of the two: the placements are far enough
    // apart that comparing their sizes flips somewhere in the middle of the
    // tablet range, and a composition that jumps as the window crosses 954px
    // is worse than either of them.
    const aside = clear !== null && w >= BESIDE_FROM
      ? clear + (want === null ? CHIP_LAP : COPY_GAP)
      : null;
    place(fit(aside), aside);
    // What the page needs to fade against: the belt past the machines is empty
    // line, and a hero that runs it out to the window's edge reads as a
    // picture with nothing in half of it. So the far fade starts where the
    // machines stop, and the page is told where that is - as a share of the
    // band, which is what a CSS width wants.
    const tail = Math.max(0, ((1 - spanNdc().right) / 2) * 100);
    host.style.setProperty('--hero-tail', `${tail.toFixed(2)}%`);
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
