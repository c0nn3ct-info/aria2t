import type { Meta, StoryObj } from '@storybook/react-vite';
import { StatsSection } from './stats-section';

// "See what is happening right now": a minute of throughput and the four
// figures beside it.
//
// Every value comes from one tick (`src/components/landing/use-tick.ts`), so
// the chart's last sample and the tile under it are the same number rather
// than two numbers that happen to look alike. The chart is decorative: the
// figures above it and the tiles below say the same thing in words.
const meta = {
  title: 'Landing/Stats',
  component: StatsSection,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof StatsSection>;

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
 * The panel and its tiles. Two hundred samples at 300ms is the minute the axis
 * labels claim, and the y-scale is fixed at 40 MiB/s so a rise in the line is
 * a rise in speed rather than a rescale.
 */
export const Band: Story = {};

/** Two tiles per row instead of four, and the chart at full bleed. */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
