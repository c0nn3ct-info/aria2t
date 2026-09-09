// The ball-to-cube morph a part flies through. Pure geometry, so it is tested
// on its own rather than by reading vertices back off a booted scene.
import { describe, expect, it } from 'vitest';
import { BufferGeometry, Float32BufferAttribute, SphereGeometry } from 'three';
import { ballToCube, morphAt } from './hero-fly-morph';

const HALF: [number, number, number] = [0.08, 0.11, 0.08];

describe('ballToCube', () => {
  const sphere = new SphereGeometry(0.08, 16, 12);
  const { position, normal } = ballToCube(sphere, HALF);

  it('keeps the sphere’s vertices one for one, which is what lets it be a morph target at all', () => {
    const src = sphere.getAttribute('position');
    expect(position.count).toBe(src.count);
    expect(normal.count).toBe(src.count);
  });

  it('lands every vertex on the surface of the box, never inside it', () => {
    for (let i = 0; i < position.count; i++) {
      const on = [position.getX(i) / HALF[0], position.getY(i) / HALF[1], position.getZ(i) / HALF[2]];
      // Exactly one axis is at full extent — that is what "on a face" means —
      // and none is past it.
      expect(Math.max(...on.map(Math.abs))).toBeCloseTo(1, 6);
    }
  });

  it('gives each vertex the axis-aligned normal of the face it landed on', () => {
    for (let i = 0; i < normal.count; i++) {
      const n = [normal.getX(i), normal.getY(i), normal.getZ(i)];
      // A unit vector along one axis: two zeros and a ±1. Interpolating from
      // the sphere's radial normals to these is what hardens the faces.
      expect(n.filter((c) => c !== 0)).toHaveLength(1);
      expect(Math.abs(n.reduce((a, c) => a + c, 0))).toBe(1);
    }
  });

  it('picks one face for a vertex sitting exactly on an edge, rather than two', () => {
    // A UV sphere rarely lands one here, but a corner vertex would otherwise
    // claim every axis at once and come out with a non-unit normal.
    const corner = new BufferGeometry();
    corner.setAttribute('position', new Float32BufferAttribute([0.05, 0.05, 0.05], 3));
    const n = ballToCube(corner, HALF).normal;
    expect([n.getX(0), n.getY(0), n.getZ(0)]).toEqual([1, 0, 0]);
  });

  it('leaves a degenerate vertex at the origin where it is instead of emitting NaN', () => {
    // NaN in a position attribute takes the whole mesh off screen, which is a
    // far worse failure than a vertex that does not move.
    const origin = new BufferGeometry();
    origin.setAttribute('position', new Float32BufferAttribute([0, 0, 0], 3));
    const { position: p, normal: n } = ballToCube(origin, HALF);
    expect([p.getX(0), p.getY(0), p.getZ(0)]).toEqual([0, 0, 0]);
    // Every axis ties at zero, so the first one wins and the normal stays a
    // unit vector rather than becoming (0,0,0) or three ones.
    expect([n.getX(0), n.getY(0), n.getZ(0)]).toEqual([0, 0, 0]);
  });
});

describe('morphAt', () => {
  it('holds the ball’s shape through the launch', () => {
    expect(morphAt(0)).toBe(0);
    expect(morphAt(0.12)).toBe(0);
  });

  it('is a cube for the last stretch, so it is not still resolving as it seats itself', () => {
    expect(morphAt(0.8)).toBe(1);
    expect(morphAt(1)).toBe(1);
  });

  it('hardens smoothly through the middle of the arc, easing at both ends', () => {
    expect(morphAt(0.46)).toBeCloseTo(0.5, 6);
    let last = -1;
    for (let k = 0; k <= 1.0001; k += 0.02) {
      const v = morphAt(k);
      expect(v).toBeGreaterThanOrEqual(last);
      last = v;
    }
  });

  it('clamps rather than overshooting, since a flight can tick past its end', () => {
    expect(morphAt(-1)).toBe(0);
    expect(morphAt(4)).toBe(1);
  });
});
