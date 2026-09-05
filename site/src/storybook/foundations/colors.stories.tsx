import { useEffect, useRef, useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Grid, Stack } from '@/storybook/layout';
import { Caption, keyOf, Section, useRootTokens } from './shared';

/**
 * Colour roles, painted from the live custom properties.
 *
 * Every swatch fills with `hsl(var(--role))` and every printed value comes from
 * `getComputedStyle`, so the page shows what the site itself renders after
 * `applyTheme`/`applyAccent` rather than a copy of the numbers in
 * `globals.css`. Switch Theme or Accent in the toolbar and both the fills and
 * the values move.
 *
 * The neutrals and the status families are theme-fixed; `[data-accent]` sweeps
 * primary, secondary and tertiary and tints the surface ladder with them, which
 * is why the greys are not the same grey under Cyan as under Neutral.
 */
const meta = {
  title: 'Foundations/Colors',
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

/**
 * The CSS a token is painted with. Most roles are stored as a bare HSL triple
 * (`220 25% 99%`) so Tailwind can compose `hsl(var(--x) / <alpha>)`; the
 * `--tui-*` palette below is stored as finished colours instead, and has to be
 * used raw.
 */
function color(token: string, literal = false): string {
  return literal ? `var(${token})` : `hsl(var(${token}))`;
}

/** One role: the fill, its name, and its resolved value. */
function Swatch({
  token,
  on,
  value,
  height = 60,
  literal = false,
}: {
  token: string;
  on?: string;
  value: string;
  height?: number;
  literal?: boolean;
}) {
  return (
    <Stack gap={4}>
      <div
        className="text-label-large"
        style={{
          background: color(token, literal),
          color: on ? color(on, literal) : undefined,
          border: '1px solid hsl(var(--outline-variant))',
          borderRadius: 'var(--shape-sm)',
          height,
          display: 'flex',
          alignItems: 'flex-end',
          padding: 8,
        }}
      >
        {on ? 'Aa' : null}
      </div>
      <code className="text-label-small text-on-surface">{token}</code>
      <Caption>{value}</Caption>
    </Stack>
  );
}

/** A container role with the text role that is meant to sit on it. */
function Pair({
  container,
  on,
  values,
}: {
  container: string;
  on?: string;
  values: Record<string, string>;
}) {
  return (
    <div
      style={{
        background: color(container),
        color: on ? color(on) : 'hsl(var(--on-surface))',
        border: '1px solid hsl(var(--outline-variant))',
        borderRadius: 'var(--shape-sm)',
        padding: 12,
      }}
    >
      <Stack gap={2}>
        <div className="text-title-small">{on ? 'Text on this role' : 'No paired text role'}</div>
        <code className="text-label-small">{container}</code>
        <code className="text-label-small">{values[container] || '—'}</code>
        {on ? (
          <>
            <code className="text-label-small">{on}</code>
            <code className="text-label-small">{values[on] || '—'}</code>
          </>
        ) : null}
      </Stack>
    </div>
  );
}

const SURFACE_TONES = [
  { token: '--background', on: '--on-background' },
  { token: '--surface', on: '--on-surface' },
  { token: '--surface-container-lowest', on: '--on-surface' },
  { token: '--surface-container-low', on: '--on-surface' },
  { token: '--surface-container', on: '--on-surface' },
  { token: '--surface-container-high', on: '--on-surface' },
  { token: '--surface-container-highest', on: '--on-surface' },
  { token: '--surface-variant', on: '--on-surface-variant' },
] as const;

const ON_ROLES = ['--on-background', '--on-surface', '--on-surface-variant'] as const;

const LINE_ROLES = ['--outline', '--outline-variant'] as const;

const SURFACE_TOKENS = [
  ...SURFACE_TONES.flatMap((tone) => [tone.token, tone.on]),
  ...ON_ROLES,
  ...LINE_ROLES,
];

/**
 * The tone-based surface set plus the text roles that ride on it. `--surface`
 * and `--background` are the same tone by design; the five containers step up
 * from it — an elevated `Card` and an `m3/Section` panel sit on `low`, a filled
 * `Card` and a surface `Fab` on `high`, the count pill in a section header and
 * a `Badge variant="mono"` on `highest`, and the browser mock's little pointer
 * on `lowest`.
 *
 * `--surface-variant` is the one tone with no caller on the site: nothing
 * writes `bg-surface-variant`, though `--on-surface-variant` (its text role) is
 * the second most used colour in the whole codebase.
 */
export const Surfaces: Story = {
  render: (_args, { globals }) => <SurfacesPage tokenKey={keyOf(globals)} />,
};

function SurfacesPage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(SURFACE_TOKENS, tokenKey);
  return (
    <Stack gap={24}>
      <Section title="Surface tones">
        <Grid columns={4} gap={16}>
          {SURFACE_TONES.map((tone) => (
            <Swatch key={tone.token} token={tone.token} on={tone.on} value={values[tone.token]} />
          ))}
        </Grid>
      </Section>
      <Section title="Text on surfaces">
        <Grid columns={4} gap={16}>
          {ON_ROLES.map((token) => (
            <Swatch key={token} token={token} value={values[token]} height={40} />
          ))}
        </Grid>
      </Section>
      <Section title="Lines">
        <Grid columns={4} gap={16}>
          {LINE_ROLES.map((token) => (
            <Swatch key={token} token={token} value={values[token]} height={40} />
          ))}
        </Grid>
      </Section>
    </Stack>
  );
}

