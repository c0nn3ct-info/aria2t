import type { Meta, StoryObj } from '@storybook/react-vite';
import { ArrowDown, ArrowUp, Download, Pause, Plus, X } from 'lucide-react';
import { Row, Stack } from '@/storybook/layout';
import { Card } from './card';
import { Fab } from './fab';

const meta = {
  title: 'Primitives/Fab',
  component: Fab,
  // A FAB is an icon and nothing else, so the label is not optional decoration:
  // without `aria-label` the control has no accessible name at all.
  args: { children: <Pause />, 'aria-label': 'Pause all downloads' },
  tags: ['autodocs'],
} satisfies Meta<typeof Fab>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Label under a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/**
 * The six container colours. The popup hero picks by meaning rather than by
 * decoration — `success` carries pause-all while the queue is running, `error`
 * the action that throws a transfer away.
 */
export const Colors: Story = {
  render: () => (
    <Row gap={20} align="start">
      <Stack gap={8} align="center">
        <Fab color="primary" aria-label="Add a download">
          <Plus />
        </Fab>
        <Label>primary</Label>
      </Stack>
      <Stack gap={8} align="center">
        <Fab color="surface" aria-label="Download">
          <Download />
        </Fab>
        <Label>surface</Label>
      </Stack>
      <Stack gap={8} align="center">
        <Fab color="secondary" aria-label="Resume all downloads">
          <ArrowDown />
        </Fab>
        <Label>secondary</Label>
      </Stack>
      <Stack gap={8} align="center">
        <Fab color="tertiary" aria-label="Seed">
          <ArrowUp />
        </Fab>
        <Label>tertiary</Label>
      </Stack>
      <Stack gap={8} align="center">
        <Fab color="success" aria-label="Pause all downloads">
          <Pause />
        </Fab>
        <Label>success</Label>
      </Stack>
      <Stack gap={8} align="center">
        <Fab color="error" aria-label="Remove download">
          <X />
        </Fab>
        <Label>error</Label>
      </Stack>
    </Row>
  ),
};

/** Three sizes; the icon scales with the shell. `regular` is the default. */
export const Sizes: Story = {
  render: () => (
    <Row gap={20} align="end">
      <Stack gap={8} align="center">
        <Fab size="small" aria-label="Pause all downloads">
          <Pause />
        </Fab>
        <Label>small</Label>
      </Stack>
      <Stack gap={8} align="center">
        <Fab size="regular" aria-label="Pause all downloads">
          <Pause />
        </Fab>
        <Label>regular</Label>
      </Stack>
      <Stack gap={8} align="center">
        <Fab size="large" aria-label="Pause all downloads">
          <Pause />
        </Fab>
        <Label>large</Label>
      </Stack>
    </Row>
  ),
};

/**
 * The one composition the site renders: the extension popup's hero, where the
 * FAB is the whole-queue action beside the transfer totals. Green because the
 * button stops what is running — the hero's own colour rule.
 */
export const InTheHero: Story = {
  render: () => (
    // Card and the type classes are the popup mock's own (popup-mock.tsx), so
    // this reads at the measurements the real hero ships with.
    <Card variant="elevated" padding="md" style={{ width: 340 }}>
      <Row gap={16} style={{ justifyContent: 'space-between', flexWrap: 'nowrap' }}>
        <Stack gap={8}>
          <div className="text-label-small uppercase tracking-[0.16em] text-on-surface-variant">
            aria2t
          </div>
          <div className="text-headline-small font-medium leading-tight tracking-tight">
            3 downloading
          </div>
          <Row gap={12}>
            <span className="inline-flex items-center gap-1 text-label-medium tabular-nums text-primary">
              <ArrowDown className="h-3 w-3" aria-hidden />
              <span dir="ltr">13.6 MiB/s</span>
            </span>
            <span className="inline-flex items-center gap-1 text-label-medium tabular-nums text-on-surface-variant">
              <ArrowUp className="h-3 w-3" aria-hidden />
              <span dir="ltr">3.9 MiB/s</span>
            </span>
          </Row>
        </Stack>
        <Fab color="success" aria-label="Pause all downloads">
          <Pause />
        </Fab>
      </Row>
    </Card>
  ),
};

/** Disabled drops to 50% and stops taking pointer events — nothing to pause. */
export const Disabled: Story = {
  render: () => (
    <Row gap={20}>
      <Fab color="success" disabled aria-label="Pause all downloads">
        <Pause />
      </Fab>
      <Fab color="primary" disabled aria-label="Add a download">
        <Plus />
      </Fab>
    </Row>
  ),
};

/** `asChild` hands the shape to an anchor, for a FAB that navigates. */
export const AsLink: Story = {
  render: () => (
    <Fab asChild color="primary">
      <a href="#install" aria-label="Install aria2t">
        <Download />
      </a>
    </Fab>
  ),
};

/** The controls panel drives one live FAB. */
export const Playground: Story = {
  args: { color: 'primary', size: 'regular' },
};
