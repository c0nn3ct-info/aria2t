import type { Meta, StoryObj } from '@storybook/react-vite';
import { LandingHero } from './hero';

// The hero: the page's claim over a WebGL production line.
//
// The scene is one composition at every size, and the copy sits on it behind
// washes made of the page's own ground at an alpha. It aims itself: the hero
// measures its heading's first line and the scene bisects its camera aim until
// the pylon crowns land on it, which is why the tips track the heading through
// every wrap and every language.
//
// The scene needs WebGL. Where the browser has none the gradients stand alone,
// and where a frame costs more than the frame it draws - a machine rendering in
// software - the loop stops and leaves a still.
const meta = {
  title: 'Landing/Hero',
  component: LandingHero,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof LandingHero>;

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
 * The hero at desktop width: the copy in a column on the left, the machinery to
 * its right, the belt running out of the bottom-left corner under the source
 * chips.
 */
export const Band: Story = {};

/**
 * At phone width the same composition holds, with the wash turned to run top to
 * bottom instead of across. The scene is not a separate mobile drawing - it is
 * the same one, framed by an aspect-interpolated camera.
 */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
