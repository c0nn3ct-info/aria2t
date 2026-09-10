import type { Meta, StoryObj } from '@storybook/react-vite';
import { QueueB2Section } from './queue-b2-section';

// "One queue for every download", the split-spine draw: the routes as a legend
// down a narrow rail, the queue as one ruled list beside them.
//
// This is the draw the page ships. `Landing/Queue` is the one it replaced -
// same headline, same body, same five downloads, same strings, different
// composition - and is kept as the band's previous shape; the difference
// between the two stories is the shape, never the words.
const meta = {
  title: 'Landing/Queue B2',
  component: QueueB2Section,
  // The band lays itself out inside the page's reading column, so the
  // preview's default `padded` would inset it twice.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage on <html> and every band reads it from
  // there: dark, with the site's blue accent.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof QueueB2Section>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A phone-sized preview frame, defined per story rather than globally. */
const PHONE = {
  viewport: {
    options: {
      phone: { name: 'Phone', styles: { width: '390px', height: '844px' }, type: 'mobile' },
    },
  },
};

/**
 * The rail on the left, the queue on the right. Every row carries the route it
 * arrived by - `link`, `torrent`, `magnet`, `input` - which is what makes the
 * headline's claim visible in the object rather than in the copy above it.
 */
export const Band: Story = {};

/**
 * The rail stacks over the queue, and each row folds to two lines: name,
 * status and progress on the first, the figures on the second. Six columns
 * never survive 390px, and a sideways scroll would be worse than a fold.
 */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
