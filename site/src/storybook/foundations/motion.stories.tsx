import { useEffect, useState, type ReactNode } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { ConnectionVisual } from '@/components/m3/connection-visual';
import { Button } from '@/components/ui/button';
import { Grid, Row, Stack } from '@/storybook/layout';
import { keyOf, Section, useRootTokens } from './shared';

/**
 * Durations, easings and the standing loops.
 *
 * Motion is the one foundation a still page cannot show, so every bar here
 * travels on a loop and they all turn on the same beat — comparing two curves
 * or two lengths is only honest if they start together. Each bar's
 * `transition-duration` and `transition-timing-function` are `var()` references
 * to the token it is labelled with, and the value printed beside it comes from
 * `getComputedStyle`, so neither the movement nor the number is a copy of what
 * is in `globals.css`.
 *
 * Pause stops every loop on the page. It starts paused when the machine asks
 * for reduced motion.
 */
const meta = {
  title: 'Foundations/Motion',
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const REDUCED_MOTION = '(prefers-reduced-motion: reduce)';

const TRACK = 320;
const THUMB = 28;

/** Longer than `--dur-x-long`, so even the slowest bar settles before it turns. */
const BEAT_MS = 1100;

/**
 * One bar. `duration` and `easing` are passed through as `var()` references, so
 * the travel is timed by the token and not by a number copied out of it.
 */
function Bar({ duration, easing, moved }: { duration: string; easing: string; moved: boolean }) {
  return (
    <div
      className="rounded-pill bg-surface-container-high"
      style={{
        position: 'relative',
        width: TRACK,
        maxWidth: '100%',
        height: 16,
        overflow: 'hidden',
      }}
    >
      <div
        className="rounded-pill bg-primary"
        style={{
          position: 'absolute',
          top: 2,
          left: 2,
          width: THUMB,
          height: 12,
          transform: moved ? `translateX(${TRACK - THUMB - 4}px)` : 'translateX(0)',
          transitionProperty: 'transform',
          transitionDuration: `var(${duration})`,
          transitionTimingFunction: `var(${easing})`,
        }}
      />
    </div>
  );
}

/**
 * The shared beat every bar on a page rides. One interval for the whole story,
 * so two bars can never drift apart; `paused` starts true under reduced motion,
 * which is also what makes the loop stoppable (WCAG 2.2.2).
 */
function useBeat() {
  const [paused, setPaused] = useState(() => window.matchMedia(REDUCED_MOTION).matches);
  const [moved, setMoved] = useState(false);
  useEffect(() => {
    if (paused) return;
    const id = window.setInterval(() => setMoved((previous) => !previous), BEAT_MS);
    return () => window.clearInterval(id);
  }, [paused]);
  return { moved, paused, toggle: () => setPaused((previous) => !previous) };
}

/** Pause/Play for the travel stories. */
function Transport({ paused, onToggle }: { paused: boolean; onToggle: () => void }) {
  return (
    <Row gap={8}>
      <Button variant="filled-tonal" size="s" onClick={onToggle}>
        {paused ? 'Play' : 'Pause'}
      </Button>
      <span className="text-body-small text-on-surface-variant">
        Every bar turns on the same beat, so the difference is the token.
      </span>
    </Row>
  );
}

/**
 * Live `prefers-reduced-motion` state, plus what the stylesheet does about it.
 * Under `reduce` the bars still travel — at 60ms, which reads as a snap.
 */
function ReducedMotionNote() {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    const query = window.matchMedia(REDUCED_MOTION);
    setReduced(query.matches);
    const onChange = (event: MediaQueryListEvent) => setReduced(event.matches);
    query.addEventListener('change', onChange);
    return () => query.removeEventListener('change', onChange);
  }, []);
  return (
    <div className="rounded-md bg-surface-container" style={{ padding: 12 }}>
      <Stack gap={4}>
        <div className="text-title-small text-on-surface">
          prefers-reduced-motion: {reduced ? 'reduce' : 'no-preference'}
        </div>
        <div className="text-body-small text-on-surface-variant">
          Reduce is not a blanket kill. The three infinite decorations are held on their first
          frame with animation-play-state: paused, because an ambient loop that runs forever is
          exactly what WCAG 2.2.2 asks to be stoppable. Everything else keeps its transition, only
          clamped to 60ms — short enough not to read as movement, long enough that the state change
          the transition exists to communicate still happens. A blanket animation: 0.01ms would
          erase that change, and a switch that snaps with no feedback is worse than one that eases.
        </div>
      </Stack>
    </div>
  );
}

