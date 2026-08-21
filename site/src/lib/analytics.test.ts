import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const webdriver = { value: false };

beforeEach(() => {
  vi.resetModules();
  document.head.innerHTML = '';
  Object.defineProperty(navigator, 'webdriver', {
    configurable: true,
    get: () => webdriver.value,
  });
  webdriver.value = false;
});

afterEach(() => {
  vi.resetModules();
});

const script = () => document.head.querySelector('script');

describe('initAmplitude', () => {
  it('appends the CDN script once', async () => {
    const { initAmplitude } = await import('./analytics');
    initAmplitude();
    initAmplitude();

    const tags = document.head.querySelectorAll('script');
    expect(tags).toHaveLength(1);
    expect(tags[0].src).toContain('cdn.amplitude.com/script/');
    expect(tags[0].async).toBe(true);
  });

  it('initialises amplitude with session replay once the script loads', async () => {
    const { initAmplitude } = await import('./analytics');
    const add = vi.fn();
    const init = vi.fn();
    const plugin = vi.fn(() => 'replay');
    (window as unknown as Record<string, unknown>).amplitude = { add, init };
    (window as unknown as Record<string, unknown>).sessionReplay = { plugin };

    initAmplitude();
    script()!.onload!(new Event('load'));

    expect(plugin).toHaveBeenCalledWith({ sampleRate: 1 });
    expect(add).toHaveBeenCalledWith('replay');
    expect(init).toHaveBeenCalledWith(expect.any(String), {
      fetchRemoteConfig: true,
      autocapture: false,
    });
  });

  it('stays out of the prerender — no events from a webdriver', async () => {
    webdriver.value = true;
    const { initAmplitude } = await import('./analytics');
    initAmplitude();
    expect(script()).toBeNull();
  });
});
