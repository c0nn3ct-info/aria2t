import { cn } from '@/lib/utils';
import { ITEMS, fmtSize, useQueueScene, type Kind } from '@/lib/queue-scene';

// The app's own palettes (tui/internal/ui/theme.go): Tokyo Night Day in light
// mode, Tokyo Night in dark. Values live as --tui-* vars in globals.css so the
// mock switches with the site theme in pure CSS (no hydration dependency).
const C = {
  accent: 'var(--tui-accent)',
  bg: 'var(--tui-bg)',
  fg: 'var(--tui-fg)',
  bright: 'var(--tui-fg-bright)',
  dim: 'var(--tui-fg-dim)',
  faint: 'var(--tui-fg-faint)',
  border: 'var(--tui-border)',
  sel: 'var(--tui-sel)',
  green: 'var(--tui-green)',
  yellow: 'var(--tui-yellow)',
  red: 'var(--tui-red)',
  cyan: 'var(--tui-cyan)',
  magenta: 'var(--tui-magenta)',
} as const;

// Character grid mirroring the real list screen (tui/internal/ui/list.go):
// marker(2)+NAME+' '+STATUS(9)+' '+bar+pct(5)+SIZE+SPEED+CONN+ETA, all
// space-padded like the TUI does with pad/lpad. 78 columns per row line.
// A real terminal geometry, not an invented one. The TUI derives
// nameW = width-72 and barW = 20 (shrinking only below nameW 20), while
// STATUS/SIZE/SPEED/CONN/ETA are fixed at 9/9/12/7/9 whatever the width
// (list.go). COLS = 100 therefore yields nameCol 18 and barW 20, and the row
// comes to 93 cells inside the panel's 96, which is what `aria2t` prints in a
// 100-column terminal.
const COLS = 100;
const STATUS_W = 9;
const NAME_W = COLS - 72 - STATUS_W - 1; // nameCol = nameW - statusW - 1 = 18
const BAR_W = 20;
const SIZE_W = 9;
const SPEED_W = 12;
const CONN_W = 7;
const ETA_W = 9;

export function pad(s: string, w: number): string {
  return s.length >= w ? s.slice(0, w) : s + ' '.repeat(w - s.length);
}

export function lpad(s: string, w: number): string {
  return s.length >= w ? s.slice(0, w) : ' '.repeat(w - s.length) + s;
}

export function trunc(s: string, w: number): string {
  return s.length <= w ? pad(s, w) : s.slice(0, w - 1) + '…';
}

// Exported for their own tests: these mirror the TUI's own formatters, and
// their edges (over-width names, an empty or full bar, sub-minute ETAs) are not
// reachable through the mock's fixed data.
// The TUI's Bar(): '━'-filled with a '╸' cap over a '─' track.
export function bar(frac: number): { filled: string; empty: string } {
  const cells = Math.round(Math.min(1, Math.max(0, frac)) * BAR_W);
  if (cells <= 0) return { filled: '', empty: '─'.repeat(BAR_W) };
  if (cells >= BAR_W) return { filled: '━'.repeat(BAR_W), empty: '' };
  return { filled: '━'.repeat(cells - 1) + '╸', empty: '─'.repeat(BAR_W - cells) };
}

export function fmtSpeed(bps: number): string {
  if (bps <= 0) return '-';
  if (bps >= 1048576) return `${(bps / 1048576).toFixed(1)} MiB/s`;
  if (bps >= 1024) return `${Math.round(bps / 1024)} KiB/s`;
  return `${Math.round(bps)} B/s`;
}

export function fmtEta(remainBytes: number, bps: number): string {
  if (bps <= 0) return '-';
  const s = Math.round(remainBytes / bps);
  if (s >= 3600) return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
  if (s >= 60) return `${Math.floor(s / 60)}m ${s % 60}s`;
  return `${s}s`;
}

// The Active/All/Waiting key-bar, in the real bar's order
// (tui/internal/ui/list.go listModel.keybar). Order matters: the bar is
// width-adaptive and drops from the RIGHT, so the tail the mock cuts at its 100
// columns is genuinely absent from the real screen at that width too.
export const LIST_HINTS: ReadonlyArray<[string, string]> = [
  ['a', 'add'],
  ['space', 'pause'],
  ['↵', 'details'],
  ['d', 'remove'],
  ['l', 'limit'],
  ['/', 'filter'],
  ['y', 'copy url'],
  ['g', 'stats'],
  ['s', 'servers'],
  [',', 'settings'],
  ['?', 'help'],
  ['q', 'quit'],
  ['S', 'scheduler'],
  ['t', 'seeding'],
];

