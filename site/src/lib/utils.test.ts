import { describe, expect, it } from 'vitest';
import config from '../../tailwind.config';
import { FONT_SIZES, cn, dedupe } from './utils';

describe('cn', () => {
  it('joins conditional class values', () => {
    const off = 0 as number;
    expect(cn('a', off > 0 && 'b', undefined, ['c', null], { d: true, e: false })).toBe('a c d');
  });

  it('lets the last of two conflicting utilities win', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4');
  });

  it('treats the M3 type scale as one font-size group', () => {
    expect(cn('text-body-small', 'text-title-large')).toBe('text-title-large');
  });
});

describe('dedupe', () => {
  it('trims, drops blanks and folds case-insensitive duplicates', () => {
    expect(dedupe([' a ', 'A', '', '  ', 'b'])).toEqual(['a', 'b']);
  });

  it('returns an empty list for empty input', () => {
    expect(dedupe([])).toEqual([]);
  });
});

describe('cn and the type scale', () => {
  it('knows every font size the config defines', () => {
    const keys = Object.keys(config.theme?.extend?.fontSize ?? {});
    expect(keys.length).toBeGreaterThan(0);
    expect(keys.filter((k) => !(FONT_SIZES as readonly string[]).includes(k))).toEqual([]);
  });

  it('keeps a size and a colour side by side', () => {
    expect(cn('text-mini', 'text-on-surface-variant')).toBe('text-mini text-on-surface-variant');
    expect(cn('text-meta text-primary', 'text-on-surface')).toBe('text-meta text-on-surface');
  });
});
