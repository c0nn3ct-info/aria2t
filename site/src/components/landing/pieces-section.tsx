// "See how much is already here": the detail screen's piece map and peer list.
// The map fills on the shared tick and the peer rows move with it, so the
// percentage in the header and the cells under it never disagree. It fills to
// 100 and stops there, which is the torrent finishing - the peers carry on
// uploading, which is the point the list beside it makes.
import { ArrowUp, Check, LayoutGrid } from 'lucide-react';
import { cn } from '@/lib/utils';
import { peerRows, pieceCells, piecePct, peers, type Cell } from '@/lib/landing-motion';
import {
  MockStage,
  LandingSection,
  MockCard,
  MockHeader,
  PointList,
  SectionHeading,
  type Point,
} from './shell';
import { useTick } from './use-tick';
import { t } from '@/i18n';

/** Cells the map is drawn from: 128 shown for 256 real pieces, two apiece. */
const CELLS = 128;
const PIECES = 256;

const CELL_CLASS: Record<Cell, string> = {
  have: 'bg-primary',
  edge: 'bg-primary/45',
  none: 'bg-surface-container-high',
};

export function PiecesSection() {
  const tick = useTick();
  const pct = piecePct(tick);
  const have = Math.round((pct / 100) * PIECES);
  const rows = peerRows(tick);
  const peerCount = peers(tick);

  const points: readonly Point[] = [
    {
      icon: LayoutGrid,
      tone: 'primary',
      text: t('landing.pieces.p1').replace('{{count}}', String(have)),
    },
    { icon: ArrowUp, tone: 'tertiary', text: t('landing.pieces.p2') },
    { icon: Check, tone: 'success', text: t('landing.pieces.p3') },
  ];

  return (
    <LandingSection id="details">
      {/* Heading first in the DOM so the section reads in order on a phone;
          on a wide screen the panel takes the left column. */}
      <div className="grid items-center gap-10 lg:grid-cols-[420px_minmax(0,1fr)] lg:gap-14">
        <div className="min-w-0 lg:order-2">
          {/* Wrapped rather than given the attribute: `SectionHeading` is
              shared, and where a band arrives is the page's business. */}
          <div data-enter>
            <SectionHeading
              title={t('landing.pieces.h2')}
              body={t('landing.pieces.body')}
              className="max-w-[520px]"
            />
          </div>
          <PointList points={points} className="mt-7" />
        </div>

        <div data-enter className="min-w-0 lg:order-1">
          <MockStage>
            <MockCard>
              <MockHeader
                title={t('landing.pieces.card')}
                aside={
                  <span dir="ltr" className="font-mono text-[11px] text-primary">
                    {`${Math.round(pct)}%`}
                  </span>
                }
              />
              <div className="p-4">
                {/* Decorative: the panel's own header states the percentage and
                    the list beside it states the piece count, so the grid adds
                    texture rather than information. */}
                <div aria-hidden className="grid grid-cols-[repeat(16,minmax(0,1fr))] gap-[3px]">
                  {pieceCells(CELLS, pct).map((c, i) => (
                    <i key={i} className={cn('aspect-square rounded-[2px]', CELL_CLASS[c])} />
                  ))}
                </div>
                <div className="mt-3 flex justify-between font-mono text-[10px] text-on-surface-variant">
                  <span dir="ltr" className="min-w-0 truncate">
                    ubuntu-24.04.2…iso
                  </span>
                  <span dir="ltr">{`${have} / ${PIECES}`}</span>
                </div>
              </div>

              <div className="px-4 pb-1.5 text-label-small uppercase tracking-[0.14em] text-on-surface-variant">
                {t('landing.pieces.peers')}
              </div>
              {rows.map((p) => (
                <div
                  key={p.ip}
                  className="flex items-center gap-3 border-t border-outline-variant px-4 py-2.5"
                >
                  <span dir="ltr" className="flex-1 font-mono text-[11px]">
                    {p.ip}
                  </span>
                  <span className="h-[5px] w-[68px] shrink-0 overflow-hidden rounded-pill bg-surface-container-high sm:w-[90px]">
                    <i
                      className={cn(
                        'block h-full rounded-pill transition-[width] duration-[280ms] ease-linear',
                        p.up ? 'bg-tertiary' : 'bg-primary',
                      )}
                      style={{ width: p.width }}
                    />
                  </span>
                  <span
                    dir="ltr"
                    className={cn(
                      'w-[72px] shrink-0 text-end font-mono text-[10px] tabular-nums',
                      p.up ? 'text-tertiary' : 'text-primary',
                    )}
                  >
                    {p.speed}
                  </span>
                </div>
              ))}
              <div className="flex items-center gap-2.5 border-t border-outline-variant px-4 py-3.5 font-mono text-[10px] text-on-surface-variant">
                <span>{t('landing.pieces.dht').replace('{{count}}', String(peerCount))}</span>
                <span className="ms-auto text-tertiary">{t('landing.pieces.ratio')}</span>
              </div>
            </MockCard>
          </MockStage>
        </div>
      </div>
    </LandingSection>
  );
}
