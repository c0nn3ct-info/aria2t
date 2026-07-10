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

interface Live {
  d0: number; // row 0 download speed
  d2: number; // row 2 download speed
  up: number; // seeding upload (top-right ▲ aggregate)
  p0: number; // row 0 percent
  p2: number; // row 2 percent
}

const P0 = 62.4;
const P2 = 18.2;
const BYTES0 = 680 * 1048576;
const BYTES2 = 1.2 * 1073741824;

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
  return { d0, d2, up, p0: P0, p2: P2 };
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
        const p0 = l.p0 >= 99 ? 12 : l.p0 + (d0 / BYTES0) * 100 * 0.18;
        const p2 = l.p2 >= 99 ? 4 : l.p2 + (d2 / BYTES2) * 100 * 0.18;
        return { d0, d2, up, p0, p2 };
      });
    }, 1000);
    return () => clearInterval(id);
  }, []);
  return live;
}

type Status = 'active' | 'seeding' | 'paused' | 'done';

const STATUS_COLOR: Record<Status, string> = {
  active: C.green,
  seeding: C.magenta,
  paused: C.yellow,
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
  } else if (status === 'paused') {
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

export function ListMock({ className }: { className?: string }) {
  const live = useLiveList();

  // Header line: brand │ endpoint ▪ connected … ▼ down ▲ up (app.go header()).
  const down = `▼ ${fmtSpeed(live.d0 + live.d2)}`;
  const up = `▲ ${fmtSpeed(live.up)}`;
  const headerLeft = 'aria2t │ localhost:6800 (built-in) ▪ connected';
  const headerGap = Math.max(1, COLS - 1 - headerLeft.length - down.length - 1 - up.length);

  const rows: RowData[] = [
    {
      name: 'debian-13.1.0-amd64-netinst.iso',
      status: 'active',
      pct: live.p0,
      size: '680 MiB',
      speed: fmtSpeed(live.d0),
      conn: '1',
      eta: fmtEta(BYTES0 * (1 - live.p0 / 100), live.d0),
      selected: true,
    },
    {
      name: 'ubuntu-24.04.2-live-server-arm64.iso',
      status: 'seeding',
      pct: 100,
      size: '3.1 GiB',
      speed: '-',
      conn: '0:34',
      eta: '-',
    },
    {
      name: 'archlinux-2026.07.01-x86_64.iso',
      status: 'active',
      pct: live.p2,
      size: '1.2 GiB',
      speed: fmtSpeed(live.d2),
      conn: '1',
      eta: fmtEta(BYTES2 * (1 - live.p2 / 100), live.d2),
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
      name: 'tails-amd64-6.15.img',
      status: 'done',
      pct: 100,
      size: '1.4 GiB',
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
  const keybarGap = Math.max(1, COLS - 1 - hintsLen - 3);

  return (
    <div dir="ltr" className={cn('select-none [container-type:inline-size]', className)}>
      <div
        className="whitespace-pre font-mono text-[length:min(12px,2.05cqw)] leading-[1.5]"
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
        <div className="mt-[0.2em]">
          <span> </span>
          <span className="font-bold" style={{ backgroundColor: C.accent, color: C.bg }}>
            {' All 5 '}
          </span>
          <span style={{ color: C.dim }}>{' [ Active 3 ] [ Waiting 1 ] [ Stopped 1 ]'}</span>
        </div>

        {/* the list panel */}
        <div
          className="mt-[0.35em] rounded-md border px-[1ch] py-[0.15em]"
          style={{ borderColor: C.border }}
        >
          <div style={{ color: C.dim }}>{colHead}</div>
          {rows.map((r) => (
            <Row key={r.name} {...r} />
          ))}
        </div>

        {/* key bar */}
        <div className="mt-[0.35em]">
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
          <span style={{ color: C.dim }}>1/5</span>
        </div>
      </div>
    </div>
  );
}
