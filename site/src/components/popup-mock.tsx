// The extension's browser-action popup, rebuilt in the site's design system.
// It mirrors the real surface (extension/src/popup) part for part: the 380x600
// shell, an elevated hero over its live speed wave carrying the transfer status
// and the whole-queue action, the download list, and a footer whose primary is
// Add. Every control is drawn with the primitive the real one uses: the hero's
// FAB through `fabVariants`, a row's pause through `iconButtonVariants`, View
// all through `buttonVariants`, so the mock inherits the real popup's
// measurements instead of approximating them, and cannot drift from it by a
// tweak to one side only.
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  Disc,
  ExternalLink,
  FileArchive,
  Folder,
  Magnet,
  Pause,
  Play,
  Plus,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { AmbientWave } from '@/components/ambient-wave';
import { buttonVariants } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { fabVariants } from '@/components/ui/fab';
import { iconButtonVariants } from '@/components/ui/icon-button';
import {
  ITEMS,
  WAVE_MAX,
  fmtSize,
  toggleAll,
  toggleItem,
  useQueueScene,
  type IconKind,
  type Kind,
} from '@/lib/queue-scene';
import { getLocale, isRtl, t } from '../i18n';
import { cn } from '@/lib/utils';

// Status → tone, from the extension's own status-word.tsx. Seeding is
// tertiary and distinct from active, which is the point: a torrent whose
// download finished is still working.
// `pillClass`, verbatim (extension/src/popup/status-word.tsx). The queue store
// calls a finished download `done`, which is the word the terminal prints; the
// extension's own name for that state is `complete`.
const PILL: Record<Kind, string> = {
  active: 'bg-success-container text-success-on-container',
  seeding: 'bg-tertiary-container text-tertiary-on-container',
  paused: 'bg-warning-container text-warning-on-container',
  waiting: 'bg-surface-container-high text-on-surface-variant',
  error: 'bg-error-container text-error-on-container',
  done: 'bg-primary-container text-primary-on-container',
};

// StatusWord's `sm` pill, verbatim (extension/src/popup/status-word.tsx).
const PILL_SIZE =
  'inline-flex h-5 shrink-0 items-center rounded-pill px-2 text-label-small font-medium';

// `barClass`, verbatim (extension/src/popup/download-row.tsx).
const BAR: Record<Kind, string> = {
  active: 'bg-primary',
  seeding: 'bg-tertiary',
  paused: 'bg-warning',
  waiting: 'bg-surface-container-high',
  error: 'bg-error',
  done: 'bg-success',
};

const ICON: Record<IconKind, LucideIcon> = {
  disc: Disc,
  folder: Folder,
  magnet: Magnet,
  archive: FileArchive,
};

// The extension's own fmtSpeed (extension/src/lib/aria2/format.ts): one decimal
// from MiB up, none below.
export function fmtSpeed(bps: number): string {
  if (bps <= 0) return '-';
  if (bps >= 1048576) return `${(bps / 1048576).toFixed(1)} MiB/s`;
  if (bps >= 1024) return `${Math.round(bps / 1024)} KiB/s`;
  return `${Math.round(bps)} B/s`;
}

/**
 * What a row prints where a speed would go. A download that is not moving has
 * no speed to show, so it shows what is on disk instead - and one that has not
 * started has nothing on disk either, which is what aria2 reports as 0 B.
 *
 * Exported so its own test can name every case directly: the popup only ever
 * renders the four rows `VISIBLE` names, none of which is waiting, errored,
 * or finished at the moment it opens.
 */
export function figureFor(i: number, kind: Kind, speed: number): string {
  if (kind === 'active') return `↓ ${fmtSpeed(speed)}`;
  if (kind === 'seeding') return `↑ ${fmtSpeed(speed)}`;
  if (kind === 'waiting' || kind === 'error') return '0 B';
  return fmtSize(ITEMS[i].bytes);
}

/** Which of the shared queue's downloads the popup shows. Four, not the
 * first four: index 3 (Nature.Docs, also active) is skipped so the row set
 * reads active/seeding/active/paused instead of three actives and a paused. */
const VISIBLE: readonly number[] = [0, 1, 2, 4];

