import type { Meta, StoryObj } from '@storybook/react-vite';
import { InstallPage } from './install';

// The whole /install route, rendered inside the site's real `Layout`.
//
// `InstallPage` takes no props: every string comes from the i18n singleton,
// every colour from the tokens `applyTheme`/`applyAccent` write onto <html>,
// and the preview's toolbar already drives all three of theme, accent and
// locale. So there is nothing worth a control here, and the one axis a story
// can still vary is the width the page is laid out at — which is the axis this
// page has rules for: `sm:grid-cols-2` on the closing cards, and eight `<pre>`
// blocks holding commands longer than a phone is wide.
const meta = {
  title: 'Pages/Install',
  component: InstallPage,
  // `fullscreen`, not the preview's `padded`: the page brings the site's own
  // sticky header, `max-w-3xl` measure and footer with it, and an inset around
  // that chrome reads as a broken sticky header rather than as a story frame.
  parameters: { layout: 'fullscreen' },
  tags: ['autodocs'],
} satisfies Meta<typeof InstallPage>;

export default meta;

type Story = StoryObj<typeof meta>;

// Declared per story rather than in `.storybook/preview`: width is this file's
// subject, not a global every component story should inherit a frame for.
const PHONE = {
  viewport: {
    options: {
      phone: { name: 'Phone', styles: { width: '390px', height: '844px' }, type: 'mobile' },
    },
  },
};

/**
 * The route as the site ships it: the *Before you start* card (aria2 itself, a
 * 256-colour terminal, Go only if you build from source), then the three
 * numbered `Section`s — install aria2 → get the binary → first run — each with
 * a copyable command per platform, the outlined *GitHub Releases* link, and the
 * Updating / Uninstalling cards two-up at the foot.
 *
 * The interactive parts are the eight code blocks and that one link. Each block
 * pins a copy `IconButton` in its `pe-12` gutter; the button swaps to a check
 * for 1.6 s and announces the change through an `aria-live` region, since
 * relabelling a button announces nothing on its own.
 *
 * Switch theme, accent and locale from the toolbar — the page reads all three
 * off `<html>`, so no story here sets them.
 */
export const Default: Story = {};

/**
 * The same page at 390 px, where the responsive rules earn their keep: the
 * closing cards drop from `sm:grid-cols-2` to one column, the measure loses its
 * `max-w-3xl` slack, and the long commands — `curl -fsSL
 * https://aria2t.c0nn3ct.info/install.sh | bash`, the two-line `git clone`
 * build — scroll inside their own `<pre>` instead of widening the page.
 *
 * The viewport frame is a canvas tool, so this is a Canvas-tab story: Docs
 * renders it at full width like the one above.
 */
export const Mobile: Story = {
  parameters: PHONE,
  globals: { viewport: { value: 'phone' } },
};
