import { useLayoutEffect, type ReactNode } from 'react';
import type { Decorator, Preview } from '@storybook/react-vite';
import '../src/styles/globals.css';
import { isRtl, LOCALES, setLocale, type Locale } from '../src/i18n';
import {
  applyAccent,
  applyTheme,
  watchSystemTheme,
  type Accent,
  type Theme,
} from '../src/lib/theme';

// The site has no theme context: `applyTheme`/`applyAccent` mutate <html>, and
// components read the resulting CSS custom properties. Running the real
// functions (instead of an addon that swaps a wrapper class) keeps the toolbar
// honest, and mutating <html> rather than remounting keeps every
// `transition-colors` on the story observable as the theme flips.
//
// `useLayoutEffect`, not `useEffect`: globals.css resolves dark from
// `prefers-color-scheme` whenever <html> carries no `data-theme`, so an effect
// that runs after the commit would let a light story paint one dark frame on a
// dark-OS machine. The same holds for `[data-accent]` tokens.
//
// The hooks live in a component rather than in the decorator function itself: a
// decorator is a plain function Storybook calls, so only a component is
// guaranteed its own hook slot (and the lint rule that enforces it is right).
function Themed({ theme, accent, children }: { theme: Theme; accent: Accent; children: ReactNode }) {
  useLayoutEffect(() => {
    applyTheme(theme);
    watchSystemTheme(theme);
  }, [theme]);

  useLayoutEffect(() => {
    applyAccent(accent);
  }, [accent]);

  return <>{children}</>;
}

const withTheme: Decorator = (Story, context) => (
  <Themed theme={context.globals.theme as Theme} accent={context.globals.accent as Accent}>
    <Story />
  </Themed>
);

// i18n is a module-level dictionary with no provider, so `setLocale` has to
// land before the story renders — an effect would fire too late. `key={locale}`
// then remounts the story, so a component that reads `t()` once (in a state
// initialiser or an effect) picks the new dictionary up.
const withLocale: Decorator = (Story, context) => {
  const locale = context.globals.locale as Locale;
  setLocale(locale);
  document.documentElement.lang = locale;
  document.documentElement.dir = isRtl(locale) ? 'rtl' : 'ltr';
  return <Story key={locale} />;
};

// Toolbar items, one per member of the union they set. `Record<Theme, …>` and
// `Record<Accent, …>` are what makes that a type error rather than a silent
// mismatch: a fourth theme or a fourth accent in `src/lib/theme.ts` fails `tsc`
// here until it has an entry, and a mistyped value ('dakr') fails as an unknown
// key. Object key order is the order the toolbar lists them in.
const THEME_TITLES: Record<Theme, string> = {
  light: 'Light',
  dark: 'Dark',
  system: 'System',
};

const ACCENT_TITLES: Record<Accent, string> = {
  neutral: 'Neutral',
  purple: 'Purple',
  cyan: 'Cyan',
};

// The locale items derive from `LOCALES` for the same reason: a seventh
// language fails `tsc` here until it is named.
const LOCALE_TITLES: Record<Locale, string> = {
  en: 'English',
  ru: 'Русский',
  'zh-CN': '中文',
  es: 'Español',
  ar: 'العربية (RTL)',
  fa: 'فارسی (RTL)',
};

/** `{ value, title }` items in declaration order. */
function toolbarItems(titles: Record<string, string>): { value: string; title: string }[] {
  return Object.entries(titles).map(([value, title]) => ({ value, title }));
}

const preview: Preview = {
  // withTheme is last, so it wraps withLocale: a locale switch remounts the
  // story without discarding the theme decorator's effect state.
  decorators: [withLocale, withTheme],
  globalTypes: {
    theme: {
      name: 'Theme',
      toolbar: { icon: 'mirror', dynamicTitle: true, items: toolbarItems(THEME_TITLES) },
    },
    accent: {
      name: 'Accent',
      toolbar: { icon: 'paintbrush', dynamicTitle: true, items: toolbarItems(ACCENT_TITLES) },
    },
    // No `dir` global: `withLocale` sets `<html dir>` from the locale, which is
    // the only place the site derives direction from.
    locale: {
      name: 'Locale',
      toolbar: {
        icon: 'globe',
        dynamicTitle: true,
        items: LOCALES.map((code) => ({ value: code, title: LOCALE_TITLES[code] })),
      },
    },
  },
  initialGlobals: { theme: 'light', accent: 'neutral', locale: 'en' },
  parameters: {
    layout: 'padded',
    // The site paints its own surfaces; a backgrounds swatch on top of that
    // only ever contradicts the active theme.
    backgrounds: { disable: true },
    docs: { toc: true },
  },
  tags: ['autodocs'],
};

export default preview;
