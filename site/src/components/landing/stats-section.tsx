// "See what is happening right now": the minute of download on the left, and
// the three figures that are not the download on the right.
//
// It used to draw both series on one 0-40 MiB/s axis and then repeat all four
// figures as a row of tiles under the chart. Two things were wrong with that.
// Upload never leaves 3.7-5.8, so on that axis its line was 13 of the chart's
// 220 units - 6%, measured on the running page - a flat rule under the
// download's fill, with a legend promising a second series. And the download
// figure was printed twice, at 40px in the card and again in the first tile
// sixty pixels below it.
//
// So each figure is stated once, and each gets the drawing its own numbers can
// carry: the download keeps the minute as a wave, the upload takes the same
// minute as bars scaled to its own window (`barHeights`), and the two counts
// are panels with nothing to draw. One material for all three panels beside the
// chart - the page's filled containers are what the hero's buttons are made of,
// so a figure in one reads as something to press.
import { useId } from 'react';
import { cn } from '@/lib/utils';
import {
  CHART_H,
  CHART_W,
  barHeights,
  connections,
  fillPath,
  history,
  linePath,
  mib,
  peers,
} from '@/lib/landing-motion';
import { LandingSection, SectionHeading } from './shell';
import { useTick } from './use-tick';
import { t } from '@/i18n';

/** The wave's y-scale in MiB/s: headroom over the busiest sample. */
const SCALE = 40;

/** Bars in the upload tile. Twenty-six of the minute's two hundred samples. */
const BARS = 26;

/**
 * One figure, everywhere in the band: an uppercase label and the number with
 * its unit set smaller and dimmer beside it. The lead figure and the three
 * panels used to set both of those differently - two trackings for the same
 * label, and the unit at the figure's own size in the panels but smaller in
 * the card. One component means the four cannot drift again.
 *
 * The three panels carry a caption under the figure, the lead none: the wave
 * and its axis are already what sits under that one, and "4 downloads running"
 * next to a chart of those four downloads was the fourth line of copy in a row.
 */
interface StatProps {
  label: string;
  /** The digits. `MiB/s` is the unit, not part of the number. */
  value: string;
  unit?: string;
  /** A caption under the figure. The lead figure has the wave instead. */
  note?: string;
  /** `lead` is the wave card's figure; `panel` the three beside it. */
  size: 'lead' | 'panel';
  /** Which accent the figure takes, if any. */
  tone?: 'down' | 'up' | 'plain';
}

const FIGURE_SIZE: Record<StatProps['size'], string> = {
  lead: 'text-[clamp(40px,5.8vw,56px)] tracking-[-0.04em]',
  panel: 'text-[38px] tracking-[-0.03em]',
};

/** The gap scales with the numeral: 6px beside 56px is a collision. */
const UNIT_SIZE: Record<StatProps['size'], string> = {
  lead: 'ms-3 text-[17px]',
  panel: 'ms-2 text-[15px]',
};

const FIGURE_TONE: Record<NonNullable<StatProps['tone']>, string> = {
  down: 'text-primary',
  up: 'text-tertiary',
  plain: '',
};

function Stat({ label, value, unit, note, size, tone = 'plain' }: StatProps) {
  return (
    <div className="min-w-0">
      <span className="block text-[13px] font-medium uppercase leading-none tracking-[0.14em] text-on-surface-variant">
        {label}
      </span>
      <div dir="ltr" className="mt-3.5 flex items-baseline">
        <b className={cn('font-mono font-medium tabular-nums', FIGURE_SIZE[size], FIGURE_TONE[tone])}>
          {value}
        </b>
        {unit && (
          <span className={cn('font-mono text-on-surface-variant', UNIT_SIZE[size])}>
            {unit}
          </span>
        )}
      </div>
      {/* A caption, not a second label: no tracking, no uppercase, reading
          weight. */}
      {note && <p className="mt-3 text-[13px] leading-[1.4] text-on-surface-variant">{note}</p>}
    </div>
  );
}

