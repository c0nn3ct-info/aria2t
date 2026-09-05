import type { ReactNode } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { userEvent, within } from 'storybook/test';
import { Row, Stack } from '@/storybook/layout';
import { Aria2tLogo } from './aria2t-logo';
import { GithubLink } from './github-link';
import { LanguageSwitcher } from './language-switcher';

const meta = {
  title: 'Blocks/LanguageSwitcher',
  component: LanguageSwitcher,
  tags: ['autodocs'],
} satisfies Meta<typeof LanguageSwitcher>;

export default meta;

type Story = StoryObj<typeof meta>;

// The menu is local state with no `open` prop, so a story reaches it the way a
// reader does — by clicking the trigger. Nothing clicks a menu *item*: those
// are real links, and following one would navigate the preview iframe.
async function openMenu(canvasElement: HTMLElement) {
  await userEvent.click(within(canvasElement).getByRole('button'));
}

/** Enough room under the trigger for the menu to hang without clipping. */
function Anchored({ children }: { children: ReactNode }) {
  return <Stack style={{ minHeight: 300 }}>{children}</Stack>;
}

/**
 * Resting state: one `standard` icon button, `aria-expanded="false"`, no menu
 * in the DOM at all. Its label and tooltip come from `nav.lang_switch_aria`, so
 * the toolbar's Locale is what translates them — the button itself never
 * changes shape.
 */
export const Closed: Story = {
  render: () => <LanguageSwitcher />,
};

/**
 * The menu open, listing all six shipped languages in their own script
 * (English, Русский, 中文, Español, العربية, فارسی) — never translated names,
 * because someone looking for their language is looking for the word they
 * know.
 *
 * The current one is marked twice over: `aria-current="true"` for a screen
 * reader, `text-on-surface font-medium` against the others'
 * `text-on-surface-variant` for everyone else. Switch the toolbar's Locale and
 * the mark moves with it — the switcher reads the i18n singleton the preview
 * sets, it holds no locale of its own.
 *
 * Each item is a real link carrying its own `hreflang`, paired to the current
 * page rather than dumped at the home page: `/install/` → `/ru/install/`, with
 * English staying at the unprefixed root. Here that pairing runs against
 * Storybook's own path, so the hrefs read `/ru/iframe.html`.
 */
export const Open: Story = {
  render: () => (
    <Anchored>
      <LanguageSwitcher />
    </Anchored>
  ),
  play: ({ canvasElement }) => openMenu(canvasElement),
};

/**
 * The composition the site renders, with the menu down: wordmark at the start
 * of the bar, the two nav actions at the end. The menu is anchored `end-0`, not
 * `right-0`, so it hangs from the trigger's outer edge and stays inside the
 * bar — and in العربية or فارسی the whole row mirrors and the menu flips to the
 * left, which is the case worth stepping through with the Locale toolbar.
 */
export const InTheHeaderBar: Story = {
  render: () => (
    <Anchored>
      <div
        className="bg-surface-container-low"
        style={{
          borderBottom: '1px solid hsl(var(--outline-variant))',
          padding: '0 16px',
          height: 64,
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}
      >
        <Row gap={8}>
          <div className="text-primary" style={{ width: 24, height: 24 }}>
            <Aria2tLogo className="h-full w-full" />
          </div>
          <span className="text-title-medium tracking-tight text-on-surface">Aria2t</span>
        </Row>
        <Row gap={4} style={{ marginInlineStart: 'auto' }}>
          <GithubLink />
          <LanguageSwitcher />
        </Row>
      </div>
    </Anchored>
  ),
  play: ({ canvasElement }) => openMenu(canvasElement),
};
