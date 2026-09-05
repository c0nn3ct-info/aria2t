import { useEffect, useRef, useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { cn } from '@/lib/utils';
import { Grid, Row, Stack } from '@/storybook/layout';
import { Caption, keyOf, Section, useRootTokens } from './shared';

/**
 * The two mechanisms that give a surface its edges: the `--shape-*` corner
 * scale and the `--shadow-*` elevation steps.
 *
 * `borderRadius` and `boxShadow` in `tailwind.config.ts` are nothing but
 * `var(--shape-*)` and `var(--shadow-*)`, so the utilities below and the values
 * printed beside them come from the same place. Each sample also reports what
 * `getComputedStyle` resolved on the rendered element, which is what catches a
 * class that is *not* wired to a token.
 */
const meta = {
  title: 'Foundations/Shape & Elevation',
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

/** One resolved property of a rendered element, re-read when the globals change. */
function useResolved(property: string, tokenKey: string) {
  const ref = useRef<HTMLDivElement>(null);
  const [value, setValue] = useState('');
  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    setValue(getComputedStyle(element).getPropertyValue(property).trim());
  }, [property, tokenKey]);
  return { ref, value };
}

// `rounded-*` maps 1:1 onto the `--shape-*` scale.
const RADII = [
  { className: 'rounded-xs', token: '--shape-xs', use: 'No caller on the site' },
  { className: 'rounded-sm', token: '--shape-sm', use: 'No caller on the site' },
  {
    className: 'rounded-md',
    token: '--shape-md',
    use: 'FAQ rows, code blocks, the skip link, square buttons at xs/s',
  },
  {
    className: 'rounded-lg',
    token: '--shape-lg',
    use: 'Card, the terminal mock, square buttons at m',
  },
  {
    className: 'rounded-xl',
    token: '--shape-xl',
    use: 'm3/Section, the popup and browser frames, Fab small',
  },
  {
    className: 'rounded-pill',
    token: '--shape-pill',
    use: 'Buttons, badges, the surface switch, the brand link',
  },
] as const;

const RADII_TOKENS = RADII.map((radius) => radius.token);

/** One radius sample: the box, its class, the token value and the resolved one. */
function RadiusSample({
  className,
  tokenKey,
  caption,
  use,
}: {
  className: string;
  tokenKey: string;
  caption: string;
  use: string;
}) {
  const { ref, value } = useResolved('border-radius', tokenKey);
  return (
    <Stack gap={6}>
      <div
        ref={ref}
        className={className}
        style={{
          height: 72,
          background: 'hsl(var(--surface-container-high))',
          border: '1px solid hsl(var(--outline-variant))',
        }}
      />
      <code className="text-label-small text-on-surface">{className}</code>
      <Caption>{caption}</Caption>
      <Caption>{`resolved: ${value || '—'}`}</Caption>
      <span className="text-body-small text-on-surface-variant">{use}</span>
    </Stack>
  );
}

// Tailwind's own steps, left at their defaults because the config extends only
// xs…xl and pill. They are reachable, and three components reach for them.
const OFF_SCALE = [
  {
    className: 'rounded-2xl',
    use: 'Button size xl and square l, IconButton square l, Fab regular',
  },
  { className: 'rounded-3xl', use: 'Button and IconButton square xl, Fab large' },
] as const;

/**
 * The six corner sizes. `--shape-pill` is 999px rather than 50%, so a pill
 * stays a pill at any width instead of turning into an ellipse. `--radius: 16px`
 * also sits in `:root`, left over from the shadcn scaffold the primitives grew
 * out of; nothing reads it.
 *
 * The second section is the part worth knowing. `tailwind.config.ts` extends
 * `borderRadius` with `xs`…`xl` and `pill` but leaves `2xl` and `3xl` at
 * Tailwind's defaults, so `rounded-xl` is the 36px token while `rounded-2xl` is
 * Tailwind's 16px — a *smaller* corner one step further up the name. Every size
 * ramp that crosses that line reverses there: a `Fab` is rounder at `small`
 * (rounded-xl) than at `regular` (rounded-2xl), and `Button` is rounder at size
 * `l` than at `xl`. The numbers below are measured off the rendered box, so
 * this page will say so the day it changes.
 */
export const Radii: Story = {
  render: (_args, { globals }) => <RadiiPage tokenKey={keyOf(globals)} />,
};

function RadiiPage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(RADII_TOKENS, tokenKey);
  return (
    <Stack gap={24}>
      <Section title="Corner radius">
        <Grid columns={3} gap={16} align="start">
          {RADII.map((radius) => (
            <RadiusSample
              key={radius.className}
              className={radius.className}
              tokenKey={tokenKey}
              caption={`${radius.token}: ${values[radius.token] || '—'}`}
              use={radius.use}
            />
          ))}
        </Grid>
      </Section>
      <Section title="Outside the scale">
        <Grid columns={3} gap={16} align="start">
          {OFF_SCALE.map((radius) => (
            <RadiusSample
              key={radius.className}
              className={radius.className}
              tokenKey={tokenKey}
              caption="no --shape-* token"
              use={radius.use}
            />
          ))}
        </Grid>
      </Section>
    </Stack>
  );
}