const SOLID_ROLES = [
  { container: '--primary', on: '--on-primary' },
  { container: '--tertiary', on: '--on-tertiary' },
  { container: '--success', on: '--on-success' },
  { container: '--error', on: '--on-error' },
  // No `--on-warning`: nothing on the site puts text on solid warning, so the
  // role deliberately has no paired text tone.
  { container: '--warning', on: undefined },
  { container: '--ring', on: undefined },
] as const;

const CONTAINER_ROLES = [
  { container: '--primary-container', on: '--on-primary-container' },
  { container: '--secondary-container', on: '--on-secondary-container' },
  { container: '--tertiary-container', on: '--on-tertiary-container' },
  { container: '--success-container', on: '--on-success-container' },
  { container: '--warning-container', on: '--on-warning-container' },
  { container: '--error-container', on: '--on-error-container' },
  { container: '--info-container', on: '--on-info-container' },
] as const;

// Compatibility aliases, kept from the shadcn scaffold the primitives grew out
// of. Every one of these points at a role above with `var()`, which is why the
// resolved value of, say, `--card` is the triple of `--surface-container-low`
// rather than the text `var(--surface-container-low)`.
const ALIAS_ROLES = [
  '--foreground',
  '--card',
  '--card-foreground',
  '--popover',
  '--popover-foreground',
  '--primary-foreground',
  '--secondary',
  '--secondary-foreground',
  '--muted',
  '--muted-foreground',
  '--accent',
  '--accent-foreground',
  '--destructive',
  '--destructive-foreground',
  '--border',
  '--input',
] as const;

const ROLE_TOKENS = [
  ...SOLID_ROLES.flatMap((role) => (role.on ? [role.container, role.on] : [role.container])),
  ...CONTAINER_ROLES.flatMap((role) => [role.container, role.on]),
  ...ALIAS_ROLES,
];

/**
 * Accent roles as they are used: a fill with the text tone that belongs on it.
 * `Badge` is the widest consumer: six of its eight variants are a container and
 * its paired text role, one per download state, and the other two are the
 * outline and the mono pill.
 *
 * The status families and `--info-container` stay put across accents; primary,
 * secondary and tertiary are swept by `[data-accent]`. Three of the sixteen
 * aliases have a caller — `--foreground` and `--border` through the two base
 * rules in `globals.css`, `--primary-foreground` through the filled `Button`
 * and the skip link. The other thirteen are plumbing the Tailwind theme still
 * exposes and nothing writes.
 */
export const Roles: Story = {
  render: (_args, { globals }) => <RolesPage tokenKey={keyOf(globals)} />,
};

