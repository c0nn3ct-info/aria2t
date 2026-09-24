import { clsx, type ClassValue } from 'clsx';
import { extendTailwindMerge } from 'tailwind-merge';

/**
 * Every `fontSize` key in tailwind.config.ts. tailwind-merge only knows its
 * own `text-sm`…`text-9xl` as sizes, so any other `text-*` reads to it as a
 * colour, and `cn('text-mini', 'text-on-surface-variant')` would keep the
 * colour and drop the size. A test holds this list to the config's keys.
 */
export const FONT_SIZES = [
  'display-small',
  'display-medium',
  'display-large',
  'headline-small',
  'headline-medium',
  'headline-large',
  'title-small',
  'title-medium',
  'title-large',
  'title-dense',
  'label-small',
  'label-medium',
  'label-large',
  'body-small',
  'body-medium',
  'body-large',
  'micro',
  'status',
  'mini',
  'meta',
  'caption',
  'subtitle',
  'lead',
  'figure',
] as const;

const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      'font-size': [{ text: [...FONT_SIZES] }],
    },
  },
});

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function dedupe(items: readonly string[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of items) {
    const trimmed = item.trim();
    if (!trimmed) continue;
    const key = trimmed.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(trimmed);
  }
  return out;
}
