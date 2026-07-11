import { useEffect, useState } from 'react';
import { cn } from '@/lib/utils';

// Tokyo Night — the app's own palette (tui/internal/ui/theme.go), verbatim.
const C = {
  accent: '#7aa2f7',
  bg: '#16161e',
  fg: '#c0caf5',
  bright: '#e6e9f2',
  dim: '#565f89',
  faint: '#3b4261',
  border: '#3b4261',
  sel: '#2a2f45',
  green: '#9ece6a',
  yellow: '#e0af68',
  red: '#f7768e',
  cyan: '#7dcfee',
  magenta: '#bb9af7',
} as const;

// Character grid mirroring the real list screen (tui/internal/ui/list.go):
// marker(2)+NAME+' '+STATUS(9)+' '+bar+pct(5)+SIZE+SPEED+CONN+ETA, all
// space-padded like the TUI does with pad/lpad. 78 columns per row line.
const NAME_W = 17;
const STATUS_W = 9;
const BAR_W = 12;
const SIZE_W = 8;
const SPEED_W = 10;
const CONN_W = 5;
const ETA_W = 8;
const COLS = 80; // header/tabs/key-bar lines; panel rows are 78 + 1ch padding

function pad(s: string, w: number): string {
  return s.length >= w ? s.slice(0, w) : s + ' '.repeat(w - s.length);
}

function lpad(s: string, w: number): string {
  return s.length >= w ? s.slice(0, w) : ' '.repeat(w - s.length) + s;
}

function trunc(s: string, w: number): string {
  return s.length <= w ? pad(s, w) : s.slice(0, w - 1) + '…';
}

// The TUI's Bar(): '━'-filled with a '╸' cap over a '─' track.
function bar(frac: number): { filled: string; empty: string } {
  const cells = Math.round(Math.min(1, Math.max(0, frac)) * BAR_W);
  if (cells <= 0) return { filled: '', empty: '─'.repeat(BAR_W) };
  if (cells >= BAR_W) return { filled: '━'.repeat(BAR_W), empty: '' };
  return { filled: '━'.repeat(cells - 1) + '╸', empty: '─'.repeat(BAR_W - cells) };
}

function fmtSpeed(bps: number): string {
  if (bps <= 0) return '-';
  if (bps >= 1048576) return `${(bps / 1048576).toFixed(1)} MiB/s`;
  return `${Math.round(bps / 1024)} KiB/s`;
}

function fmtEta(remainBytes: number, bps: number): string {
  if (bps <= 0) return '-';
  const s = Math.round(remainBytes / bps);
  if (s >= 3600) return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
  if (s >= 60) return `${Math.floor(s / 60)}m ${s % 60}s`;
  return `${s}s`;
}

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

// The five downloading rows: target speed, size, starting percent.
const LIVE_TARGETS = [4_000_000, 2_500_000, 6_100_000, 1_500_000, 930_000] as const;
const LIVE_BYTES = [
  5.4 * 1073741824,
  2.3 * 1073741824,
  1.2 * 1073741824,
  4.1 * 1073741824,
  1.4 * 1073741824,
] as const;
const LIVE_P0 = [35.2, 68.4, 19.1, 55.6, 81.3] as const;

interface Live {
  speeds: number[];
  pcts: number[];
  up: number;
}

// Seed by running the walk forward deterministically, so the opening frame is
// already settled (same distribution as the live tick).
function seedLive(): Live {
  const rnd = mulberry32(0x61726961); // 'aria'
  const speeds = [...LIVE_TARGETS] as number[];
  let up = 900_000;
  for (let i = 0; i < 16; i++) {
    for (let j = 0; j < speeds.length; j++) speeds[j] = stepSpeed(speeds[j], LIVE_TARGETS[j], rnd);
    up = stepSpeed(up, 900_000, rnd);
  }
  return { speeds, pcts: [...LIVE_P0], up };
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
        const speeds = l.speeds.map((s, j) => stepSpeed(s, LIVE_TARGETS[j], rnd));
        // Visual pace, slower than real time so the bars creep rather than race;
        // wrap near the end so the demo never freezes at 100%.
        const pcts = l.pcts.map((p, j) =>
          p >= 99 ? LIVE_P0[j] * 0.2 : p + (speeds[j] / LIVE_BYTES[j]) * 100 * 0.35,
        );
        return { speeds, pcts, up: stepSpeed(l.up, 900_000, rnd) };
      });
    }, 1000);
    return () => clearInterval(id);
  }, []);
  return live;
}