// Port of hintbarEx (tui/internal/ui/app.go): keep adding hints until the next
// one would cross the budget the trailer leaves, and never drop the first.
export function fitHints(
  hints: ReadonlyArray<[string, string]>,
  cols: number,
  trailer: string,
): ReadonlyArray<[string, string]> {
  const budget = trailer ? cols - (trailer.length + 2) : cols;
  const out: Array<[string, string]> = [];
  let x = 1;
  for (const h of hints) {
    const w = h[0].length + 1 + h[1].length;
    if (out.length > 0 && x + w - 1 >= budget) break;
    out.push([h[0], h[1]]);
    x += w + 2;
  }
  return out;
}

// The same six the queue store uses; aria2 has no seventh.
type Status = Kind;

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

// Exported for its own test: an error row only shows its red bar once the
// failed download got somewhere, which the mock's fixed data never does.
export function Row({ name, status, pct, size, speed, conn, eta, selected }: RowData) {
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
  } else if (status === 'error') {
    const b = bar(pct / 100);
    progress = (
      <>
        <span style={{ color: C.red }}>{b.filled}</span>
        <span style={{ color: C.faint }}>{b.empty + '     '}</span>
      </>
    );
  } else if (status === 'waiting' || status === 'paused') {
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
  const scene = useQueueScene();

  // Header line: brand │ endpoint ▪ connected … ▼ down ▲ up (app.go header()).
  // Both figures are aria2's globals, which is why they match the extension
  // popup's to the byte: it is the same daemon and the same store.
  const down = `▼ ${fmtSpeed(scene.down)}`;
  const up = `▲ ${fmtSpeed(scene.up)}`;
  const headerLeft = 'Aria2t │ localhost:6800 (built-in) ▪ connected';
  const headerGap = Math.max(1, COLS - 1 - headerLeft.length - down.length - 1 - up.length);

  const rows: RowData[] = ITEMS.map((item, i) => {
    const { kind, pct, speed } = scene.live[i];
    return {
      name: item.name,
      status: kind,
      pct,
      // `FmtBytes(s.Total())`, whatever the status. A download that failed
      // before it read a content length has no total, which is the `0 B` the
      // real screen prints on its error row; a queued one restored from the
      // session does have one (tui-storybook/frames/screens-list-all.json).
      size: kind === 'error' ? '0 B' : fmtSize(item.bytes),
      // SPEED is the download column. A seeding torrent's upload is in the
      // header total, the way the real screen prints it.
      speed: kind === 'active' ? fmtSpeed(speed) : '-',
      conn: kind === 'active' || kind === 'seeding' ? item.conn : '-',
      // Remaining bytes over the current speed, so the countdown runs at one
      // second per second instead of drifting against the bar beside it.
      eta: kind === 'active' ? fmtEta(item.bytes * (1 - pct / 100), speed) : '-',
      selected: i === 0,
    };
  });

  // aria2's own bucketing (rpc tellActive/tellWaiting/tellStopped): seeding is
  // still active, paused counts as waiting, and stopped holds done + error.
  const tabCounts = {
    active: rows.filter((r) => r.status === 'active' || r.status === 'seeding').length,
    waiting: rows.filter((r) => r.status === 'waiting' || r.status === 'paused').length,
    stopped: rows.filter((r) => r.status === 'done' || r.status === 'error').length,
  };

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

  const trailer = `1/${rows.length}`;
  const hints = fitHints(LIST_HINTS, COLS, trailer);
  const hintsLen = hints.reduce((n, [k, l]) => n + k.length + 1 + l.length, 0) + (hints.length - 1) * 2;
  const keybarGap = Math.max(1, COLS - 1 - hintsLen - trailer.length);

  return (
    <div dir="ltr" className={cn('select-none [container-type:inline-size]', className)}>
      <div
        className="whitespace-pre font-mono text-[length:min(12px,1.64cqw)] leading-[1.65]"
        style={{ color: C.fg }}
      >
        {/* app header: brand │ endpoint ▪ connected … global speeds */}
        <div>
          <span> </span>
          <span className="font-bold" style={{ color: C.accent }}>
            Aria2t
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
            {` All ${rows.length} `}
          </span>
          <span style={{ color: C.dim }}>
            {` [ Active ${tabCounts.active} ] [ Waiting ${tabCounts.waiting} ] [ Stopped ${tabCounts.stopped} ]`}
          </span>
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
          <span style={{ color: C.dim }}>{trailer}</span>
        </div>
      </div>
    </div>
  );
}
