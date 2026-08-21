import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(here, 'src') },
  },
  test: {
    globals: true,
    // One environment for everything: jsdom still allows node builtins, so the
    // prerender script's suite works under it too.
    environment: 'jsdom',
    setupFiles: ['src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}', 'scripts/**/*.test.mjs'],
    coverage: {
      // istanbul, not v8: it instruments the source, so branch counts follow
      // the TS/TSX the tests target rather than post-esbuild ranges (which
      // invent unreachable branches that could only be silenced).
      provider: 'istanbul',
      // Untested files must count as 0%, never vanish from the report.
      all: true,
      // The prerender/sitemap generator is source too, so it counts.
      include: ['src/**/*.{ts,tsx}', 'scripts/**/*.mjs'],
      // Non-source only: the tests, their helpers, and type-only declarations.
      exclude: ['src/**/*.{test,spec}.{ts,tsx}', 'src/test/**', 'src/**/*.d.ts', 'scripts/**/*.test.mjs'],
      reporter: ['text', 'html'],
      // All four metrics, and nothing silenced: no istanbul-ignore comments
      // anywhere, and the exclude list above names only non-source. Code that
      // turns out to be unreachable gets restructured, not ignored.
      thresholds: { 100: true },
    },
  },
});
