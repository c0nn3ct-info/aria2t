// The faint speed wave bleeding out from behind the popup hero's card. It is
// the site's stand-in for the extension's own backdrop (SpeedSpark over
// Sparkline, extension/src/components/charts/): the same shape — total
// throughput, smoothed, sqrt-scaled, a gradient glow in the accent colour with
// the oldest edge masked out — drawn by a renderer small enough to live on a
// static site. Decorative, and pure: the parent owns the rolling buffer, so the
// readout above it and the wave under it are the same samples.
import { useId } from 'react';
import { cn } from '@/lib/utils';

const VW = 320;
const VH = 120;
/** Headroom above the tallest sample, so a peak never touches the top edge. */
const TOP_PAD = 0.12;

// Catmull-Rom → cubic bezier, tension 0.2, the same curve the extension's
// sparkline draws. Callers guard on two points, so there is no degenerate case.
function smoothPath(coords: [number, number][]): string {
  const t = 0.2;
  const d = [`M${coords[0][0].toFixed(1)} ${coords[0][1].toFixed(1)}`];
  for (let i = 0; i < coords.length - 1; i++) {
    const p0 = coords[i === 0 ? 0 : i - 1];
    const p1 = coords[i];
    const p2 = coords[i + 1];
    const p3 = coords[i + 2 < coords.length ? i + 2 : i + 1];
    const c1x = p1[0] + (p2[0] - p0[0]) * t;
    const c1y = p1[1] + (p2[1] - p0[1]) * t;
    const c2x = p2[0] - (p3[0] - p1[0]) * t;
    const c2y = p2[1] - (p3[1] - p1[1]) * t;
    d.push(
      `C${c1x.toFixed(1)} ${c1y.toFixed(1)} ${c2x.toFixed(1)} ${c2y.toFixed(1)} ${p2[0].toFixed(1)} ${p2[1].toFixed(1)}`,
    );
  }
  return d.join(' ');
}

interface Props {
  /** Total throughput per sample, oldest first. */
  points: number[];
  /** Shared y-scale ceiling; the series peak is a sane default. */
  max: number;
  className?: string;
}

export function AmbientWave({ points, max, className }: Props) {
  const gid = useId().replace(/:/g, '');
  if (points.length < 2) return null;

  const sqrtPeak = Math.sqrt(Math.max(max, 1));
  const usableH = VH * (1 - TOP_PAD);
  const stepX = VW / (points.length - 1);
  const coords: [number, number][] = points.map((v, i) => [
    i * stepX,
    VH - (Math.sqrt(Math.max(0, v)) / sqrtPeak) * usableH,
  ]);
  const line = smoothPath(coords);

  return (
    <svg
      viewBox={`0 0 ${VW} ${VH}`}
      preserveAspectRatio="none"
      aria-hidden
      // The buffer drops its oldest sample every tick; unmasked, that sample
      // pops off the left edge. Faded, the cut reads as the wave scrolling in
      // from off-screen — the extension's SpeedSpark carries the same mask.
      className={cn('[mask-image:linear-gradient(to_right,transparent,black_12%)]', className)}
    >
      <defs>
        <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="currentColor" stopOpacity={0.9} />
          <stop offset="100%" stopColor="currentColor" stopOpacity={0} />
        </linearGradient>
      </defs>
      <path d={`${line} L${VW.toFixed(1)} ${VH} L0 ${VH} Z`} fill={`url(#${gid})`} />
      <path
        d={line}
        fill="none"
        stroke="currentColor"
        strokeWidth={1.75}
        strokeLinejoin="round"
        strokeLinecap="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}
