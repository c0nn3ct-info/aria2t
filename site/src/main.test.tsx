import { beforeEach, describe, expect, it, vi } from 'vitest';

const createRoot = vi.fn(() => ({ render: vi.fn() }));
const hydrateRoot = vi.fn();
vi.mock('react-dom/client', () => ({ createRoot, hydrateRoot }));
const initAmplitude = vi.fn();
vi.mock('@/lib/analytics', () => ({ initAmplitude }));

const { mountPage } = await import('./main');

beforeEach(() => {
  vi.clearAllMocks();
  document.body.innerHTML = '';
  document.documentElement.className = '';
  document.documentElement.removeAttribute('data-theme');
});

describe('mountPage', () => {
  it('applies the system theme, starts analytics and renders into an empty root', () => {
    document.body.innerHTML = '<div id="root"></div>';
    mountPage(<p>page</p>);

    expect(document.documentElement.classList.contains('light')).toBe(true);
    expect(initAmplitude).toHaveBeenCalled();
    expect(createRoot).toHaveBeenCalledWith(document.getElementById('root'));
    expect(hydrateRoot).not.toHaveBeenCalled();
  });

  it('hydrates a prerendered root instead of replacing it', () => {
    document.body.innerHTML = '<div id="root"><main>prerendered</main></div>';
    mountPage(<p>page</p>);

    expect(hydrateRoot).toHaveBeenCalled();
    expect(createRoot).not.toHaveBeenCalled();
  });

  it('fails loudly when the page has no root', () => {
    expect(() => mountPage(<p>page</p>)).toThrow(/#root/);
  });
});
