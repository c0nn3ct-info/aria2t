// The gate's entry point. Two lines, but they are what `npm run lint` runs, so
// they get a test: import it with the work mocked and check it ran, and that
// the exit code is the one the check returned.
import { describe, expect, it, vi } from 'vitest';

const main = vi.fn(() => 0);
vi.mock('./stories-check-core.mjs', () => ({ main }));

describe('stories-check entry', () => {
  it('runs the check once and exits with its verdict', async () => {
    const exit = vi.spyOn(process, 'exit').mockImplementation(() => undefined);
    await import('./stories-check.mjs');
    expect(main).toHaveBeenCalledTimes(1);
    expect(exit).toHaveBeenCalledWith(0);
    exit.mockRestore();
  });
});
