import type { Meta, StoryObj } from '@storybook/react-vite';
import { ProseCode } from './prose-code';

// The badge the reading pages put around a path, a flag or a file name inside
// a sentence. The stories are the real strings those pages pass it.
const meta = {
  title: 'Components/ProseCode',
  component: ProseCode,
  globals: { theme: 'dark' },
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <p className="max-w-[60ch] text-body-large text-on-surface-variant">
        <Story />
      </p>
    ),
  ],
} satisfies Meta<typeof ProseCode>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Two paths and a flag in one sentence, the install page's step 3. */
export const Paths: Story = {
  args: {
    text: "Configuration is stored in ~/.config/aria2t/config.json (override the path with --config); the managed daemon's session lives under ~/.config/aria2t/daemon/.",
  },
};

/** A file name at the head of a sentence, as the privacy page lists them. */
export const FileName: Story = {
  args: {
    text: 'picks.json — file selections still awaiting an answer, so the picker opens again after a restart.',
  },
};

/**
 * A sentence with nothing to mark up comes back as itself: the pages hand it
 * every string in a block, not the ones they know carry a token.
 */
export const NothingToMark: Story = {
  args: { text: 'Aria2t stores no history database or cache of downloaded content.' },
};

/**
 * The same sentence in Arabic. The path is preceded by U+200E there, and that
 * mark belongs to the prose run rather than to the badge.
 */
export const RightToLeft: Story = {
  args: {
    text: 'يوجد كل شيء تحت ‎~/.config/aria2t/ على جهازك:',
  },
  decorators: [
    (Story) => (
      <p dir="rtl" className="max-w-[60ch] text-body-large text-on-surface-variant">
        <Story />
      </p>
    ),
  ],
};
