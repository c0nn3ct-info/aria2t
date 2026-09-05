import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@/storybook/layout';
import { BrowserMock } from './browser-mock';
import { PopupMock } from './popup-mock';

const meta = {
  title: 'Blocks/BrowserMock',
  component: BrowserMock,
  // The frame exists to hold the extension popup, so that is the default child;
  // the story that renders it empty overrides this explicitly.
  args: { children: <PopupMock /> },
  tags: ['autodocs'],
} satisfies Meta<typeof BrowserMock>;

export default meta;

type Story = StoryObj<typeof meta>;

// The frame is fluid, so the width is the story. The home hero gives it two,
// and both are measured off `main` (`lg:max-w-5xl` = 1024px, `sm:px-6`):
// (1024 - 48 - gap-10) / 2.05 for the right column of `lg:grid-cols-[1.05fr_1fr]`.
const HERO_COLUMN = 456;
const STACKED_CARD = 560;

/**
 * The home hero at `lg` and up: the frame in the layout's right column, with
 * the popup hanging off the toolbar button. This is the composition
 * `pages/home.tsx` renders — the 380px popup very nearly fills a 456px window,
 * which is the intended read (a popup is large relative to its browser).
 */
export const Default: Story = {
  render: (args) => (
    <Stack style={{ width: HERO_COLUMN }}>
      <BrowserMock {...args} />
    </Stack>
  ),
};

/**
 * Below `lg` the hero stacks and the mock gets the wider `max-w-[560px]` card.
 * Same popup, more window around it — the only thing width changes is how much
 * of the abstract page shows beside the popup.
 */
export const StackedCard: Story = {
  render: (args) => (
    <Stack style={{ width: STACKED_CARD }}>
      <BrowserMock {...args} />
    </Stack>
  ),
};

/**
 * The frame with no child, which is everything the component itself draws:
 * traffic lights, reload, the locked URL pill, the pinned aria2t toolbar button
 * with its "daemon is up" dot, the pointer that anchors a popup to that button,
 * and the deliberately unreadable page behind.
 */
export const EmptyFrame: Story = {
  args: { children: null },
  render: (args) => (
    <Stack style={{ width: HERO_COLUMN }}>
      <BrowserMock {...args} />
    </Stack>
  ),
};
