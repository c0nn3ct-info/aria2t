import type { Meta, StoryObj } from '@storybook/react-vite';
import { PrivacyPage } from './privacy';

// Same shape as the other `Pages/*` files, and for the same reason: the page
// takes no props. Every string comes from the i18n singleton, every colour from
// the tokens on <html>, and there is no component state at all — the privacy
// route is prose, three `Section`s and two cards. The toolbar is the whole API:
// theme, accent and locale. `ar`/`fa` also flip `<html dir>`, which the page
// leans on more than most — the numbered "where traffic goes" list is indented
// with logical `ps-5`, and the bullet rows are `flex items-start gap-2`, so both
// mirror without a single `rtl:` override.
const meta = {
  title: 'Pages/Privacy',
  component: PrivacyPage,
  // The page renders the site's own sticky header, content column and footer
  // edge to edge; the preview's default `padded` would inset that chrome from
  // the frame and hide the fact that the header spans the full width.
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
} satisfies Meta<typeof PrivacyPage>;

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
 * The privacy route as the site ships it: the h1, the tracked uppercase
 * "Last updated" line and the lede, then the three claims the page exists to
 * make, one `Section` each — what Aria2t stores (`~/.config/aria2t/config.json`,
 * `daemon/`, `picks.json`, as a dot-bulleted list), where network traffic goes
 * (a numbered list of the only three kinds: your downloads, JSON-RPC to the
 * daemon, this website), and what it does not do (a `Ban` glyph per row, the
 * one list on the site that repeats its icon instead of a bullet). The Open
 * source and Contact cards sit two-up at the foot, above the closing Changes
 * note.
 *
 * Switch theme, accent and locale from the toolbar — the page reads all three
 * out of `<html>`, so nothing here needs to set them.
 */
export const Page: Story = {};

/**
 * The same page at phone width (Canvas tab — the viewport frame is a canvas
 * tool, so Docs always renders full width). The Open source / Contact grid
 * drops from `sm:grid-cols-2` to one column and the content column loses its
 * `max-w-3xl` slack, so this is where the list rows earn their `items-start`:
 * the config-path lines wrap to two and three lines each, and the bullet dot
 * and the `Ban` glyph stay pinned to the first line instead of centring against
 * the whole wrapped paragraph.
 */
export const Phone: Story = {
  parameters: PHONE,
  globals: { viewport: { value: 'phone', isRotated: false } },
};
