import type { ReactNode } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Card } from '@/components/ui/card';
import { mulberry32, stepSpeed } from '@/lib/mock-motion';
import { Grid, Stack } from '@/storybook/layout';
import { AmbientWave } from './ambient-wave';

const MIB = 1024 * 1024;

/** One stretch of a run: `count` samples walked toward `target` bytes per second. */
type Leg = [count: number, target: number];

/**
 * Throughput samples built with the site's own motion helpers
 * (`src/lib/mock-motion.ts`) instead of hand-written numbers: `stepSpeed` is the
 * AR(1) walk the popup hero seeds its buffer with, so a story's wave carries the
 * shape a real minute of aria2 traffic has, and `mulberry32` makes it identical
 * on every render — including the one the smoke test mounts.
 */
function traffic(seed: number, legs: readonly Leg[]): number[] {
  const rnd = mulberry32(seed);
  const out: number[] = [];
  let v = legs[0][1];
  for (const [count, target] of legs) {
    for (let i = 0; i < count; i++) {
      v = stepSpeed(v, target, rnd);
      out.push(v);
    }
  }
  return out;
}

const peak = (points: number[]) => Math.max(...points);
const mib = (bps: number) => `${(bps / MIB).toFixed(1)} MiB/s`;

/**
 * A minute at 1 Hz — the window the extension's traffic buffer keeps, and the
 * one the popup hero hands over: a busy queue, a dip while a mirror renegotiates,
 * then a faster stretch.
 */
const QUEUE = traffic(0x77617665, [
  [22, 29 * MIB],
  [12, 21 * MIB],
  [26, 33 * MIB],
]);

const meta = {
  title: 'Blocks/AmbientWave',
  component: AmbientWave,
  // `h-full w-full` is the class the popup hero itself passes: the svg has no
  // intrinsic size and `preserveAspectRatio="none"`, so it takes whatever box it
  // is put in. Every story below supplies that box through `Field`.
  args: { points: QUEUE, max: peak(QUEUE), className: 'h-full w-full' },
  tags: ['autodocs'],
} satisfies Meta<typeof AmbientWave>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Caption under a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/** The sized, accent-coloured field the wave fills — it draws in `currentColor`. */
function Field({ height = 120, children }: { height?: number; children: ReactNode }) {
  return (
    <div className="text-primary" style={{ height }}>
      {children}
    </div>
  );
}

/** The default reading: a minute of total throughput, scaled against its own peak. */
export const LiveTraffic: Story = {
  render: () => (
    <Stack gap={8} style={{ maxWidth: 460 }}>
      <Field>
        <AmbientWave points={QUEUE} max={peak(QUEUE)} className="h-full w-full" />
      </Field>
      <Label>{`60 samples · peak ${mib(peak(QUEUE))}`}</Label>
    </Stack>
  ),
};

/**
 * Silhouettes worth recognising. The smoothing is deliberately heavy, so a
 * stalled mirror decays over several samples rather than dropping off a cliff.
 */
export const TrafficShapes: Story = {
  render: () => (
    <Grid columns={2} gap={20} align="stretch" style={{ maxWidth: 640 }}>
      {[
        { label: 'one fast mirror, holding', points: traffic(0x1101, [[60, 24 * MIB]]) },
        {
          label: 'a torrent finding peers',
          points: traffic(0x1102, [
            [6, 0.6 * MIB],
            [6, 2 * MIB],
            [8, 6 * MIB],
            [10, 14 * MIB],
            [30, 22 * MIB],
          ]),
        },
        {
          label: 'a mirror drops, the retry lands',
          points: traffic(0x1103, [
            [18, 18 * MIB],
            [16, 0.3 * MIB],
            [26, 16 * MIB],
          ]),
        },
        {
          label: 'the queue drains, file by file',
          points: traffic(0x1104, [
            [12, 26 * MIB],
            [12, 17 * MIB],
            [12, 9 * MIB],
            [12, 4 * MIB],
            [12, 1.5 * MIB],
          ]),
        },
      ].map(({ label, points }) => (
        <Stack key={label} gap={8}>
          <Field height={88}>
            <AmbientWave points={points} max={peak(points)} className="h-full w-full" />
          </Field>
          <Label>{label}</Label>
        </Stack>
      ))}
    </Grid>
  ),
};

