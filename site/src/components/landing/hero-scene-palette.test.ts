// The two stages, tested directly rather than through a booted scene: these
// are the branches, and a WebGL boot per branch buys nothing over reading the
// values the boot would have consumed.
import { afterEach, describe, expect, it } from 'vitest';
import { Color } from 'three';
import { isDark, scenePalette } from './hero-scene-palette';

/** Sets the tokens `globals.css` would have set for the light stage. */
function lightTokens({ accents = true }: { accents?: boolean } = {}) {
  document.documentElement.style.setProperty('--background', '217 30% 99%');
  if (accents) document.documentElement.style.setProperty('--primary', '217 60% 42%');
}

afterEach(() => {
  for (const p of ['--background', '--primary']) document.documentElement.style.removeProperty(p);
});

describe('isDark', () => {
  it('defaults to dark when the background token is unset (this suite loads no stylesheet)', () => {
    expect(isDark()).toBe(true);
  });

  it('defaults to dark when the token is set but unparseable', () => {
    document.documentElement.style.setProperty('--background', 'rebeccapurple');
    expect(isDark()).toBe(true);
  });

  it('reads light off a bright --background token', () => {
    lightTokens();
    expect(isDark()).toBe(false);
  });

  it('reads dark off a dim --background token', () => {
    document.documentElement.style.setProperty('--background', '240 15% 5%');
    expect(isDark()).toBe(true);
  });
});

describe('the dark stage', () => {
  it('is the comp’s own fixed set, and glows additively', () => {
    const p = scenePalette();
    expect(p.bg).toBe(0x0b0b0f);
    expect(p.additive).toBe(true);
    expect(p.carbon).toEqual({ color: 0x151a26, roughness: 0.55, metalness: 0.3 });
    expect(p.bone.color).toBe(0xd8e0f2);
    expect(p.rig).toEqual({
      sky: 0xa8c7fa,
      ground: 0x05050a,
      hemi: 0.5,
      key: 1.05,
      // The rims are the stage's own accents, which is most of what makes it
      // read as neon.
      rimAColor: p.magenta,
      rimA: 22,
      rimBColor: p.cyan,
      rimB: 16,
    });
  });

  it('gives the chassis the carbon the belt body wore before `belt` existed, so its arrival changed nothing here', () => {
    const p = scenePalette();
    expect(p.belt).toEqual(p.carbon);
    // Opaque: the glass housing is the light stage's answer, not this one's.
    expect(p.belt.transparent).toBeUndefined();
  });

  it('paints the machines’ trim in the accents this stage always used', () => {
    const p = scenePalette();
    expect(p.traceA).toBe(0xa8c7fa);
    expect(p.traceB).toBe(0xc6b2ff);
  });

  it('needs no conversion, since every alpha and every burn is authored for it', () => {
    const p = scenePalette();
    expect(p.glowAlpha).toBe(1);
    expect(p.bloomAlpha).toBe(1);
    expect(p.spillAlpha).toBe(1);
    expect(p.trimEmissive).toBe(1);
    expect(p.metalScale).toBe(1);
  });
});