const DURATIONS = [
  { token: '--dur-x-short', use: 'No caller on the site' },
  {
    token: '--dur-short',
    use: 'Button and IconButton press, the FAQ chevron, m3/Section, the state layer',
  },
  { token: '--dur-med', use: 'Card elevation, the Fab, the ConnectionVisual core colour' },
  { token: '--dur-long', use: 'No caller on the site' },
  { token: '--dur-x-long', use: 'No caller — the bars in Easings run at it' },
] as const;

const DURATION_TOKENS = DURATIONS.map((duration) => duration.token);

/**
 * The five duration steps, all on `--ease-emph`, so only the length differs.
 * Two of them carry the whole site: `--dur-short` for anything the pointer
 * causes, `--dur-med` for anything a surface does. The other three exist so the
 * scale is a scale.
 *
 * `duration-short` and `duration-med` in the components come from
 * `transitionDuration` in `tailwind.config.ts`, which restates these as literal
 * ms values rather than as `var()` references — the one place in the system
 * where two lists have to be kept in step by hand, and the reason this page
 * prints the custom property rather than the utility.
 */
export const Durations: Story = {
  render: (_args, { globals }) => <DurationsPage tokenKey={keyOf(globals)} />,
};

function DurationsPage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(DURATION_TOKENS, tokenKey);
  const { moved, paused, toggle } = useBeat();
  return (
    <Stack gap={20}>
      <Transport paused={paused} onToggle={toggle} />
      <Section title="Duration">
        <Stack gap={16}>
          {DURATIONS.map((duration) => (
            <Stack key={duration.token} gap={6}>
              <Row gap={8}>
                <code className="text-label-small text-on-surface">{duration.token}</code>
                <code className="text-label-small text-on-surface-variant">
                  {values[duration.token] || '—'}
                </code>
                <span className="text-body-small text-on-surface-variant">{duration.use}</span>
              </Row>
              <Bar duration={duration.token} easing="--ease-emph" moved={moved} />
            </Stack>
          ))}
        </Stack>
      </Section>
      <ReducedMotionNote />
    </Stack>
  );
}

const EASINGS = [
  { token: '--ease-emph', use: 'The default: colour, elevation, the state layer' },
  { token: '--ease-emph-decel', use: 'Arrivals — the pulse-ring keyframes' },
  { token: '--ease-spring', use: 'Press feedback on Button and IconButton' },
  { token: '--ease-spring-standard', use: 'Only .shape-morph, which has no caller' },
] as const;

const EASING_TOKENS = EASINGS.map((easing) => easing.token);

/**
 * The same travel at `--dur-x-long`, so the shape of each curve is visible.
 * Both springs overshoot and come back, by very different amounts:
 * `--ease-spring` is the generic backOut (control point 1.56), which passes its
 * target by about 10% before settling, while `--ease-spring-standard` (1.06)
 * barely reaches past it at all.
 *
 * Neither belongs on a `transition` shorthand that also carries colour, where
 * an overshoot interpolates past the target and back — which is exactly why
 * `Button` names its properties one by one
 * (`transition-[transform,background-color,box-shadow,border-color,color,opacity]`)
 * instead of writing `transition-all` next to `ease-spring`.
 */
export const Easings: Story = {
  render: (_args, { globals }) => <EasingsPage tokenKey={keyOf(globals)} />,
};

function EasingsPage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(EASING_TOKENS, tokenKey);
  const { moved, paused, toggle } = useBeat();
  return (
    <Stack gap={20}>
      <Transport paused={paused} onToggle={toggle} />
      <Section title="Easing">
        <Stack gap={16}>
          {EASINGS.map((easing) => (
            <Stack key={easing.token} gap={6}>
              <Row gap={8}>
                <code className="text-label-small text-on-surface">{easing.token}</code>
                <code className="text-label-small text-on-surface-variant">
                  {values[easing.token] || '—'}
                </code>
                <span className="text-body-small text-on-surface-variant">{easing.use}</span>
              </Row>
              <Bar duration="--dur-x-long" easing={easing.token} moved={moved} />
            </Stack>
          ))}
        </Stack>
      </Section>
      <ReducedMotionNote />
    </Stack>
  );
}

