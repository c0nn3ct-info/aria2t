import { useEffect, useState } from 'react';
import { cn } from '@/lib/utils';

// Tokyo Night — the app's own palette (internal/ui/theme.go), verbatim.
const C = {
  accent: '#7aa2f7',
  bg: '#16161e',
  fg: '#c0caf5',
  bright: '#e6e9f2',
  dim: '#565f89',
  faint: '#3b4261',
  sel: '#2a2f45',
  green: '#9ece6a',
  yellow: '#e0af68',
  cyan: '#7dcfee',
  magenta: '#bb9af7',
} as const;

// Deterministic PRNG so the prerendered frame matches client hydration (no
// flash / mismatch) — the live tick switches to Math.random after mount.
function mulberry32(seed: number): () => number {
  return () => {
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

// One step of a speed random-walk: AR(1) low-pass around a target rate.
function stepSpeed(prev: number, target: number, rnd: () => number): number {
  const next = prev * 0.8 + target * 0.2 + (rnd() - 0.5) * target * 0.3;
  return Math.max(target * 0.4, Math.min(target * 1.6, next));
}

function fmtSpeed(bps: number): string {
  if (bps >= 1048576) return `${(bps / 1048576).toFixed(1)} MiB/s`;
  return `${Math.round(bps / 1024)} KiB/s`;
}

function fmtEta(remainBytes: number, bps: number): string {
  if (bps <= 0) return '-';
  const s = Math.round(remainBytes / bps);
  if (s >= 3600) return `${Math.floor(s / 3600)}h${Math.floor((s % 3600) / 60)}m`;
  if (s >= 60) return `${Math.floor(s / 60)}m${s % 60}s`;
  return `${s}s`;
}

type RowStatus = 'active' | 'seeding' | 'paused' | 'done';

interface StaticRow {
  name: string;
  size: string;
  bytes: number;
  status: RowStatus;
  pct: number; // fixed rows only
}

const ROWS: readonly StaticRow[] = [
  { name: 'debian-13.1.0-amd64-netinst.iso', size: '680 MiB', bytes: 680 * 1048576, status: 'active', pct: 62.4 },
  { name: 'ubuntu-24.04.2-live-server-arm64.iso', size: '3.1 GiB', bytes: 3.1 * 1073741824, status: 'seeding', pct: 100 },
  { name: 'archlinux-2026.07.01-x86_64.iso', size: '1.2 GiB', bytes: 1.2 * 1073741824, status: 'active', pct: 18.2 },
  { name: 'linuxmint-22.1-cinnamon-64bit.iso', size: '2.8 GiB', bytes: 2.8 * 1073741824, status: 'paused', pct: 35.7 },
  { name: 'tails-amd64-6.15.img', size: '1.4 GiB', bytes: 1.4 * 1073741824, status: 'done', pct: 100 },
];

interface Live {
  d0: number; // row 0 download speed
  d2: number; // row 2 download speed
  up: number; // seeding row upload speed
  p0: number; // row 0 percent
  p2: number; // row 2 percent
}

// Seed by running the walk forward deterministically, so the opening frame is
// already settled (same distribution as the live tick).
function seedLive(): Live {
  const rnd = mulberry32(0x61726961); // 'aria'
  let d0 = 5_400_000;
  let d2 = 2_100_000;
  let up = 900_000;
  for (let i = 0; i < 16; i++) {
    d0 = stepSpeed(d0, 5_400_000, rnd);
    d2 = stepSpeed(d2, 2_100_000, rnd);
    up = stepSpeed(up, 900_000, rnd);
  }
  return { d0, d2, up, p0: ROWS[0].pct, p2: ROWS[2].pct };
}

function useLiveList(): Live {
  const [live, setLive] = useState<Live>(seedLive);
  useEffect(() => {
    // Keep the prerender frozen at the deterministic seed: the prerendered DOM
    // must equal the first client render or hydration mismatches.
    if (navigator.webdriver) return;
    const rnd = () => Math.random();
    const id = setInterval(() => {
      setLive((l) => {
        const d0 = stepSpeed(l.d0, 5_400_000, rnd);
        const d2 = stepSpeed(l.d2, 2_100_000, rnd);
        const up = stepSpeed(l.up, 900_000, rnd);
        // Visual pace, slower than real time so the bars creep rather than race;
        // wrap near the end so the demo never freezes at 100%.
        const p0 = l.p0 >= 99 ? 12 : l.p0 + (d0 / ROWS[0].bytes) * 100 * 0.18;
        const p2 = l.p2 >= 99 ? 4 : l.p2 + (d2 / ROWS[2].bytes) * 100 * 0.18;
        return { d0, d2, up, p0, p2 };
      });
    }, 1000);
    return () => clearInterval(id);
  }, []);
  return live;
}

const STATUS_COLOR: Record<RowStatus, string> = {
  active: C.green,
  seeding: C.magenta,
  paused: C.yellow,
  done: C.green,
};

function Bar({ pct, color }: { pct: number; color: string }) {
  return (
    <span className="flex min-w-0 flex-1 items-center gap-1.5">
      <span className="h-[5px] min-w-0 flex-1 overflow-hidden rounded-full" style={{ backgroundColor: C.faint }}>
        <span
          className="block h-full rounded-full transition-[width] duration-long ease-emph"
          style={{ width: `${Math.min(100, pct)}%`, backgroundColor: color }}
        />
      </span>
      <span className="w-8 shrink-0 text-right" style={{ color: C.dim }}>
        {`${Math.floor(pct)}%`}
      </span>
    </span>
  );
}

interface RowView {
  row: StaticRow;
  pct: number;
  speed: string;
  speedColor: string;
  eta: string;
  selected?: boolean;
}

function Row({ row, pct, speed, speedColor, eta, selected }: RowView) {
  const barColor = pct >= 100 ? (row.status === 'seeding' ? C.magenta : C.green) : C.accent;
  return (
    <div
      className="flex items-center gap-2 rounded-[3px] px-1.5 py-[3px]"
      style={selected ? { backgroundColor: C.sel } : undefined}
    >
      <span className="w-2 shrink-0" style={{ color: C.accent }}>
        {selected ? '▸' : ''}
      </span>
      <span className="min-w-0 flex-[2.4] truncate" style={{ color: selected ? C.bright : C.fg }}>
        {row.name}
      </span>
      <span className="w-14 shrink-0" style={{ color: STATUS_COLOR[row.status] }}>
        {row.status}
      </span>
      <Bar pct={pct} color={barColor} />
      <span className="hidden w-16 shrink-0 text-right sm:block" style={{ color: C.dim }}>
        {row.size}
      </span>
      <span className="w-[4.75rem] shrink-0 whitespace-nowrap text-right sm:w-20" style={{ color: speedColor }}>
        {speed}
      </span>
      <span className="hidden w-12 shrink-0 text-right sm:block" style={{ color: C.dim }}>
        {eta}
      </span>
    </div>
  );
}

const KEYS: ReadonlyArray<[string, string]> = [
  ['a', 'add'],
  ['space', 'pause'],
  ['d', 'remove'],
  ['l', 'limit'],
  ['/', 'filter'],
  ['?', 'help'],
];

export function ListMock({ className }: { className?: string }) {
  const live = useLiveList();
  const totalDown = live.d0 + live.d2;

  const views: RowView[] = [
    {
      row: ROWS[0],
      pct: live.p0,
      speed: fmtSpeed(live.d0),
      speedColor: C.cyan,
      eta: fmtEta(ROWS[0].bytes * (1 - live.p0 / 100), live.d0),
      selected: true,
    },
    { row: ROWS[1], pct: 100, speed: `↑ ${fmtSpeed(live.up)}`, speedColor: C.magenta, eta: '-' },
    {
      row: ROWS[2],
      pct: live.p2,
      speed: fmtSpeed(live.d2),
      speedColor: C.cyan,
      eta: fmtEta(ROWS[2].bytes * (1 - live.p2 / 100), live.d2),
    },
    { row: ROWS[3], pct: ROWS[3].pct, speed: '-', speedColor: C.dim, eta: '-' },
    { row: ROWS[4], pct: 100, speed: '-', speedColor: C.dim, eta: '-' },
  ];

  return (
    <div dir="ltr" className={cn('select-none font-mono text-[11px] leading-[1.35]', className)}>
      {/* brand + tabs + global speeds */}
      <div className="flex items-center gap-2 pb-2">
        <span className="font-bold" style={{ color: C.accent }}>
          aria2t
        </span>
        <span
          className="rounded-[3px] px-1.5 font-bold"
          style={{ backgroundColor: C.accent, color: C.bg }}
        >
          1 All
        </span>
        <span style={{ color: C.dim }}>2 Active</span>
        <span className="hidden sm:inline" style={{ color: C.dim }}>
          3 Waiting
        </span>
        <span className="hidden sm:inline" style={{ color: C.dim }}>
          4 Stopped
        </span>
        <span className="ms-auto" style={{ color: C.cyan }}>
          {`▼ ${fmtSpeed(totalDown)}`}
        </span>
        <span style={{ color: C.magenta }}>{`▲ ${fmtSpeed(live.up)}`}</span>
      </div>

      {/* column header */}
      <div className="flex items-center gap-2 px-1.5 pb-1" style={{ color: C.dim }}>
        <span className="w-2 shrink-0" />
        <span className="min-w-0 flex-[2.4]">NAME</span>
        <span className="w-14 shrink-0">STATUS</span>
        <span className="min-w-0 flex-1">PROGRESS</span>
        <span className="hidden w-16 shrink-0 text-right sm:block">SIZE</span>
        <span className="w-[4.75rem] shrink-0 whitespace-nowrap text-right sm:w-20">SPEED</span>
        <span className="hidden w-12 shrink-0 text-right sm:block">ETA</span>
      </div>

      <div className="space-y-px">
        {views.map((v) => (
          <Row key={v.row.name} {...v} />
        ))}
      </div>

      {/* status line + key bar */}
      <div className="flex items-center gap-1 pt-2" style={{ color: C.dim }}>
        <span style={{ color: C.green }}>✓</span>
        <span>connected · built-in daemon</span>
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 pt-1.5">
        {KEYS.map(([k, label]) => (
          <span key={k} className="whitespace-nowrap">
            <span className="font-bold" style={{ color: C.accent }}>
              {k}
            </span>{' '}
            <span style={{ color: C.dim }}>{label}</span>
          </span>
        ))}
      </div>
    </div>
  );
}
