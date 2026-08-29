// The prerender generator. Everything but main() is pure string work, so it is
// tested directly; main() runs against a mocked puppeteer and file system.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LOCALES } from '../src/i18n/index.ts';

const PAGES = ['home', 'install', 'extension', 'privacy', 'license'];

const written = new Map();
const writeFile = vi.fn((path, body) => {
  written.set(String(path), String(body));
  return Promise.resolve();
});
// The core loads the real locale catalogues through readFile, so JSON reads pass
// through to disk; only writes are captured.
vi.mock('node:fs/promises', async (importOriginal) => {
  const actual = await importOriginal();
  const readFile = vi.fn((p, enc) =>
    String(p).endsWith('.json')
      ? actual.readFile(p, enc)
      : Promise.resolve('<!doctype html><html><head></head><body></body></html>'),
  );
  return { readFile, writeFile, default: { readFile, writeFile } };
});

const chromeAt = { path: null, all: false };
const statSync = vi.fn((p) => {
  if (chromeAt.all || (chromeAt.path && String(p) === chromeAt.path)) {
    return { isFile: () => true };
  }
  throw new Error('ENOENT');
});
vi.mock('node:fs', () => ({ statSync, default: { statSync } }));

const closed = { server: 0, browser: 0 };
const listen = vi.fn((port, cb) => cb());
const createServer = vi.fn((handler) => ({
  listen,
  handler,
  close: () => {
    closed.server += 1;
  },
}));
vi.mock('node:http', () => ({ createServer, default: { createServer } }));
vi.mock('serve-handler', () => ({ default: vi.fn() }));

// The page stub runs the callbacks the script hands it, so the readiness
// predicate and the serializer are exercised rather than skipped.
const pageStub = {
  goto: vi.fn(() => Promise.resolve()),
  waitForFunction: vi.fn((fn) => Promise.resolve(fn())),
  evaluate: vi.fn((fn) => Promise.resolve(fn())),
  close: vi.fn(() => Promise.resolve()),
};
const launch = vi.fn(() =>
  Promise.resolve({
    newPage: () => Promise.resolve(pageStub),
    close: () => {
      closed.browser += 1;
      return Promise.resolve();
    },
  }),
);
vi.mock('puppeteer', () => ({ default: { launch: (...a) => launch(...a) } }));

const core = await import('./prerender-core.mjs');

beforeEach(() => {
  written.clear();
  closed.server = 0;
  closed.browser = 0;
  chromeAt.path = null;
  chromeAt.all = false;
  delete process.env.PUPPETEER_EXECUTABLE_PATH;
  launch.mockClear();
  pageStub.goto.mockClear();
});

afterEach(() => {
  vi.restoreAllMocks?.();
});

describe('pathFor and diskPath', () => {
  it('keeps English at the root and prefixes every other locale', () => {
    expect(core.pathFor('home', 'en')).toBe('/');
    expect(core.pathFor('install', 'en')).toBe('/install/');
    expect(core.pathFor('home', 'ru')).toBe('/ru/');
    expect(core.pathFor('install', 'fa')).toBe('/fa/install/');
  });

  it('maps every page and locale to a distinct file on disk', () => {
    const seen = new Set();
    for (const page of PAGES) {
      for (const locale of LOCALES) {
        const p = core.diskPath(page, locale);
        expect(p.endsWith('.html')).toBe(true);
        seen.add(p);
      }
    }
    expect(seen.size).toBe(PAGES.length * LOCALES.length);
  });
});

describe('dictionary fallbacks', () => {
  it('falls back for a page the catalogues do not describe', () => {
    const meta = core.getMeta('nonexistent', 'en');
    expect(meta.title).toBe('Aria2t');
    expect(meta.description).toBe('');

    const blocks = core.jsonLdBlocks('nonexistent', 'en', '1.2.3');
    expect(JSON.stringify(blocks)).toContain('nonexistent');
  });

  it('still describes the OG image for every shipped locale', () => {
    for (const locale of LOCALES) {
      expect(core.buildHeadInjection('home', locale, '1.2.3'), locale).toContain('og:image:alt');
    }
  });
});

describe('getMeta', () => {
  it('gives every page in every locale a title and a description', () => {
    for (const page of PAGES) {
      for (const locale of LOCALES) {
        const meta = core.getMeta(page, locale);
        expect(meta.title, `${page}/${locale}`).toBeTruthy();
        expect(meta.description, `${page}/${locale}`).toBeTruthy();
      }
    }
  });

  it('localizes: the same page reads differently in another language', () => {
    expect(core.getMeta('home', 'ru').title).not.toBe(core.getMeta('home', 'en').title);
  });
});

describe('escaping', () => {
  it('escapes attribute and text contexts', () => {
    // Ampersands first, so an escape is never double-escaped. The attribute form
    // leaves ">" alone — it cannot end an attribute value.
    expect(core.escapeHtmlAttr('a"b&c<d>')).toBe('a&quot;b&amp;c&lt;d>');
    expect(core.escapeHtmlText('<script>&')).toBe('&lt;script&gt;&amp;');
  });
});

describe('jsonLdBlocks', () => {
  it('emits structured data for every page and locale', () => {
    for (const page of PAGES) {
      for (const locale of LOCALES) {
        const blocks = core.jsonLdBlocks(page, locale, '1.2.3');
        expect(Array.isArray(blocks), `${page}/${locale}`).toBe(true);
        expect(blocks.length, `${page}/${locale}`).toBeGreaterThan(0);
        for (const b of blocks) {
          expect(b['@context'], `${page}/${locale}`).toBe('https://schema.org');
          expect(b['@type'], `${page}/${locale}`).toBeTruthy();
          // Must survive serialization into the page's <script type=ld+json>.
          expect(() => JSON.stringify(b), `${page}/${locale}`).not.toThrow();
        }
      }
    }
  });
});

