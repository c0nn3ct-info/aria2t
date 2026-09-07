// "One queue for every download": the three ways something enters aria2, drawn
// side by side, and then the single queue they all land in. The first card is
// live, the three mirrors of one file each moving at its own speed, because
// that is the claim the card makes.
import type { ReactNode } from 'react';
import { Check, ExternalLink, FileText, Magnet } from 'lucide-react';
import { cn } from '@/lib/utils';
import { mirrorRows, mirrorTotal } from '@/lib/landing-motion';
import { fmtSize } from '@/lib/queue-scene';
import { FILES, TOTAL_BYTES, pickedBytes, togglePick, usePick } from '@/lib/pick-scene';
import { LandingSection } from './shell';
import { useTick } from './use-tick';
import { t } from '@/i18n';

/** The mono footer strip every input card ends with. */
function CardFooter({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        'mt-auto flex items-center gap-2 border-t border-outline-variant pt-3 font-mono text-[10px] text-on-surface-variant',
        className,
      )}
    >
      {children}
    </div>
  );
}

/**
 * The step between what goes in and what comes out. Mirrored on a
 * right-to-left page: the arrow points the way the row is read, and the glyph
 * itself does not turn around.
 */
function Arrow() {
  return (
    <span aria-hidden className="inline-block text-outline-variant rtl:-scale-x-100">
      →
    </span>
  );
}

/** Input 01: one link, several mirrors, one file. */
function MirrorsCard({ tick }: { tick: number }) {
  const rows = mirrorRows(tick);
  return (
    <div className="flex min-h-[220px] flex-col gap-3.5 rounded-md border border-outline-variant bg-surface-container-low p-[18px]">
      <div className="flex h-[30px] items-center gap-2 overflow-hidden rounded-pill bg-surface-container-high px-3 font-mono text-[10px] text-on-surface-variant">
        <ExternalLink className="h-3 w-3 shrink-0 text-primary" aria-hidden />
        <span dir="ltr" className="truncate">
          ubuntu-24.04.2-desktop-amd64.iso
        </span>
      </div>
      <div className="flex flex-1 flex-col justify-center gap-3">
        {rows.map((m) => (
          <div key={m.name} className="flex items-center gap-2.5">
            <span dir="ltr" className="w-[74px] shrink-0 font-mono text-[10px] text-on-surface-variant">
              {m.name}
            </span>
            <span className="h-2 flex-1 overflow-hidden rounded-pill bg-surface-container-high">
              <i
                className="block h-full rounded-pill bg-primary transition-[width] duration-[280ms] ease-linear"
                style={{ width: m.width }}
              />
            </span>
            <span dir="ltr" className="w-16 shrink-0 text-end font-mono text-[10px] tabular-nums text-primary">
              {m.speed}
            </span>
          </div>
        ))}
      </div>
      <CardFooter>
        <span>{t('landing.queue.in1.sources')}</span>
        <Arrow />
        <span className="text-on-surface">{t('landing.queue.in1.result')}</span>
        <span dir="ltr" className="ms-auto text-primary">{`↓ ${mirrorTotal(rows)} MiB/s`}</span>
      </CardFooter>
    </div>
  );
}

/**
 * Input 02: a torrent, its file tree, and what the selection costs.
 *
 * The boxes are the same three the file-selection band further down draws, and
 * the same selection: this card and that picker are one torrent, so ticking a
 * file here is the same act as ticking it there. Everything in the footer is
 * computed, so the card cannot claim a total its own boxes contradict.
 */
