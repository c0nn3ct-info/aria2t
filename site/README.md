# aria2t-site

The aria2t site (Vite + React + Tailwind, Material 3), served at <https://aria2t.c0nn3ct.info>. Five pages (home, install, extension, privacy, license) × six locales (en, ru, zh-CN, es, ar, fa — exactly the set the extension ships, with ar/fa right-to-left), prerendered to static HTML.

## Commands

```bash
npm ci
npm run dev        # dev server on http://localhost:5181
npm run build      # tsc + vite build + prerender → dist/
npm run lint       # eslint — must be clean, warnings fail too
npm run typecheck  # both TS projects: the site's own, and the tests + stories
npm run storybook  # component workshop on http://localhost:6007
npm run preview
```

`npm run build` runs `npm run typecheck` first, which is two `tsc` projects rather than one: `tsconfig.json` covers the site's own sources with `skipLibCheck` off, and `tsconfig.dev.json` adds the tests, the stories and `.storybook/` with it on — Storybook's and vitest's shipped declarations do not check clean, and a devDependency's broken types must not weaken the check over `src/`.

Storybook is one of the three refs the shell in `../storybook` composes; it loads `vite.storybook.config.ts` rather than the site's 30-input multi-page build, and scans the stories through its own Tailwind instance (the shipped CSS excludes them). `src/storybook/stories.smoke.test.tsx` mounts every story in jsdom on `npm test`, which is what keeps stories out of the coverage denominator from meaning "unchecked".

The prerender step (`scripts/prerender.mjs`) serves `dist/`, renders every route in headless Chrome, and injects canonical/hreflang/OpenGraph/JSON-LD metadata plus sitemaps. It prefers a system Chrome; set `PUPPETEER_EXECUTABLE_PATH` to override.

## Layout

- `pages/**/index.html` — 24 hand-checked route shells (per-locale title/description; the locale auto-redirect script lives only in the English shells).
- `src/entries/` — one entry per page; reads `<html lang>` to pick the dictionary.
- `src/i18n/` — flat key→string dictionaries, one per locale; all six must hold the same key set.
- `src/styles/globals.css` + `tailwind.config.ts` — the Material 3 design tokens (color roles, type scale, shape, motion, elevation).
- `src/components/` — M3 UI primitives (`ui/`), the live terminal mock (`terminal-mock`, `list-mock`), the architecture diagram, FAQ, and chrome.

Deployed to GitHub Pages by the public repo's `deploy.yml` on pushes to `main` touching `site/**`.
