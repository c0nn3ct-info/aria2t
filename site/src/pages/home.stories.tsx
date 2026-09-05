import type { Meta, StoryObj } from '@storybook/react-vite';
import { HomePage } from './home';

// The landing page end to end, inside the real `Layout`: the hero and its
// surface switch (extension popup <-> terminal list), the source badges, the
// "what you get" list, the architecture diagram, the FAQ and the closing CTA.
//
// `HomePage` takes no props — the hero's surface is component state and every
// string comes from the i18n singleton — so there is nothing to put in the
// controls panel. The toolbar is the whole API: theme, accent and locale are
// read off `<html>`, and `ar`/`fa` also flip `<html dir>`, which mirrors the
// hero arrows and every logical padding on the page.
const meta = {
  title: 'Pages/Home',
  component: HomePage,
  // The page renders the site's own sticky header, content column and footer
  // edge to edge; the preview's default `padded` would inset that chrome and
  // hide the fact that the header spans the full width.
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
} satisfies Meta<typeof HomePage>;

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
 * The landing page as the site ships it. Above `lg` the hero is two columns
 * with the surface switch and its mock on the right; the switch swaps the
 * browser popup for the terminal download list while both CTAs stay put,
 * because they are two different things to get rather than two views of one.
 * The store button is deliberately disabled — the listing is not live, and a
 * CTA that 404s is worse than one that plainly cannot be pressed yet.
 */
export const Page: Story = {};

/**
 * The same page at phone width (Canvas tab — the viewport frame is a canvas
 * tool, so Docs always renders full width). Below `lg` the hero collapses to
 * one column and the switch plus mock move inline between the h1 and the lede:
 * the markup carries both placements and hides one, so this is the only story
 * that shows the second. The source badges wrap over several rows, and the
 * terminal list — sized off a container query, `min(12px, 1.64cqw)` — shrinks
 * its monospace grid to fit rather than wrapping a row or widening the page.
 */
export const Phone: Story = {
  parameters: PHONE,
  globals: { viewport: { value: 'phone', isRotated: false } },
};
