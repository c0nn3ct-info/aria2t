// The redirect runs as an inline <script> in the prerendered HTML, before any
// bundle loads — so it is a string, and the way to test it is to run it.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AUTO_REDIRECT_SCRIPT } from './auto-redirect';

type Loc = { pathname: string; search: string; hash: string; replace: (u: string) => void };

function run(lang: string, path = '/', stored: string | null = null): { loc: Loc; saved?: string } {
  const replace = vi.fn();
  const loc: Loc = { pathname: path, search: '', hash: '', replace };
  const store = new Map<string, string>();
  if (stored) store.set('aria2t-locale', stored);

  // Running the shipped snippet is the point of this suite.
  new Function('location', 'navigator', 'localStorage', AUTO_REDIRECT_SCRIPT)(
    loc,
    { language: lang },
    {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
    },
  );
  return { loc, saved: store.get('aria2t-locale') };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('auto-redirect snippet', () => {
  it('sends a Russian browser to /ru/', () => {
    const { loc, saved } = run('ru-RU');
    expect(saved).toBe('ru');
    expect(loc.replace).toHaveBeenCalledWith('/ru/');
  });

  it('maps each supported language to its prefix', () => {
    for (const [lang, loc] of [
      ['zh-Hans', 'zh-CN'],
      ['es-ES', 'es'],
      ['de-AT', 'de'],
      ['ja', 'ja'],
    ] as const) {
      const r = run(lang, '/install/');
      expect(r.saved, lang).toBe(loc);
      expect(r.loc.replace).toHaveBeenCalledWith(`/${loc}/install/`);
    }
  });

  it('leaves an English browser at the root', () => {
    const { loc, saved } = run('en-GB');
    expect(saved).toBe('en');
    expect(loc.replace).not.toHaveBeenCalled();
  });

  it('treats an unknown language as English', () => {
    const { saved, loc } = run('fr-FR');
    expect(saved).toBe('en');
    expect(loc.replace).not.toHaveBeenCalled();
  });

  it('never redirects twice — a stored choice wins', () => {
    const { loc } = run('ru-RU', '/', 'en');
    expect(loc.replace).not.toHaveBeenCalled();
  });

  it('copes with a browser that reports no language', () => {
    const { saved } = run('');
    expect(saved).toBe('en');
  });

  it('stays silent when storage throws', () => {
    const replace = vi.fn();
    const loc = { pathname: '/', search: '', hash: '', replace };
    expect(() =>
      // See above.
      new Function('location', 'navigator', 'localStorage', AUTO_REDIRECT_SCRIPT)(loc, { language: 'ru' }, {
        getItem: () => {
          throw new Error('blocked');
        },
      }),
    ).not.toThrow();
    expect(replace).not.toHaveBeenCalled();
  });
});
