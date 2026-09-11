// "One queue for every download": the claim and its three routes down the left
// column, and the queue beside them as one list whose every row carries the
// route it arrived by.
//
// It replaced a draw that put three live input cards over a strip of five
// cells. The strip was the band's payoff and five cards across a row have no
// first and no next, which is the one thing "one list, one order" has to make
// visible; the cards also said each route twice, once as a mock and once as a
// sentence under it. The two draws shipped side by side for a while and the
// list won, so the strings here are the ones that band already had - the copy
// was never what the comparison was about.
//
// Built from the page's own furniture: `SectionHeading` and `PointList` for the
// column of copy, the material `MockCard` is made of for the mock beside it,
// and for the route marks the three icons and three tones the old draw gave its
// input cards - so a row and the point that explains it wear the same mark.
//
// The rows are the first five of `queue-scene`, the store the popup and the
// terminal list already read, so they walk on the same beat and this band and
// the one above it cannot disagree about what is downloading. A static card
// would also have been the only motionless mock on the page.
//
// Its header and footer sum the five rows under them rather than the store's
// globals. The popup can print globals because its header is the daemon's and
// its rows are a window onto a longer queue; this card says "5 downloads" and
// then shows five, so its totals have to be theirs.
import { ExternalLink, FileText, Magnet, type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
// The two figures the terminal list prints, from the file that mirrors its
// formatter - a landing mock of the queue has no business rounding bytes its
// own way.
import { fmtEta, fmtSpeed } from '@/components/list-mock';
import { ITEMS, fmtSize, useQueueScene, type Item, type Kind, type Live, type Route } from '@/lib/queue-scene';
import { LandingSection, PointList, SectionHeading, type Point } from './shell';
import { t } from '@/i18n';

/**
 * A route's mark. Both BitTorrent routes take the magnet - the icon the old
 * draw gave that card, and the one the rail's middle point carries: a torrent
 * and a magnet are one route with two front doors.
 */
const ROUTE_ICON: Record<Route, LucideIcon> = {
  link: ExternalLink,
  torrent: Magnet,
  magnet: Magnet,
  input: FileText,
};

/**
 * Identity, not state: the mark says where a download came from, and the status
 * column keeps the page's status colours to itself. The three tints are the
 * ones `PointList` gives the points beside them, in the same order.
 */
const ROUTE_TINT: Record<Route, string> = {
  link: 'text-primary',
  torrent: 'text-tertiary',
  magnet: 'text-tertiary',
  input: 'text-success-text',
};

/**
 * How many of the queue's downloads the card shows. Five, because
 * `landing.queue.strip.count` says five in all six locales and the figures
 * under the rows are the sum of exactly these.
 */
const SHOWN = 5;

/** aria2's own status words, coloured the way the rest of the page colours them. */
const STATUS_TEXT: Record<Kind, string> = {
  active: 'text-success-text',
  seeding: 'text-tertiary',
  waiting: 'text-on-surface-variant',
  paused: 'text-on-surface-variant',
  error: 'text-error-text',
  done: 'text-on-surface-variant',
};

const STATUS_BAR: Record<Kind, string> = {
  active: 'bg-primary',
  seeding: 'bg-tertiary',
  // Nothing is moving in these three, so the bar states its position and stops
  // asking to be looked at.
  waiting: 'bg-surface-container-highest',
  paused: 'bg-surface-container-highest',
  done: 'bg-success',
  error: 'bg-error',
};

/** Only a transfer gets the accent; a still row's dash is not a measurement. */
const SPEED_TINT: Record<Kind, string | undefined> = {
  active: 'text-primary',
  seeding: 'text-tertiary',
  waiting: undefined,
  paused: undefined,
  error: undefined,
  done: undefined,
};

/**
 * The speed cell. Down while active, up while seeding, and a dash for the rest
 * - the arrow is what says which, since one column carries both.
 */
export function speedCell(kind: Kind, speed: number): string {
  if (kind === 'active') return `\u2193 ${fmtSpeed(speed)}`;
  if (kind === 'seeding') return `\u2191 ${fmtSpeed(speed)}`;
  return '-';
}

/**
 * What a download reports for size. A row that failed before it read a content
 * length has no total, which is the `0 B` the real screen prints there.
 */
export function sizeCell(kind: Kind, bytes: number): string {
  return kind === 'error' ? '0 B' : fmtSize(bytes);
}

/**
 * The last column. A download in flight counts down - remaining bytes over the
 * speed it is doing right now, so the figure runs at a second per second
 * against the bar beside it. A finished torrent has no time left to state and
 * gives its ratio instead, which is the one thing still changing about it.
 */
export function etaCell(kind: Kind, item: Item, live: Live): string {
  if (kind === 'active') return fmtEta(item.bytes * (1 - live.pct / 100), live.speed);
  if (kind === 'seeding') return t('landing.pieces.ratio');
  return '-';
}

/** A ratio is a translated phrase; everything else in that column is a figure. */
export function etaIsFigure(kind: Kind): boolean {
  return kind !== 'seeding';
}

/**
 * What the rows under the header add up to: download for what is transferring,
 * upload for what is seeding. The same split `queue-scene` makes for the whole
 * queue, over the slice this card draws.
 */
export function shownTotals(live: readonly Live[]): { down: number; up: number } {
  let down = 0;
  let up = 0;
  for (const l of live) {
    if (l.kind === 'active') down += l.speed;
    else if (l.kind === 'seeding') up += l.speed;
  }
  return { down, up };
}

/**
 * The column names are the terminal client's own, untranslated for the same
 * reason `active` and `seeding` are: they are what the daemon and the TUI
 * print. Hidden below `md`, where the row folds and each figure sits next to
 * the thing it measures.
 */
const COLUMNS = ['name', 'status', 'progress', 'size', 'speed', 'eta'] as const;

/**
 * One line per column at `md` and up; two stacked lines below it.
 *
 * Every width here is fixed on purpose. Each row is its own grid - they are
 * list items, not one table - so a `1fr` or `auto` track is measured per row
 * and the columns stop lining up between them: the ETA cell alone ranges from
 * a countdown to a ratio to a translated phrase. Only the name flexes, because
 * it is the one cell allowed to truncate.
 */
const GRID =
  'grid-cols-[minmax(0,1fr)_auto] md:grid-cols-[minmax(0,1fr)_64px_96px_72px_92px_84px]';

export function QueueSection() {
  // The first five of the queue, walking on the store's own beat.
  const scene = useQueueScene();
  const rows = ITEMS.slice(0, SHOWN).map((item, i) => ({ item, live: scene.live[i] }));
  const totals = shownTotals(rows.map((r) => r.live));

  // The three routes in the order the rows first use them, each carrying the
  // mark its rows carry.
  const points: readonly Point[] = [
    { icon: ROUTE_ICON.link, tone: 'primary', text: t('landing.queue.in1.body') },
    { icon: ROUTE_ICON.torrent, tone: 'tertiary', text: t('landing.queue.in2.body') },
    { icon: ROUTE_ICON.input, tone: 'success', text: t('landing.queue.in3.body') },
  ];

  return (
    <LandingSection id="sources">
      {/* Centred, not top-aligned: five rows are shorter than the copy beside
          them, and a top-aligned card leaves the column under it empty. */}
      <div className="grid items-center gap-10 lg:grid-cols-[340px_minmax(0,1fr)] lg:gap-14">
        <div className="min-w-0">
          {/* Wrapped rather than given the attribute: `SectionHeading` is
              shared, and where a band arrives is the page's business. */}
          <div data-enter>
            <SectionHeading title={t('landing.queue.h2')} body={t('landing.queue.body')} />
          </div>
          <PointList points={points} className="mt-7" />
        </div>

        {/* The queue: one object, one order, every row carrying its route. */}
        <div
          data-enter
          className="min-w-0 overflow-hidden rounded-md border border-outline-variant bg-background text-on-surface shadow-e4"
        >
          <div className="flex items-center gap-2.5 border-b border-outline-variant px-4 py-3.5 sm:px-5">
            <span className="flex-1 text-title-small font-semibold">
              {t('landing.queue.strip.title')}
            </span>
            <span dir="ltr" className="font-mono text-[11px] tabular-nums text-primary">
              {`↓ ${fmtSpeed(totals.down)}`}
            </span>
          </div>

          <div className="px-4 pb-1 pt-3 sm:px-5">
            <div
              aria-hidden
              className={cn(
                'hidden gap-x-3 pb-2 font-mono text-[10px] uppercase tracking-[0.1em] text-on-surface-variant md:grid',
                GRID,
              )}
            >
              {COLUMNS.map((c, i) => (
                <span key={c} className={i > 2 ? 'text-end' : undefined}>
                  {c}
                </span>
              ))}
            </div>

            <ul>
              {rows.map(({ item, live }) => {
                const Icon = ROUTE_ICON[item.route];
                const still = live.kind !== 'active' && live.kind !== 'seeding';
                return (
                  <li
                    key={item.name}
                    className={cn(
                      'grid items-center gap-x-3 gap-y-2 border-t border-outline-variant py-3',
                      GRID,
                    )}
                  >
                    <span className="flex min-w-0 items-center gap-2.5">
                      <Icon className={cn('h-4 w-4 shrink-0', ROUTE_TINT[item.route])} aria-hidden />
                      {/* The mark is what is seen; the word is for a reader that
                          cannot see which mark it is. */}
                      <span className="sr-only">{item.route}</span>
                      <span
                        dir="ltr"
                        className={cn(
                          'truncate text-body-medium',
                          still && 'text-on-surface-variant',
                        )}
                      >
                        {item.name}
                      </span>
                    </span>
                    <span dir="ltr" className={cn('font-mono text-[10.5px]', STATUS_TEXT[live.kind])}>
                      {live.kind}
                    </span>
                    {/* Full width under the name on a phone, its own column at `md`. */}
                    <span className="col-span-2 h-[5px] overflow-hidden rounded-pill bg-surface-container-high md:col-span-1">
                      {/* The store moves a bar by a second's worth at a time, so
                          the fill is eased across that second rather than
                          stepping once a second. */}
                      <i
                        className={cn(
                          'block h-full rounded-pill transition-[width] duration-[950ms] ease-linear motion-reduce:transition-none',
                          STATUS_BAR[live.kind],
                        )}
                        style={{ width: `${live.pct}%` }}
                      />
                    </span>
                    {/* One line of figures on a phone; three cells at `md`, where
                        `contents` dissolves their wrapper into the row's grid. */}
                    <span className="col-span-2 flex justify-between gap-3 font-mono text-[11px] tabular-nums text-on-surface-variant md:contents">
                      <span dir="ltr" className="md:text-end">
                        {sizeCell(live.kind, item.bytes)}
                      </span>
                      <span dir="ltr" className={cn('md:text-end', SPEED_TINT[live.kind])}>
                        {speedCell(live.kind, live.speed)}
                      </span>
                      <span
                        dir={etaIsFigure(live.kind) ? 'ltr' : undefined}
                        className="whitespace-nowrap md:text-end"
                      >
                        {etaCell(live.kind, item, live)}
                      </span>
                    </span>
                  </li>
                );
              })}
            </ul>
          </div>

          {/* The daemon's own totals, under the rows they are the sum of. */}
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1 border-t border-outline-variant px-4 py-3 font-mono text-[10px] tabular-nums text-on-surface-variant sm:px-5">
            <span>{t('landing.queue.strip.count')}</span>
            <span dir="ltr" className="ms-auto text-tertiary">
              {`↑ ${fmtSpeed(totals.up)}`}
            </span>
          </div>
        </div>
      </div>
    </LandingSection>
  );
}
