import { afterEach, describe, expect, it } from 'vitest';
import { render, screen, within } from '@/test/render';
import { GITHUB_URL } from '@/constants';
import { LOCALES, setLocale, t } from './i18n';
import { LOCALE_OPTIONS } from './components/language-switcher';
import { Layout } from './layout';

afterEach(() => setLocale('en'));

describe('Layout', () => {
  it('frames the page with a header, its content and a footer', () => {
    render(
      <Layout current="home">
        <p>page body</p>
      </Layout>,
    );
    expect(screen.getByRole('banner')).toBeInTheDocument();
    expect(within(screen.getByRole('main')).getByText('page body')).toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toBeInTheDocument();
  });

  it('drops the reading column when a page lays itself out edge to edge', () => {
    render(
      <Layout current="home" bleed>
        <p>hero</p>
      </Layout>,
    );
    const main = screen.getByRole('main');
    expect(main.className).not.toMatch(/max-w-/);
    expect(main.className).not.toMatch(/px-/);
    // the chrome around it is unchanged
    expect(screen.getByRole('banner')).toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toBeInTheDocument();
  });

  it('links home, to GitHub and to every page', () => {
    render(
      <Layout current="install">
        <p>x</p>
      </Layout>,
    );
    const hrefs = screen.getAllByRole('link').map((a) => a.getAttribute('href'));
    expect(hrefs).toContain('/');
    expect(hrefs).toContain('/install/');
    expect(hrefs).toContain('/privacy/');
    expect(hrefs).toContain('/license/');
    expect(hrefs).toContain(GITHUB_URL);
  });

  it('keeps every in-site link inside the active locale', () => {
    setLocale('ru');
    render(
      <Layout current="home">
        <p>x</p>
      </Layout>,
    );
    // Every link but the footer's language row, which exists to leave the
    // active locale for another one.
    const languages = screen.getByRole('navigation', { name: t('footer.languages') });
    const internal = screen
      .getAllByRole('link')
      .filter((a) => !languages.contains(a))
      .map((a) => a.getAttribute('href') ?? '')
      .filter((h) => h.startsWith('/'));
    expect(internal.length).toBeGreaterThan(0);
    for (const href of internal) expect(href.startsWith('/ru/')).toBe(true);
  });

  it('lists every locale of the current page as plain crawlable markup', () => {
    setLocale('ru');
    render(
      <Layout current="privacy">
        <p>x</p>
      </Layout>,
    );
    const languages = within(
      screen.getByRole('navigation', { name: t('footer.languages') }),
    ).getAllByRole('link');

    // The switcher in the header builds its menu on click, so without this row
    // the prerendered HTML links no translation at all.
    expect(languages.map((a) => a.getAttribute('href'))).toEqual([
      '/privacy/',
      '/ru/privacy/',
      '/zh-CN/privacy/',
      '/es/privacy/',
      '/ar/privacy/',
      '/fa/privacy/',
    ]);
    expect(languages.map((a) => a.getAttribute('hreflang'))).toEqual(
      LOCALE_OPTIONS.map((l) => l.code),
    );
    // The active locale is the one already being read.
    expect(languages.filter((a) => a.getAttribute('aria-current') === 'true')).toHaveLength(1);
  });

  it('leaks no raw i18n key in any locale', () => {
    for (const locale of LOCALES) {
      setLocale(locale);
      const { container, unmount } = render(
        <Layout current="home">
          <p>x</p>
        </Layout>,
      );
      expect(container.textContent, locale).not.toMatch(/(nav|footer)\.[a-z_]+/);
      unmount();
    }
  });
});
