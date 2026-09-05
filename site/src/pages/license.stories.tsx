import type { Meta, StoryObj } from '@storybook/react-vite';
import { LicensePage } from './license';

// Same shape as the other `Pages/*` files: the page takes no props, pulls every
// string from the i18n singleton and every colour from the tokens on <html>.
// The toolbar covers theme, accent and locale; the stories cover width.
//
// The one structural thing worth watching here is heading depth. The Apache
// `Section` renders its header at the default `headingLevel` 2, and the four
// summaries inside it (grant, conditions, warranty, liability) are h3s — so the
// page is h1 → h2 → h3 with no level skipped, which is what `pages.test.tsx`
// and WCAG 1.3.1 are both after.
const meta = {
  title: 'Pages/License',
  component: LicensePage,
  // The page renders the site's own sticky header, content column and footer
  // edge to edge; the preview's default `padded` would inset that chrome from
  // the frame.
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
} satisfies Meta<typeof LicensePage>;

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
 * The license route as the site ships it: the plain-English summary first — a
 * numbered list of three points (free to use, patent grant included, no
 * warranty), each a bold `text-on-surface` lead-in against
 * `text-on-surface-variant` body — then the long Apache-2.0 `Section` with the
 * copyright line, the "full text governs" preamble and four h3 summaries, and a
 * closing link to the binding LICENSE file in the repository.
 *
 * The two cards at the foot are the reason the page is longer than a link to
 * GitHub: Application source, and Engine and libraries — the only place on the
 * site that names the third-party licences, aria2 under GPL-2.0 as a separate
 * process, Bubble Tea / Bubbles / Lip Gloss under MIT, Tokyo Night under
 * Apache-2.0.
 *
 * Switch theme, accent and locale from the toolbar — the page reads all three
 * out of `<html>`, so nothing here needs to set them.
 */
export const Page: Story = {};

/**
 * The same page at phone width (Canvas tab — the viewport frame is a canvas
 * tool, so Docs always renders full width). The Application source / Engine and
 * libraries grid collapses from `sm:grid-cols-2` to one column, and the two
 * `list-decimal`/`list-disc` lists keep their hanging indent off `ps-5` while
 * every item wraps to three or four lines — the densest prose on the site at
 * the narrowest width it ships at.
 */
export const Phone: Story = {
  parameters: PHONE,
  globals: { viewport: { value: 'phone', isRotated: false } },
};