function TorrentCard() {
  const picked = usePick();
  const count = picked.filter(Boolean).length;
  const bytes = pickedBytes(picked);
  return (
    <div className="flex min-h-[220px] flex-col gap-3 rounded-md border border-outline-variant bg-surface-container-low p-[18px]">
      <div className="flex items-center gap-2.5 border-b border-outline-variant pb-3">
        <Magnet className="h-4 w-4 shrink-0 text-tertiary" aria-hidden />
        <span className="min-w-0 flex-1 truncate text-xs font-semibold">{t('landing.demo.torrent')}</span>
        <span className="font-mono text-[10px] text-on-surface-variant">
          {t('landing.queue.in2.files').replace('{{count}}', String(FILES.length))}
        </span>
      </div>
      {FILES.map((f, i) => {
        const on = picked[i];
        return (
          <button
            key={f.name}
            type="button"
            role="checkbox"
            aria-checked={on}
            onClick={() => togglePick(i)}
            className="-mx-1 flex items-center gap-3 rounded-xs px-1 py-0.5 text-start transition-colors hover:bg-surface-container-high focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-primary"
          >
            {on ? (
              <span className="grid h-5 w-5 shrink-0 place-items-center rounded-[6px] bg-primary text-primary-foreground">
                <Check className="h-3.5 w-3.5" strokeWidth={2.6} aria-hidden />
              </span>
            ) : (
              <span aria-hidden className="h-5 w-5 shrink-0 rounded-[6px] border-[1.5px] border-outline" />
            )}
            <span
              dir="ltr"
              className={cn(
                'min-w-0 flex-1 truncate text-xs',
                !on && 'text-on-surface-variant line-through',
              )}
            >
              {f.name}
            </span>
            <span
              dir="ltr"
              className={cn(
                'font-mono text-[10px]',
                on ? 'text-on-surface-variant' : 'text-on-surface-variant/80',
              )}
            >
              {fmtSize(f.bytes)}
            </span>
          </button>
        );
      })}
      <CardFooter>
        <span className="whitespace-nowrap">
          {t('landing.queue.in2.selected')
            .replace('{{n}}', String(count))
            .replace('{{total}}', String(FILES.length))}
        </span>
        <Arrow />
        <span dir="ltr" className="whitespace-nowrap text-on-surface">
          {fmtSize(bytes)}
        </span>
        {/* Only when something is being skipped: "-0 B" is not a
            saving, it is a sentence with nothing in it. */}
        {bytes < TOTAL_BYTES && (
          <span dir="ltr" className="ms-auto whitespace-nowrap text-success-text">
            {`\u2212${fmtSize(TOTAL_BYTES - bytes)}`}
          </span>
        )}
      </CardFooter>
    </div>
  );
}

const INPUT_LINES: ReadonlyArray<[text: string, link: boolean]> = [
  ['https://mirror.one/ubuntu.iso', true],
  ['  dir=/downloads/iso', false],
  ['https://mirror.two/fedora.iso', true],
  ['  max-connection-per-server=8', false],
];

/** Input 03: aria2's own batch format, options and all. */
function InputFileCard() {
  return (
    <div className="flex min-h-[220px] flex-col overflow-hidden rounded-md border border-outline-variant bg-surface-container-low">
      <div className="flex items-center gap-2.5 border-b border-outline-variant bg-surface-container px-4 py-3">
        <FileText className="h-4 w-4 shrink-0 text-primary" aria-hidden />
        <span dir="ltr" className="flex-1 font-mono text-[11px]">
          downloads.input
        </span>
        <span className="font-mono text-[10px] text-on-surface-variant">
          {t('landing.queue.in3.entries')}
        </span>
      </div>
      <div dir="ltr" className="flex flex-1 flex-col gap-1.5 px-4 py-3 font-mono text-[10px] text-on-surface-variant">
        {INPUT_LINES.map(([text, link], i) => (
          <div key={text} className="flex gap-2.5">
            <span className="w-3 shrink-0 text-end text-on-surface-variant/80">{i + 1}</span>
            <span className={cn('min-w-0 truncate whitespace-pre', link && 'text-primary')}>{text}</span>
          </div>
        ))}
      </div>
      <CardFooter className="mt-0 px-4 pb-3">
        <span>{t('landing.queue.in3.from')}</span>
        <Arrow />
        <span className="text-on-surface">{t('landing.queue.in3.to')}</span>
        <span className="ms-auto text-tertiary">{t('landing.queue.in3.note')}</span>
      </CardFooter>
    </div>
  );
}

interface Input {
  title: string;
  body: string;
  card: ReactNode;
}

interface QueueItem {
  status: 'active' | 'seeding' | 'paused';
  name: string;
  pct: number;
  /**
   * The technical half of the row's caption: a percentage, a size, a time, a
   * ratio. It stays left-to-right, because on a right-to-left page the bidi
   * algorithm splits "18% · 680 MiB" into three runs and reorders them into
   * "MiB 680 · 18%".
   */
  note: string;
  /** A translated word after it, which follows the page's own direction. */
  extra?: string;
}

const STATUS_TEXT: Record<QueueItem['status'], string> = {
  active: 'text-success-text',
  seeding: 'text-tertiary',
  paused: 'text-on-surface-variant',
};

const STATUS_BAR: Record<QueueItem['status'], string> = {
  active: 'bg-primary',
  seeding: 'bg-tertiary',
  // Deliberately the faintest of the three: a paused download is the one thing
  // in the strip that is not moving.
  paused: 'bg-surface-container-highest',
};

