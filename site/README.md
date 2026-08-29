# aria2t-site

The aria2t site (Vite + React + Tailwind, Material 3), served at <https://aria2t.c0nn3ct.info>. Five pages (home, install, extension, privacy, license) × six locales (en, ru, zh-CN, es, ar, fa — exactly the set the extension ships, with ar/fa right-to-left), prerendered to static HTML.

## Commands

```bash
npm ci
npm run dev        # dev server on http://localhost:5181
npm run build      # tsc + vite build + prerender → dist/
npm run lint       # eslint — must be clean, warnings fail too
npm run preview
```

The prerender step (`scripts/prerender.mjs`) serves `dist/`, renders every route in headless Chrome, and injects canonical/hreflang/OpenGraph/JSON-LD metadata plus sitemaps. It prefers a system Chrome; set `PUPPETEER_EXECUTABLE_PATH` to override.

## Layout

- `pages/**/index.html` — 24 hand-checked route shells (per-locale title/description; the locale auto-redirect script lives only in the English shells).
- `src/entries/` — one entry per page; reads `<html lang>` to pick the dictionary.
- `src/i18n/` — flat key→string dictionaries, one per locale; all six must hold the same key set.
- `src/styles/globals.css` + `tailwind.config.ts` — the Material 3 design tokens (color roles, type scale, shape, motion, elevation).
- `src/components/` — M3 UI primitives (`ui/`), the live terminal mock (`terminal-mock`, `list-mock`), the architecture diagram, FAQ, and chrome.

Deployed to GitHub Pages by the public repo's `deploy.yml` on pushes to `main` touching `site/**`.
