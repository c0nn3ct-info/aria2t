import en from './en.json';

export const LOCALES = ['en', 'ru', 'zh-CN', 'es', 'ar', 'fa'] as const;
export type Locale = (typeof LOCALES)[number];

export const RTL_LOCALES: readonly Locale[] = ['ar', 'fa'];

export function isRtl(locale: Locale): boolean {
  return RTL_LOCALES.includes(locale);
}

export function isLocale(x: string): x is Locale {
  return (LOCALES as readonly string[]).includes(x);
}

type Dictionary = Record<string, string>;

/* English ships with every page and the rest arrive on demand: a page is read
 * in one language, and the other five were in the chunk every first paint
 * waited on. English is what `t` reads until the page's own locale is in. */
const DICTIONARIES: Partial<Record<Locale, Dictionary>> = { en: en as Dictionary };

const LOADERS: Record<Exclude<Locale, 'en'>, () => Promise<{ default: Dictionary }>> = {
  ru: () => import('./ru.json'),
  'zh-CN': () => import('./zh-CN.json'),
  es: () => import('./es.json'),
  ar: () => import('./ar.json'),
  fa: () => import('./fa.json'),
};

/** Fetches a locale's dictionary. Call it before `setLocale` puts it in force. */
export async function loadLocale(locale: Locale): Promise<void> {
  if (DICTIONARIES[locale]) return;
  DICTIONARIES[locale] = (await LOADERS[locale as Exclude<Locale, 'en'>]()).default;
}

const NON_EN_LOCALES = LOCALES.filter((l): l is Exclude<Locale, 'en'> => l !== 'en');

let currentLocale: Locale = 'en';

export function setLocale(locale: Locale): void {
  currentLocale = locale;
}

export function getLocale(): Locale {
  return currentLocale;
}

export function t(key: string): string {
  // A locale that has not been loaded reads as English rather than as keys.
  const value = (DICTIONARIES[currentLocale] ?? DICTIONARIES.en!)[key];
  if (value === undefined) {
    if (import.meta.env.DEV) console.warn(`[i18n] missing key: ${key} (${currentLocale})`);
    return key;
  }
  return value;
}

/** Strip a known non-English locale prefix from a path → its base form (`/`, `/install/`, …). */
export function stripLocale(path: string): string {
  for (const l of NON_EN_LOCALES) {
    if (path === `/${l}` || path === `/${l}/`) return '/';
    if (path.startsWith(`/${l}/`)) return path.slice(l.length + 1);
  }
  return path;
}

/** Prefix a base (English) path with a locale. English stays at the root. */
export function withLocale(base: string, locale: Locale): string {
  if (locale === 'en') return base;
  if (base === '/') return `/${locale}/`;
  return `/${locale}${base}`;
}

export function localePath(path: string): string {
  return withLocale(path, currentLocale);
}