export function PopupMock({ className }: { className?: string }) {
  const scene = useQueueScene();
  const anyRunning = scene.live.some((l) => l.kind === 'active' || l.kind === 'seeding');

  return (
    <div
      // The popup states its own direction, the way the real one does:
      // `root.dir = isRtl(lng)` in extension/src/lib/theme.ts, from the
      // extension's UI language rather than from anything around it. Inheriting
      // instead left it left-to-right inside the browser frame, whose chrome is
      // pinned that way, while the same popup drawn on its own mirrored.
      dir={isRtl(getLocale()) ? 'rtl' : 'ltr'}
      className={cn(
        // The real surface is 380 wide with these paddings
        // (extension/src/popup/app.tsx); everything below is measured off it.
        // The border and the radius are the site's own framing - in a browser
        // it is Chrome that draws the popup's edge.
        //
        // 380x600 exactly, because that is the surface: Chrome caps a popup at
        // 600px tall and the real shell is fixed there (app.tsx). What does not
        // fit scrolls, in the list, the way it does in the extension - a taller
        // mock would be showing a window Chrome will not open.
        //
        // `box-content` is what makes those the surface rather than the box:
        // the border below is the site's own framing, and with the default
        // border-box it ate a pixel off each edge, leaving every measurement
        // inside 2px short of the real popup's.
        //
        // `max-w-full` is what happens when the caller has less room than 380:
        // the popup narrows instead of being clipped or pushing its container
        // open. The home hero's frame is under 380 on a phone.
        'pointer-events-auto box-content flex h-[600px] w-[380px] max-w-full select-none flex-col rounded-lg border border-outline-variant bg-background text-on-surface shadow-e3',
        className,
      )}
    >
      {/* hero: eyebrow, what the queue is doing, the two speeds, and the
          whole-queue action as the primary (extension/src/popup/hero.tsx) */}
      <section className="shrink-0 px-4 pb-2 pt-4">
        <Card variant="elevated" padding="md" className="relative overflow-hidden">
          {/* the ambient backdrop, decorative and faint, at the same
              intensity the extension's SpeedSpark carries */}
          <div className="pointer-events-none absolute inset-x-0 bottom-0 h-2/3 text-primary opacity-[0.15] rtl:-scale-x-100">
            <AmbientWave
              points={[...scene.wave]}
              max={WAVE_MAX}
              className="h-full w-full"
            />
          </div>
          <div className="relative flex min-w-0 items-center gap-4">
            <div className="min-w-0 flex-1 space-y-2">
              <div className="text-label-small uppercase tracking-[0.16em] text-on-surface-variant">
                {t('popup.eyebrow')}
              </div>
              <div className="truncate text-headline-small font-medium leading-tight tracking-tight">
                {t('popup.downloading').replace('{{count}}', String(scene.downloading))}
              </div>
              <div className="flex items-center gap-3 text-label-medium tabular-nums">
                <span className="inline-flex items-center gap-1 text-primary">
                  <ArrowDown className="h-3 w-3" aria-hidden />
                  <span dir="ltr">{fmtSpeed(scene.down)}</span>
                </span>
                <span className="inline-flex items-center gap-1 text-on-surface-variant">
                  <ArrowUp className="h-3 w-3" aria-hidden />
                  <span dir="ltr">{fmtSpeed(scene.up)}</span>
                </span>
              </div>
            </div>
            {/* green = the action that stops what is running, per the hero's
                own colour rule. Real: it pauses every active/seeding row and,
                clicked again, brings back only the ones it paused. */}
            <button
              type="button"
              onClick={toggleAll}
              aria-label={t(anyRunning ? 'popup.pause' : 'popup.resume')}
              className={fabVariants({ color: 'success', size: 'regular' })}
            >
              {anyRunning ? <Pause /> : <Play />}
            </button>
          </div>
        </Card>
      </section>

      <div className="flex min-h-0 flex-1 flex-col px-2 pb-2 pt-3">
        <div className="flex items-center justify-between gap-2 px-2 pb-2">
          <span className="text-label-small uppercase text-on-surface-variant">
            {t('popup.downloads')}
          </span>
          {/* A real, focusable button - this is a screen the mock has no
              other view of, so the click has nowhere to go. */}
          <button type="button" className={buttonVariants({ variant: 'text', size: 'xs' })}>
            {t('popup.viewAll')}
            <ArrowRight className="rtl:-scale-x-100" aria-hidden />
          </button>
        </div>

        {/* Scrolls, as the real one does (a `ScrollArea` in app.tsx) - only in
            principle now that four rows fit the 600px shell with nothing left
            over. `overscroll-contain` used to sit here on the theory that it
            hands scrolling back to the page at either end; `contain` does the
            opposite; it stops a scroll from *ever* reaching the page once
            this element is at its own limit - which, at zero overflow, it
            always is. The wheel test that caught it: hover the list, scroll,
            and the page under it does not move at all. Left at the default
            (`auto`), a wheel over an empty list scrolls the page normally,
            and the list still scrolls itself first if it ever holds enough
            rows to overflow again. */}
        <div className="min-h-0 flex-1 overflow-y-auto">
          <ul className="space-y-1">
            {VISIBLE.map((i) => {
              const item = ITEMS[i];
              const Icon = ICON[item.icon];
              const { kind, pct, speed } = scene.live[i];
              const figure = figureFor(i, kind, speed);
              const canToggle = kind === 'active' || kind === 'seeding' || kind === 'paused';
              return (
                <li key={item.name} className="flex items-center gap-3 rounded-lg px-4 py-3">
                  <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-surface-container-high text-on-surface-variant">
                    <Icon className="h-5 w-5" aria-hidden />
                  </span>
                  <span className="flex min-w-0 flex-1 flex-col gap-1">
                    <span className="flex w-full items-center gap-3">
                      {/* a download name states its own direction, whatever the
                          UI's is */}
                      <span dir="auto" className="min-w-0 flex-1 truncate text-title-medium leading-tight">
                        {item.name}
                      </span>
                      <span className={cn(PILL_SIZE, PILL[kind])}>{t(`popup.status.${kind}`)}</span>
                    </span>
                    <span className="flex w-full items-center gap-2">
                      {/* fixed track, so every row's bar is the same length and
                          the fills can be read against each other */}
                      <span className="h-1.5 w-24 shrink-0 overflow-hidden rounded-pill bg-surface-container-high">
                        <span
                          className={cn('block h-full rounded-pill', BAR[kind])}
                          style={{ width: `${pct}%` }}
                        />
                      </span>
                      <span className="flex min-w-0 items-center gap-1.5 text-label-small text-on-surface-variant">
                        {/* one expression, not `{n}%`: a value beside a
                            literal is two text nodes, and the prerendered DOM
                            serializes them as one, which is a hydration
                            mismatch that throws the whole prerender away */}
                        <span dir="ltr" className="shrink-0">{`${Math.floor(pct)}%`}</span>
                        <span className="text-outline" aria-hidden>·</span>
                        <span dir="ltr" className="truncate">{figure}</span>
                      </span>
                    </span>
                  </span>
                  {/* the one action a row offers, on a tonal surface so it reads
                      as something to press rather than a glyph. Real for a
                      moving or paused row - pause/resume that one download in
                      place. A waiting/errored/finished row has no action this
                      mock can perform, so it stays a plain glyph. */}
                  {canToggle ? (
                    <button
                      type="button"
                      onClick={() => toggleItem(i)}
                      aria-label={`${t(kind === 'paused' ? 'popup.resume' : 'popup.pause')}: ${item.name}`}
                      className={iconButtonVariants({ variant: 'filled-tonal', size: 's' })}
                    >
                      {kind === 'paused' ? <Play /> : <Pause />}
                    </button>
                  ) : (
                    <span
                      aria-hidden
                      className={iconButtonVariants({ variant: 'filled-tonal', size: 's' })}
                    >
                      <Play />
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      </div>

      {/* Add at its content width, the panel taking the rest of the row. The
          real popup's is a <footer>; here it would be a second contentinfo
          landmark on a page that already has one, so it stays a plain row.
          Both are real, focusable buttons - opening the add dialog or the
          full panel is a screen this mock has no other view of, so a click
          has nowhere to go. */}
      <div className="mt-auto flex shrink-0 items-center gap-2 px-4 py-3">
        <button
          type="button"
          className={cn(buttonVariants({ variant: 'filled', size: 's' }), 'min-w-0')}
        >
          <Plus />
          <span className="truncate">{t('popup.add')}</span>
        </button>
        <button
          type="button"
          className={cn(buttonVariants({ variant: 'filled-tonal', size: 's' }), 'min-w-0 flex-1')}
        >
          <span className="truncate">{t('popup.panel')}</span>
          <ExternalLink />
        </button>
      </div>
    </div>
  );
}
