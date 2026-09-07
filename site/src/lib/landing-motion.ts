// Motion for the landing page's live figures: the mirror bars, the minute of
// throughput behind the stats chart, the piece map and the peer rows. Unlike
// the two mocks (`mock-motion.ts`), which random-walk after mount, everything
// here is a pure function of one tick counter `t`. That buys two things: the
// first frame (t = 0) is the same on the server and on the client, so the
// prerender hydrates without a seed/PRNG dance; and every live section on the
// page can share one clock and still agree about what "now" looks like.

/**
 * A smooth, deterministic wobble in [-1.18, 1.18]: five sines at unrelated
 * frequencies, phase-shifted by `k` so two figures never move in step.
 */
export function wob(t: number, k: number): number {
  return (
    Math.sin(t * 0.913 + k) * 0.42 +
    Math.sin(t * 0.271 + k * 2.3) * 0.3 +
    Math.sin(t * 1.771 + k * 0.7) * 0.16 +
    Math.sin(t * 3.119 + k * 1.9) * 0.08 +
    Math.sin(t * 0.077 + k * 3.7) * 0.22
  );
}

/** Global download/upload in MiB/s at tick `t`. */
export function sample(t: number): { d: number; u: number } {
  return { d: 28.7 + 6.4 * wob(t * 0.37, 1.2), u: 4.7 + 1.15 * wob(t * 0.31, 3.4) };
}

/** Samples in the chart: one per tick, the design's minute at 300 ms. */
export const HISTORY_N = 200;

/** The last `HISTORY_N` samples ending at tick `t`, oldest first. */
export function history(t: number): { d: number[]; u: number[] } {
  const d: number[] = [];
  const u: number[] = [];
  for (let i = 0; i < HISTORY_N; i++) {
    const s = sample(t - (HISTORY_N - 1) + i);
    d.push(s.d);
    u.push(s.u);
  }
  return { d, u };
}

export const CHART_W = 1000;
export const CHART_H = 220;

/** A polyline through `vals` scaled to `max`, in the chart's viewBox. */
export function linePath(vals: readonly number[], max: number): string {
  const n = vals.length;
  let d = '';
  for (let i = 0; i < n; i++) {
    const x = (i / (n - 1)) * CHART_W;
    const y = CHART_H - Math.max(0, Math.min(1, vals[i] / max)) * CHART_H;
    d += (i ? ' L' : 'M') + x.toFixed(1) + ' ' + y.toFixed(1);
  }
  return d;
}

/** `linePath` closed down to the baseline, for the area under the line. */
export function fillPath(vals: readonly number[], max: number): string {
  return `${linePath(vals, max)} L${CHART_W.toFixed(1)} ${CHART_H.toFixed(1)} L0.0 ${CHART_H.toFixed(1)} Z`;
}

/** Formats a MiB/s figure the way the mocks do: one decimal. */
export function mib(v: number): string {
  return v.toFixed(1);
}

export interface MirrorRow {
  name: string;
  /** Speed, formatted. */
  speed: string;
  /** Bar width, as a CSS percentage. */
  width: string;
}

const MIRRORS: ReadonlyArray<[name: string, base: number, k: number]> = [
  ['mirror.one', 6.2, 0.4],
  ['mirror.two', 4.1, 2.1],
  ['mirror.three', 2.7, 4.6],
];

/** The three mirrors of one file, each with its own live speed. */
export function mirrorRows(t: number): MirrorRow[] {
  return MIRRORS.map(([name, base, k]) => {
    const sp = Math.max(0.3, base * (1 + 0.22 * wob(t * 0.4, k)));
    return { name, speed: `${mib(sp)} MiB/s`, width: `${Math.min(100, (sp / 7.4) * 100).toFixed(1)}%` };
  });
}

/** The sum of the mirror speeds, formatted. */
export function mirrorTotal(rows: readonly MirrorRow[]): string {
  return mib(rows.reduce((a, m) => a + parseFloat(m.speed), 0));
}

export interface PeerRow {
  ip: string;
  /** True for the peer we upload to; it takes the upload colour. */
  up: boolean;
  speed: string;
  width: string;
}

const PEERS: ReadonlyArray<[ip: string, base: number, up: boolean, k: number]> = [
  ['185.14.•.•', 4.2, false, 1.1],
  ['92.63.•.•', 2.7, false, 2.7],
  ['51.15.•.•', 1.1, true, 4.2],
];

export function peerRows(t: number): PeerRow[] {
  return PEERS.map(([ip, base, up, k]) => {
    const sp = Math.max(0.2, base * (1 + 0.26 * wob(t * 0.36, k)));
    // No direction glyph: it made the upload row's cell wider than the two
    // download rows and pushed the column out of alignment. The row's colour
    // carries the direction instead.
    return {
      ip,
      up,
      speed: `${mib(sp)} MiB/s`,
      width: `${Math.min(100, (sp / 5.6) * 100).toFixed(1)}%`,
    };
  });
}

export function connections(t: number): number {
  return Math.round(42 + 5 * wob(t * 0.22, 6.1));
}

export function peers(t: number): number {
  return Math.round(34 + 6 * wob(t * 0.18, 8.4));
}

/**
 * The piece map's completion, in percent: it starts at 63 and creeps 0.031 a
 * tick, reaching 100 in about six minutes and staying there.
 *
 * It used to wrap back to 5 so the map would never park at 100. That made the
 * pieces on disk fall from 255 to 13 every thirty-three seconds, and bytes on
 * disk do not un-download. A torrent that finishes is exactly what the section
 * beside it claims happens next: it moves to seeding, and the peers keep
 * uploading.
 */
export function piecePct(t: number): number {
  return Math.min(100, 63 + 0.031 * t);
}

export type Cell = 'have' | 'edge' | 'none';

/**
 * The piece map as the TUI draws it (tui/internal/ui/detail.go): the pieces we
 * have, then a six-cell ragged frontier being fetched, then nothing yet.
 */
export function pieceCells(count: number, pct: number): Cell[] {
  const have = Math.round((pct / 100) * count);
  return Array.from({ length: count }, (_, i) =>
    i < have ? 'have' : i < have + 6 ? 'edge' : 'none',
  );
}