describe('the light stage', () => {
  it('inverts the machines: shells near-white, pictograms near-black', () => {
    lightTokens();
    const p = scenePalette();
    expect(p.carbon).toEqual({ color: 0xe6e9f2, roughness: 0.5, metalness: 0.06 });
    expect(p.graphite.color).toBe(0xc7cee0);
    expect(p.bone).toEqual({ color: 0x2b3446, roughness: 0.36, metalness: 0.12, emissive: 0x1a2233, emissiveIntensity: 0.05 });
  });

  it('turns the chassis to pale glass, where the dark stage’s near-black is a wedge across the page', () => {
    lightTokens();
    const p = scenePalette();
    expect(p.belt.transparent).toBe(true);
    expect(p.belt.opacity).toBeLessThan(0.5);
    expect(p.belt.color).not.toBe(0x151a26);
    // The slats ride on it, and they are the part that inverts.
    expect(p.graphite.color).toBe(0xc7cee0);
  });

  it('takes the metal out, since a metal with no environment map only renders dark', () => {
    lightTokens();
    expect(scenePalette().metalScale).toBeLessThan(0.25);
  });

  it('keeps more of the trim’s burn than the parts’, since the strips are the only colour there is', () => {
    lightTokens();
    const p = scenePalette();
    expect(p.trimEmissive).toBeGreaterThan(0.22);
    expect(p.trimEmissive).toBeLessThan(1);
  });

  it('lands its spill between the trim and the bloom — the one glow a white stage needs', () => {
    lightTokens();
    const p = scenePalette();
    expect(p.spillAlpha).toBeLessThan(1);
    expect(p.spillAlpha).toBeGreaterThan(p.bloomAlpha);
  });

  it('takes its trim from the models rather than the page’s muted accents, which is what keeps it ink', () => {
    lightTokens();
    const p = scenePalette();
    expect(p.traceA).toBe(0x3a5fd6);
    expect(p.traceB).toBe(0x6f42d8);
    // The page's own light `--primary` is a muted mid-blue, which is what
    // this deliberately is not: saturation is the only contrast a trace has
    // left on a white ground.
    expect(p.traceA).not.toBe(new Color().setHSL(217 / 360, 0.6, 0.42).getHex());
  });

  it('scales its glow alphas up, since normal blending over white starts from nothing', () => {
    lightTokens();
    expect(scenePalette().glowAlpha).toBeGreaterThan(1);
  });

  it('keeps its backdrop bloom nearly invisible, where the dark stage leans on it', () => {
    lightTokens();
    const lit = scenePalette().haloStops;
    document.documentElement.style.removeProperty('--background');
    const dark = scenePalette().haloStops;
    const alpha = (s: string) => parseFloat(s.slice(s.lastIndexOf(',') + 1));
    expect(alpha(lit[0])).toBeLessThan(alpha(dark[0]) / 4);
  });

  it('draws its glow normally, since adding to near-white is a no-op', () => {
    lightTokens();
    expect(scenePalette().additive).toBe(false);
  });

  it('trades the dark stage’s rims for fill: near-white shells clip under a spotlight and grey out without one', () => {
    lightTokens();
    const { rig } = scenePalette();
    const dark = (document.documentElement.style.removeProperty('--background'), scenePalette().rig);
    expect(rig.rimA).toBeLessThan(dark.rimA);
    expect(rig.rimB).toBeLessThan(dark.rimB);
    expect(rig.key).toBeLessThan(dark.key);
    expect(rig.ground).toBeGreaterThan(dark.ground);
    // The other direction, and the one that stops anything the key misses
    // from reading as grime: this is a studio, and the fill is the whole of it.
    expect(rig.hemi).toBeGreaterThan(dark.hemi);
  });

  it('takes its background from the page’s own token, which is all it reads', () => {
    lightTokens();
    expect(scenePalette().bg).toBe(new Color().setHSL(217 / 360, 0.3, 0.99).getHex());
  });

  it('rims in near-neutral, so a white shell does not come out pastel', () => {
    lightTokens();
    const p = scenePalette();
    for (const c of [p.rig.rimAColor, p.rig.rimBColor, p.rig.sky]) {
      // Read as sRGB bytes, which is how the value is written. `Color` would
      // hold these linearised, where even a barely-cool near-white spreads
      // far enough between channels to make the bound meaningless.
      const ch = [(c >> 16) & 0xff, (c >> 8) & 0xff, c & 0xff];
      expect(Math.max(...ch) - Math.min(...ch)).toBeLessThan(32);
      expect(Math.min(...ch)).toBeGreaterThan(0xd0);
    }
    expect(p.rig.rimAColor).not.toBe(p.magenta);
  });

  it('takes its bloom almost all the way out, since escaped light needs dark to escape into', () => {
    lightTokens();
    expect(scenePalette().bloomAlpha).toBeLessThan(0.5);
  });

  it('darkens the scene-only accents, which have no token to read', () => {
    lightTokens();
    const p = scenePalette();
    expect(p.cyan).toBe(0x0e7490);
    expect(p.magenta).toBe(0xa21caf);
    expect(p.haloStops[0]).toContain('31,143,174');
  });

  it('needs no accent token at all — the machines are the models’, not the page’s', () => {
    lightTokens({ accents: false });
    const p = scenePalette();
    expect(p.carbon.color).toBe(0xe6e9f2);
    expect(p.traceA).toBe(0x3a5fd6);
  });
});
