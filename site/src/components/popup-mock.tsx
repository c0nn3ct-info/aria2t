// The extension's browser-action popup, rebuilt in the site's design system.
// It mirrors the real surface (extension/src/popup) part for part: the 380x600
// shell, an elevated hero over its live speed wave carrying the transfer status
// and the whole-queue action, the download list, and a footer whose primary is
// Add. Every control is drawn with the primitive the real one uses — the hero's
// FAB through `fabVariants`, a row's pause through `iconButtonVariants`, View
// all through `buttonVariants` — so the mock inherits the real popup's
// measurements instead of approximating them, and cannot drift from it by a
// tweak to one side only.
import { useEffect, useState } from 'react';
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
import { Button, buttonVariants } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { fabVariants } from '@/components/ui/fab';
import { iconButtonVariants } from '@/components/ui/icon-button';
import { mulberry32, stepSpeed } from '@/lib/mock-motion';
import { t } from '../i18n';
import { cn } from '@/lib/utils';

// Status → tone, from the extension's own status-word.tsx. Seeding is
// tertiary and distinct from active, which is the point: a torrent whose
// download finished is still working.
type Kind = 'active' | 'seeding' | 'paused';

const PILL: Record<Kind, string> = {
  active: 'bg-success-container text-success-on-container',
  seeding: 'bg-tertiary-container text-tertiary-on-container',
  paused: 'bg-warning-container text-warning-on-container',
};

// StatusWord's `sm` pill, verbatim (extension/src/popup/status-word.tsx).
const PILL_SIZE = 'inline-flex h-5 shrink-0 items-center rounded-pill px-2 text-[11px] font-medium';

const BAR: Record<Kind, string> = {
  active: 'bg-primary',
  seeding: 'bg-tertiary',
  paused: 'bg-warning',
};

interface RowSpec {
  name: string;
  icon: LucideIcon;
  kind: Kind;
  /** Bytes per second this row walks around; 0 for a row that is not moving. */
  target: number;
  pct: number;
  /** Shown instead of a speed when the row is not moving. */
  note?: string;
}

const ROWS: readonly RowSpec[] = [
  { name: 'ubuntu-24.04.2-desktop-amd64.iso', icon: Disc, kind: 'active', target: 13_600_000, pct: 63 },
  { name: 'Fedora-Workstation-Live-42.torrent', icon: Folder, kind: 'seeding', target: 4_000_000, pct: 100 },
  { name: '4K Wallpaper Megapack', icon: Magnet, kind: 'active', target: 5_100_000, pct: 23 },
  { name: 'Nature.Docs.S01.1080p.WEB-DL', icon: Folder, kind: 'active', target: 6_400_000, pct: 41 },
  { name: 'raspios-bookworm-arm64.img.xz', icon: FileArchive, kind: 'paused', target: 0, pct: 18, note: '680 MiB' },
];

const UP_TARGET = 4_000_000;
/**
 * Samples behind the hero. The extension's service worker keeps exactly this
 * many at 1 Hz (`SPEED_HISTORY`, extension/src/lib/traffic-buffer.ts), so the
 * wave here spans the same minute of history the real one draws.
 */
const WAVE_N = 60;

// The extension's own fmtSpeed (extension/src/lib/aria2/format.ts): one decimal
// from MiB up, none below.
export function fmtSpeed(bps: number): string {
  if (bps <= 0) return '0 B/s';
  if (bps >= 1048576) return `${(bps / 1048576).toFixed(1)} MiB/s`;
  if (bps >= 1024) return `${Math.round(bps / 1024)} KiB/s`;
  return `${Math.round(bps)} B/s`;
}

interface Live {
  speeds: number[];
  pcts: number[];
  up: number;
  /** Total throughput per second, oldest first — what the wave draws. */
  wave: number[];
}

