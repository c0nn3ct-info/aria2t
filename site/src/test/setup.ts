// Global test setup: the jsdom gaps the site actually hits, plus RTL teardown.
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

// ── matchMedia ───────────────────────────────────────────────────────────────
// lib/theme.ts resolves the system theme through it and subscribes to changes,
// so the query list has to be able to change its answer. A `(min-width: Npx)`
// query is answered off `window.innerWidth` instead - the landing hero asks
// for the `lg:` breakpoint, and a stub that handed it the dark-mode flag would
// tie the layout it picks to the theme.
type MqlListener = (ev: MediaQueryListEvent) => void;
const mqlListeners = new Set<MqlListener>();
let systemDark = false;

export function setSystemDark(dark: boolean): void {
  systemDark = dark;
  const ev = { matches: dark, media: '(prefers-color-scheme: dark)' } as MediaQueryListEvent;
  for (const l of [...mqlListeners]) l(ev);
}

window.matchMedia = ((query: string) => {
  const minWidth = /\(min-width:\s*(\d+)px\)/.exec(query);
  return {
    get matches() {
      return minWidth ? window.innerWidth >= Number(minWidth[1]) : systemDark;
    },
    media: query,
    onchange: null,
    addEventListener: (_: string, l: MqlListener) => void mqlListeners.add(l),
    removeEventListener: (_: string, l: MqlListener) => void mqlListeners.delete(l),
    addListener: (l: MqlListener) => void mqlListeners.add(l),
    removeListener: (l: MqlListener) => void mqlListeners.delete(l),
    dispatchEvent: () => true,
  } as unknown as MediaQueryList;
}) as typeof window.matchMedia;

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
