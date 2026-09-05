import type { Meta, StoryObj } from '@storybook/react-vite';
import { ArrowUp, Check, Clock, Magnet, Pause, TriangleAlert } from 'lucide-react';
import { Row, Stack } from '@/storybook/layout';
import { Badge } from './badge';

const meta = {
  title: 'Primitives/Badge',
  component: Badge,
  args: { children: 'active' },
  tags: ['autodocs'],
} satisfies Meta<typeof Badge>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Label above a row of samples. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/** The eight variants, each carrying the download state it is meant to colour. */
export const Variants: Story = {
  render: () => (
    <Row>
      <Badge variant="default">waiting</Badge>
      <Badge variant="primary">active</Badge>
      <Badge variant="outline">paused</Badge>
      <Badge variant="success">done</Badge>
      <Badge variant="warning">seeding</Badge>
      <Badge variant="info">metadata</Badge>
      <Badge variant="destructive">error</Badge>
      <Badge variant="mono">sha-256</Badge>
    </Row>
  ),
};

/**
 * Both sizes. `md` is the default and the only one the site renders today;
 * `sm` is the 20px pill for a badge sitting inside a dense list row.
 */
export const Sizes: Story = {
  render: () => (
    <Stack gap={16}>
      <Stack gap={8}>
        <Label>sm</Label>
        <Row>
          <Badge size="sm" variant="primary">
            active
          </Badge>
          <Badge size="sm" variant="success">
            done
          </Badge>
          <Badge size="sm" variant="mono">
            6800
          </Badge>
        </Row>
      </Stack>
      <Stack gap={8}>
        <Label>md</Label>
        <Row>
          <Badge size="md" variant="primary">
            active
          </Badge>
          <Badge size="md" variant="success">
            done
          </Badge>
          <Badge size="md" variant="mono">
            6800
          </Badge>
        </Row>
      </Stack>
    </Stack>
  ),
};

/**
 * The base recipe carries `gap-1`, so an icon is just a first child. It has no
 * `[&_svg]` sizing of its own — pass lucide's `size` so the glyph matches the
 * label rather than the 24px default.
 */
export const WithIcons: Story = {
  render: () => (
    <Row>
      <Badge variant="primary">
        <ArrowUp size={12} /> 4.2 MiB/s
      </Badge>
      <Badge variant="success">
        <Check size={12} /> verified
      </Badge>
      <Badge variant="default">
        <Clock size={12} /> queued
      </Badge>
      <Badge variant="outline">
        <Pause size={12} /> paused
      </Badge>
      <Badge variant="destructive">
        <TriangleAlert size={12} /> tracker unreachable
      </Badge>
      <Badge variant="mono">
        <Magnet size={12} /> magnet:?xt=urn:btih:
      </Badge>
    </Row>
  ),
};

/**
 * The one composition the site renders: the home page's "works with" strip.
 * Note it is `outline` plus `font-mono`, not the `mono` variant — the strip
 * wants the mono face on a hairline pill, not on a filled container.
 */
export const SourceTags: Story = {
  render: () => (
    <Row gap={8}>
      {[
        'HTTP(S)',
        'FTP',
        'SFTP',
        'BitTorrent',
        'magnet:',
        '.torrent',
        'Metalink',
        'aria2 input file',
      ].map((source) => (
        <Badge key={source} variant="outline" size="md" className="font-mono">
          {source}
        </Badge>
      ))}
    </Row>
  ),
};

/** The controls panel drives one live badge. */
export const Playground: Story = {
  args: { variant: 'default', size: 'md' },
};
