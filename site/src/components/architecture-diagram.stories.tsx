import type { CSSProperties } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { t } from '@/i18n';
import { Stack } from '@/storybook/layout';
import { ArchitectureDiagram } from './architecture-diagram';

// The diagram takes no props: it is one fixed picture of how the product is
// wired, so its stories are contexts rather than states. Two things do vary
// under it — the locale (through the prose it illustrates) and the viewport
// (the `lg:` row/column switch) — and both are driven from outside the
// component, which is what the stories below put a handle on.
//
// Its own box labels are hardcoded rather than dictionary keys, on purpose:
// they are protocol names and the daemon's own vocabulary (JSON-RPC, HTTP(S),
// FTP, BitTorrent, DHT, Metalink), so the locale toolbar moves the copy around
// the diagram without touching the boxes inside it.

const meta = {
  title: 'Blocks/ArchitectureDiagram',
  component: ArchitectureDiagram,
  tags: ['autodocs'],
} satisfies Meta<typeof ArchitectureDiagram>;

export default meta;

type Story = StoryObj<typeof meta>;

// `max-w-2xl` on the paragraph as an inline width: the site's Tailwind `content`
// skips stories, so measurements written only here are kept out of `className`.
const PROSE: CSSProperties = { maxWidth: '42rem' };

/**
 * The picture on its own: the two front ends as peers in one labelled column,
 * the JSON-RPC hop between them and the daemon (the only connector carrying a
 * live `ConnectionVisual`), the managed aria2c daemon on your machine, then the
 * mirrors and peers it pulls from — drawn muted, because that column is the one
 * part of the row aria2t does not run.
 *
 * At `lg` and up the four groups sit in a row, as here; below it the row becomes
 * a column and each connector swaps its right arrow for a down arrow. That is a
 * viewport media query, so narrow the browser window to see it — a narrower
 * story container will not trigger it.
 */
export const Default: Story = {};

/**
 * The composition the home page actually renders (`#how-it-works`): heading and
 * body straight from the dictionary, diagram underneath as the illustration of
 * that paragraph. This is the story to check after a copy change — switch the
 * locale toolbar and read the prose the boxes have to sit beneath.
 */
export const InSection: Story = {
  render: () => (
    <Stack gap={16}>
      <h2 className="text-headline-small font-medium tracking-tight">{t('home.how.h2')}</h2>
      <p className="text-body-medium text-on-surface-variant" style={PROSE}>
        {t('home.how.body')}
      </p>
      <ArchitectureDiagram />
    </Stack>
  ),
};

/**
 * The same section in an RTL container. The heading and paragraph mirror; the
 * diagram does not, because it pins itself to `dir="ltr"` — front ends → daemon
 * → internet is a sequence of events, not a line of text, and mirroring it would
 * aim the arrows back at the front ends. The wrapper forces the case at any
 * locale; picking العربية or فارسی in the toolbar reaches the same layout
 * through `<html dir>`, with real RTL copy above it.
 */
export const RightToLeft: Story = {
  render: () => (
    <div dir="rtl">
      <Stack gap={16}>
        <h2 className="text-headline-small font-medium tracking-tight">{t('home.how.h2')}</h2>
        <p className="text-body-medium text-on-surface-variant" style={PROSE}>
          {t('home.how.body')}
        </p>
        <ArchitectureDiagram />
      </Stack>
    </div>
  ),
};