function RolesPage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(ROLE_TOKENS, tokenKey);
  return (
    <Stack gap={24}>
      <Section title="Solid roles">
        <Grid columns={3} gap={16}>
          {SOLID_ROLES.map((role) => (
            <Pair key={role.container} container={role.container} on={role.on} values={values} />
          ))}
        </Grid>
      </Section>
      <Section title="Containers and their text">
        <Grid columns={3} gap={16}>
          {CONTAINER_ROLES.map((role) => (
            <Pair key={role.container} container={role.container} on={role.on} values={values} />
          ))}
        </Grid>
      </Section>
      <Section title="Aliases">
        <Grid columns={4} gap={16}>
          {ALIAS_ROLES.map((token) => (
            <Swatch key={token} token={token} value={values[token]} height={40} />
          ))}
        </Grid>
      </Section>
    </Stack>
  );
}

const STATUS_FAMILIES = [
  { fill: '--success', text: '--success-text', word: 'done' },
  { fill: '--warning', text: '--warning-text', word: 'seeding' },
  { fill: '--error', text: '--error-text', word: 'error' },
] as const;

const STATUS_TOKENS = STATUS_FAMILIES.flatMap((family) => [family.fill, family.text]);

const STATUS_GROUNDS = [
  '--surface',
  '--surface-container-low',
  '--surface-container-high',
  '--surface-container-highest',
] as const;

/**
 * Why each status hue exists twice. `--success`, `--warning` and `--error` are
 * fill and icon colours: at their light-theme lightness they land between 2.5:1
 * and 4.4:1 as text on an elevated surface, under the 4.5:1 AA floor. The
 * `*-text` trio is the same hue taken dark enough to read as body text on every
 * tone in the set — worst case 4.94:1 on `--surface-container-high`.
 *
 * Both rows below are set on the same four grounds. In light theme the fill row
 * visibly thins out as the ground rises and the text row does not; in dark the
 * two match for success and warning (dark already had the headroom) and only
 * error is lifted, 65% → 72% lightness.
 */
export const StatusText: Story = {
  render: (_args, { globals }) => <StatusTextPage tokenKey={keyOf(globals)} />,
};

function StatusTextPage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(STATUS_TOKENS, tokenKey);
  return (
    <Stack gap={24}>
      {STATUS_FAMILIES.map((family) => (
        <Section key={family.fill} title={`${family.fill} vs ${family.text}`}>
          <Stack gap={12}>
            <Grid columns={4} gap={12}>
              {STATUS_GROUNDS.map((ground) => (
                <div
                  key={ground}
                  style={{
                    background: color(ground),
                    border: '1px solid hsl(var(--outline-variant))',
                    borderRadius: 'var(--shape-sm)',
                    padding: 12,
                  }}
                >
                  <Stack gap={6}>
                    <div className="text-body-medium" style={{ color: color(family.fill) }}>
                      {family.word} — fill role
                    </div>
                    <div className="text-body-medium" style={{ color: color(family.text) }}>
                      {family.word} — text role
                    </div>
                    <code className="text-label-small text-on-surface-variant">{ground}</code>
                  </Stack>
                </div>
              ))}
            </Grid>
            <Caption>{`${family.fill}: ${values[family.fill] || '—'}  ·  ${family.text}: ${values[family.text] || '—'}`}</Caption>
          </Stack>
        </Section>
      ))}
    </Stack>
  );
}

const DIR_TOKENS = ['--dir', '--dir-on', '--dir-container', '--dir-on-container'] as const;

const DIRECTIONS = [
  { className: 'dir-proxy', label: 'Points at the primary family' },
  { className: 'dir-direct', label: 'Points at the tertiary family' },
  { className: 'dir-block', label: 'Points at the error family' },
] as const;

/**
 * `--dir-*` is a scoped alias set, not a root role: `.dir-proxy`, `.dir-direct`
 * and `.dir-block` each aim the same four variables at a different family, and
 * there is no `:root` fallback — a `bg-dir-container` with no such ancestor
 * renders transparent. `Card variant="accent"` is the only consumer, which is
 * why its own story wraps it in a `.dir-*` scope.
 *
 * The three class names come from the shared design system rather than from
 * anything aria2t models; here they are simply three swappable colour families
 * one component can be pointed at. Values below are read off the scoped element,
 * since `<html>` never carries them.
 */
