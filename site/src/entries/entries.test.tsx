// Each entry sets the locale from the document's lang and mounts its page. They
// are three lines each, but they are what the browser actually runs.
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mounted: unknown[] = [];
vi.mock('../main', () => ({ mountPage: (p: unknown) => void mounted.push(p) }));

const ENTRIES = [
  ['home', () => import('./home')],
  ['install', () => import('./install')],
  ['extension', () => import('./extension')],
  ['privacy', () => import('./privacy')],
  ['license', () => import('./license')],
] as const;

beforeEach(() => {
  mounted.length = 0;
  vi.resetModules();
});

describe.each(ENTRIES)('%s entry', (name, load) => {
  it('takes the locale from the document and mounts its page', async () => {
    document.documentElement.lang = 'ru';
    await load();

    const { getLocale } = await import('../i18n');
    expect(getLocale(), name).toBe('ru');
    expect(mounted).toHaveLength(1);
  });
});

describe.each(ENTRIES)('%s entry with an unknown document language', (name, load) => {
  it('falls back to English', async () => {
    document.documentElement.lang = 'fr';
    await load();
    const { getLocale } = await import('../i18n');
    expect(getLocale(), name).toBe('en');
    expect(mounted).toHaveLength(1);
  });
});