/** A figure beside its own drawing, in one of the three panels. */
function Panel({ chart, ...stat }: StatProps & { chart?: number[] }) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-md border border-outline-variant bg-surface-container-low p-5">
      <Stat {...stat} />
      {chart && (
        // Decorative: the figure beside it is the same minute's last sample, so
        // the bars add texture rather than information.
        <div aria-hidden dir="ltr" className="flex h-12 w-[42%] shrink-0 items-end gap-[3px]">
          {chart.map((h, i) => (
            <i
              key={i}
              className="block min-w-[2px] flex-1 rounded-sm bg-tertiary"
              style={{ height: `${h}%` }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function StatsSection() {
  const tick = useTick();
  const gid = useId().replace(/:/g, '');
  const h = history(tick);
  const down = h.d[h.d.length - 1];
  const up = h.u[h.u.length - 1];

  return (
    <LandingSection id="live">
      <div data-enter className="mb-8">
        <SectionHeading title={t('landing.stats.h2')} body={t('landing.stats.body')} />
      </div>

      {/* The wave takes the wide track; the three figures stack beside it, and
          the column ends level with the chart card on a wide screen. */}
      <div className="grid items-stretch gap-4 lg:grid-cols-[minmax(0,1.62fr)_minmax(0,1fr)]">
        <div
          data-enter
          className="flex flex-col rounded-lg border border-outline-variant bg-surface-container-low px-6 pb-5 pt-6 sm:px-7"
        >
          <Stat
            size="lead"
            tone="down"
            label={t('landing.stats.minute')}
            value={mib(down)}
            unit="MiB/s"
          />

          {/* Decorative: the figure above states the last sample and the panels
              beside it state the rest, so the drawing carries nothing a screen
              reader would otherwise miss. */}
          <svg
            viewBox={`0 0 ${CHART_W} ${CHART_H}`}
            preserveAspectRatio="none"
            aria-hidden
            className="mt-5 block h-[168px] w-full flex-1 rtl:-scale-x-100"
          >
            <defs>
              <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity={0.34} />
                <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity={0} />
              </linearGradient>
            </defs>
            <path d={fillPath(h.d, SCALE)} fill={`url(#${gid})`} />
            <path
              d={linePath(h.d, SCALE)}
              fill="none"
              stroke="hsl(var(--primary))"
              strokeWidth={2}
              strokeLinejoin="round"
              vectorEffect="non-scaling-stroke"
            />
          </svg>
          <div className="mt-3 flex justify-between font-mono text-[11px] text-on-surface-variant">
            <span>{t('landing.stats.ago')}</span>
            <span>{t('landing.stats.now')}</span>
          </div>
        </div>

        {/* Two rows of equal height rather than three panels pushed to the
            ends: the column has to finish level with the chart beside it, and
            `content-between` bought that with a hole in its middle. */}
        <div data-enter-stagger className="grid grid-rows-2 gap-4">
          <Panel
            size="panel"
            tone="up"
            label={t('landing.stats.legend_up')}
            value={mib(up)}
            unit="MiB/s"
            note={t('landing.stats.tile_up_note')}
            chart={barHeights(h.u, BARS)}
          />
          {/* Two abreast, at both ends of the range: they are the shortest
              figures in the band and a full-width panel each would leave the
              column taller than the chart. */}
          <div className="grid grid-cols-2 gap-4">
            <Panel
              size="panel"
              label={t('landing.stats.tile_conns')}
              value={String(connections(tick))}
              note={t('landing.stats.tile_conns_note')}
            />
            <Panel
              size="panel"
              label={t('landing.stats.tile_peers')}
              value={String(peers(tick))}
              note={t('landing.stats.tile_peers_note')}
            />
          </div>
        </div>
      </div>
    </LandingSection>
  );
}