// Seed by running the walk forward deterministically: the opening frame is both
// already settled and already holding a full minute of wave, the way a real
// popup opens onto a history the worker kept while it was closed.
function seedLive(): Live {
  const rnd = mulberry32(0x70757021); // 'pup!'
  const speeds = ROWS.map((r) => r.target);
  let up = UP_TARGET;
  const wave: number[] = [];
  for (let i = 0; i < WAVE_N; i++) {
    for (let j = 0; j < speeds.length; j++) {
      speeds[j] = ROWS[j].target > 0 ? stepSpeed(speeds[j], ROWS[j].target, rnd) : 0;
    }
    up = stepSpeed(up, UP_TARGET, rnd);
    wave.push(speeds.reduce((a, b) => a + b, 0) + up);
  }
  return { speeds, pcts: ROWS.map((r) => r.pct), up, wave };
}

function useLivePopup(): Live {
  const [live, setLive] = useState<Live>(seedLive);
  useEffect(() => {
    // Frozen for the prerender capture, exactly as the TUI mock is: the
    // prerendered DOM must equal the first client render.
    if (navigator.webdriver) return;
    const rnd = () => Math.random();
    const id = setInterval(() => {
      setLive((l) => {
        const speeds = l.speeds.map((s, i) => (ROWS[i].target > 0 ? stepSpeed(s, ROWS[i].target, rnd) : 0));
        const up = stepSpeed(l.up, UP_TARGET, rnd);
        return {
          speeds,
          // A seeding row is already whole, and a paused one is not moving; only
          // the downloading rows creep, and they wrap so the mock never parks at
          // 100% for the rest of the visit.
          pcts: l.pcts.map((p, i) =>
            ROWS[i].kind !== 'active' ? p : p >= 99 ? ROWS[i].pct * 0.3 : p + 0.4,
          ),
          up,
          // One sample in, the oldest out — the wave scrolls rather than grows.
          wave: [...l.wave.slice(1), speeds.reduce((a, b) => a + b, 0) + up],
        };
      });
    }, 1000);
    return () => clearInterval(id);
  }, []);
  return live;
}

