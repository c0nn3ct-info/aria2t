// One queue, seen from two places.
//
// The extension popup and the terminal client are two views of a single aria2
// daemon, which is what the page's "One daemon holds everything" band claims.
// Both mocks used to walk their own numbers, so the popup showed three
// downloads at 26 MiB/s while the list beside it showed six at 14 - the one
// thing that band says cannot happen. They read this store instead, so the
// figures agree by construction and carry across when the visitor switches
// surfaces.
//
// Everything here is one queue of twelve downloads. The popup shows the first
// few, the way the real popup does, and its headline figures are the daemon's
// globals (aria2's `getGlobalStat`), not a sum of the rows that happen to fit.
import { useSyncExternalStore } from 'react';
import { motionAllowed, mulberry32, stepSpeed } from './mock-motion';

const MiB = 1048576;
const GiB = 1073741824;

export type Kind = 'active' | 'seeding' | 'waiting' | 'paused' | 'error' | 'done';
export type IconKind = 'disc' | 'folder' | 'magnet' | 'archive';

export interface Item {
  name: string;
  /** Total size. A waiting or failed download reports nothing yet. */
  bytes: number;
  /** Bytes per second it walks around while transferring. */
  target: number;
  icon: IconKind;
  /** A torrent or a magnet: it seeds on completion rather than stopping. */
  torrent: boolean;
  /** The list's CONN cell: connections, or `seeds:peers` for a torrent. */
  conn: string;
}

export interface Live {
  kind: Kind;
  pct: number;
  /** Bytes per second: download while active, upload while seeding, else 0. */
  speed: number;
}

export interface Snapshot {
  live: readonly Live[];
  /** aria2's global download and upload speeds. */
  down: number;
  up: number;
  /** Downloads moving data right now. */
  downloading: number;
  /** Total throughput per second, oldest first: what the popup's wave draws. */
  wave: readonly number[];
}

/** What a seeding torrent uploads, as a share of what it downloaded at. */
const SEED_SHARE = 0.52;

/** aria2 runs this many at once; when one finishes, the next one starts. */
const CONCURRENT = 5;

/**
 * Samples behind the popup hero. The extension's service worker keeps this
 * many at 1 Hz (`SPEED_HISTORY`, extension/src/lib/traffic-buffer.ts).
 */
const WAVE_N = 60;

/**
 * The wave's ceiling, fixed rather than the current window's peak. A window
 * maximum pins the tallest sample to the top on every frame, so the height
 * stops meaning throughput and the whole curve jumps when a peak scrolls out.
 * Comfortably above the sum of the targets below.
 */
export const WAVE_MAX = 44 * MiB;

/** The queue. The first five are what the popup has room for. */
export const ITEMS: readonly Item[] = [
  { name: 'ubuntu-24.04.2-desktop-amd64.iso', bytes: 5.4 * GiB, target: 9.8 * MiB, icon: 'disc', torrent: false, conn: '1' },
  { name: 'Fedora-Workstation-Live-42.torrent', bytes: 2.3 * GiB, target: 9.6 * MiB, icon: 'folder', torrent: true, conn: '4:31' },
  { name: '4K Wallpaper Megapack', bytes: 12.6 * GiB, target: 4.6 * MiB, icon: 'magnet', torrent: true, conn: '2:18' },
  { name: 'Nature.Docs.S01.1080p.WEB-DL', bytes: 8.4 * GiB, target: 6.8 * MiB, icon: 'folder', torrent: true, conn: '6:24' },
  { name: 'raspios-bookworm-arm64.img.xz', bytes: 680 * MiB, target: 3.4 * MiB, icon: 'archive', torrent: false, conn: '-' },
  { name: 'archlinux-2026.07.01-x86_64.iso', bytes: 1.2 * GiB, target: 3.1 * MiB, icon: 'disc', torrent: false, conn: '1' },
  { name: 'kali-linux-2026.2-installer.iso', bytes: 4.1 * GiB, target: 2.4 * MiB, icon: 'disc', torrent: false, conn: '1' },
  { name: 'debian-13.1.0-amd64-netinst.iso', bytes: 680 * MiB, target: 1.7 * MiB, icon: 'disc', torrent: true, conn: '0:34' },
  { name: 'libreoffice-25.8.1-macos-aarch64.dmg', bytes: 380 * MiB, target: 5.2 * MiB, icon: 'archive', torrent: false, conn: '-' },
  { name: 'linuxmint-22.1-cinnamon-64bit.iso', bytes: 2.8 * GiB, target: 4.5 * MiB, icon: 'disc', torrent: false, conn: '-' },
  { name: 'mirrorlist-nope.iso', bytes: 0, target: 0, icon: 'disc', torrent: false, conn: '-' },
  { name: 'gparted-live-1.7.0-amd64.iso', bytes: 527 * MiB, target: 0, icon: 'disc', torrent: false, conn: '-' },
];

