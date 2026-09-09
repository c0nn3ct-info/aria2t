// Which stage the hero's WebGL scene is standing on, and every colour that
// depends on the answer.
//
// This lives apart from `hero-scene-three.ts` for two reasons. The scene file
// is large and this is the one part of it with no three.js in it beyond
// `Color`'s HSL maths, so both halves get smaller and clearer. And the
// per-theme values are exactly what wants direct unit tests: the branches here
// are cheap to cover on their own, where covering them through a booted scene
// means one full WebGL boot per branch.
//
// Two sources feed a palette. The *accents* come from the same
// `[data-accent='blue']` CSS tokens the surrounding bands use
// (`globals.css`), so the WebGL ground and the page's own background can never
// drift the way two hand-picked hex constants could. The *machine* colours
// come from the authoring models: `.claude/3d/dark` and `.claude/3d/light`
// each carry the same three GLBs, same geometry, different materials, and the
// six values below are those materials' `baseColorFactor`/`emissiveFactor`
// converted out of linear light. So the machines are not tinted versions of
// one dark model — light is its own authored material set, in which the
// shells invert to near-white, the pictograms invert to near-black, and the
// accent emissives drop by about 4.4x.
import { Color } from 'three';

/** A structural material as the authoring GLBs describe one. The two
 * transparency fields are not in the models: only the belt's chassis sets
 * them, and only on the light stage. */
export interface Neutral {
  color: number;
  roughness: number;
  metalness: number;
  emissive?: number;
  emissiveIntensity?: number;
  transparent?: boolean;
  opacity?: number;
}

/** The stage lighting, which the models say nothing about: neither GLB
 * carries a `KHR_lights_punctual` light or a camera, so the rig is the
 * scene's own and has to be re-tuned per stage. Near-white shells at
 * metalness 0.06 clip to flat white under the dark stage's rim intensities. */
export interface Rig {
  /** Hemisphere sky, and the bounce off the floor under it. */
  sky: number;
  ground: number;
  hemi: number;
  key: number;
  /**
   * The two rim lights. Their *colours* are per-stage and not merely their
   * intensities: on the dark stage they are the scene's magenta and cyan, and
   * spraying saturated coloured light across the machines is most of what
   * makes it read as neon. Over near-white shells the same two lights tint
   * every face pink or mint — the shells have nothing dark to hide it in — and
   * a crate that should be white comes out pastel. So the light stage rims in
   * near-neutral and lets the trim carry all the colour there is.
   */
  rimAColor: number;
  rimA: number;
  rimBColor: number;
  rimB: number;
}

