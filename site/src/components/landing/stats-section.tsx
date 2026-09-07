// "See what is happening right now": a minute of throughput and the four
// figures beside it. Every value comes from the shared tick, so the chart's
// last sample and the tile under it are the same number rather than two
// numbers that happen to look alike.
import { useId } from 'react';
import { cn } from '@/lib/utils';
import {
  CHART_H,
  CHART_W,
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

/** The y-scale both series share, in MiB/s: headroom over the busiest sample. */
const SCALE = 40;

interface TileProps {
  label: string;
  value: string;
  note: string;
  tone: 'primary' | 'tertiary' | 'plain';
}

const TILE_TONE: Record<TileProps['tone'], string> = {
  primary: 'bg-primary-container text-primary-on-container',
  tertiary: 'bg-tertiary-container text-tertiary-on-container',
  plain: 'border border-outline-variant bg-surface-container-low',
};

function Tile({ label, value, note, tone }: TileProps) {
  return (
    <div className={cn('rounded-md p-5', TILE_TONE[tone])}>
      <span
        className={cn(
          'text-label-small uppercase tracking-[0.14em]',
          tone === 'plain' && 'text-on-surface-variant',
        )}
      >
        {label}
      </span>
      <b dir="ltr" className="mt-2 block font-mono text-2xl font-medium tabular-nums">
        {value}
      </b>
      <small
        className={cn('mt-1 block text-label-small', tone === 'plain' && 'text-on-surface-variant')}
      >
        {note}
      </small>
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

      <div data-enter className="rounded-lg border border-outline-variant bg-surface-container-low px-6 pb-5 pt-6 sm:px-7">
        <div className="flex flex-wrap items-start justify-between gap-6">
          <div>
            <span className="text-label-small uppercase tracking-[0.16em] text-on-surface-variant">
              {t('landing.stats.minute')}
            </span>
            <div dir="ltr" className="mt-2 flex items-baseline gap-2.5">
              <b className="font-mono text-[clamp(30px,5vw,44px)] font-medium tabular-nums tracking-[-0.03em] text-primary">
                {mib(down)}
              </b>
              <span className="font-mono text-[15px] text-on-surface-variant">MiB/s</span>
            </div>
          </div>
          <div className="flex gap-4 text-label-medium text-on-surface-variant">
            <span className="inline-flex items-center gap-2">
              <i aria-hidden className="block h-[3px] w-3 rounded-sm bg-primary" />
              {t('landing.stats.legend_down')}
            </span>
            <span className="inline-flex items-center gap-2">
              <i aria-hidden className="block h-[3px] w-3 rounded-sm bg-tertiary" />
              {t('landing.stats.legend_up')}
            </span>
          </div>
        </div>

        {/* Decorative: the two figures above and the four tiles below say the
            same thing in words, so the drawing itself carries no information a
            screen reader would otherwise miss. */}
        <svg
          viewBox={`0 0 ${CHART_W} ${CHART_H}`}
          preserveAspectRatio="none"
          aria-hidden
          className="mt-5 block h-[200px] w-full rtl:-scale-x-100"
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
          <path
            d={linePath(h.u, SCALE)}
            fill="none"
            stroke="hsl(var(--tertiary))"
            strokeWidth={2}
            strokeLinejoin="round"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
        <div className="mt-2.5 flex justify-between font-mono text-[10px] text-on-surface-variant">
          <span>{t('landing.stats.ago')}</span>
          <span>{t('landing.stats.now')}</span>
        </div>
      </div>

      <div data-enter-stagger="wipe" className="mt-4 grid grid-cols-2 gap-3.5 lg:grid-cols-4">
        <Tile
          tone="primary"
          label={t('landing.stats.legend_down')}
          value={`${mib(down)} MiB/s`}
          note={t('landing.stats.tile_down_note')}
        />
        <Tile
          tone="tertiary"
          label={t('landing.stats.legend_up')}
          value={`${mib(up)} MiB/s`}
          note={t('landing.stats.tile_up_note')}
        />
        <Tile
          tone="plain"
          label={t('landing.stats.tile_conns')}
          value={String(connections(tick))}
          note={t('landing.stats.tile_conns_note')}
        />
        <Tile
          tone="plain"
          label={t('landing.stats.tile_peers')}
          value={String(peers(tick))}
          note={t('landing.stats.tile_peers_note')}
        />
      </div>
    </LandingSection>
  );
}
