import type { Meta, StoryObj } from '@storybook/react-vite';
import { QueueSection } from './queue-section';

// "One queue for every download": the routes as a legend down the left column,
// the queue as one list beside them, every row carrying the route it arrived
// by.
const meta = {
  title: 'Landing/Queue',
  component: QueueSection,
  // The band lays itself out inside the page's reading column, so the
  // preview's default `padded` would inset it twice.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage on <html> and every band reads it from
  // there: dark, with the site's blue accent.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof QueueSection>;

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
