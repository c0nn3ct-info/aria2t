import type { ReactNode } from 'react';
import {
  Download,
  FileText,
  Github,
  Home,
  Languages,
  Mail,
  ShieldCheck,
  type LucideIcon,
} from 'lucide-react';
import { Aria2tLogo } from '@/components/aria2t-logo';
import { CONTACT_MAILTO, GITHUB_URL, ORG_SITE } from '@/constants';
import { cn } from '@/lib/utils';
import { getLocale, localePath, t, withLocale } from './i18n';
import { GUTTER } from './components/landing/shell';
import { LanguageSwitcher, LOCALE_OPTIONS } from './components/language-switcher';
import { GithubLink } from './components/github-link';

type PageKey = 'home' | 'install' | 'privacy' | 'license';

interface LayoutProps {
  current: PageKey;
  /**
   * Full-bleed main: the page lays out its own sections edge to edge (a hero
   * that paints under the header, bands of alternating width) instead of
   * living in the reading column every other page uses.
   */
  bleed?: boolean;
  children: ReactNode;
}

interface NavLink {
  key: Exclude<PageKey, 'home'> | 'home';
  path: string;
  labelKey: string;
  icon: LucideIcon;
  /** Which footer column this link belongs to; the header shows every link
   * that isn't `home` (the logo already is that link) regardless of section. */
  section: 'product' | 'resources';
}

// One list for the header's page nav and the footer's two columns, so the
// three can never drift apart.
const NAV_LINKS: readonly NavLink[] = [
  { key: 'home', path: '/', labelKey: 'footer.home', icon: Home, section: 'product' },
  { key: 'install', path: '/install/', labelKey: 'nav.docs', icon: Download, section: 'product' },
  { key: 'privacy', path: '/privacy/', labelKey: 'nav.privacy', icon: ShieldCheck, section: 'resources' },
  { key: 'license', path: '/license/', labelKey: 'nav.license', icon: FileText, section: 'resources' },
];

