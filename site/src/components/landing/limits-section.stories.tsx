import type { Meta, StoryObj } from '@storybook/react-vite';
import { LimitsSection } from './limits-section';

// "Leave bandwidth for everything else": the per-download presets, the global
// cap and the scheduler that moves it by time of day.
//
// All three are controls rather than pictures of controls. The slider is a
// range input under three spans that carry the look, which is what makes it
// keyboard-operable and correct in both directions without a vendor
// pseudo-element in sight. Nothing here animates on its own: a limit that
// drifted would say the opposite of what the band claims.
const meta = {
  title: 'Landing/Limits',
  component: LimitsSection,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof LimitsSection>;

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
 * Pick a preset, drag the global cap (its far end is no limit at all, the empty
 * string aria2 takes for unlimited), and switch the schedule off to watch the
 * window's bars released to their unthrottled heights.
 */
export const Band: Story = {};

/** The panel under its copy, controls at the same size. */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
