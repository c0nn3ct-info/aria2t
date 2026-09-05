import type { Meta, StoryObj } from '@storybook/react-vite';
import { Row, Stack } from '@/storybook/layout';
import { Aria2tLogo } from './aria2t-logo';

const meta = {
  title: 'Blocks/Aria2tLogo',
  component: Aria2tLogo,
  tags: ['autodocs'],
} satisfies Meta<typeof Aria2tLogo>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Caption under a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/**
 * A sized, coloured box for the mark.
 *
 * The svg carries no `width`/`height` of its own, only a `viewBox`, so it takes
 * whatever box it is given — `h-full w-full` here, the same pair the site's own
 * `popup-mock` passes. And every shape in it is `currentColor`, so the colour
 * comes from the box as well: the site writes both at once
 * (`<Aria2tLogo className="h-6 w-6 text-primary" />` in the header).
 */
function Mark({ size, tone = 'text-primary' }: { size: number; tone?: string }) {
  return (
    <div className={tone} style={{ width: size, height: size }}>
      <Aria2tLogo className="h-full w-full" />
    </div>
  );
}

/** A token pair the mark is dropped onto, mark and caption stacked. */
function Swatch({ tone, field, name }: { tone: string; field?: string; name: string }) {
  return (
    <Stack gap={8} align="start">
      <div className={field} style={{ padding: 16, borderRadius: 12 }}>
        <Mark size={40} tone={tone} />
      </div>
      <Label>{name}</Label>
    </Stack>
  );
}

/**
 * The mark as the site header draws it: 24px, `text-primary`. A ring of
 * rounded cells, a download arrow inside it, and the block of chunks to its
 * right — one file arriving in pieces, which is what aria2 does.
 */
export const Default: Story = {
  render: () => <Mark size={24} />,
};

/**
 * The sizes it is actually asked for. 14px is the favicon slot in the fake
 * browser tab (`h-3.5 w-3.5`), 24px the header (`h-6 w-6`); the rest is
 * headroom. The stroke widths are fixed in user units, so the ring thins as it
 * scales up rather than staying a hairline — and below ~14px the individual
 * cells stop resolving and the mark reads as a solid ring.
 */
export const Sizes: Story = {
  render: () => (
    <Row gap={24} align="end">
      {[14, 24, 40, 64, 96].map((size) => (
        <Stack key={size} gap={8} align="center">
          <Mark size={size} />
          <Label>{`${size}px`}</Label>
        </Stack>
      ))}
    </Row>
  ),
};

/**
 * `currentColor` throughout, so the mark inherits whatever token the container
 * sets — no fills to keep in step with the theme, and the accent toolbar
 * repaints the first swatch without the svg knowing. The last two are the
 * filled cases, where the mark has to hold against a container colour rather
 * than the page background.
 */
export const TokenColours: Story = {
  render: () => (
    <Row gap={20} align="start">
      <Swatch tone="text-primary" name="text-primary" />
      <Swatch tone="text-on-surface" name="text-on-surface" />
      <Swatch tone="text-on-surface-variant" name="text-on-surface-variant" />
      <Swatch tone="text-primary-foreground" field="bg-primary" name="on bg-primary" />
      <Swatch
        tone="text-secondary-on-container"
        field="bg-secondary-container"
        name="on bg-secondary-container"
      />
    </Row>
  ),
};

/**
 * The lockup the header renders: the mark at 24px in `text-primary` beside the
 * wordmark, the whole thing a link home. The word is **Aria2t** — lowercase
 * `aria2t` is reserved for the binary, the command and `~/.config/aria2t/`,
 * never for prose or a wordmark.
 */
export const HeaderLockup: Story = {
  render: () => (
    <Row gap={8}>
      <Mark size={24} />
      <span className="text-title-medium tracking-tight text-on-surface">Aria2t</span>
    </Row>
  ),
};
