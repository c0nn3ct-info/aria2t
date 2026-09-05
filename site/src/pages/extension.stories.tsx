import type { Meta, StoryObj } from '@storybook/react-vite';
import { ExtensionPage } from './extension';

// The whole /extension route, rendered inside the site's real `Layout`. Same
// shape as `install.stories.tsx`, and for the same reason: the page takes no
// props, reads its copy from the i18n singleton and its colours from the tokens
// on <html>. The toolbar covers theme, accent and locale; these stories cover
// the width, which is what the page's own rules react to.
const meta = {
  title: 'Pages/Extension',
  component: ExtensionPage,
  // `fullscreen`, not the preview's `padded`: the page carries the site's
  // sticky header, measure and footer, and an inset around that chrome reads as
  // a broken header rather than as a story frame.
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
} satisfies Meta<typeof ExtensionPage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Declared per story, for the same reason as in `install.stories.tsx`. */
const PHONE = {
  viewport: {
    options: {
      phone: { name: 'Phone', styles: { width: '390px', height: '844px' }, type: 'mobile' },
    },
  },
};

/**
 * The route as the site ships it: the *What it does* card (capture magnets and
 * large browser downloads, pick torrent files before the bytes move, reach an
 * aria2 on a seedbox or NAS), then the three `Section`s — add the extension
 * from the Web Store → install the native host → reload the extension — and the
 * Updating / Several browsers cards two-up at the foot.
 *
 * Three things are interactive: the outlined Web Store button (an anchor
 * rendered `asChild`, with an external-link glyph) and the copy buttons on the
 * two host-install commands, which swap to a check for 1.6 s and announce it
 * through an `aria-live` region.
 *
 * The body deliberately carries no "source on GitHub" link, unlike /install's
 * Releases button: the extension is not mirrored to the public repo and ships
 * through the Web Store, with the `aria2t` binary itself acting as the native
 * host. The only GitHub link on screen is the site-wide one in the header.
 */
export const Default: Story = {};

/**
 * The same page at 390 px. The closing cards drop from `sm:grid-cols-2` to one
 * column, and the two host commands — including
 * `$env:ARIA2T_EXT_ID='<extension-id>'; iwr -useb
 * https://aria2t.c0nn3ct.info/windows.ps1 | iex`, the longest string on the
 * site — scroll inside their own `<pre>` with the copy button still pinned to
 * the `pe-12` gutter rather than sliding out of reach with the text.
 *
 * The viewport frame is a canvas tool, so this is a Canvas-tab story: Docs
 * renders it at full width like the one above.
 */
export const Mobile: Story = {
  parameters: PHONE,
  globals: { viewport: { value: 'phone' } },
};
