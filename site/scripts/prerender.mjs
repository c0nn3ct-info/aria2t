#!/usr/bin/env node
// Entry point for `npm run build`: prerender every page and write the sitemaps.
// All of the work lives in prerender-core.mjs, which is importable on its own.
import { main } from './prerender-core.mjs';

await main();
