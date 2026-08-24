import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const pagesDir = path.resolve(here, 'pages');

// Keep in sync with LOCALES in src/i18n/index.ts (config is plain, can't import the .ts type list).
const LOCALES = ['en', 'ru', 'zh-CN', 'es', 'de', 'ja'] as const;
const PAGES: Record<string, string> = {
  home: 'index.html',
  install: 'install/index.html',
  extension: 'extension/index.html',
  privacy: 'privacy/index.html',
  license: 'license/index.html',
};

// Route HTML live under pages/ (the Vite root); output mirrors them into dist/ at the same URLs.
const input: Record<string, string> = {};
for (const locale of LOCALES) {
  for (const [page, rel] of Object.entries(PAGES)) {
    const key = locale === 'en' ? page : `${locale}-${page}`;
    const file = locale === 'en' ? rel : `${locale}/${rel}`;
    input[key] = path.resolve(pagesDir, file);
  }
}

// Dev-only: the route shells reference ../src/… relative to their depth; the
// browser normalizes those URLs to /src/… which lives outside the Vite root
// (pages/), so the dev server would fall back to HTML. Resolve them to the
// real src/ dir. Build is unaffected (Rollup resolves the HTML imports on
// the filesystem).
const srcOutsideRoot = {
  name: 'src-outside-root',
  apply: 'serve' as const,
  resolveId(id: string) {
    if (id.startsWith('/src/')) return path.resolve(here, id.slice(1));
    return null;
  },
};

export default defineConfig({
  base: '/',
  root: pagesDir,
  publicDir: path.resolve(here, 'public'),
  plugins: [srcOutsideRoot, react()],
  resolve: {
    alias: { '@': path.resolve(here, 'src') },
  },
  build: {
    outDir: path.resolve(here, 'dist'),
    emptyOutDir: true,
    rollupOptions: {
      input,
      output: {
        // Split the framework out of the per-route layout chunk. Every page is
        // prerendered, so this bundle exists only to hydrate — keeping React in
        // its own chunk lets it stay cached across routes and across deploys
        // that only touch page code.
        manualChunks: {
          react: ['react', 'react-dom', 'react-dom/client'],
        },
      },
    },
  },
  server: {
    port: 5181,
    strictPort: true,
    // Route HTML reference the sibling ../src via relative module scripts.
    fs: { allow: [here] },
  },
});