export function QueueSection() {
  const tick = useTick();
  // The status words are aria2's own, printed in English by the daemon and by
  // the terminal client; the list mock beside them shows the same.
  const queue: readonly QueueItem[] = [
    { status: 'active', name: 'ubuntu-24.04.2.iso', pct: 63, note: '63% · 4m 12s' },
    { status: 'seeding', name: 'fedora-42.torrent', pct: 100, note: t('landing.pieces.ratio') },
    { status: 'active', name: '4K Wallpaper Megapack', pct: 23, note: '23% · magnet' },
    {
      status: 'active',
      name: 'Nature.Docs.S01',
      pct: 41,
      note: '41%',
      extra: t('landing.queue.strip.peers'),
    },
    { status: 'paused', name: 'raspios-arm64.img.xz', pct: 18, note: '18% · 680 MiB' },
  ];

  const inputs: readonly Input[] = [
    {
      title: t('landing.queue.in1.title'),
      body: t('landing.queue.in1.body'),
      card: <MirrorsCard tick={tick} />,
    },
    { title: t('landing.queue.in2.title'), body: t('landing.queue.in2.body'), card: <TorrentCard /> },
    { title: t('landing.queue.in3.title'), body: t('landing.queue.in3.body'), card: <InputFileCard /> },
  ];

  return (
    <LandingSection id="sources">
      <div className="grid items-start gap-8 lg:grid-cols-2 lg:gap-14">
        <h2 data-enter className="text-balance text-[clamp(28px,4.6vw,46px)] font-semibold leading-[1.08] tracking-[-0.035em]">
          {t('landing.queue.h2')}
        </h2>
        <p data-enter className="text-pretty text-[17px] leading-[1.6] text-on-surface-variant sm:text-[19px]">
          {t('landing.queue.body')}
        </p>
      </div>

      <ol data-enter-stagger="wipe" className="mt-12 grid gap-x-9 gap-y-12 md:grid-cols-3">
        {inputs.map((input, i) => (
          <li
            key={input.title}
            className={cn('flex flex-col', i > 0 && 'md:border-s md:border-outline-variant md:ps-9')}
          >
            <div>{input.card}</div>
            <h3 className="mt-6 text-[19px] font-semibold tracking-[-0.02em]">{input.title}</h3>
            <p className="mt-2.5 text-body-medium leading-[1.75] text-on-surface-variant">{input.body}</p>
          </li>
        ))}
      </ol>

      <div data-enter className="mt-12 overflow-hidden rounded-lg border border-outline-variant bg-surface-container-low">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-b border-outline-variant px-6 py-4">
          <span className="text-title-small font-semibold">{t('landing.queue.strip.title')}</span>
          <span className="ms-auto font-mono text-[11px] text-on-surface-variant">
            {t('landing.queue.strip.count')}
            <span dir="ltr">{' · '}</span>
            <span dir="ltr" className="text-primary">
              ↓ 24.0 MiB/s
            </span>
            <span dir="ltr">{' · '}</span>
            <span dir="ltr" className="text-tertiary">
              ↑ 3.8 MiB/s
            </span>
          </span>
        </div>
        <ul className="grid grid-cols-2 gap-px bg-outline-variant lg:grid-cols-5">
          {queue.map((d) => (
            <li
              key={d.name}
              // Five cards into two columns leaves the last cell empty, and the
              // grid's gap colour shows through it as a stray block. The last
              // card takes the whole row instead.
              className="flex flex-col gap-2.5 bg-surface-container-low p-5 last:col-span-2 lg:last:col-span-1"
            >
              <span dir="ltr" className={cn('font-mono text-[10px]', STATUS_TEXT[d.status])}>
                {d.status}
              </span>
              <span
                dir="ltr"
                className={cn(
                  'truncate text-[13px] font-medium',
                  d.status === 'paused' && 'text-on-surface-variant',
                )}
              >
                {d.name}
              </span>
              <div className="h-[5px] overflow-hidden rounded-pill bg-surface-container-high">
                <i
                  className={cn('block h-full rounded-pill', STATUS_BAR[d.status])}
                  style={{ width: `${d.pct}%` }}
                />
              </div>
              <span className="flex flex-wrap items-baseline gap-1.5 font-mono text-[10px] text-on-surface-variant">
                <span dir="ltr">{d.note}</span>
                {d.extra && (
                  <>
                    <span aria-hidden>·</span>
                    <span>{d.extra}</span>
                  </>
                )}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </LandingSection>
  );
}
