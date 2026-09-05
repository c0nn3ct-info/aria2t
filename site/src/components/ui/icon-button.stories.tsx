import type { Meta, StoryObj } from '@storybook/react-vite';
import { Copy, Github, Languages, Pause, Play, Trash2 } from 'lucide-react';
import { Row, Stack } from '@/storybook/layout';
import { IconButton } from './icon-button';

const meta = {
  title: 'Primitives/IconButton',
  component: IconButton,
  // The icon is the whole label, so every call site owes the control an
  // `aria-label` (and a `title`, for the pointer users the icon is aimed at).
  args: { children: <Pause />, 'aria-label': 'Pause' },
  tags: ['autodocs'],
} satisfies Meta<typeof IconButton>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Label under a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/**
 * The four variants. `standard` is the default and what the site header uses;
 * `filled-tonal` is the popup row's pause, sitting on a surface it has to lift
 * off of.
 */
export const Variants: Story = {
  render: () => (
    <Row gap={20} align="start">
      <Stack gap={8} align="center">
        <IconButton variant="filled" aria-label="Pause">
          <Pause />
        </IconButton>
        <Label>filled</Label>
      </Stack>
      <Stack gap={8} align="center">
        <IconButton variant="filled-tonal" aria-label="Resume">
          <Play />
        </IconButton>
        <Label>filled-tonal</Label>
      </Stack>
      <Stack gap={8} align="center">
        <IconButton variant="outlined" aria-label="Remove download">
          <Trash2 />
        </IconButton>
        <Label>outlined</Label>
      </Stack>
      <Stack gap={8} align="center">
        <IconButton variant="standard" aria-label="GitHub">
          <Github />
        </IconButton>
        <Label>standard</Label>
      </Stack>
    </Row>
  ),
};

/**
 * The size ramp. `s` is the default and the header's; `xs` is the copy button
 * tucked into a code block. `l` and `xl` exist for a hero-scale control the
 * site has not needed yet.
 */
export const Sizes: Story = {
  render: () => (
    <Row gap={16} align="end">
      <Stack gap={8} align="center">
        <IconButton variant="filled-tonal" size="xs" aria-label="Pause">
          <Pause />
        </IconButton>
        <Label>xs</Label>
      </Stack>
      <Stack gap={8} align="center">
        <IconButton variant="filled-tonal" size="s" aria-label="Pause">
          <Pause />
        </IconButton>
        <Label>s</Label>
      </Stack>
      <Stack gap={8} align="center">
        <IconButton variant="filled-tonal" size="m" aria-label="Pause">
          <Pause />
        </IconButton>
        <Label>m</Label>
      </Stack>
      <Stack gap={8} align="center">
        <IconButton variant="filled-tonal" size="l" aria-label="Pause">
          <Pause />
        </IconButton>
        <Label>l</Label>
      </Stack>
      <Stack gap={8} align="center">
        <IconButton variant="filled-tonal" size="xl" aria-label="Pause">
          <Pause />
        </IconButton>
        <Label>xl</Label>
      </Stack>
    </Row>
  ),
};

/**
 * `square` trades the pill for a radius that grows with the size — a compound
 * variant per step, so an `xs` square is not a scaled-down `xl` one.
 */
export const Shapes: Story = {
  render: () => (
    <Stack gap={16}>
      <Stack gap={8}>
        <Label>round</Label>
        <Row gap={12} align="end">
          <IconButton variant="filled" shape="round" size="xs" aria-label="Pause">
            <Pause />
          </IconButton>
          <IconButton variant="filled" shape="round" size="s" aria-label="Pause">
            <Pause />
          </IconButton>
          <IconButton variant="filled" shape="round" size="m" aria-label="Pause">
            <Pause />
          </IconButton>
          <IconButton variant="filled" shape="round" size="l" aria-label="Pause">
            <Pause />
          </IconButton>
        </Row>
      </Stack>
      <Stack gap={8}>
        <Label>square</Label>
        <Row gap={12} align="end">
          <IconButton variant="filled" shape="square" size="xs" aria-label="Pause">
            <Pause />
          </IconButton>
          <IconButton variant="filled" shape="square" size="s" aria-label="Pause">
            <Pause />
          </IconButton>
          <IconButton variant="filled" shape="square" size="m" aria-label="Pause">
            <Pause />
          </IconButton>
          <IconButton variant="filled" shape="square" size="l" aria-label="Pause">
            <Pause />
          </IconButton>
        </Row>
      </Stack>
    </Stack>
  ),
};

/** Disabled drops to 50% and stops taking pointer events, at every variant. */
export const Disabled: Story = {
  render: () => (
    <Row gap={12}>
      <IconButton variant="filled" disabled aria-label="Pause">
        <Pause />
      </IconButton>
      <IconButton variant="filled-tonal" disabled aria-label="Resume">
        <Play />
      </IconButton>
      <IconButton variant="outlined" disabled aria-label="Remove download">
        <Trash2 />
      </IconButton>
      <IconButton variant="standard" disabled aria-label="GitHub">
        <Github />
      </IconButton>
    </Row>
  ),
};

/**
 * The site header's nav actions: a GitHub link that is an anchor wearing the
 * button's shape (`asChild`), beside the language switcher's real button.
 */
export const InTheHeader: Story = {
  render: () => (
    <Row gap={4}>
      <IconButton asChild variant="standard" size="s" aria-label="GitHub" title="GitHub">
        <a href="https://github.com/c0nn3ct-info/aria2t" target="_blank" rel="noreferrer noopener">
          <Github />
        </a>
      </IconButton>
      <IconButton
        type="button"
        variant="standard"
        size="s"
        aria-label="Change language"
        aria-haspopup="menu"
        aria-expanded={false}
        title="Change language"
      >
        <Languages />
      </IconButton>
    </Row>
  ),
};

/**
 * The install page's copy button: `xs` and `standard`, pinned inside the code
 * block so it never competes with the command it copies.
 */
export const OnACodeBlock: Story = {
  render: () => (
    // The wrapper and <pre> classes are copied verbatim from install.tsx, which
    // is what keeps them inside Tailwind's `content` and therefore real here.
    <div className="group relative rounded-md bg-surface-container-highest" style={{ width: 420 }}>
      <pre className="overflow-x-auto px-3 py-3 pe-12 text-body-small font-mono text-on-surface">
        <code>curl -fsSL https://aria2t.c0nn3ct.info/install.sh | bash</code>
      </pre>
      <IconButton
        type="button"
        variant="standard"
        size="xs"
        aria-label="Copy command"
        title="Copy"
        className="absolute end-1.5 top-1.5 text-on-surface-variant"
      >
        <Copy />
      </IconButton>
    </div>
  ),
};

/** The controls panel drives one live icon button. */
export const Playground: Story = {
  args: { variant: 'standard', size: 's', shape: 'round' },
};
