import type { Meta, StoryObj } from '@storybook/react-vite';
import { SurfacesSection } from './surfaces';

// "One daemon holds everything": the two front ends over one aria2, with the
// switch that chooses which one the band shows.
//
// Both mocks read the same queue store (`src/lib/queue-scene.ts`), so the
// figures in the popup and in the terminal list are the same figures. Flip the
// switch and watch the header speeds carry across rather than reset - that
// continuity is the band's whole argument, and two independent walks could not
// produce it.
const meta = {
  title: 'Landing/Surfaces',
  component: SurfacesSection,
  // The band draws itself edge to edge inside the page's own reading column,
  // so the preview's default `padded` would inset it and hide that.
  parameters: { layout: 'fullscreen' },
  // The landing declares its stage once on its root and every band reads it
  // off <html>: dark, with the comp's blue accent. Switch either from the
  // toolbar to see the band on the site's other palettes.
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof SurfacesSection>;

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
 * The band as the page ships it, on the extension. The popup renders at its
 * real 380x600 inside a browser window barely wider than itself, which is the
 * proportion a visitor sees on their own screen; the strip of page behind it is
 * the frame's own grey bars, because a fuller page drawn there gets sliced by
 * the popup's left edge.
 *
 * Press Terminal for the other surface: the same twelve downloads as a
 * 100-column character grid, every column and glyph matched to
 * `tui/internal/ui/list.go`.
 */
export const Band: Story = {};

/**
 * At phone width the browser frame is the wrong container - it is wider than
 * the viewport - so the popup goes alone at its real width and the strip
 * scrolls sideways, the way the terminal's does. A popup narrowed to 280 is no
 * longer the thing being shown.
 */
export const Phone: Story = {
  parameters: PHONE,
  globals: { theme: 'dark', accent: 'blue', viewport: { value: 'phone', isRotated: false } },
};