describe('buildHeadInjection', () => {
  it('carries canonical, alternates and Open Graph for every locale', () => {
    for (const locale of LOCALES) {
      const head = core.buildHeadInjection('home', locale, '1.2.3');
      expect(head).toContain('rel="canonical"');
      expect(head).toContain('hreflang="x-default"');
      expect(head).toContain('property="og:');
      for (const other of LOCALES) expect(head).toContain(`hreflang="${other}"`);
    }
  });

  it('points each page at its own canonical URL', () => {
    expect(core.buildHeadInjection('install', 'ar', '1')).toContain('/ar/install/');
  });
});

describe('injectIntoHead', () => {
  const html = '<!doctype html>\n<html><head><title>old</title><meta name="description" content="old"></head><body></body></html>';

  it('replaces the title and description and adds the injection', () => {
    const out = core.injectIntoHead(html, '<link rel="canonical" href="x">', 'new title', 'new desc');
    expect(out).toContain('<title>new title</title>');
    expect(out).toContain('new desc');
    expect(out).not.toContain('>old<');
    expect(out).toContain('rel="canonical"');
  });

  it('still injects into a head with nothing to replace', () => {
    const bare = '<html><head></head><body></body></html>';
    expect(core.injectIntoHead(bare, '<meta name="x">', 't', 'd')).toContain('<meta name="x">');
  });
});

describe('sitemaps', () => {
  it('lists every page in every locale, with priorities and lastmod', () => {
    const xml = core.buildSitemap('2026-08-05');
    expect(xml).toContain('<?xml');
    expect((xml.match(/<url>/g) ?? []).length).toBe(PAGES.length * LOCALES.length);
    expect(xml).toContain('<lastmod>2026-08-05</lastmod>');
    expect(xml).toContain('<priority>1.0</priority>');
    for (const locale of LOCALES) expect(xml).toContain(`hreflang="${locale}"`);
  });

  it('writes an index pointing at the sitemap', () => {
    const xml = core.buildSitemapIndex('2026-08-05');
    expect(xml).toContain('<sitemapindex');
    expect(xml).toContain('sitemap.xml');
    expect(xml).toContain('2026-08-05');
  });
});

describe('findSystemChrome', () => {
  it('returns nothing when no known Chrome is installed', () => {
    delete process.env.PUPPETEER_EXECUTABLE_PATH;
    expect(core.findSystemChrome()).toBeUndefined();
  });

  it('prefers an explicitly configured binary', () => {
    process.env.PUPPETEER_EXECUTABLE_PATH = '/opt/chrome';
    expect(core.findSystemChrome()).toBe('/opt/chrome');
    delete process.env.PUPPETEER_EXECUTABLE_PATH;
  });

  it('knows where Chrome lives on each platform', () => {
    chromeAt.all = true;
    expect(core.findSystemChrome('darwin')).toContain('/Applications/');
    expect(core.findSystemChrome('linux')).toBe('/usr/bin/google-chrome');
    expect(core.findSystemChrome('win32')).toContain('chrome.exe');
  });
});

describe('startServer', () => {
  it('serves dist on the given port', async () => {
    const server = await core.startServer(4321);
    expect(listen).toHaveBeenCalledWith(4321, expect.any(Function));

    // Every request is handed to serve-handler, rooted at dist.
    const handler = (await import('serve-handler')).default;
    const req = {};
    const res = {};
    server.handler(req, res);
    expect(handler).toHaveBeenCalledWith(req, res, { public: expect.stringContaining('dist') });

    server.close();
    expect(closed.server).toBe(1);
  });
});

describe('main', () => {
  it('prerenders every page and writes all three sitemaps', async () => {
    // The readiness predicate looks for a mounted #root.
    document.body.innerHTML = '<div id="root"><main>x</main></div>';
    await core.main();

    const pages = [...written.keys()].filter((p) => p.endsWith('.html'));
    expect(pages).toHaveLength(PAGES.length * LOCALES.length);
    expect([...written.keys()].some((p) => p.endsWith('sitemap.xml'))).toBe(true);
    expect([...written.keys()].some((p) => p.endsWith('site.xml'))).toBe(true);
    expect([...written.keys()].some((p) => p.endsWith('sitemap_index.xml'))).toBe(true);

    // Every rendered page carries its injected head.
    for (const html of pages.map((p) => written.get(p))) {
      expect(html).toContain('rel="canonical"');
    }
    // And it always closes what it opened.
    expect(closed.browser).toBe(1);
    expect(closed.server).toBe(1);
    expect(launch).toHaveBeenCalledWith({ headless: true });
  });

  it('uses a system Chrome when it finds one', async () => {
    chromeAt.all = true;
    await core.main();
    expect(launch).toHaveBeenCalledWith(
      expect.objectContaining({ headless: true, executablePath: expect.any(String) }),
    );
  });

  it('still closes the browser and server when a page fails', async () => {
    pageStub.goto.mockRejectedValueOnce(new Error('navigation failed'));
    await expect(core.main()).rejects.toThrow('navigation failed');
    expect(closed.browser).toBe(1);
    expect(closed.server).toBe(1);
  });
});
