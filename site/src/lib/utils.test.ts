import { describe, expect, it } from 'vitest';
import { cn, dedupe } from './utils';

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
