// "One queue for every download", drawn the other way: the split spine.
//
// `queue-section.tsx` puts three input cards over a five-cell strip. This band
// keeps the page's two-column meter instead - the claim and its three routes
// read down the left column, and the queue takes the wide track beside them as
// one list whose every row carries the route it arrived by. The sentence is the
// same; what changes is that a list drawn as a list has an order, which five
// cells across a row do not.
//
// It is built from the page's own furniture: `SectionHeading` and `PointList`
// for the column of copy, and the material `MockCard` is made of for the mock
// beside it - bordered, lifted, opening on a header strip. The route marks are
// the three icons and the three tones the other draw already gives its input
// cards, so a row and the point that explains it wear the same mark.
//
// It is the draw the page ships. `queue-section.tsx` is the one it replaced and
// stays in the repo as what the band used to be; nothing imports it from the
// page. That comparison is also why nothing here adds an i18n key and the five
// downloads are declared locally - the two draws had to say the same words for
// the choice between them to be about composition, so every string below is one
// the other draw already shipped.
import { ExternalLink, FileText, Magnet, type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { LandingSection, PointList, SectionHeading, type Point } from './shell';
import { t } from '@/i18n';

/** How a download got into the queue. aria2's own words, like the status column's. */
type Route = 'link' | 'torrent' | 'magnet' | 'input';

/**
 * A route's mark. Both BitTorrent routes take the magnet - the icon the other
 * draw gives that card, and the one the rail's middle point carries: a torrent
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

interface Row {
  route: Route;
  /** The word the strip prints, in aria2's English, as the other draw does. */
  status: 'active' | 'seeding' | 'paused';
  name: string;
  pct: number;
  size: string;
  /** Down while active, up while seeding; a paused download reports neither. */
  speed: string;
  /** aria2's ETA, or what stands in for it once there is none: a ratio, peers. */
  eta: string;
  /**
   * Whether that ETA is a figure rather than a translated phrase. A figure
   * stays left-to-right - the bidi algorithm splits "4m 12s" and reorders it on
   * an Arabic page - while a phrase follows the page.
   */
  etaFigure: boolean;
}

const STATUS_TEXT: Record<Row['status'], string> = {
  active: 'text-success-text',
  seeding: 'text-tertiary',
  paused: 'text-on-surface-variant',
};

const STATUS_BAR: Record<Row['status'], string> = {
  active: 'bg-primary',
  seeding: 'bg-tertiary',
  // The faintest of the three: a paused download is the one row not moving.
  paused: 'bg-surface-container-highest',
};

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
 * list items, not one table - so a `1fr` or `auto` column is measured per row
 * and the columns stop lining up between them: the ETA cell alone ranges from
 * "4m 12s" to "ratio 1.84" to a translated phrase. Only the name track flexes,
 * because it is the one cell allowed to truncate.
 */
const GRID =
  'grid-cols-[minmax(0,1fr)_auto] md:grid-cols-[minmax(0,1fr)_72px_110px_76px_84px_96px]';

export function QueueB2Section() {
  const rows: readonly Row[] = [
    {
      route: 'link',
      status: 'active',
      name: 'ubuntu-24.04.2.iso',
      pct: 63,
      size: '6.1 GiB',
      speed: '↓ 20.2 M',
      eta: '4m 12s',
      etaFigure: true,
    },
    {
      route: 'torrent',
      status: 'seeding',
      name: 'fedora-42.torrent',
      pct: 100,
      size: '2.1 GiB',
      speed: '↑ 3.8 M',
      eta: t('landing.pieces.ratio'),
      etaFigure: false,
    },
    {
      route: 'magnet',
      status: 'active',
      name: '4K Wallpaper Megapack',
      pct: 23,
      size: '18 GiB',
      speed: '↓ 2.4 M',
      eta: '1h 22m',
      etaFigure: true,
    },
    {
      route: 'torrent',
      status: 'active',
      name: 'Nature.Docs.S01',
      pct: 41,
      size: '31 GiB',
      speed: '↓ 1.4 M',
      eta: t('landing.queue.strip.peers'),
      etaFigure: false,
    },
    {
      route: 'input',
      status: 'paused',
      name: 'raspios-arm64.img.xz',
      pct: 18,
      size: '680 MiB',
      // Nothing is being transferred, and "0" would be a measurement.
      speed: '-',
      eta: '-',
      etaFigure: true,
    },
  ];

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
              ↓ 24.0 MiB/s
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
              {rows.map((d) => {
                const Icon = ROUTE_ICON[d.route];
                return (
                  <li
                    key={d.name}
                    className={cn(
                      'grid items-center gap-x-3 gap-y-2 border-t border-outline-variant py-3',
                      GRID,
                    )}
                  >
                    <span className="flex min-w-0 items-center gap-2.5">
                      <Icon className={cn('h-4 w-4 shrink-0', ROUTE_TINT[d.route])} aria-hidden />
                      {/* The mark is what is seen; the word is for a reader that
                          cannot see which mark it is. */}
                      <span className="sr-only">{d.route}</span>
                      <span
                        dir="ltr"
                        className={cn(
                          'truncate text-body-medium',
                          d.status === 'paused' && 'text-on-surface-variant',
                        )}
                      >
                        {d.name}
                      </span>
                    </span>
                    <span dir="ltr" className={cn('font-mono text-[10.5px]', STATUS_TEXT[d.status])}>
                      {d.status}
                    </span>
                    {/* Full width under the name on a phone, its own column at `md`. */}
                    <span className="col-span-2 h-[5px] overflow-hidden rounded-pill bg-surface-container-high md:col-span-1">
                      <i
                        className={cn('block h-full rounded-pill', STATUS_BAR[d.status])}
                        style={{ width: `${d.pct}%` }}
                      />
                    </span>
                    {/* One line of figures on a phone; three cells at `md`, where
                        `contents` dissolves this wrapper into the row's grid. */}
                    <span className="col-span-2 flex justify-between gap-3 font-mono text-[11px] tabular-nums text-on-surface-variant md:contents">
                      <span dir="ltr" className="md:text-end">
                        {d.size}
                      </span>
                      <span
                        dir="ltr"
                        className={cn(
                          'md:text-end',
                          d.status === 'paused' ? undefined : 'text-primary',
                        )}
                      >
                        {d.speed}
                      </span>
                      <span
                        dir={d.etaFigure ? 'ltr' : undefined}
                        className="whitespace-nowrap md:text-end"
                      >
                        {d.eta}
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
              ↑ 3.8 MiB/s
            </span>
          </div>
        </div>
      </div>
    </LandingSection>
  );
}
