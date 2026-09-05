import type { ComponentProps, CSSProperties, ReactNode } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@/storybook/layout';
import { fmtEta, fmtSpeed, ListMock, Row as DownloadRow } from './list-mock';

// ListMock reproduces the real list screen (tui/internal/ui/list.go) as a
// character grid: 100 columns, the same fixed STATUS/SIZE/SPEED/CONN/ETA
// widths, the same colours read from --tui-* so the mock follows the toolbar's
// theme. It takes no props but `className` — its states are the states of the
// rows it draws, which is why the exported `Row` carries most of these stories.

const meta = {
  title: 'Blocks/ListMock',
  component: ListMock,
  tags: ['autodocs'],
} satisfies Meta<typeof ListMock>;

export default meta;

type Story = StoryObj<typeof meta>;

// The terminal grid as inline style: the site's Tailwind `content` skips
// stories, so `font-mono whitespace-pre` written only here would never reach
// the shipped CSS.
const SCREEN: CSSProperties = {
  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace',
  fontSize: 12,
  lineHeight: 1.65,
  whiteSpace: 'pre',
  color: 'var(--tui-fg)',
  backgroundColor: 'var(--tui-bg)',
  border: '1px solid var(--tui-border)',
  borderRadius: 6,
  padding: '6px 10px',
  overflowX: 'auto',
};

/** Rows outside the full mock still need the terminal's grid to line up. */
function Screen({ children }: { children: ReactNode }) {
  return <div style={SCREEN}>{children}</div>;
}

/** Label above a group of rows. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

type RowProps = ComponentProps<typeof DownloadRow>;

/**
 * The list as the home page renders it. It seeds from a deterministic PRNG so
 * the prerendered frame and the first client render match, then ticks once a
 * second — speeds walk, bars creep and wrap near the end rather than parking at
 * 100%. It holds still under `navigator.webdriver`, which is how the prerender
 * and the screenshot pass capture a stable frame.
 */
export const Live: Story = {};

/**
 * Every status word and the progress cell that goes with it. `error` is drawn
 * mid-transfer on purpose: a failed download keeps the part it did fetch as a
 * red bar, and the mock's own data never reaches that state.
 */
export const RowStates: Story = {
  render: () => {
    const rows: RowProps[] = [
      {
        name: 'ubuntu-24.04.2-desktop-amd64.iso',
        status: 'active',
        pct: 35.2,
        size: '5.4 GiB',
        speed: '4.1 MiB/s',
        conn: '1',
        eta: '14m 8s',
      },
      {
        name: 'debian-13.1.0-amd64-netinst.iso',
        status: 'seeding',
        pct: 100,
        size: '680 MiB',
        speed: '-',
        conn: '0:34',
        eta: '-',
      },
      {
        name: 'raspios-bookworm-arm64-full.img.xz',
        status: 'waiting',
        pct: 0,
        size: '0 B',
        speed: '-',
        conn: '-',
        eta: '-',
      },
      {
        name: 'linuxmint-22.1-cinnamon-64bit.iso',
        status: 'paused',
        pct: 35.7,
        size: '2.8 GiB',
        speed: '-',
        conn: '-',
        eta: '-',
      },
      {
        name: 'mirrorlist-nope.iso',
        status: 'error',
        pct: 42,
        size: '1.9 GiB',
        speed: '-',
        conn: '-',
        eta: '-',
      },
      {
        name: 'gparted-live-1.7.0-amd64.iso',
        status: 'done',
        pct: 100,
        size: '527 MiB',
        speed: '-',
        conn: '-',
        eta: '-',
      },
    ];
    return (
      <Screen>
        {rows.map((r) => (
          <DownloadRow key={r.name} {...r} />
        ))}
      </Screen>
    );
  },
};

/**
 * The bar across its range: an empty track, the `╸` cap that marks the head of
 * a partial transfer, and a solid bar at the top. The percent and the ETA are
 * computed by the mock's own formatters, so they stay consistent with the bar.
 */
export const ProgressRamp: Story = {
  render: () => {
    const bytes = 5.4 * 1073741824;
    const bps = 4_300_000;
    return (
      <Screen>
        {[0, 1, 17, 50, 99, 100].map((pct) => (
          <DownloadRow
            key={pct}
            name="ubuntu-24.04.2-desktop-amd64.iso"
            status="active"
            pct={pct}
            size="5.4 GiB"
            speed={fmtSpeed(bps)}
            conn="1"
            eta={fmtEta(bytes * (1 - pct / 100), bps)}
          />
        ))}
      </Screen>
    );
  },
};

/**
 * The cursor row. Selection is the screen's one affordance: a `▸` marker, the
 * name in the bright foreground, and the whole line banded in `--tui-sel`.
 */
export const Selection: Story = {
  render: () => {
    const row: RowProps = {
      name: 'fedora-workstation-42-x86_64.iso',
      status: 'active',
      pct: 68.4,
      size: '2.3 GiB',
      speed: '2.5 MiB/s',
      conn: '1',
      eta: '4m 51s',
    };
    return (
      <Stack gap={16}>
        <Stack gap={6}>
          <Label>selected</Label>
          <Screen>
            <DownloadRow {...row} selected />
          </Screen>
        </Stack>
        <Stack gap={6}>
          <Label>not selected</Label>
          <Screen>
            <DownloadRow {...row} />
          </Screen>
        </Stack>
      </Stack>
    );
  },
};

/**
 * The list is sized off a container query (`min(12px, 1.64cqw)`), so a narrow
 * hero shrinks the type instead of wrapping a row — a wrapped row would break
 * the column grid. Below roughly 730px every row shrinks together; above it the
 * type parks at 12px.
 */
export const Widths: Story = {
  render: () => (
    <Stack gap={16}>
      <Stack gap={6}>
        <Label>420px — phone hero</Label>
        <Stack style={{ maxWidth: 420 }}>
          <ListMock />
        </Stack>
      </Stack>
      <Stack gap={6}>
        <Label>760px — desktop hero</Label>
        <Stack style={{ maxWidth: 760 }}>
          <ListMock />
        </Stack>
      </Stack>
    </Stack>
  ),
};