/**
 * `max` is the y-scale ceiling, not a maximum the samples are clipped to. The
 * series peak is the sane default; a link-rate ceiling flattens the same run
 * into the floor, and a ceiling *under* the series sends its peaks out through
 * the top of the viewBox.
 */
export const SharedCeiling: Story = {
  render: () => (
    <Stack gap={20} style={{ maxWidth: 460 }}>
      {[
        { label: `max = the series peak (${mib(peak(QUEUE))})`, max: peak(QUEUE) },
        { label: 'max = 125 MB/s, a gigabit link', max: 125_000_000 },
        { label: 'max = a third of the peak — the tops are cut off', max: peak(QUEUE) / 3 },
      ].map(({ label, max }) => (
        <Stack key={label} gap={8}>
          <Field height={96}>
            <AmbientWave points={QUEUE} max={max} className="h-full w-full" />
          </Field>
          <Label>{label}</Label>
        </Stack>
      ))}
    </Stack>
  ),
};

/**
 * What the site actually renders: the wave pinned to the bottom two thirds of
 * the popup hero's card, tinted with the accent and faint enough to read as a
 * backdrop rather than as a chart. The readout above it and the wave under it
 * are the same samples, which is the whole reason the parent owns the buffer.
 */
export const BehindTheHero: Story = {
  render: () => (
    <Card
      variant="elevated"
      padding="md"
      style={{ position: 'relative', overflow: 'hidden', maxWidth: 348 }}
    >
      <div
        className="text-primary"
        style={{
          position: 'absolute',
          left: 0,
          right: 0,
          bottom: 0,
          height: '66%',
          opacity: 0.15,
          pointerEvents: 'none',
        }}
      >
        <AmbientWave points={QUEUE} max={peak(QUEUE)} className="h-full w-full" />
      </div>
      <Stack gap={8} style={{ position: 'relative' }}>
        <div
          className="text-label-small text-on-surface-variant"
          style={{ textTransform: 'uppercase', letterSpacing: '0.16em' }}
        >
          aria2t
        </div>
        <div className="text-headline-small font-medium">5 downloads</div>
        <div
          className="text-label-medium text-on-surface-variant"
          style={{ fontVariantNumeric: 'tabular-nums' }}
        >
          {`${mib(QUEUE[QUEUE.length - 1])} total`}
        </div>
      </Stack>
    </Card>
  ),
};

/**
 * Under two samples there is nothing to draw between, so the component renders
 * nothing at all — the state the popup is in for one tick after a cold connect,
 * before the buffer has a second reading.
 */
export const NotEnoughSamples: Story = {
  render: () => (
    <Stack gap={20} style={{ maxWidth: 460 }}>
      {[
        { label: 'points = [] — no history yet', points: [] },
        { label: 'points = [one sample] — still nothing to join', points: [29 * MIB] },
      ].map(({ label, points }) => (
        <Stack key={label} gap={8}>
          <div
            className="text-on-surface-variant"
            style={{ height: 88, border: '1px dashed', borderRadius: 8, opacity: 0.4 }}
          >
            <AmbientWave points={points} max={29 * MIB} className="h-full w-full" />
          </div>
          <Label>{label}</Label>
        </Stack>
      ))}
    </Stack>
  ),
};

/** The controls panel drives one live wave — edit `points` to reshape it. */
export const Playground: Story = {
  render: (args) => (
    <div style={{ maxWidth: 460 }}>
      <Field>
        <AmbientWave {...args} />
      </Field>
    </div>
  ),
};
