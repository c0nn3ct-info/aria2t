// The build's entry point. Two lines, but they are what `npm run build` runs,
// so they get a test: import it with the work mocked and check it ran.
import { describe, expect, it, vi } from 'vitest';

const main = vi.fn(() => Promise.resolve());
vi.mock('./prerender-core.mjs', () => ({ main }));

describe('prerender entry', () => {
  it('runs the generator once', async () => {
    await import('./prerender.mjs');
    expect(main).toHaveBeenCalledTimes(1);
  });
});
