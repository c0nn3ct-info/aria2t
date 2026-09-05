import type { Meta, StoryObj } from '@storybook/react-vite';
import { userEvent } from 'storybook/test';
import { FaqSection } from './faq-section';

// The section takes no props: eight `<details>` built from `home.faq.*`, all
// closed on mount. So an open answer is only reachable the way a reader reaches
// it — by clicking a summary — and the stories that show one do exactly that in
// a `play`.

const meta = {
  title: 'Blocks/FaqSection',
  component: FaqSection,
  tags: ['autodocs'],
} satisfies Meta<typeof FaqSection>;

export default meta;

type Story = StoryObj<typeof meta>;

/**
 * Indices, not question text: the same entry then opens in every locale the
 * toolbar offers, instead of the story silently opening nothing once the copy
 * is translated.
 */
async function open(canvasElement: HTMLElement, indices: readonly number[]) {
  const summaries = canvasElement.querySelectorAll('summary');
  for (const i of indices) await userEvent.click(summaries[i]);
}

/**
 * How the section first paints: eight questions, every answer collapsed, each
 * chevron pointing down. Nothing here is measured — the questions come from the
 * dictionary, so the locale toolbar is what decides whether a row wraps.
 */
export const Default: Story = {};

/**
 * One entry open, as it looks after a click: the chevron rotates 180° over
 * `duration-short`, and the answer hangs under the question at the summary's own
 * indent (`ps-11`, so it clears the chevron and mirrors under RTL). The entry is
 * "Does it work with an aria2 I already run?" — the answer carrying the
 * `--url ws://host:6800/jsonrpc --secret …` invocation, and the longest of the
 * eight in most locales.
 */
export const Expanded: Story = {
  play: ({ canvasElement }) => open(canvasElement, [2]),
};

/**
 * Every answer open at once. The `<details>` share no `name`, so they are
 * independent and opening one never closes another. This is the story to
 * proofread the whole FAQ in — and to sweep in a locale that runs long
 * (Русский, فارسی) for an answer that outgrows its row.
 */
export const AllExpanded: Story = {
  play: ({ canvasElement }) => open(canvasElement, [0, 1, 2, 3, 4, 5, 6, 7]),
};

/**
 * Phone width, with one long answer open. Most questions wrap to two or three
 * lines here, and the chevron stays pinned to the first line (`items-start` plus
 * `mt-0.5`) instead of centring itself against the whole block.
 */
export const Narrow: Story = {
  render: () => (
    <div style={{ maxWidth: 360 }}>
      <FaqSection />
    </div>
  ),
  play: ({ canvasElement }) => open(canvasElement, [3]),
};
