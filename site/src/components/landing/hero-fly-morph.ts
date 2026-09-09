// The shape a part takes on its way from a pylon to the blueprint: it leaves
// as a ball and arrives as a cube.
//
// Done as a morph target rather than by swapping geometries, so the change is
// continuous and costs one attribute rather than a second mesh and a
// crossfade. The trick that makes it cheap is that a sphere's own vertices
// already sit one-to-one on a cube: scale each by `r / max(|x|,|y|,|z|)` and it
// lands on the face of the cube of half-extent `r`, in the same order, with
// the same winding. So the cube is not a second model at all — it is the same
// mesh with one attribute of offsets, and `morphTargetInfluences[0]` slides
// between them.
//
// A UV sphere's quads straddle the cube's edges, so the far end reads as a
// cube with slightly soft edges rather than a hard one. That is the right
// shape here: every crate on the belt is a `roundedBox`, and a part that
// hardened into a razor-edged cube would not match the thing it becomes.
import { BufferAttribute, Float32BufferAttribute } from 'three';
import type { BufferGeometry } from 'three';

/**
 * Builds the cube end of the morph for a sphere geometry, as the
 * `morphAttributes` entries three.js wants: positions projected onto the box
 * of the given half-extents, and the axis-aligned face normals that go with
 * them.
 *
 * Both are *absolute* targets, not deltas — `morphTargetsRelative` stays
 * false, which is what a geometry built this way reports by default.
 */
export function ballToCube(
  sphere: BufferGeometry,
  half: [x: number, y: number, z: number],
): { position: Float32BufferAttribute; normal: Float32BufferAttribute } {
  const src = sphere.getAttribute('position') as BufferAttribute;
  const n = src.count;
  const pos = new Float32Array(n * 3);
  const nor = new Float32Array(n * 3);
  for (let i = 0; i < n; i++) {
    const x = src.getX(i);
    const y = src.getY(i);
    const z = src.getZ(i);
    const ax = Math.abs(x);
    const ay = Math.abs(y);
    const az = Math.abs(z);
    const m = Math.max(ax, ay, az);
    // A vertex at the origin has no direction to project along. A sphere has
    // none, but the guard keeps a degenerate input from emitting NaN, which
    // would take the whole mesh off screen rather than fail visibly.
    const k = m === 0 ? 0 : 1 / m;
    pos[i * 3] = x * k * half[0];
    pos[i * 3 + 1] = y * k * half[1];
    pos[i * 3 + 2] = z * k * half[2];
    // The face it landed on is the axis it was furthest along, so the cube's
    // normal there is that axis. Interpolating from the sphere's radial
    // normals to these is what makes the faces read flat as it hardens.
    nor[i * 3] = m === ax ? Math.sign(x) : 0;
    nor[i * 3 + 1] = m === ay && m !== ax ? Math.sign(y) : 0;
    nor[i * 3 + 2] = m === az && m !== ax && m !== ay ? Math.sign(z) : 0;
  }
  return {
    position: new Float32BufferAttribute(pos, 3),
    normal: new Float32BufferAttribute(nor, 3),
  };
}

/**
 * How far along the morph a flight of progress `k` is.
 *
 * Finished before it lands, and not linearly: the ball holds its shape
 * through the launch, hardens through the middle of the arc, and is a cube
 * for the last stretch, so what the eye catches is a transformation in flight
 * rather than a shape that is still resolving as it seats itself.
 */
export function morphAt(k: number): number {
  const t = Math.min(1, Math.max(0, (k - 0.12) / 0.68));
  return t * t * (3 - 2 * t);
}
