import type { Meta, StoryObj } from '@storybook/react-vite';
import { ArrowRight, Download, Puzzle, Terminal } from 'lucide-react';
import { Row, Stack } from '@/storybook/layout';
import { Button } from './button';

const meta = {
  title: 'Primitives/Button',
  component: Button,
  args: { children: 'Download' },
  tags: ['autodocs'],
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Label above a row of samples. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

export const Variants: Story = {
  render: () => (
    <Row>
      <Button variant="filled">Install</Button>
      <Button variant="filled-tonal">Get the extension</Button>
      <Button variant="outlined">Read the docs</Button>
      <Button variant="text">Skip</Button>
      <Button variant="elevated">Download</Button>
      <Button variant="destructive">Remove</Button>
      <Button variant="ghost">Later</Button>
    </Row>
  ),
};

/** The size ramp the landing blocks pick from — `s` is the default. */
export const Sizes: Story = {
  render: () => (
    <Row>
      <Button size="xs">Extra small</Button>
      <Button size="s">Small</Button>
      <Button size="m">Medium</Button>
      <Button size="l">Large</Button>
      <Button size="xl">Extra large</Button>
    </Row>
  ),
};

/** `square` swaps the pill radius for the size's cornered shape. */
export const Shapes: Story = {
  render: () => (
    <Stack gap={16}>
      <Stack gap={8}>
        <Label>round</Label>
        <Row>
          <Button shape="round" size="xs">
            Extra small
          </Button>
          <Button shape="round" size="s">
            Small
          </Button>
          <Button shape="round" size="m">
            Medium
          </Button>
        </Row>
      </Stack>
      <Stack gap={8}>
        <Label>square</Label>
        <Row>
          <Button shape="square" size="xs">
            Extra small
          </Button>
          <Button shape="square" size="s">
            Small
          </Button>
          <Button shape="square" size="m">
            Medium
          </Button>
        </Row>
      </Stack>
    </Stack>
  ),
};

/** Icons are plain children; the size recipe sizes them. */
export const WithIcons: Story = {
  render: () => (
    <Row>
      <Button variant="filled">
        <Download /> Install aria2t
      </Button>
      <Button variant="filled-tonal">
        <Puzzle /> Add to Chrome
      </Button>
      <Button variant="outlined">
        <Terminal /> Copy command
      </Button>
      <Button variant="text">
        Learn more <ArrowRight />
      </Button>
    </Row>
  ),
};

/** Disabled drops to 50% and stops taking pointer events, at every variant. */
export const Disabled: Story = {
  render: () => (
    <Row>
      <Button variant="filled" disabled>
        Install
      </Button>
      <Button variant="outlined" disabled>
        Read the docs
      </Button>
      <Button variant="text" disabled>
        Skip
      </Button>
    </Row>
  ),
};

/**
 * `asChild` hands the recipe to whatever element is passed — the landing CTAs
 * are anchors that look like buttons, not buttons that navigate.
 */
export const AsLink: Story = {
  render: () => (
    <Row>
      <Button asChild variant="filled">
        <a href="#install">Install aria2t</a>
      </Button>
      <Button asChild variant="outlined">
        <a href="#extension">Browser extension</a>
      </Button>
    </Row>
  ),
};

/** The controls panel drives one live button. */
export const Playground: Story = {
  args: { variant: 'filled', size: 's', shape: 'round' },
};
