import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { applyAccent, applyTheme, watchSystemTheme } from './theme';
import { setSystemDark } from '@/test/setup';

const root = () => document.documentElement;

beforeEach(() => {
  root().className = '';
  root().removeAttribute('data-theme');
  root().removeAttribute('data-accent');
});

afterEach(() => watchSystemTheme('light'));

describe('applyAccent', () => {
  it('sets a data-accent block, and removes it for neutral', () => {
    applyAccent('purple');
    expect(root().getAttribute('data-accent')).toBe('purple');
    applyAccent('cyan');
    expect(root().getAttribute('data-accent')).toBe('cyan');
    applyAccent('neutral');
    expect(root().hasAttribute('data-accent')).toBe(false);
  });
});

describe('applyTheme', () => {
  it('pins an explicit theme with both the attribute and the class', () => {
    applyTheme('dark');
    expect(root().getAttribute('data-theme')).toBe('dark');
    expect(root().classList.contains('dark')).toBe(true);

    applyTheme('light');
    expect(root().getAttribute('data-theme')).toBe('light');
    expect(root().classList.contains('dark')).toBe(false);
  });

  it('resolves system against the media query and pins no attribute', () => {
    setSystemDark(true);
    applyTheme('system');
    expect(root().hasAttribute('data-theme')).toBe(false);
    expect(root().classList.contains('dark')).toBe(true);

    setSystemDark(false);
    applyTheme('system');
    expect(root().classList.contains('light')).toBe(true);
  });
});

describe('watchSystemTheme', () => {
  it('repaints when the OS theme changes while on system', () => {
    watchSystemTheme('system');
    setSystemDark(true);
    expect(root().classList.contains('dark')).toBe(true);
    setSystemDark(false);
    expect(root().classList.contains('light')).toBe(true);
  });

  it('unsubscribes when switching to an explicit theme', () => {
    watchSystemTheme('system');
    watchSystemTheme('dark');
    applyTheme('dark');
    setSystemDark(true);
    expect(root().getAttribute('data-theme')).toBe('dark');
  });

  it('replaces an existing subscription rather than stacking them', () => {
    watchSystemTheme('system');
    watchSystemTheme('system');
    setSystemDark(true);
    expect(root().classList.contains('dark')).toBe(true);
  });

  it('does nothing when called with an explicit theme first', () => {
    watchSystemTheme('light');
    setSystemDark(true);
    expect(root().className).toBe('');
  });
});
