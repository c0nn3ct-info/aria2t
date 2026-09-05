import type { Meta, StoryObj } from '@storybook/react-vite';
import { ArrowRight } from 'lucide-react';
import { Row, Stack } from '@/storybook/layout';
import { ConnectionVisual, type ConnState } from './connection-visual';

const meta = {
  title: 'M3/ConnectionVisual',
  component: ConnectionVisual,
  args: { state: 'connected' },
  tags: ['autodocs'],
} satisfies Meta<typeof ConnectionVisual>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Caption under a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

const STATES: { state: ConnState; caption: string }[] = [
  { state: 'idle', caption: 'No daemon yet' },
  { state: 'connecting', caption: 'Probing 127.0.0.1:6800' },
  { state: 'connected', caption: 'aria2c answering RPC' },
  { state: 'error', caption: 'aria2c not found' },
];

/**
 * The four RPC states, which are the whole component. `idle` breathes the
 * middle ring only and dims the other two; the three live states pulse all
 * three rings, staggered a third of a cycle apart, at their own cadence —
 * `connecting` 1.6s in primary, `connected` 3.2s in success green, `error`
 * 1.4s in error red. Under `prefers-reduced-motion` globals.css pauses the
 * rings on their first frame rather than deleting them, so the colour and the
 * core still tell the state apart.
 */
export const States: Story = {
  render: () => (
    <Row gap={24} align="flex-start">
      {STATES.map(({ state, caption }) => (
        <Stack key={state} gap={8} align="center">
          <ConnectionVisual state={state} size={132} />
          <Label>{state}</Label>
          <Label>{caption}</Label>
        </Stack>
      ))}
    </Row>
  ),
};

/**
 * Every measurement — ring diameter, core, power glyph — is derived from
 * `size`, so one component covers a 188px hero visual and a 20px inline status
 * dot without a second set of styles.
 */
export const Sizes: Story = {
  render: () => (
    <Row gap={24} align="center">
      {[20, 44, 96, 188].map((size) => (
        <Stack key={size} gap={8} align="center">
          <ConnectionVisual state="connected" size={size} />
          <Label>{`${size}px`}</Label>
        </Stack>
      ))}
    </Row>
  ),
};

/**
 * The one place the site renders it: the architecture diagram's featured
 * connector, a 20px `connected` visual pinned in front of the JSON-RPC pill
 * that links a front end to the managed daemon.
 */
export const InlineConnector: Story = {
  render: () => (
    <Row gap={4}>
      <ConnectionVisual state="connected" size={20} className="shrink-0" />
      <span className="whitespace-nowrap rounded-pill border border-success/40 bg-background px-2 py-0.5 text-label-small text-on-surface">
        JSON-RPC
      </span>
      <ArrowRight className="h-3.5 w-3.5 text-on-surface-variant" aria-hidden />
    </Row>
  ),
};

/** The controls panel drives one live visual. */
export const Playground: Story = {
  args: { state: 'connecting', size: 188 },
};
