// Shared scaffolding for the Foundations token pages. Not a `.stories.tsx`
// file, so Storybook's glob never picks it up and its exports cannot be read as
// stories; `src/storybook/**` is negated in the shipped Tailwind content and
// excluded from coverage, same as the story files that import it.
import { useEffect, useState, type ReactNode } from 'react';
import { Stack } from '@/storybook/layout';

/**
 * Fingerprint of the globals that rewrite `<html>`. The preview's decorators
 * run `applyTheme`, `applyAccent` and set `lang`/`dir`, so a change to any of
 * the three restyles the document and invalidates every value a story has read
 * out of the cascade.
 */
export function keyOf(globals: { [name: string]: unknown }): string {
  return [globals.theme, globals.accent, globals.locale].join('/');
}

/**
 * Resolved values of custom properties on `<html>`, re-read whenever `tokenKey`
 * changes. The theme decorator writes `<html>` in a layout effect, so a read
 * from this effect is a read of the settled cascade — media-query dark and
 * `[data-accent]` overrides included. Pass a module-level `names` array: it is
 * an effect dependency.
 */
export function useRootTokens(names: readonly string[], tokenKey: string): Record<string, string> {
  const [values, setValues] = useState<Record<string, string>>({});
  useEffect(() => {
    const style = getComputedStyle(document.documentElement);
    setValues(Object.fromEntries(names.map((name) => [name, style.getPropertyValue(name).trim()])));
  }, [names, tokenKey]);
  return values;
}

/** A titled group of samples. */
export function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Stack gap={8}>
      <div className="text-label-large text-on-surface-variant">{title}</div>
      {children}
    </Stack>
  );
}

/** The small mono line under a sample that carries its token and resolved value. */
export function Caption({ children }: { children: string }) {
  return <code className="text-label-small text-on-surface-variant">{children || '—'}</code>;
}
