// Mounts every story in jsdom.
//
// `tsc`, `build-storybook` and `storybook dev --smoke-test` all pass without
// ever rendering a story, so a story that throws on its first frame ships
// green. This closes that gap without a browser: portable stories give each
// story its project annotations, loaders and decorators, and Testing Library
// mounts it. Loaders run — that is what sets the locale and the theme — but
// `play` does not: the assertion here is that a story renders at all, and
// interaction belongs in Storybook.
//
// It is also what keeps the story files honest under a coverage config that
// excludes them: they are excluded from the *denominator*, not from being run.
import type { ReactElement } from 'react';
import { afterEach, expect, it } from 'vitest';
import { composeStories, setProjectAnnotations } from '@storybook/react-vite';
import { cleanup, render, waitFor } from '@testing-library/react';
import * as projectAnnotations from '../../.storybook/preview';

setProjectAnnotations([projectAnnotations]);

/**
 * Stories that cannot render here. Empty: every story in the site mounts in
 * jsdom. Anything that stops being true — a story that needs a real canvas or a
 * real layout engine — goes in as `'<title> › <name>': 'one-line reason'`.
 */
const ALLOW_LIST: Record<string, string> = {};

type StoryModule = Parameters<typeof composeStories>[0];

/** The slice of a composed story this test touches. */
type ComposedStory = ((props?: Record<string, never>) => ReactElement | null) & {
  storyName: string;
  load: () => Promise<void>;
};

// A lazy glob awaited below, not `{ eager: true }`: Vite hoists an eager glob's
// imports above this file's own, and importing the story modules after
// `.storybook/preview` — which imports the stylesheet and the i18n singleton —
// is the order Storybook itself uses.
const modules = import.meta.glob<StoryModule>('../**/*.stories.tsx');

const cases: [name: string, Story: ComposedStory][] = (
  await Promise.all(
    Object.keys(modules)
      .sort()
      .map(async (path) => {
        const mod = await modules[path]();
        const title = mod.default?.title ?? path;
        return Object.values(composeStories(mod)).map((story) => {
          const Story = story as ComposedStory;
          return [`${title} › ${Story.storyName}`, Story] as [string, ComposedStory];
        });
      }),
  )
).flat();

const mountable = cases.filter(([name]) => !(name in ALLOW_LIST));

afterEach(cleanup);

// A renamed or deleted story would leave its allow-list entry excluding
// nothing, and the story it was meant to skip would then be mounted (or, worse,
// a real exclusion would look like it was still in force). Every key has to
// name a story the glob above actually found.
it('has no stale ALLOW_LIST entries', () => {
  const found = new Set(cases.map(([name]) => name));
  expect(Object.keys(ALLOW_LIST).filter((name) => !found.has(name))).toEqual([]);
});

// There is no story file at all until the first one lands, and a glob that
// matches nothing would leave this suite green while asserting nothing.
it('found stories to mount', () => {
  expect(cases.length).toBeGreaterThan(0);
});

// `%s` over a tuple rather than `$name` over an object: `$`-interpolation
// quotes the value and truncates it to a stub like `'Primitives/Butt…'`.
it.each(mountable)('%s', async (_name, Story) => {
  await Story.load();
  const { baseElement } = render(<Story />);
  // Elements under `baseElement` (document.body), not under the container: a
  // story that renders through a portal leaves the container empty. The render
  // root is the one element always present.
  //
  // An explicit timeout, not the 1s default: this assertion runs once per story
  // across every story in the package, and it is a liveness check — the story
  // either renders within a couple of frames or it is genuinely broken. A 1s
  // default is short enough that machine load, not a real bug, can fail it.
  await waitFor(
    () => {
      expect(baseElement.querySelectorAll('*').length).toBeGreaterThan(1);
    },
    { timeout: 3000 },
  );
});
