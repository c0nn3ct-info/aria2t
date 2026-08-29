import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  getLocale,
  isLocale,
  isRtl,
  localePath,
  LOCALES,
  setLocale,
  stripLocale,
  t,
  withLocale,
} from './index';

afterEach(() => setLocale('en'));

describe('locale identification', () => {
  it('recognises exactly the shipped locales', () => {
    for (const l of LOCALES) expect(isLocale(l)).toBe(true);
    expect(isLocale('fr')).toBe(false);
    expect(isLocale('')).toBe(false);
  });

  it('reports Arabic and Persian as RTL and nothing else', () => {
    // The site ships exactly the extension's locales, two of which read
    // right-to-left; the layout is logical-property-only so it mirrors for free.
    for (const l of LOCALES) expect(isRtl(l)).toBe(l === 'ar' || l === 'fa');
  });

  it('defaults to English and follows setLocale', () => {
    expect(getLocale()).toBe('en');
    setLocale('fa');
    expect(getLocale()).toBe('fa');
  });
});

describe('translation', () => {
  it('resolves a key in every locale', () => {
    for (const l of LOCALES) {
      setLocale(l);
      expect(t('nav.install')).not.toBe('nav.install');
    }
  });

  it('falls back to the key itself when a string is missing, and warns in dev', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    expect(t('no.such.key')).toBe('no.such.key');
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('no.such.key'));
    warn.mockRestore();
  });

  it('stays quiet about a missing string in a production build', async () => {
    vi.stubEnv('DEV', false);
    vi.resetModules();
    const fresh = await import('./index');
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);

    expect(fresh.t('no.such.key')).toBe('no.such.key');
    expect(warn).not.toHaveBeenCalled();

    warn.mockRestore();
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it('keeps every catalogue in step with English', async () => {
    const en = (await import('./en.json')).default as Record<string, string>;
    for (const l of LOCALES) {
      const dict = (await import(`./${l}.json`)).default as Record<string, string>;
      expect(Object.keys(dict).sort(), l).toEqual(Object.keys(en).sort());
    }
  });
});

describe('locale paths', () => {
  it('keeps English at the root and prefixes the rest', () => {
    expect(withLocale('/', 'en')).toBe('/');
    expect(withLocale('/install/', 'en')).toBe('/install/');
    expect(withLocale('/', 'ru')).toBe('/ru/');
    expect(withLocale('/install/', 'zh-CN')).toBe('/zh-CN/install/');
  });

  it('strips a locale prefix back to the base path', () => {
    expect(stripLocale('/ru')).toBe('/');
    expect(stripLocale('/ru/')).toBe('/');
    expect(stripLocale('/ru/install/')).toBe('/install/');
    expect(stripLocale('/zh-CN/privacy/')).toBe('/privacy/');
  });

  it('leaves an unprefixed or unknown path alone', () => {
    expect(stripLocale('/')).toBe('/');
    expect(stripLocale('/install/')).toBe('/install/');
    expect(stripLocale('/fr/install/')).toBe('/fr/install/');
    // "en" is never a prefix, so a path that merely starts with it survives.
    expect(stripLocale('/extension/')).toBe('/extension/');
  });

  it('round-trips every locale through both helpers', () => {
    for (const l of LOCALES) {
      for (const base of ['/', '/install/', '/extension/', '/privacy/', '/license/']) {
        expect(stripLocale(withLocale(base, l))).toBe(base);
      }
    }
  });

  it('localePath follows the current locale', () => {
    expect(localePath('/install/')).toBe('/install/');
    setLocale('ar');
    expect(localePath('/install/')).toBe('/ar/install/');
    expect(localePath('/')).toBe('/ar/');
  });
});
