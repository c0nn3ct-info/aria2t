import type { Meta, StoryObj } from '@storybook/react-vite';
import { Download, Info, MonitorCheck, Wrench } from 'lucide-react';
import { Grid, Row, Stack } from '@/storybook/layout';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './card';

const meta = {
  title: 'Primitives/Card',
  component: Card,
  args: {
    children: (
      <CardHeader>
        <CardTitle>ubuntu-24.04.1-desktop-amd64.iso</CardTitle>
        <CardDescription>5.7 GiB · 12.4 MiB/s · 6 min left</CardDescription>
      </CardHeader>
    ),
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Card>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Label above a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/** A card's body, so each surface in a line-up has something to hold. */
function Sample({ title, body }: { title: string; body: string }) {
  return (
    <CardHeader>
      <CardTitle>{title}</CardTitle>
      <CardDescription>{body}</CardDescription>
    </CardHeader>
  );
}

/**
 * All five surfaces. `elevated` is the default; `filled` and `outlined` are
 * what the pages actually reach for. `accent` reads the `--dir-*` custom
 * properties, which only exist under a `dir-proxy`/`dir-direct`/`dir-block`
 * ancestor — see the AccentDirection story — so it is shown wrapped in one.
 */
export const Variants: Story = {
  render: () => (
    <Grid columns={2} gap={16} align="stretch">
      <Card variant="elevated">
        <Sample title="Elevated" body="The default — a low container under the e1 shadow." />
      </Card>
      <Card variant="filled">
        <Sample title="Filled" body="No shadow, a high container — the install callouts." />
      </Card>
      <Card variant="outlined">
        <Sample title="Outlined" body="Surface plus a hairline — the Privacy notes." />
      </Card>
      <Card variant="tonal">
        <Sample title="Tonal" body="Primary container, for a panel that carries colour." />
      </Card>
      <div className="dir-proxy">
        <Card variant="accent">
          <Sample title="Accent" body="Container colour from the nearest dir-* ancestor." />
        </Card>
      </div>
    </Grid>
  ),
};

/**
 * The padding ramp. `none` is for a card that frames content owning its own
 * insets — a code block, a table, an image that should reach the radius.
 */
export const Padding: Story = {
  render: () => (
    <Grid columns={2} gap={16} align="stretch">
      <Card variant="outlined" padding="none">
        <div style={{ padding: '12px 16px' }} className="text-label-medium text-on-surface-variant">
          padding=&quot;none&quot;
        </div>
        <div
          style={{ padding: '12px 16px' }}
          className="border-t border-outline-variant font-mono text-body-medium"
        >
          aria2t --url ws://seedbox.local:6800
        </div>
      </Card>
      <Card variant="outlined" padding="sm">
        <Sample title="sm" body="p-4 — a compact row in a list of servers." />
      </Card>
      <Card variant="outlined" padding="md">
        <Sample title="md" body="p-5 — the default, and what every page uses." />
      </Card>
      <Card variant="outlined" padding="lg">
        <Sample title="lg" body="p-6 — a card that stands alone on the page." />
      </Card>
    </Grid>
  ),
};

/**
 * Every part in one card. `CardContent` and `CardFooter` bring their own top
 * margin, so a composition needs no spacing between the pieces.
 */
export const Anatomy: Story = {
  render: () => (
    <div style={{ maxWidth: 420 }}>
      <Card variant="elevated" padding="md">
        <CardHeader>
          <Badge variant="success" size="sm">
            connected
          </Badge>
          <CardTitle>Seedbox</CardTitle>
          <CardDescription>ws://seedbox.local:6800</CardDescription>
        </CardHeader>
        <CardContent>
          <Stack gap={6}>
            <Row gap={8} align="baseline">
              <span className="text-label-medium text-on-surface-variant">Active</span>
              <span className="text-body-medium tabular-nums">3 downloads</span>
            </Row>
            <Row gap={8} align="baseline">
              <span className="text-label-medium text-on-surface-variant">Down</span>
              <span className="text-body-medium tabular-nums">12.4 MiB/s</span>
            </Row>
            <Row gap={8} align="baseline">
              <span className="text-label-medium text-on-surface-variant">Up</span>
              <span className="text-body-medium tabular-nums">1.1 MiB/s</span>
            </Row>
          </Stack>
        </CardContent>
        <CardFooter>
          <Button variant="filled" size="xs">
            Connect
          </Button>
          <Button variant="text" size="xs">
            Edit
          </Button>
        </CardFooter>
      </Card>
    </div>
  ),
};

/**
 * `accent` is the only variant whose colour comes from outside the recipe: it
 * paints `--dir-container`, which `.dir-proxy` / `.dir-direct` / `.dir-block`
 * set. One card, three ancestors.
 */
export const AccentDirection: Story = {
  render: () => (
    <Grid columns={3} gap={16} align="stretch">
      {(
        [
          ['dir-proxy', 'Through the managed daemon'],
          ['dir-direct', 'Straight to the mirror'],
          ['dir-block', 'Refused by the tracker'],
        ] as const
      ).map(([klass, body]) => (
        <Stack key={klass} gap={8}>
          <Label>{klass}</Label>
          <div className={klass}>
            <Card variant="accent" padding="sm">
              <CardTitle>{body}</CardTitle>
            </Card>
          </div>
        </Stack>
      ))}
    </Grid>
  ),
};

/**
 * The two shapes the site really renders: the filled checklist that opens the
 * Install page, and the outlined note that closes Extension and Privacy.
 */
export const SitePanels: Story = {
  render: () => (
    <Stack gap={16} style={{ maxWidth: 560 }}>
      <Card variant="filled" padding="md">
        <CardHeader>
          <CardTitle>
            <Row gap={8}>
              <Info size={16} /> Before you start
            </Row>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Stack gap={8}>
            <Row gap={8} align="flex-start">
              <Download size={16} />
              <span className="text-body-medium text-on-surface-variant">
                aria2 does the downloading — aria2t drives it.
              </span>
            </Row>
            <Row gap={8} align="flex-start">
              <MonitorCheck size={16} />
              <span className="text-body-medium text-on-surface-variant">
                A terminal that speaks 256 colours and reports a mouse.
              </span>
            </Row>
            <Row gap={8} align="flex-start">
              <Wrench size={16} />
              <span className="text-body-medium text-on-surface-variant">
                Go 1.22+, only if you build from source.
              </span>
            </Row>
          </Stack>
        </CardContent>
      </Card>

      <Card variant="outlined" padding="md">
        <CardHeader>
          <Badge variant="outline" size="sm">
            Chrome, Edge, Brave
          </Badge>
          <CardTitle>Which browsers work</CardTitle>
          <CardDescription>
            Any Chromium browser with native messaging. A snap or Flatpak install cannot reach the
            host — the sandbox hides /usr/local/bin.
          </CardDescription>
        </CardHeader>
      </Card>
    </Stack>
  ),
};

/** The controls panel drives one live card. */
export const Playground: Story = {
  args: { variant: 'elevated', padding: 'md' },
};
