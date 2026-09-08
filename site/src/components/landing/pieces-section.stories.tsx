import type { Meta, StoryObj } from '@storybook/react-vite';
import { PiecesSection } from './pieces-section';

// "See how much is already here": the detail screen's piece map and peer list.
//
// The map fills to 100 and parks there, which is the torrent finishing - the
// peers carry on uploading, which is the claim the list beside it makes. It
// used to wrap back to 5, dropping 255 pieces to 13 every thirty-three
// seconds, and bytes on disk do not un-download.
const meta = {
  title: 'Landing/Pieces',
  component: PiecesSection,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof PiecesSection>;

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
 * 128 cells for 256 pieces, two apiece, with a six-cell frontier at the edge of
 * what is on disk - the shape `tui/internal/ui/detail.go` draws. The peer rows
 * move with the map, so the percentage in the header and the cells under it
 * never disagree.
 */
export const Band: Story = {};

/** The panel first, then its copy: the order the DOM already has. */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