export interface Palette {
  /**
   * The page's own background, and the only token this scene still reads.
   * The fog and the clear colour have to agree with the band around the
   * canvas or the machines sit in a visible rectangle.
   *
   * Everything else used to follow `--primary`/`--tertiary` as well, on the
   * theory that the scene should recolour with the site. It never applied to
   * the dark stage, which takes its whole palette from the constants below —
   * so it only ever governed the light one, where it was actively harmful:
   * the light `--primary` is a muted mid-blue, and a muted mid-blue is what a
   * white machine has no way to wear. The site sets one accent for the whole
   * page anyway, so nothing was varying.
   */
  bg: number;
  cyan: number;
  magenta: number;
  /** The one crate type colour with no CSS token *and* no theme-flipping
   * sibling: the four are cyan/magenta/violet plus this green. */
  green: number;
  /**
   * The two accents the machines' own trim is painted in — the models call
   * them periwinkle and lilac.
   *
   * These are deliberately not `primary`/`tertiary`. On the dark stage a
   * trace reads because it is self-lit against near-black, so the page's own
   * accent tokens work fine and keeping them means the machines recolour with
   * the site. On a white stage nothing is brighter than the ground: contrast
   * has to come from saturation instead, and the page's light `--primary` is a
   * muted mid-blue that goes straight to grey mush behind a dimmed emissive.
   * So the light stage takes the authoring models' own vivid trim colours,
   * which is what makes the refs' blue read as ink rather than haze. On the
   * dark stage they are still the values that stage always used, unchanged.
   */
  traceA: number;
  traceB: number;
  carbon: Neutral;
  graphite: Neutral;
  bone: Neutral;
  /**
   * The conveyor's chassis: the closed loop the slats ride on.
   *
   * Both model sets paint it the same near-black, and on the dark stage that
   * is what it stays — the belt reads as a belt because its frame is darker
   * than the slats crossing it. On the light stage the same value is a black
   * wedge across a white page, heavy enough to become the loudest thing in
   * the band, so there it goes to pale glass instead: the slat pitch and the
   * shadow under the loop carry the read, and the return run showing faintly
   * through the housing is the closed tread loop the model details.
   */
  belt: Neutral;
  /** Bodies of the small parts that fly pylon -> blueprint. Scene-only, so
   * the models do not name them; they follow the shells' inversion far enough
   * that a part still reads against its stage. */
  part: number;
  fly: number;
  rig: Rig;
  /** Additive on the dark stage, where a glow adds to near-black and reads as
   * light. Adding to near-white is a no-op, so the light stage draws the same
   * meshes normally and a trace reads as saturated ink instead. The authored
   * alphas are shared: at the low end the two blends land at comparable
   * subtlety, and at the high end (the lit slats' 0.55) normal blending reads
   * stronger, which is what a daylight trace wants. */
  additive: boolean;
  /**
   * What the scene's own materials multiply their authored `metalness` by.
   *
   * There is no environment map in this scene, and a metal with nothing to
   * reflect renders dark: metalness takes the diffuse away and hands it to a
   * specular term that has no light to gather. On the dark stage that costs
   * nothing, because dark is where everything already is. On a white one the
   * parts and the traces come out muddy grey in the middle of clean white
   * machinery — which is exactly why the light models drop their own
   * metalness so far (carbon 0.3 to 0.06, graphite 0.35 to 0.08). Those
   * values arrive with the models; this is the same correction for the
   * materials the scene mixes itself.
   */
  metalScale: number;
  /**
   * What an authored *spill* alpha has to be multiplied by on this stage.
   *
   * A spill is the soft, near-coplanar widening of a flat trim line — the
   * thing that makes a painted strip read as a lit one. It is the one part of
   * the glow layer a white stage genuinely needs: emissive alone cannot make
   * a strip look self-lit against a ground that is already brighter than it,
   * but a soft wash of the strip's own hue beside it reads as light spilling
   * onto the shell. It sits between the other two — well under the additive
   * value, which smears into a wide band over white, and well over the bloom
   * scale, which would erase it.
   */
  spillAlpha: number;
  /**
   * How hard *trim* burns — the strips, rings and seams the machines are
   * painted with, as against `emissive`, which is for the parts.
   *
   * Kept separate because the two want opposite things on a white stage. A
   * part is a small white shell and wants its emissive out of the way. A
   * strip is the only colour on the machine and has to look lit, so it keeps
   * much more of its burn: enough to sit clearly above its shell, short of
   * the value that would wash a saturated hue back out to pale.
   */
  trimEmissive: number;
  /**
   * What an authored *bloom* alpha has to be multiplied by on this stage.
   *
   * A bloom is the oversized transparent shell around something bright — the
   * sphere around an emitter bead, the box around a part in flight, the spill
   * beside a light-line. It is light that escaped, so it only means anything
   * where there is dark for it to escape into. Blended over white it is not a
   * glow at all, just a pale card floating in front of the object, which is
   * what turned the pylon heads into smudges. So the light stage takes it
   * almost all the way out, while flat trim — the marks the machine actually
   * has painted on it — goes the other way, up.
   */
  bloomAlpha: number;
  /**
   * What an authored glow alpha has to be multiplied by on this stage.
   *
   * The alphas below are written for additive blending, where 0.08 of a
   * bright colour over near-black is plainly visible because it is added to
   * nothing. Blended normally over white the same 0.08 is 92% background —
   * invisible. Perceived presence needs several times the alpha, so this is
   * the one number that converts between the two, clamped at fully opaque.
   */
  glowAlpha: number;
  /** The backdrop bloom's two gradient stops, `rgba()` strings for a 2D
   * canvas. They are the accents rather than the tokens because this is the
   * only place the two rim colours appear together. */
  haloStops: [inner: string, mid: string];
}