type Status = 'active' | 'seeding' | 'waiting' | 'paused' | 'error' | 'done';

const STATUS_COLOR: Record<Status, string> = {
  active: C.green,
  seeding: C.magenta,
  waiting: C.yellow,
  paused: C.yellow,
  error: C.red,
  done: C.green,
};

interface RowData {
  name: string;
  status: Status;
  pct: number;
  size: string;
  speed: string;
  conn: string;
  eta: string;
  selected?: boolean;
}

function Row({ name, status, pct, size, speed, conn, eta, selected }: RowData) {
  const nameStyle = { color: selected ? C.bright : C.fg };
  let progress: JSX.Element;
  if (status === 'done') {
    progress = <span style={{ color: C.green }}>{'━'.repeat(BAR_W) + '     '}</span>;
  } else if (status === 'seeding') {
    progress = (
      <>
        <span style={{ color: C.accent }}>{'━'.repeat(BAR_W)}</span>
        <span style={{ color: C.dim }}>{' 100%'}</span>
      </>
    );
  } else if (status === 'waiting' || status === 'paused' || status === 'error') {
    progress = <span style={{ color: C.faint }}>{'─'.repeat(BAR_W) + '     '}</span>;
  } else {
    const b = bar(pct / 100);
    const pctText = ` ${String(Math.floor(pct)).padStart(3)}%`;
    progress = (
      <>
        <span style={{ color: C.accent }}>{b.filled}</span>
        <span style={{ color: C.faint }}>{b.empty}</span>
        <span style={{ color: C.dim }}>{pctText}</span>
      </>
    );
  }
  return (
    <div style={selected ? { backgroundColor: C.sel } : undefined}>
      <span style={{ color: C.accent }}>{selected ? '▸ ' : '  '}</span>
      <span style={nameStyle}>{trunc(name, NAME_W) + ' '}</span>
      <span style={{ color: STATUS_COLOR[status] }}>{pad(status, STATUS_W) + ' '}</span>
      {progress}
      <span style={{ color: C.dim }}>{lpad(size, SIZE_W)}</span>
      <span style={{ color: C.cyan }}>{lpad(speed, SPEED_W)}</span>
      <span style={{ color: C.dim }}>{lpad(conn, CONN_W)}</span>
      <span style={{ color: C.dim }}>{lpad(eta, ETA_W)}</span>
    </div>
  );
}

const LIVE_NAMES = [
  'ubuntu-24.04.2-desktop-amd64.iso',
  'fedora-workstation-42-x86_64.iso',
  'archlinux-2026.07.01-x86_64.iso',
  'kali-linux-2026.2-installer.iso',
  'tails-amd64-6.15.img',
] as const;
const LIVE_SIZES = ['5.4 GiB', '2.3 GiB', '1.2 GiB', '4.1 GiB', '1.4 GiB'] as const;

