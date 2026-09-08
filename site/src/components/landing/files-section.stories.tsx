import type { Meta, StoryObj } from '@storybook/react-vite';
import { FilesSection } from './files-section';

// "Download only the files you want": the picker the extension opens before a
// torrent moves any data.
//
// The rows are checkboxes, and the whole row is the target - both a bigger one
// to hit and the only way the name and the size become the checkbox's
// accessible name. Untick everything and the commit button says so rather than
// pretending: an empty selection is the one thing aria2 will not start.
const meta = {
  title: 'Landing/Files',
  component: FilesSection,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof FilesSection>;

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
 * The picker with two of three files chosen. The count, the byte total and the
 * button's label are computed from the boxes, which is how the card stopped
 * claiming "1.5 GiB" for a 1.4 GiB selection.
 */
export const Band: Story = {};

/** The picker above its copy, at phone width. */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
