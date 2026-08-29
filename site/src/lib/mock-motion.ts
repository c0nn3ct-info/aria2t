// Shared motion for the two live mocks on the home page (the TUI list and the
// extension popup). Both have the same problem: the frame is prerendered, so
// the opening client render must match the server's byte for byte or React
// throws away the markup with a hydration mismatch. Both solve it the same way,
// by seeding from a deterministic PRNG and only switching to Math.random after
// mount.

/** Deterministic PRNG, so a seeded frame is identical on server and client. */
export function mulberry32(seed: number): () => number {
  return () => {
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/**
 * One step of a speed random-walk: an AR(1) low-pass pulling toward `target`,
 * clamped to a band around it. The low-pass is what keeps the numbers from
 * jittering like a random sequence and makes them read as a real transfer.
 */
export function stepSpeed(prev: number, target: number, rnd: () => number): number {
  const next = prev * 0.8 + target * 0.2 + (rnd() - 0.5) * target * 0.3;
  return Math.max(target * 0.4, Math.min(target * 1.6, next));
}

/**
 * Run a walk forward `steps` times from the seed so the first painted frame is
 * already settled. Without this the mock opens on its start value and visibly
 * shakes itself out over the first few seconds.
 */
export function settle(start: number, target: number, rnd: () => number, steps = 16): number {
  let v = start;
  for (let i = 0; i < steps; i++) v = stepSpeed(v, target, rnd);
  return v;
}