export function Layout({ current, bleed = false, children }: LayoutProps) {
  const homeHref = localePath('/');
  const locale = getLocale();
  // Locale-less path of the page being rendered, so every language link points
  // at this page's translation rather than at the six home pages.
  const currentPath = NAV_LINKS.find((l) => l.key === current)!.path;
  return (
    <div className="flex min-h-screen flex-col bg-background text-on-surface">
      {/* First focusable thing on the page: the header/footer links otherwise
          stand between a keyboard user and the content, on every page. */}
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:start-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-primary-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      >
        {t('nav.skip_to_content')}
      </a>
      {/* Opaque, and no backdrop blur. The blur was only there to hide what a
          95% fill let through, and over the hero's live canvas it made the
          first frame of every scroll a 55-90ms one in Chrome while its
          surface was built. */}
      <header className="sticky top-0 z-20 flex h-16 items-center gap-2 border-b border-outline-variant bg-surface-container-low px-4 sm:px-6">
        <div className="inline-flex items-center gap-2">
          <a
            href={homeHref}
            className="m3-state-layer inline-flex min-h-11 items-center gap-2 rounded-pill px-2 text-on-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            aria-label={t('nav.home_aria')}
          >
            <Aria2tLogo className="h-6 w-6 text-primary" />
            <span className="text-title-medium tracking-tight">Aria2t</span>
          </a>
          {/* Under 360px the lockup and the two 44px buttons need 6px more
              than the bar has, so the org half goes; the footer carries it. */}
          <span aria-hidden="true" className="text-title-medium text-on-surface-variant/50 max-[359px]:hidden">
            ×
          </span>
          <a
            href={ORG_SITE}
            target="_blank"
            rel="noreferrer noopener"
            className="inline-flex min-h-11 items-center rounded-sm px-1 text-label-large text-on-surface-variant underline-offset-4 max-[359px]:hidden hover:text-on-surface hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          >
            c0nn3ct.info
          </a>
        </div>
        {/* Below md the footer carries the same links, so the header drops
            them rather than crowding the bar. At sm they fit in English and
            not in Russian or Spanish: "Документация · Приватность · Лицензия"
            took the bar 50px past a 640px window, and the page with it. */}
        <nav aria-label={t('nav.site_nav_aria')} className="ms-4 hidden items-center gap-1 md:flex">
          {NAV_LINKS.filter((l) => l.key !== 'home').map((l) => (
            <a
              key={l.key}
              href={localePath(l.path)}
              aria-current={current === l.key ? 'page' : undefined}
              className={cn(
                'm3-state-layer inline-flex min-h-11 items-center rounded-pill px-3 text-label-large focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                current === l.key ? 'text-on-surface' : 'text-on-surface-variant',
              )}
            >
              {t(l.labelKey)}
            </a>
          ))}
        </nav>
        <div className="ms-auto flex items-center gap-1">
          <GithubLink />
          <LanguageSwitcher />
        </div>
      </header>

      {/* A reading page stands in the landing's own column - same 1160, same
          gutter - so its content starts exactly where a band's does at every
          width. Matching only the padding was not enough: the column used to
          be capped at 768 below `lg`, which put a page of prose up to 111px
          further in than the band above it on the home page (measured at
          1000px wide).

          The measure is kept by an inner block, and that block is centred in
          the column. Taking the column's start edge instead - the way the
          landing's heading blocks sit inside a band - left the whole of the
          column's slack on the end side: 113px of gutter on one edge and 168
          on the other at 1305, which reads as broken padding rather than as a
          measure. A page of full-width cards has no ragged end to justify it,
          so the slack is split. */}
      <main
        id="main"
        className={
          bleed
            ? 'w-full flex-1'
            : cn('mx-auto w-full max-w-[1160px] flex-1 py-10 sm:py-12', GUTTER)
        }
      >
        {bleed ? children : <div className="mx-auto w-full max-w-5xl">{children}</div>}
      </main>

      {/* One column and one gutter for every page, the same ones `main` above
          takes: the footer's rule closes the content, so a footer on a
          narrower column than the page ends short of what it is closing - 52px
          short on a reading page at 1300, measured. The measure is again the
          inner block's, not the column's, so a page of prose and its footer
          start on the same edge. */}
      <footer
        className={cn('mx-auto w-full max-w-[1160px] py-8 text-label-medium text-on-surface-variant', GUTTER)}
      >
        <div
          className={cn(
            // Two columns on a phone, the brand across both: a free wrap put
            // one column under another at whatever width each happened to be.
            'grid grid-cols-2 items-start gap-x-6 gap-y-6 border-t border-outline-variant pt-6',
            'sm:flex sm:flex-wrap sm:gap-x-12',
            !bleed && 'mx-auto max-w-5xl',
          )}
        >
          {/* Brand block: aria2t.c0nn3ct.info is the product, c0nn3ct.info is
              who made it, so the pairing (echoed from the header) stands in
              for the old "by c0nn3ct.info" byline. */}
          <div className="col-span-2 flex max-w-[280px] flex-col gap-3">
            <div className="inline-flex items-center gap-2 text-on-surface">
              <Aria2tLogo className="h-5 w-5 text-primary" />
              <span className="text-title-medium tracking-tight">Aria2t</span>
              <span aria-hidden="true" className="text-title-medium text-on-surface-variant/50">
                ×
              </span>
              <a
                className="text-label-large text-on-surface-variant underline-offset-4 hover:text-on-surface hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                href={ORG_SITE}
                target="_blank"
                rel="noreferrer noopener"
              >
                c0nn3ct.info
              </a>
            </div>
            {/* Full strength, not 70%: at 11px on the light theme's own ground
                the dimmed token is 3.47:1, under the 4.5 AA asks for. The
                hierarchy here is size and weight, and it survives without
                borrowing contrast it cannot spare. */}
            <p className="text-label-small text-on-surface-variant">{t('home.description')}</p>
          </div>
          <nav aria-label={t('footer.product')}>
            <div className="mb-2 text-label-small uppercase tracking-[0.12em] text-on-surface-variant">
              {t('footer.product')}
            </div>
            <ul className="sm:space-y-1.5">
              {NAV_LINKS.filter((l) => l.section === 'product').map((l) => (
                <li key={l.key}>
                  <a
                    className="inline-flex min-h-11 items-center gap-2 underline-offset-4 hover:underline sm:min-h-[24px] sm:py-1"
                    href={l.key === 'home' ? homeHref : localePath(l.path)}
                    aria-current={current === l.key ? 'page' : undefined}
                  >
                    <l.icon className="h-3.5 w-3.5" />
                    {t(l.labelKey)}
                  </a>
                </li>
              ))}
            </ul>
          </nav>
          <nav aria-label={t('footer.resources')}>
            <div className="mb-2 text-label-small uppercase tracking-[0.12em] text-on-surface-variant">
              {t('footer.resources')}
            </div>
            <ul className="sm:space-y-1.5">
              {NAV_LINKS.filter((l) => l.section === 'resources').map((l) => (
                <li key={l.key}>
                  <a
                    className="inline-flex min-h-11 items-center gap-2 underline-offset-4 hover:underline sm:min-h-[24px] sm:py-1"
                    href={localePath(l.path)}
                    aria-current={current === l.key ? 'page' : undefined}
                  >
                    <l.icon className="h-3.5 w-3.5" />
                    {t(l.labelKey)}
                  </a>
                </li>
              ))}
            </ul>
          </nav>
          <nav aria-label={t('footer.contacts')}>
            <div className="mb-2 text-label-small uppercase tracking-[0.12em] text-on-surface-variant">
              {t('footer.contacts')}
            </div>
            <ul className="sm:space-y-1.5">
              <li>
                <a
                  className="inline-flex min-h-11 items-center gap-2 underline-offset-4 hover:underline sm:min-h-[24px] sm:py-1"
                  href={GITHUB_URL}
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  <Github className="h-3.5 w-3.5" />
                  GitHub
                </a>
              </li>
              <li>
                <a
                  className="inline-flex min-h-11 items-center gap-2 underline-offset-4 hover:underline sm:min-h-[24px] sm:py-1"
                  href={CONTACT_MAILTO}
                >
                  <Mail className="h-3.5 w-3.5" />
                  help@c0nn3ct.info
                </a>
              </li>
            </ul>
          </nav>
        </div>
        {/* The header switcher builds its menu on click, so the prerendered HTML
            carries no link between the locales at all — only hreflang. These are
            the same six URLs as plain markup. One wrapped row rather than a
            fifth column: six stacked items made the footer twice as tall and
            broke the column layout below 900px. */}
        <nav
          aria-label={t('footer.languages')}
          className={cn(
            'mt-6 flex flex-wrap items-center gap-x-2 gap-y-1 border-t border-outline-variant pt-4 text-label-small',
            // Its rule closes the same measure the row of columns above does.
            !bleed && 'mx-auto max-w-5xl',
          )}
        >
          {/* The icon carries the row; the group name lives on the nav's
              aria-label, so screen readers still announce it. */}
          <Languages className="me-1 h-3.5 w-3.5 shrink-0" aria-hidden />
          {LOCALE_OPTIONS.map((l, i) => (
            <span key={l.code} className="inline-flex items-center gap-2">
              {i > 0 && (
                <span aria-hidden className="text-outline-variant">
                  ·
                </span>
              )}
              <a
                // 44px on a phone, where the footer is navigated by thumb;
                // from `sm` 24px, not the 16 the line box gives it: a target
                // below 24x24 fails WCAG 2.5.8, and these six sit one beside
                // the other with a separator between them.
                className="inline-flex min-h-11 items-center sm:min-h-[24px] underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                href={withLocale(currentPath, l.code)}
                hrefLang={l.code}
                lang={l.code}
                aria-current={l.code === locale ? 'true' : undefined}
              >
                {l.label}
              </a>
            </span>
          ))}
        </nav>
      </footer>
    </div>
  );
}
