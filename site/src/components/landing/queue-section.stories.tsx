import type { Meta, StoryObj } from '@storybook/react-vite';
import { QueueSection } from './queue-section';

// "One queue for every download": the three ways something enters aria2, then
// the queue they all land in.
//
// The torrent card's boxes are real, and they share their selection with the
// file picker further down the page (`src/lib/pick-scene.ts`) - one torrent,
// one answer. Everything in that card's footer is computed from the boxes, so
// the count, the total and the saving cannot contradict them.
const meta = {
  title: 'Landing/Queue',
  component: QueueSection,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
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
 * The three inputs side by side. Card 01 is live: three mirrors of one file,
 * each moving at its own speed, totalled in the footer. Card 02 is the file
 * tree, and its boxes tick. Card 03 is aria2's own batch format.
 *
 * Below them, the queue those three feed, five downloads wide.
 */
export const Band: Story = {};

/** One column, the cards stacked in reading order. */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
