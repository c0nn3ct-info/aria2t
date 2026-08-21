import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import tseslint from 'typescript-eslint';

// Same posture as extension/eslint.config.js — presets plus two explicit
// escalations, no rule turned off, run with --max-warnings 0. See the note
// there for why the react-hooks rules are named instead of preset-spread.
export default tseslint.config(
  // Build output only — no source is excluded from linting.
  { ignores: ['dist'] },
  js.configs.recommended,
  tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      globals: { ...globals.browser, ...globals.node },
    },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      // The preset ships this as a warning; an unsound dep array is a bug.
      'react-hooks/exhaustive-deps': 'error',
    },
  },
  // The prerender/sitemap generator is plain Node ESM, outside tsconfig's
  // include — linting is the only check it gets. It spans two environments:
  // the driver runs in Node, while the callbacks handed to puppeteer's
  // waitForFunction/evaluate are serialized and run in the page, so `document`
  // is genuinely in scope there.
  {
    files: ['scripts/**/*.mjs'],
    languageOptions: { globals: { ...globals.node, ...globals.browser } },
  },
  {
    files: ['*.config.js'],
    languageOptions: { globals: globals.node },
  },
);