const DARK: Palette = {
  bg: 0x0b0b0f,
  cyan: 0x7dcfee,
  magenta: 0xbb9af7,
  green: 0x56c990,
  traceA: 0xa8c7fa,
  traceB: 0xc6b2ff,
  carbon: { color: 0x151a26, roughness: 0.55, metalness: 0.3 },
  graphite: { color: 0x252c3e, roughness: 0.42, metalness: 0.35 },
  bone: { color: 0xd8e0f2, roughness: 0.34, metalness: 0.14, emissive: 0x9db4e6, emissiveIntensity: 0.2 },
  // The chassis the belt body already wore here, so naming it changed nothing
  // on this stage.
  belt: { color: 0x151a26, roughness: 0.55, metalness: 0.3 },
  part: 0x2b3f5e,
  fly: 0x3a5378,
  // The two rim colours are this palette's own magenta and cyan, spelled out
  // rather than referenced so the stage is one flat table.
  rig: { sky: 0xa8c7fa, ground: 0x05050a, hemi: 0.5, key: 1.05, rimAColor: 0xbb9af7, rimA: 22, rimBColor: 0x7dcfee, rimB: 16 },
  additive: true,
  metalScale: 1,
  spillAlpha: 1,
  trimEmissive: 1,
  bloomAlpha: 1,
  glowAlpha: 1,
  haloStops: ['rgba(125,207,238,0.4)', 'rgba(187,154,247,0.14)'],
};

/** The light models' own material set, read out of `.claude/3d/light`. */
const LIGHT_MACHINES = {
  // The four crate types, re-picked as ink rather than light: saturated at
  // roughly 40% lightness, which is where a colour still separates from white
  // once its emissive has been dimmed out of the way.
  green: 0x15803d,
  traceA: 0x3a5fd6,
  traceB: 0x6f42d8,
  carbon: { color: 0xe6e9f2, roughness: 0.5, metalness: 0.06 },
  graphite: { color: 0xc7cee0, roughness: 0.44, metalness: 0.08 },
  bone: { color: 0x2b3446, roughness: 0.36, metalness: 0.12, emissive: 0x1a2233, emissiveIntensity: 0.05 },
  belt: { color: 0xdfe6f2, roughness: 0.34, metalness: 0.02, transparent: true, opacity: 0.3 },
  // Just off the shell white, and no darker: there are dozens of them, they
  // are small, and on the dark stage their whole identity is the cyan glow a
  // white ground takes away — so here they are near the white of the crate
  // they compact into and let shading alone separate them from the ground.
  part: 0xdde4ef,
  fly: 0xd6dfec,
  // A white studio, and the fill is the whole of it. The rim lights sit down
  // near the belt, so anything the key does not reach — the parts up in the
  // blueprint at `BUILD_Y`, and every face pointing away from it — has only
  // the hemisphere to light it. At the dark stage's 0.5 that leaves them at
  // about half their own colour, which on white reads as grey grime rather
  // than as shadow. Lifting the fill is what makes the machinery sit *in*
  // daylight instead of under a spotlight in an empty room; the key stays
  // where it is so the forms keep their modelling.
  rig: { sky: 0xe8eeff, ground: 0xdfe4f0, hemi: 1.5, key: 0.85, rimAColor: 0xf4f1f8, rimA: 5, rimBColor: 0xeef4fa, rimB: 4 },
  additive: false,
  metalScale: 0.18,
  spillAlpha: 0.5,
  trimEmissive: 0.5,
  bloomAlpha: 0.25,
  glowAlpha: 3,
  // Barely there. The dark stage's backdrop bloom is what gives the machines
  // somewhere to stand; blended normally over white the same plane is a milky
  // wash across the whole band, and the refs' ground is clean.
  haloStops: ['rgba(31,143,174,0.05)', 'rgba(122,79,201,0.025)'],
} satisfies Omit<Palette, 'bg' | 'cyan' | 'magenta'>;

/**
 * Parses a `"H S% L%"` custom property (the format every token in
 * `globals.css` is written in) into the fractions `Color.setHSL` wants.
 * Returns null when the property is unset or unparseable — the jsdom test
 * host has no stylesheet loaded by default, and `isDark`/`scenePalette`
 * both fall back to the dark palette in that case, which is what every
 * pre-existing scene test already boots against.
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

export function scenePalette(): Palette {
  if (isDark()) return DARK;
  return {
    bg: hslHex(readHSL('--background')!),
    // No CSS token: these two are scene-only accents, saturated rather than
    // merely darkened so they hold their own against a near-white ground.
    cyan: 0x0e7490,
    magenta: 0xa21caf,
    ...LIGHT_MACHINES,
  };
}
