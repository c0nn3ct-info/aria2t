import { afterEach, describe, expect, it } from 'vitest';
import { render, screen, within } from '@/test/render';
import { GITHUB_URL } from '@/constants';
import { LOCALES, setLocale } from './i18n';
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

  it('links home, to GitHub and to every page', () => {
    render(
      <Layout current="install">
        <p>x</p>
      </Layout>,
    );
    const hrefs = screen.getAllByRole('link').map((a) => a.getAttribute('href'));
    expect(hrefs).toContain('/');
    expect(hrefs).toContain('/install/');
    expect(hrefs).toContain('/extension/');
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
    const internal = screen
      .getAllByRole('link')
      .map((a) => a.getAttribute('href') ?? '')
      .filter((h) => h.startsWith('/'));
    expect(internal.length).toBeGreaterThan(0);
    for (const href of internal) expect(href.startsWith('/ru/')).toBe(true);
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