/** Where the queue stands when the page opens. */
const START: ReadonlyArray<{ kind: Kind; pct: number }> = [
  { kind: 'active', pct: 63 },
  { kind: 'seeding', pct: 100 },
  { kind: 'active', pct: 23 },
  { kind: 'active', pct: 41 },
  { kind: 'paused', pct: 18 },
  { kind: 'active', pct: 19 },
  { kind: 'active', pct: 56 },
  { kind: 'seeding', pct: 100 },
  { kind: 'waiting', pct: 0 },
  { kind: 'paused', pct: 36 },
  { kind: 'error', pct: 0 },
  { kind: 'done', pct: 100 },
];

/** What a download's speed walks around, given what it is doing. */
function targetFor(i: number, kind: Kind): number {
  if (kind === 'active') return ITEMS[i].target;
  if (kind === 'seeding') return ITEMS[i].target * SEED_SHARE;
  return 0;
}

function totals(live: readonly Live[]): Pick<Snapshot, 'down' | 'up' | 'downloading'> {
  let down = 0;
  let up = 0;
  let downloading = 0;
  for (const l of live) {
    // A seeding torrent's throughput is upload. Counting it as download was
    // what made the popup's headline 5 MiB/s higher than its own rows.
    if (l.kind === 'active') {
      down += l.speed;
      downloading++;
    } else if (l.kind === 'seeding') {
      up += l.speed;
    }
  }
  return { down, up, downloading };
}

/** One second of the queue. */
function step(prev: Snapshot, rnd: () => number): Snapshot {
  const live: Live[] = prev.live.map((l, i) => {
    if (l.kind === 'active') {
      const speed = stepSpeed(l.speed, ITEMS[i].target, rnd);
      // The bar advances at the rate its own speed and size imply, so the
      // percentage, the speed and the ETA beside them are one set of numbers.
      const pct = l.pct + (speed / ITEMS[i].bytes) * 100;
      if (pct < 100) return { kind: l.kind, pct, speed };
      // Completion is the only way a bar leaves 100%, and it does not go back:
      // a torrent starts seeding, anything else stops.
      return ITEMS[i].torrent
        ? { kind: 'seeding', pct: 100, speed: targetFor(i, 'seeding') }
        : { kind: 'done', pct: 100, speed: 0 };
    }
    if (l.kind === 'seeding') {
      return { kind: l.kind, pct: 100, speed: stepSpeed(l.speed, targetFor(i, 'seeding'), rnd) };
    }
    return l;
  });

  // aria2 starts the next one in the queue as soon as a slot frees up. Without
  // it the scene would drain to a screen full of finished downloads.
  if (live.filter((l) => l.kind === 'active').length < CONCURRENT) {
    const next = live.findIndex((l) => l.kind === 'waiting');
    if (next >= 0) live[next] = { kind: 'active', pct: 0, speed: ITEMS[next].target };
  }

  const sums = totals(live);
  return {
    live,
    ...sums,
    // One sample in, the oldest out: the wave scrolls rather than grows.
    wave: [...prev.wave.slice(1), sums.down + sums.up],
  };
}

/**
 * The opening frame. The speed walk is run forward so the numbers are already
 * settled and the wave already holds a full minute, the way a popup opens onto
 * history the worker kept while it was closed; the percentages stay where they
 * are declared, because a page that opened on a different point in the queue
 * every time would not be the same scene twice.
 */
function seed(): Snapshot {
  const rnd = mulberry32(0x61726961); // 'aria'
  let live: Live[] = START.map((s, i) => ({ ...s, speed: targetFor(i, s.kind) }));
  const wave: number[] = [];
  for (let n = 0; n < WAVE_N; n++) {
    live = live.map((l, i) => ({ ...l, speed: stepSpeed(l.speed, targetFor(i, l.kind), rnd) }));
    const { down, up } = totals(live);
    wave.push(down + up);
  }
  return { live, ...totals(live), wave };
}

let snapshot: Snapshot = seed();
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setInterval> | undefined;

function beat(): void {
  // A background tab still runs timers on some engines; advancing the queue
  // there only burns battery on a screen nobody is looking at.
  if (document.hidden) return;
  snapshot = step(snapshot, Math.random);
  for (const l of listeners) l();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  if (listeners.size === 1 && motionAllowed()) timer = setInterval(beat, 1000);
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0 && timer !== undefined) {
      clearInterval(timer);
      timer = undefined;
    }
  };
}

function getSnapshot(): Snapshot {
  return snapshot;
}

/** The queue as both mocks see it. */
export function useQueueScene(): Snapshot {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

/**
 * Sizes, byte for byte as both real surfaces print them: `FmtBytes` in
 * tui/internal/ui/format.go and `fmtBytes` in
 * extension/src/lib/aria2/format.ts. One decimal below ten and none from ten
 * up, which is why 12.6 GiB reads `13 GiB` rather than `12.6 GiB`.
 */
export function fmtSize(bytes: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
  let v = Math.max(0, bytes);
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  if (i === 0) return `${Math.round(v)} B`;
  return `${v < 10 ? v.toFixed(1) : v.toFixed(0)} ${units[i]}`;
}
