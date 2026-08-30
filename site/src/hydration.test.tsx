// The site ships prerendered: `scripts/prerender.mjs` drives a real browser,
// takes the rendered DOM back out as HTML, and the visitor's React hydrates it.
// That round trip through the browser's own serializer is where the markup can
// stop matching what React expects — most easily by adjacent text nodes, since
// `{value}%` or `{word}{' '}` renders as two nodes and comes back as one. React
// answers that by throwing the whole prerendered tree away and client-rendering
// the root, which is silent, invisible in the tests, and exactly the thing the
// prerender exists to avoid.
//
// So: render each page, serialize it, parse it back — the same round trip —
// then hydrate onto it and listen. Nothing needs a build for this.
import { StrictMode } from 'react';
import { createRoot, hydrateRoot } from 'react-dom/client';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act } from '@/test/render';
import { LOCALES, setLocale } from '@/i18n';
import { HomePage } from '@/pages/home';
import { InstallPage } from '@/pages/install';
import { ExtensionPage } from '@/pages/extension';
import { PrivacyPage } from '@/pages/privacy';
import { LicensePage } from '@/pages/license';

const PAGES = [
  ['home', HomePage],
  ['install', InstallPage],
  ['extension', ExtensionPage],
  ['privacy', PrivacyPage],
  ['license', LicensePage],
] as const;

/**
 * The one difference React is allowed to find, and it is cosmetic: an inline
 * `style` comes back from the serializer as `width: 63%;` while React builds
 * `width:63%` to compare against. React logs that in development and hydrates
 * anyway — a bar's width is set through the CSSOM either way. Anything else,
 * including a new *kind* of prop warning, is a real mismatch.
 */
const STYLE_FORMAT = /Prop `style` did not match|Warning: Prop `%s` did not match[\s\S]*style/;

async function roundTrip(page: React.ReactElement): Promise<string[]> {
  const rendered = document.createElement('div');
  document.body.appendChild(rendered);
  const root = createRoot(rendered);
  await act(async () => {
    root.render(<StrictMode>{page}</StrictMode>);
  });

  // Through the serializer and back: this is what the prerender writes to disk
  // and what the browser hands React on the next visit.
  const target = document.createElement('div');
  target.innerHTML = rendered.innerHTML;
  document.body.appendChild(target);

  const errs: string[] = [];
  const spy = vi.spyOn(console, 'error').mockImplementation((...args: unknown[]) => {
    errs.push(args.map(String).join(' '));
  });
  const hydrated = hydrateRoot(target, <StrictMode>{page}</StrictMode>);
  await act(async () => undefined);
  spy.mockRestore();

  act(() => {
    root.unmount();
    hydrated.unmount();
  });
  rendered.remove();
  target.remove();
  return errs.filter((e) => !STYLE_FORMAT.test(e));
}

afterEach(() => setLocale('en'));

describe.each(PAGES)('%s page', (_name, Page) => {
  it('hydrates onto its own prerendered markup', async () => {
    expect(await roundTrip(<Page />)).toEqual([]);
  });
});

describe('every locale', () => {
  // A translation is markup too: a string that lands beside a link or a number
  // splits the same way, and only in that language.
  it.each(LOCALES)('hydrates the home page in %s', async (locale) => {
    setLocale(locale);
    expect(await roundTrip(<HomePage />)).toEqual([]);
  });
});
