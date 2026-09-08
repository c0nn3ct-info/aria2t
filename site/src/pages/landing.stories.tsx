import type { Meta, StoryObj } from '@storybook/react-vite';
import { LandingPage } from './landing';

// The home page, whole: the hero, the two surfaces over one daemon, the three
// ways into the queue, a live minute of throughput, the file picker, the piece
// map, the limits panel and the FAQ.
//
// It takes no props, like every other `Pages/*` file. Blue is the site's
// permanent accent (set once in `main.tsx`, outside this component), and the
// page follows the theme toolbar like every other `Pages/*` story; the locale
// toolbar moves it too, and `ar`/`fa` also flip `<html dir>`, which every band
// mirrors without a `rtl:` override except the four places a glyph or a scene
// has to turn around.
//
// Bands arrive as they come up, on a `view()` timeline where the browser has
// one. Scroll the canvas rather than the Docs page to see that.
const meta = {
  title: 'Pages/Landing',
  component: LandingPage,
  // The page is edge to edge from the header to the footer.
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
} satisfies Meta<typeof LandingPage>;

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
 * The page as it ships. Two live mocks read one queue store, so the popup and
 * the terminal list report the same daemon; the stats band, the piece map and
 * the mirror card run off one tick; and the file picker and the limits panel
 * are controls you can actually work.
 *
 * Switch locale from the toolbar - all six are translated, including the two
 * that read right to left.
 */
export const Page: Story = {};

/**
 * The same page at phone width. Every two-column band collapses to one, the
 * browser frame gives way to the popup at its real size in a sideways-scrolling
 * strip, and the terminal keeps its 620px so its columns stay aligned rather
 * than reflowing its type down to six pixels.
 */
export const Phone: Story = {
  parameters: PHONE,
  globals: { viewport: { value: 'phone', isRotated: false } },
};
