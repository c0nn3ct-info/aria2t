// One treatment for the paths, flags and file names that appear inside a
// sentence.
//
// The reading pages carry a dozen of them - `~/.config/aria2t/config.json`,
// `--config`, `picks.json` - and they were set two ways: the uninstall card
// wrapped its path in a mono badge, every other sentence left the same kind of
// token as plain prose. Same role, two treatments, on the same page.
//
// The tokens are Latin and identical in all six locales (a translated sentence
// says the path, it does not translate it), so the split is on the text rather
// than on the string being marked up per locale. Arabic and Persian precede
// them with U+200E to keep the bidi run intact; that mark stays in the prose
// run, outside the match.
import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

/**
 * A home-relative path, a relative binary, a long flag, a file name of a kind
 * this product actually has, or a bare directory - `daemon/` heads one item of
 * a list whose other two items are file names.
 *
 * Two details do the work. The trailing character class keeps a sentence's
 * final full stop out of a path (`~/.config/aria2t/daemon/.` ends at the
 * slash), and the bare directory has to be followed by a space or the end of
 * the string, which is what leaves `and/or` and `https://host` alone.
 */
const TOKEN =
  /(~\/[\w./-]*[\w/]|\.\/[\w.-]*[\w]|--[a-z][\w-]*|\b[\w-]+\.(?:json|txt|conf|log|torrent|metalink|sh|ps1)\b|\b[\w-]+\/(?=\s|$))/g;

/** The badge itself, exported so a caller with its own markup matches it. */
export const proseCodeClass =
  'rounded bg-surface-container-highest px-1 py-0.5 font-mono text-body-small';

/**
 * Renders `text` with every such token in the badge and the rest as prose.
 * Returns a fragment, so the caller keeps its own `<p>`, `<li>` or cell.
 */
export function ProseCode({ text, className }: { text: string; className?: string }): ReactNode {
  // A capturing split puts the matches on the odd indices and the prose
  // between them on the even ones, including the empty strings at the edges.
  const parts = text.split(TOKEN);
  return (
    <>
      {parts.map((part, i) =>
        i % 2 === 1 ? (
          <code key={i} dir="ltr" className={cn(proseCodeClass, className)}>
            {part}
          </code>
        ) : (
          part
        ),
      )}
    </>
  );
}
