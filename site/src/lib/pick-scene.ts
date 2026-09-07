// The torrent the two file pickers on the page are picking from.
//
// One torrent, one selection: the queue band draws its file tree as an input
// card and the file-selection band draws the picker itself, and both are the
// same three files. Holding the selection here rather than in each section
// means ticking a box in one and scrolling to the other does not show two
// different answers to the same question.
//
// The selection starts where it is declared, so the prerendered frame and the
// first client render agree.
import { useSyncExternalStore } from 'react';

const KiB = 1024;
const MiB = 1048576;
const GiB = 1073741824;

export interface PickFile {
  name: string;
  /** The content type the picker prints under the name. */
  type: string;
  bytes: number;
}

export const FILES: readonly PickFile[] = [
  { name: 'film.mp4', type: 'video/mp4', bytes: 1.4 * GiB },
  { name: 'subtitles.srt', type: 'text/plain', bytes: 84 * KiB },
  { name: 'extras.zip', type: 'application/zip', bytes: 320 * MiB },
];

/** What the picker opens on: the extras nobody asked for are left out. */
const START: readonly boolean[] = [true, true, false];

export const TOTAL_BYTES = FILES.reduce((a, f) => a + f.bytes, 0);

let picked: readonly boolean[] = START;
const listeners = new Set<() => void>();

function getSnapshot(): readonly boolean[] {
  return picked;
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => void listeners.delete(listener);
}

/**
 * Tick or untick one file. Emptying the selection is allowed - what a picker
 * does with an empty one is refuse to start, which is what the commit button
 * does rather than this.
 */
export function togglePick(i: number): void {
  const next = picked.map((p, j) => (j === i ? !p : p));
  picked = next;
  for (const l of listeners) l();
}

export function usePick(): readonly boolean[] {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}

/** Bytes the current selection will fetch. */
export function pickedBytes(sel: readonly boolean[]): number {
  return FILES.reduce((a, f, i) => (sel[i] ? a + f.bytes : a), 0);
}
