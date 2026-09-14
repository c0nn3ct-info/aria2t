// What the badge may and may not swallow. The pages hand `ProseCode` whole
// translated sentences, so over-matching would put a badge around a word and
// under-matching would leave a path as prose - the inconsistency it exists to
// remove.
import { describe, expect, it } from 'vitest';
import { render } from '@/test/render';
import { ProseCode, proseCodeClass } from './prose-code';

function badges(text: string): string[] {
  const { container } = render(
    <p>
      <ProseCode text={text} />
    </p>,
  );
  return [...container.querySelectorAll('code')].map((c) => c.textContent ?? '');
}

describe('ProseCode', () => {
  it('marks paths, relative binaries, long flags and file names', () => {
    expect(
      badges(
        "Configuration is stored in ~/.config/aria2t/config.json (override the path with --config); run ./aria2t and see session.txt.",
      ),
    ).toEqual(['~/.config/aria2t/config.json', '--config', './aria2t', 'session.txt']);
  });

  it('marks a bare directory, but not a slash inside a word or a URL', () => {
    expect(badges('daemon/ — the managed daemon\'s session, configuration, and log.')).toEqual([
      'daemon/',
    ]);
    expect(badges('Either and/or is fine, and https://aria2t.c0nn3ct.info stays a link.')).toEqual(
      [],
    );
  });

  it("leaves a sentence's own punctuation out of the path", () => {
    // `daemon/.` would otherwise take the full stop into the badge
    expect(badges('The session lives under ~/.config/aria2t/daemon/.')).toEqual([
      '~/.config/aria2t/daemon/',
    ]);
  });

  it('returns a sentence with nothing to mark as itself', () => {
    const text = 'Aria2t stores no history database or cache of downloaded content.';
    const { container } = render(
      <p>
        <ProseCode text={text} />
      </p>,
    );
    expect(container.querySelectorAll('code')).toHaveLength(0);
    expect(container.textContent).toBe(text);
  });

  it('keeps the bidi mark in the prose and the path left to right', () => {
    const { container } = render(
      <p dir="rtl">
        <ProseCode text={'يوجد كل شيء تحت ‎~/.config/aria2t/ على جهازك:'} />
      </p>,
    );
    const code = container.querySelector('code')!;
    expect(code.textContent).toBe('~/.config/aria2t/');
    expect(code).toHaveAttribute('dir', 'ltr');
    // the mark stays where the translation put it, before the badge
    expect(container.textContent).toContain('‎');
  });

  it('takes a class beside its own', () => {
    const { container } = render(<ProseCode text="see config.json" className="ms-1" />);
    const code = container.querySelector('code')!;
    expect(code.className).toContain('ms-1');
    expect(proseCodeClass.split(' ').every((c) => code.className.includes(c))).toBe(true);
  });
});
