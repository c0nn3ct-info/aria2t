// The extension's browser-action popup, rebuilt in the site's design system.
// It mirrors the real surface (extension/src/popup): an elevated hero carrying
// the transfer status and the whole-queue action, the download list, and a
// footer whose primary is Add. Deliberately English, like the TUI mock beside
// it: this depicts the product the way a screenshot would, and a translated
// screenshot of an untranslated terminal would read as two different products.
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
import { Button } from '@/components/ui/button';
import { mulberry32, settle, stepSpeed } from '@/lib/mock-motion';
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
}

function seedLive(): Live {
  const rnd = mulberry32(0x70757021); // 'pup!'
  return {
    speeds: ROWS.map((r) => (r.target > 0 ? settle(r.target, r.target, rnd) : 0)),
    pcts: ROWS.map((r) => r.pct),
    up: settle(4_000_000, 4_000_000, rnd),
  };
}

function useLivePopup(): Live {
  const [live, setLive] = useState<Live>(seedLive);
  useEffect(() => {
    // Frozen for the prerender capture, exactly as the TUI mock is: the
    // prerendered DOM must equal the first client render.
    if (navigator.webdriver) return;
    const rnd = () => Math.random();
    const id = setInterval(() => {
      setLive((l) => ({
        speeds: l.speeds.map((s, i) => (ROWS[i].target > 0 ? stepSpeed(s, ROWS[i].target, rnd) : 0)),
        // A seeding row is already whole, and a paused one is not moving; only
        // the downloading rows creep, and they wrap so the mock never parks at
        // 100% for the rest of the visit.
        pcts: l.pcts.map((p, i) =>
          ROWS[i].kind !== 'active' ? p : p >= 99 ? ROWS[i].pct * 0.3 : p + 0.4,
        ),
        up: stepSpeed(l.up, 4_000_000, rnd),
      }));
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
        // The real surface is 380px wide with these exact paddings
        // (extension/src/popup/app.tsx); everything below is measured off it.
        'w-[380px] select-none overflow-hidden rounded-xl border border-outline-variant bg-background text-on-surface shadow-e4',
        className,
      )}
    >
      {/* hero: eyebrow, what the queue is doing, the two speeds, and the
          whole-queue action as the primary (extension/src/popup/hero.tsx) */}
      <div className="px-4 pb-2 pt-4">
        <div className="relative overflow-hidden rounded-xl bg-surface-container-low p-4 shadow-e1">
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
            <span
              aria-hidden
              className="grid h-14 w-14 shrink-0 place-items-center rounded-2xl bg-success text-success-foreground shadow-e1"
            >
              <Pause className="h-6 w-6" />
            </span>
          </div>
        </div>
      </div>

      <div className="flex flex-col px-2 pb-2 pt-3">
        <div className="flex items-center justify-between gap-2 px-2 pb-2">
          <span className="text-label-small uppercase text-on-surface-variant">
            {t('popup.downloads')}
          </span>
          <span className="inline-flex items-center gap-1 px-2 text-label-small font-medium text-primary">
            {t('popup.viewAll')}
            <ArrowRight className="h-4 w-4 rtl:-scale-x-100" aria-hidden />
          </span>
        </div>

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
                      <span dir="ltr" className="shrink-0">{Math.floor(pct)}%</span>
                      <span className="text-outline" aria-hidden>·</span>
                      <span dir="ltr" className="truncate">{figure}</span>
                    </span>
                  </span>
                </span>
                <span
                  aria-hidden
                  className="grid h-10 w-10 shrink-0 place-items-center rounded-full text-on-surface-variant"
                >
                  {r.kind === 'paused' ? <Play className="h-5 w-5" /> : <Pause className="h-5 w-5" />}
                </span>
              </li>
            );
          })}
        </ul>
      </div>

      {/* Add at its content width, the panel taking the rest of the row */}
      <div className="flex items-center gap-2 px-4 py-3">
        <Button variant="filled" size="s" className="pointer-events-none" tabIndex={-1}>
          <Plus />
          {t('popup.add')}
        </Button>
        <Button variant="filled-tonal" size="s" className="pointer-events-none min-w-0 flex-1" tabIndex={-1}>
          <span className="truncate">{t('popup.panel')}</span>
          <ExternalLink />
        </Button>
      </div>
    </div>
  );
}