export function ListMock({ className }: { className?: string }) {
  const live = useLiveList();

  // Header line: brand │ endpoint ▪ connected … ▼ down ▲ up (app.go header()).
  const down = `▼ ${fmtSpeed(live.speeds.reduce((a, b) => a + b, 0))}`;
  const up = `▲ ${fmtSpeed(live.up)}`;
  const headerLeft = 'aria2t │ localhost:6800 (built-in) ▪ connected';
  const headerGap = Math.max(1, COLS - 1 - headerLeft.length - down.length - 1 - up.length);

  const rows: RowData[] = [
    ...LIVE_NAMES.map((name, j) => ({
      name,
      status: 'active' as Status,
      pct: live.pcts[j],
      size: LIVE_SIZES[j],
      speed: fmtSpeed(live.speeds[j]),
      conn: '1',
      eta: fmtEta(LIVE_BYTES[j] * (1 - live.pcts[j] / 100), live.speeds[j]),
      selected: j === 0,
    })),
    {
      name: 'debian-13.1.0-amd64-netinst.iso',
      status: 'seeding',
      pct: 100,
      size: '680 MiB',
      speed: '-',
      conn: '0:34',
      eta: '-',
    },
    {
      name: 'raspios-bookworm-arm64-full.img.xz',
      status: 'waiting',
      pct: 0,
      size: '0 B',
      speed: '-',
      conn: '-',
      eta: '-',
    },
    {
      name: 'libreoffice-25.8.1-macos-aarch64.dmg',
      status: 'waiting',
      pct: 0,
      size: '0 B',
      speed: '-',
      conn: '-',
      eta: '-',
    },
    {
      name: 'linuxmint-22.1-cinnamon-64bit.iso',
      status: 'paused',
      pct: 35.7,
      size: '2.8 GiB',
      speed: '-',
      conn: '-',
      eta: '-',
    },
    {
      name: 'freebsd-14.3-memstick-amd64.img',
      status: 'paused',
      pct: 0,
      size: '0 B',
      speed: '-',
      conn: '-',
      eta: '-',
    },
    {
      name: 'mirrorlist-nope.iso',
      status: 'error',
      pct: 0,
      size: '0 B',
      speed: '-',
      conn: '-',
      eta: '-',
    },
    {
      name: 'gparted-live-1.7.0-amd64.iso',
      status: 'done',
      pct: 100,
      size: '527 MiB',
      speed: '-',
      conn: '-',
      eta: '-',
    },
  ];

  const colHead =
    pad('NAME', NAME_W + 2) +
    ' ' +
    pad('STATUS', STATUS_W) +
    ' ' +
    pad('PROGRESS', BAR_W + 5) +
    lpad('SIZE', SIZE_W) +
    lpad('SPEED', SPEED_W) +
    lpad('CONN', CONN_W) +
    lpad('ETA', ETA_W);

  const hints: ReadonlyArray<[string, string]> = [
    ['a', 'add'],
    ['space', 'pause'],
    ['↵', 'details'],
    ['d', 'remove'],
    ['l', 'limit'],
    ['/', 'filter'],
    ['?', 'help'],
  ];
  const hintsLen = hints.reduce((n, [k, l]) => n + k.length + 1 + l.length, 0) + (hints.length - 1) * 2;
  const keybarGap = Math.max(1, COLS - 1 - hintsLen - 4);

  return (
    <div dir="ltr" className={cn('select-none [container-type:inline-size]', className)}>
      <div
        className="whitespace-pre font-mono text-[length:min(12px,2.05cqw)] leading-[1.65]"
        style={{ color: C.fg }}
      >
        {/* app header: brand │ endpoint ▪ connected … global speeds */}
        <div>
          <span> </span>
          <span className="font-bold" style={{ color: C.accent }}>
            aria2t
          </span>
          <span style={{ color: C.faint }}>{' │ '}</span>
          <span style={{ color: C.dim }}>{'localhost:6800 (built-in) '}</span>
          <span style={{ color: C.green }}>▪ connected</span>
          <span>{' '.repeat(headerGap)}</span>
          <span style={{ color: C.cyan }}>{down}</span>
          <span> </span>
          <span style={{ color: C.magenta }}>{up}</span>
        </div>

        {/* tabs line */}
        <div className="mt-[0.25em]">
          <span> </span>
          <span className="font-bold" style={{ backgroundColor: C.accent, color: C.bg }}>
            {' All 12 '}
          </span>
          <span style={{ color: C.dim }}>{' [ Active 6 ] [ Waiting 4 ] [ Stopped 2 ]'}</span>
        </div>

        {/* the list panel */}
        <div
          className="mt-[0.45em] rounded-md border px-[1ch] py-[0.3em]"
          style={{ borderColor: C.border }}
        >
          <div style={{ color: C.dim }}>{colHead}</div>
          {rows.map((r) => (
            <Row key={r.name} {...r} />
          ))}
        </div>

        {/* key bar */}
        <div className="mt-[0.45em]">
          <span> </span>
          {hints.map(([k, label], i) => (
            <span key={k}>
              <span className="font-bold" style={{ color: C.accent }}>
                {k}
              </span>
              <span style={{ color: C.dim }}>{` ${label}${i < hints.length - 1 ? '  ' : ''}`}</span>
            </span>
          ))}
          <span>{' '.repeat(keybarGap)}</span>
          <span style={{ color: C.dim }}>1/12</span>
        </div>
      </div>
    </div>
  );
}