function Loop({ title, note, children }: { title: string; note: string; children: ReactNode }) {
  return (
    <Stack gap={8}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: 112,
          borderRadius: 'var(--shape-md)',
          background: 'hsl(var(--surface-container-low))',
        }}
      >
        {children}
      </div>
      <code className="text-label-small text-on-surface">{title}</code>
      <span className="text-body-small text-on-surface-variant">{note}</span>
    </Stack>
  );
}

/**
 * The three named keyframe animations from `tailwind.config.ts`, and the one
 * component whose whole purpose is motion. All three are infinite, all three
 * are frozen under reduced motion, and none of them is a transition — the
 * one-shot family belongs to whatever triggers it.
 *
 * `animate-pulse-ring` reads `var(--pulse-dur, 3s)`, so the caller sets the
 * period: `ConnectionVisual` gives each connection state its own (1.6s probing,
 * 3.2s connected, 1.4s error) and staggers three rings by a third of it.
 * `animate-status-dot` is declared and reachable but has no caller on the site.
 *
 * Not everything that moves here is a token. The TUI list and the terminal on
 * the home page redraw their speed columns from a JS interval over a seeded
 * PRNG, because they are simulating a transfer rather than easing between two
 * states — CSS has no opinion to offer about 12.4 MiB/s.
 */
export const Loops: Story = {
  render: () => (
    <Stack gap={20}>
      <Grid columns={3} gap={20} align="start">
        <Loop title="animate-pulse-ring" note="Attention ring around a live target.">
          <span style={{ position: 'relative', display: 'inline-flex' }}>
            <span
              className="animate-pulse-ring rounded-pill bg-primary"
              style={{ position: 'absolute', inset: -6 }}
            />
            <span className="rounded-pill bg-primary" style={{ width: 16, height: 16 }} />
          </span>
        </Loop>
        <Loop title="animate-breathe" note="Idle glow: scale 0.94→1 at 45–70% opacity.">
          <span
            className="animate-breathe rounded-pill bg-tertiary"
            style={{ width: 32, height: 32 }}
          />
        </Loop>
        <Loop title="animate-status-dot" note="A daemon that is up and answering.">
          <span
            className="animate-status-dot rounded-pill bg-success"
            style={{ width: 12, height: 12 }}
          />
        </Loop>
        <Loop title="ConnectionVisual connecting" note="Three pulse rings at 1.6s, staggered.">
          <ConnectionVisual state="connecting" size={96} />
        </Loop>
        <Loop title="ConnectionVisual connected" note="The same rings, slowed to 3.2s.">
          <ConnectionVisual state="connected" size={96} />
        </Loop>
        <Loop title="ConnectionVisual idle" note="One breathing ring, two at rest.">
          <ConnectionVisual state="idle" size={96} />
        </Loop>
      </Grid>
      <ReducedMotionNote />
    </Stack>
  ),
};

const STATE_LAYER_TARGETS = [
  { label: 'Hover me', note: '8% of --on-surface' },
  { label: 'Tab to me', note: '10% on focus-visible' },
  { label: 'Hold me down', note: '12% while active' },
] as const;

/**
 * `.m3-state-layer` is the fourth piece of motion in the system and the only
 * one with no keyframes: an `::after` pseudo-element at `inset: 0` that fades
 * `hsl(var(--on-surface))` in over `--dur-short --ease-emph` — 8% on hover, 10%
 * on `focus-visible`, 12% while pressed.
 *
 * It exists so a tint can ride on top of a surface whose own colour is unknown,
 * which is why the header brand link and the FAQ rows use it instead of a
 * `hover:bg-*`. `isolation: isolate` on the host is what keeps the layer under
 * the label, and `border-radius: inherit` is what keeps it inside a pill.
 *
 * Nothing below moves on its own: put a pointer on one, then tab to it.
 */
export const StateLayer: Story = {
  render: () => (
    <Stack gap={20}>
      <Section title="m3-state-layer">
        <Row gap={16} align="flex-start">
          {STATE_LAYER_TARGETS.map((target) => (
            <Stack key={target.label} gap={6} align="flex-start">
              <button
                type="button"
                className="m3-state-layer rounded-pill bg-surface-container-low text-label-large text-on-surface"
                style={{ padding: '10px 20px' }}
              >
                {target.label}
              </button>
              <span className="text-body-small text-on-surface-variant">{target.note}</span>
            </Stack>
          ))}
        </Row>
      </Section>
      <ReducedMotionNote />
    </Stack>
  ),
};
