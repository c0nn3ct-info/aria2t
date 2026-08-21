// Global test setup: the jsdom gaps the site actually hits, plus RTL teardown.
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

// ── matchMedia ───────────────────────────────────────────────────────────────
// lib/theme.ts resolves the system theme through it and subscribes to changes,
// so the query list has to be able to change its answer.
type MqlListener = (ev: MediaQueryListEvent) => void;
const mqlListeners = new Set<MqlListener>();
let systemDark = false;

export function setSystemDark(dark: boolean): void {
  systemDark = dark;
  const ev = { matches: dark, media: '(prefers-color-scheme: dark)' } as MediaQueryListEvent;
  for (const l of [...mqlListeners]) l(ev);
}

window.matchMedia = ((query: string) =>
  ({
    get matches() {
      return systemDark;
    },
    media: query,
    onchange: null,
    addEventListener: (_: string, l: MqlListener) => void mqlListeners.add(l),
    removeEventListener: (_: string, l: MqlListener) => void mqlListeners.delete(l),
    addListener: (l: MqlListener) => void mqlListeners.add(l),
    removeListener: (l: MqlListener) => void mqlListeners.delete(l),
    dispatchEvent: () => true,
  }) as unknown as MediaQueryList) as typeof window.matchMedia;

// ── localStorage ─────────────────────────────────────────────────────────────
// jsdom hands back a bare object with no Storage methods under this
// environment, and the locale switcher stores the chosen language.
const localStore = new Map<string, string>();
if (typeof window.localStorage?.getItem !== 'function') {
  Object.defineProperty(window, 'localStorage', {
    configurable: true,
    value: {
      get length() {
        return localStore.size;
      },
      key: (i: number) => [...localStore.keys()][i] ?? null,
      getItem: (k: string) => localStore.get(k) ?? null,
      setItem: (k: string, v: string) => void localStore.set(k, String(v)),
      removeItem: (k: string) => void localStore.delete(k),
      clear: () => localStore.clear(),
    } satisfies Storage,
  });
}

afterEach(() => {
  cleanup();
  mqlListeners.clear();
  localStore.clear();
  systemDark = false;
});