export function PopupMock({ className }: { className?: string }) {
  const live = useLivePopup();
  const downloading = ROWS.filter((r) => r.kind === 'active').length;
  const down = live.speeds.reduce((a, b) => a + b, 0);

  return (
    <div
      className={cn(
        // The real surface is exactly 380x600 with these paddings
        // (extension/src/popup/app.tsx); everything below is measured off it.
        // The border and the radius are the site's own framing — in a browser
        // it is Chrome that draws the popup's edge.
        'flex h-[600px] w-[380px] select-none flex-col overflow-hidden rounded-xl border border-outline-variant bg-background text-on-surface shadow-e4',
        className,
      )}
    >
      {/* hero: eyebrow, what the queue is doing, the two speeds, and the
          whole-queue action as the primary (extension/src/popup/hero.tsx) */}
      <section className="shrink-0 px-4 pb-2 pt-4">
        <Card variant="elevated" padding="md" className="relative overflow-hidden">
          {/* the ambient backdrop, decorative and deliberately faint — the same
              intensity the extension's SpeedSpark carries */}
          <div className="pointer-events-none absolute inset-x-0 bottom-0 h-2/3 text-primary opacity-[0.15] rtl:-scale-x-100">
            <AmbientWave
              points={live.wave}
              max={Math.max(1, ...live.wave)}
              className="h-full w-full"
            />
          </div>
          <div className="relative flex min-w-0 items-center gap-4">
            <div className="min-w-0 flex-1 space-y-2">
              <div className="text-label-small uppercase tracking-[0.16em] text-on-surface-variant">
                {t('popup.eyebrow')}
              </div>
              <div className="truncate text-headline-small font-medium leading-tight tracking-tight">
                {t('popup.downloading').replace('{{count}}', String(downloading))}
              </div>
              <div className="flex items-center gap-3 text-label-medium tabular-nums">
                <span className="inline-flex items-center gap-1 text-primary">
                  <ArrowDown className="h-3 w-3" aria-hidden />
                  <span dir="ltr">{fmtSpeed(down)}</span>
                </span>
                <span className="inline-flex items-center gap-1 text-on-surface-variant">
                  <ArrowUp className="h-3 w-3" aria-hidden />
                  <span dir="ltr">{fmtSpeed(live.up)}</span>
                </span>
              </div>
            </div>
            {/* green = the action that stops what is running, per the hero's
                own colour rule */}
            <span aria-hidden className={fabVariants({ color: 'success', size: 'regular' })}>
              <Pause />
            </span>
          </div>
        </Card>
      </section>

      <div className="flex min-h-0 flex-1 flex-col px-2 pb-2 pt-3">
        <div className="flex items-center justify-between gap-2 px-2 pb-2">
          <span className="text-label-small uppercase text-on-surface-variant">
            {t('popup.downloads')}
          </span>
          <span className={buttonVariants({ variant: 'text', size: 'xs' })}>
            {t('popup.viewAll')}
            <ArrowRight className="rtl:-scale-x-100" aria-hidden />
          </span>
        </div>

        {/* the real list scrolls here; the mock holds the five rows that fit */}
        <div className="min-h-0 flex-1 overflow-hidden">
          <ul className="space-y-1">
            {ROWS.map((r, i) => {
              const Icon = r.icon;
              const pct = Math.min(100, live.pcts[i]);
              const figure = r.target > 0
                ? `${r.kind === 'seeding' ? '↑' : '↓'} ${fmtSpeed(live.speeds[i])}`
                : r.note;
              return (
                <li key={r.name} className="flex items-center gap-3 rounded-lg px-4 py-3">
                  <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-surface-container-high text-on-surface-variant">
                    <Icon className="h-5 w-5" aria-hidden />
                  </span>
                  <span className="flex min-w-0 flex-1 flex-col gap-1">
                    <span className="flex w-full items-center gap-3">
                      {/* a download name states its own direction, whatever the
                          UI's is */}
                      <span dir="auto" className="min-w-0 flex-1 truncate text-title-medium leading-tight">
                        {r.name}
                      </span>
                      <span className={cn(PILL_SIZE, PILL[r.kind])}>{t(`popup.status.${r.kind}`)}</span>
                    </span>
                    <span className="flex w-full items-center gap-2">
                      {/* fixed track, so every row's bar is the same length and
                          the fills can be read against each other */}
                      <span className="h-1.5 w-24 shrink-0 overflow-hidden rounded-pill bg-surface-container-high">
                        <span
                          className={cn('block h-full rounded-pill', BAR[r.kind])}
                          style={{ width: `${pct}%` }}
                        />
                      </span>
                      <span className="flex min-w-0 items-center gap-1.5 text-label-small text-on-surface-variant">
                        {/* one expression, not `{n}%`: a value beside a
                            literal is two text nodes, and the prerendered DOM
                            serializes them as one — which is a hydration
                            mismatch that throws the whole prerender away */}
                        <span dir="ltr" className="shrink-0">{`${Math.floor(pct)}%`}</span>
                        <span className="text-outline" aria-hidden>·</span>
                        <span dir="ltr" className="truncate">{figure}</span>
                      </span>
                    </span>
                  </span>
                  {/* the one action a row offers, on a tonal surface so it reads
                      as something to press rather than a glyph */}
                  <span
                    aria-hidden
                    className={iconButtonVariants({ variant: 'filled-tonal', size: 's' })}
                  >
                    {r.kind === 'paused' ? <Play /> : <Pause />}
                  </span>
                </li>
              );
            })}
          </ul>
        </div>
      </div>

      {/* Add at its content width, the panel taking the rest of the row. The
          real popup's is a <footer>; here it would be a second contentinfo
          landmark on a page that already has one, so it stays a plain row. */}
      <div className="mt-auto flex shrink-0 items-center gap-2 px-4 py-3">
        <Button variant="filled" size="s" className="pointer-events-none min-w-0" tabIndex={-1}>
          <Plus />
          <span className="truncate">{t('popup.add')}</span>
        </Button>
        <Button variant="filled-tonal" size="s" className="pointer-events-none min-w-0 flex-1" tabIndex={-1}>
          <span className="truncate">{t('popup.panel')}</span>
          <ExternalLink />
        </Button>
      </div>
    </div>
  );
}