// The clip-path in each of these is a hard-coded superellipse path, so the box
// has to be exactly the size the path was drawn for — anything else clips.
const SQUIRCLES = [
  { className: 'shape-squircle-sm', size: 36 },
  { className: 'shape-squircle-power', size: 40 },
  { className: 'shape-squircle-md', size: 44 },
  { className: 'shape-squircle-lg', size: 56 },
] as const;

/**
 * M3 Expressive squircles: a superellipse (n≈4) drawn as a `clip-path`. The
 * path is authored at one size per class, so the element has to match it —
 * 36, 40, 44 and 56px, in that order. Give one a different size and the path
 * cuts the content instead of framing it.
 *
 * All four are carried from the shared design system and none has a caller on
 * this site: the avatar-shaped component they were drawn for lives on the
 * sibling project, and aria2t's round pieces are the `ConnectionVisual` core
 * and the `Fab`, both of which use a radius. They are shown here because the
 * classes are in `globals.css` and reachable, not because the site renders one.
 *
 * `clip-path` also clips `box-shadow`, which is why an elevated squircle needs
 * the shadow on a wrapper rather than on the clipped element — the samples
 * below deliberately carry none.
 */
export const Squircles: Story = {
  render: () => (
    <Section title="Clip paths">
      <Row gap={24} align="flex-end">
        {SQUIRCLES.map((squircle) => (
          <Stack key={squircle.className} gap={6} align="center">
            <div
              className={squircle.className}
              style={{
                width: squircle.size,
                height: squircle.size,
                background: 'hsl(var(--primary-container))',
              }}
            />
            <code className="text-label-small text-on-surface">{squircle.className}</code>
            <Caption>{`${squircle.size}×${squircle.size}px`}</Caption>
          </Stack>
        ))}
      </Row>
    </Section>
  ),
};

const ELEVATIONS = [
  {
    className: 'shadow-e1',
    token: '--shadow-1',
    use: 'Card elevated, the resting elevated Button, filled buttons on hover',
  },
  {
    className: 'shadow-e2',
    token: '--shadow-2',
    use: 'The language popover, the ConnectionVisual core, elevated Button on hover',
  },
  { className: 'shadow-e3', token: '--shadow-3', use: 'Fab at rest' },
  {
    className: 'shadow-e4',
    token: '--shadow-4',
    use: 'Fab on hover, and all three device mocks',
  },
] as const;

const ELEVATION_TOKENS = ELEVATIONS.map((elevation) => elevation.token);

/**
 * Four steps, each a two-layer shadow: a tight key shadow plus a softer ambient
 * one, both fixed black at low alpha. `transition-shadow duration-med ease-emph`
 * on `Card`, and `transition-all` on `Fab`, are what make a hover step up
 * rather than snap.
 */
export const Elevation: Story = {
  render: (_args, { globals }) => <ElevationPage tokenKey={keyOf(globals)} />,
};

function ElevationPage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(ELEVATION_TOKENS, tokenKey);
  return (
    <Section title="Elevation">
      <Grid columns={2} gap={24} style={{ padding: 8 }} align="start">
        {ELEVATIONS.map((elevation) => (
          <Stack key={elevation.className} gap={8}>
            <div
              className={cn('rounded-lg bg-surface-container-low', elevation.className)}
              style={{ minHeight: 88, padding: 20 }}
            >
              <Stack gap={4}>
                <div className="text-title-small text-on-surface">{elevation.className}</div>
                <div className="text-body-small text-on-surface-variant">{elevation.use}</div>
              </Stack>
            </div>
            <Caption>{`${elevation.token}: ${values[elevation.token] || '—'}`}</Caption>
          </Stack>
        ))}
      </Grid>
    </Section>
  );
}

const GROUNDS = [
  { label: 'bg-background', className: 'bg-background' },
  { label: 'bg-surface-container-low', className: 'bg-surface-container-low' },
  { label: 'bg-surface-container-high', className: 'bg-surface-container-high' },
] as const;

/**
 * The same four shadows over three grounds, because the tones are fixed black
 * at low alpha rather than tinted by the theme: switch to Dark and they all but
 * disappear, and depth has to come from the surface ladder instead.
 *
 * That is why every mock on the site pairs its shadow with a line — the popup
 * and browser frames draw `border-outline-variant` under `shadow-e4`, and the
 * terminal mock draws `--tui-frame-border`, the app's own border colour at 60%,
 * so the frame still has an edge in dark where the shadow has none.
 */
export const Layering: Story = {
  render: () => (
    <Stack gap={24}>
      {GROUNDS.map((ground) => (
        <Section key={ground.label} title={ground.label}>
          <div className={cn('rounded-lg', ground.className)} style={{ padding: 24 }}>
            <Grid columns={4} gap={20}>
              {ELEVATIONS.map((elevation) => (
                <div
                  key={elevation.className}
                  className={cn(
                    'rounded-md bg-surface-container-lowest text-label-small text-on-surface',
                    elevation.className,
                  )}
                  style={{ padding: 12 }}
                >
                  {elevation.className}
                </div>
              ))}
            </Grid>
          </div>
        </Section>
      ))}
    </Stack>
  ),
};
