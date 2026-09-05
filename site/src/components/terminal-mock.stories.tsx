import type { CSSProperties } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@/storybook/layout';
import { ListMock } from './list-mock';
import { TerminalMock } from './terminal-mock';

// TerminalMock is chrome and nothing else: a title bar with three traffic
// lights and a padded body painted from the app's own --tui-* palette (Tokyo
// Night Day in light mode, Tokyo Night in dark), so it follows the toolbar's
// theme the way the real terminal follows its setting. Every story here is
// therefore about what the frame does to whatever it wraps — the frame has no
// state of its own to enumerate.

// Terminal type as inline style, not `font-mono whitespace-pre`: the site's
// Tailwind `content` skips stories, so a utility written only here would not
// reach the shipped CSS.
const MONO: CSSProperties = {
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
  fontSize: 12,
  lineHeight: 1.65,
  whiteSpace: 'pre',
  color: 'var(--tui-fg)',
};

/** Plain terminal output — the frame's children are ordinary nodes, not a widget. */
function Session() {
  return (
    <div style={MONO}>
      <div>
        <span style={{ color: 'var(--tui-green)' }}>$ </span>
        curl -fsSL https://aria2t.c0nn3ct.info/install.sh | bash
      </div>
      <div style={{ color: 'var(--tui-fg-dim)' }}>
        aria2c 1.37.0 found at /opt/homebrew/bin/aria2c
      </div>
      <div style={{ color: 'var(--tui-fg-dim)' }}>installed aria2t to /usr/local/bin/aria2t</div>
      <div>
        <span style={{ color: 'var(--tui-green)' }}>$ </span>
        aria2t
        <span style={{ backgroundColor: 'var(--tui-fg)' }}> </span>
      </div>
    </div>
  );
}

const meta = {
  title: 'Blocks/TerminalMock',
  component: TerminalMock,
  args: { children: <Session /> },
  tags: ['autodocs'],
} satisfies Meta<typeof TerminalMock>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The frame as the site ships it: default title, traffic lights, themed body. */
export const Default: Story = {};

/**
 * The home hero's composition — the live TUI list inside the frame. The list
 * ticks once a second after mount, so this story moves; the frozen row states
 * live in `Blocks/ListMock`.
 */
export const WithDownloadList: Story = {
  render: () => (
    <TerminalMock>
      <ListMock />
    </TerminalMock>
  ),
};

/** `title` names the window — an external aria2 endpoint instead of a local path. */
export const CustomTitle: Story = {
  args: { title: 'Aria2t — ws://seedbox:6800/jsonrpc' },
};

/**
 * At the width the hero drops to on a phone. The frame clips to its rounded
 * border while the list inside scales its type off a container query, so the
 * 100-column grid stays intact instead of wrapping rows.
 */
export const Narrow: Story = {
  render: () => (
    <Stack style={{ maxWidth: 380 }}>
      <TerminalMock>
        <ListMock />
      </TerminalMock>
    </Stack>
  ),
};
