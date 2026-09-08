import type { Meta, StoryObj } from '@storybook/react-vite';
import { FaqBand } from './faq-band';

// The closing band: the eight questions the site already answers, beside the
// two ways to ask a ninth.
//
// The questions are `FaqList` in its flush frame - the same component and the
// same strings the site has always shipped, which is why they cannot drift
// from the answers anywhere else.
const meta = {
  title: 'Landing/FAQ',
  component: FaqBand,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof FaqBand>;

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

/** The two columns: the ask-us block on the left, the accordion on the right,
 * with the first answer open. */
export const Band: Story = {};

/** One column, the accordion under the contact links. */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
