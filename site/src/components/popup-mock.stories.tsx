import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@/storybook/layout';
import { BrowserMock } from './browser-mock';
import { PopupMock } from './popup-mock';

const meta = {
  title: 'Blocks/PopupMock',
  component: PopupMock,
  tags: ['autodocs'],
} satisfies Meta<typeof PopupMock>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The hero column the home page gives the browser frame at `lg` and up. */
const HERO_COLUMN = 456;

/**
 * The popup at its real size — 380x600, the extension's own browser-action
 * measurements, with each control drawn by the primitive the extension uses
 * (`fabVariants` for the pause-everything FAB, `iconButtonVariants` for a row's
 * pause, `buttonVariants` for View all), so the mock cannot drift from the real
 * surface by a tweak to one side only.
 *
 * It is live: the queue's speeds walk once a second and the ambient wave behind
 * the hero scrolls a rolling minute of throughput — the same 60 samples the
 * extension's service worker keeps. The opening frame is seeded by running that
 * walk forward deterministically, so the popup opens already settled instead of
 * climbing up from zero.
 */
export const Default: Story = {
  render: () => (
    <Stack align="start">
      <PopupMock />
    </Stack>
  ),
};

/**
 * The one composition the site actually renders: the popup inside the fake
 * browser, anchored under the pinned toolbar button. The hero column is only
 * 456px wide, so this is the pairing worth checking — a 380px popup has to stay
 * clear of the window's start edge and keep its pointer on the button above it.
 */
export const InBrowserChrome: Story = {
  render: () => (
    <Stack style={{ width: HERO_COLUMN }}>
      <BrowserMock>
        <PopupMock />
      </BrowserMock>
    </Stack>
  ),
};