export const Direction: Story = {
  render: (_args, { globals }) => (
    <Grid columns={3} gap={16} align="start">
      {DIRECTIONS.map((direction) => (
        <DirBlock
          key={direction.className}
          label={direction.label}
          className={direction.className}
          tokenKey={keyOf(globals)}
        />
      ))}
    </Grid>
  ),
};

function DirBlock({
  label,
  className,
  tokenKey,
}: {
  label: string;
  className: string;
  tokenKey: string;
}) {
  const scope = useRef<HTMLDivElement>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  useEffect(() => {
    const element = scope.current;
    if (!element) return;
    const style = getComputedStyle(element);
    setValues(
      Object.fromEntries(DIR_TOKENS.map((name) => [name, style.getPropertyValue(name).trim()])),
    );
  }, [tokenKey]);
  return (
    <div ref={scope} className={className}>
      <Stack gap={8}>
        <code className="text-label-small text-on-surface-variant">.{className}</code>
        <div
          className="rounded-md bg-dir-container text-dir-on-container"
          style={{ padding: 12 }}
        >
          <Stack gap={2}>
            <div className="text-title-small">{label}</div>
            <div className="text-body-small">bg-dir-container / text-dir-on-container</div>
          </Stack>
        </div>
        <Grid columns={2} gap={8}>
          {DIR_TOKENS.map((token) => (
            <Swatch key={token} token={token} value={values[token]} height={32} />
          ))}
        </Grid>
      </Stack>
    </div>
  );
}

const TUI_TONES = [
  '--tui-bg',
  '--tui-surface',
  '--tui-sel',
  '--tui-border',
  '--tui-border-dim',
  '--tui-frame-border',
] as const;

const TUI_TEXT = ['--tui-fg-bright', '--tui-fg', '--tui-fg-dim', '--tui-fg-faint'] as const;

const TUI_ACCENTS = [
  '--tui-accent',
  '--tui-green',
  '--tui-yellow',
  '--tui-red',
  '--tui-cyan',
  '--tui-magenta',
] as const;

const TUI_TOKENS = [...TUI_TONES, ...TUI_TEXT, ...TUI_ACCENTS];

/**
 * The app's own palette, mirrored into the site. `tui/internal/ui/theme.go`
 * carries Tokyo Night and Tokyo Night Day verbatim; `globals.css` restates them
 * as `--tui-*` so the terminal and list mocks on the home page follow the site
 * theme in pure CSS, with no hydration dependency — the mock switches the way
 * the real TUI switches.
 *
 * These are the one set stored as finished colours (`#7aa2f7`,
 * `rgb(59 66 97 / 0.6)`) rather than as HSL triples, because nothing composes
 * an alpha over them through Tailwind. They are also the one set `[data-accent]`
 * does not touch: the TUI has its own accent and the toolbar's does not apply.
 *
 * The six accents are the TUI's semantic colours, not decoration. In the list
 * mock, which mirrors the real list screen: green is an active or finished
 * download, magenta a seeding torrent and the upload rate, yellow a waiting or
 * paused one, red a failure, cyan the download rate. The accent draws the
 * selection marker, the progress bar and the wordmark.
 */
export const TuiPalette: Story = {
  render: (_args, { globals }) => <TuiPalettePage tokenKey={keyOf(globals)} />,
};

function TuiPalettePage({ tokenKey }: { tokenKey: string }) {
  const values = useRootTokens(TUI_TOKENS, tokenKey);
  return (
    <Stack gap={24}>
      <Section title="Grounds and chrome">
        <Grid columns={3} gap={16}>
          {TUI_TONES.map((token) => (
            <Swatch key={token} token={token} value={values[token]} literal />
          ))}
        </Grid>
      </Section>
      <Section title="Foregrounds">
        <Grid columns={4} gap={16}>
          {TUI_TEXT.map((token) => (
            <Swatch
              key={token}
              token={token}
              on="--tui-bg"
              value={values[token]}
              height={40}
              literal
            />
          ))}
        </Grid>
      </Section>
      <Section title="Semantic accents">
        <Grid columns={3} gap={16}>
          {TUI_ACCENTS.map((token) => (
            <Swatch
              key={token}
              token={token}
              on="--tui-bg"
              value={values[token]}
              height={40}
              literal
            />
          ))}
        </Grid>
      </Section>
    </Stack>
  );
}
