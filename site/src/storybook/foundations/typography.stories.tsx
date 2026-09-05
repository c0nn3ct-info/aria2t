import { useEffect, useRef, useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Stack } from '@/storybook/layout';
import { keyOf, Section } from './shared';

/**
 * The type scale, measured rather than quoted.
 *
 * Each row renders its sample with the literal utility class and then reports
 * what `getComputedStyle` makes of it, so the numbers on this page cannot drift
 * from `tailwind.config.ts`. Class strings are written out in full — Storybook's
 * Tailwind instance scans this file for candidates, and a class assembled from
 * a template literal would never be generated.
 */
const meta = {
  title: 'Foundations/Typography',
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

interface Metrics {
  size: string;
  line: string;
  weight: string;
  tracking: string;
}

/** Metrics of a rendered element, re-measured when the toolbar globals change. */
function useMetrics(tokenKey: string) {
  const ref = useRef<HTMLSpanElement>(null);
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  useEffect(() => {
    const element = ref.current;
    if (!element) return;
    const style = getComputedStyle(element);
    setMetrics({
      size: style.fontSize,
      line: style.lineHeight,
      weight: style.fontWeight,
      tracking: style.letterSpacing,
    });
  }, [tokenKey]);
  return { ref, metrics };
}

function Readout({ metrics }: { metrics: Metrics | null }) {
  return (
    <code className="text-label-small text-on-surface-variant">
      {metrics
        ? `${metrics.size} / ${metrics.line} · ${metrics.weight} · ${metrics.tracking}`
        : 'measuring…'}
    </code>
  );
}

/**
 * One step of the scale: the sample on the left, the class, the measured
 * metrics and the caller on the right.
 */
function TypeRow({
  className,
  sample,
  use,
  tokenKey,
}: {
  className: string;
  sample: string;
  use: string;
  tokenKey: string;
}) {
  const { ref, metrics } = useMetrics(tokenKey);
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'minmax(0, 1fr) auto',
        alignItems: 'baseline',
        gap: 24,
        borderBottom: '1px solid hsl(var(--outline-variant))',
        paddingBottom: 12,
      }}
    >
      <span ref={ref} className={className}>
        {sample}
      </span>
      <Stack gap={2} align="flex-end">
        <code className="text-label-small text-on-surface">{className}</code>
        <Readout metrics={metrics} />
        <span className="text-body-small text-on-surface-variant">{use}</span>
      </Stack>
    </div>
  );
}

// Every `fontSize` key in tailwind.config.ts, in the order the config declares
// them — descending through display, headline, title, label and body. The class
// strings are literals so the scanner sees them.
const SCALE = [
  {
    className: 'text-display-small',
    sample: 'Download manager',
    use: 'The home hero, and only there',
  },
  { className: 'text-headline-large', sample: 'Install aria2t', use: 'Sub-page h1' },
  {
    className: 'text-headline-medium',
    sample: 'Browser extension',
    use: 'No caller on the site',
  },
  {
    className: 'text-headline-small',
    sample: 'Frequently asked questions',
    use: 'Section headings, the popup mock header',
  },
  {
    className: 'text-title-large',
    sample: 'Built-in managed daemon',
    use: 'No caller on the site',
  },
  {
    className: 'text-title-medium',
    sample: 'ubuntu-24.04.1-desktop-amd64.iso',
    use: 'CardTitle, m3/Section header, the site brand',
  },
  {
    className: 'text-title-small',
    sample: 'Connect to aria2 over JSON-RPC',
    use: 'FAQ questions, feature and step headings',
  },
  { className: 'text-label-large', sample: 'Add download', use: 'The Foundations sheets' },
  {
    className: 'text-label-medium',
    sample: '12.4 MiB/s · 42 peers',
    use: 'Surface switch, nav links, the count pill',
  },
  { className: 'text-label-small', sample: 'seeding', use: 'Badge, the diagram captions' },
  {
    className: 'text-body-large',
    sample: 'Manage aria2 from the terminal or browser.',
    use: 'Page ledes',
  },
  {
    className: 'text-body-medium',
    sample: 'Aria2t keeps its session under ~/.config/aria2t/daemon, so downloads resume across restarts.',
    use: 'Body copy, FAQ answers',
  },
  {
    className: 'text-body-small',
    sample: '18 of 42 files selected · 3.1 GiB of 7.4 GiB',
    use: 'Supporting lines on the install and extension pages',
  },
] as const;

/**
 * Display, headline, title, label and body — the whole `fontSize` scale.
 *
 * The site starts at `display-small`: the hero is the only place that needs a
 * display size, and it overrides the step outright above `sm`, which is why the
 * scale carries no display-medium or display-large that nothing would use.
 * `headline-medium` and `title-large` are the two steps in the middle with no
 * caller at all — they are kept so the ramp has no hole in it.
 *
 * Weight is baked into five of the steps by the config — `title-medium`,
 * `title-small` and the three labels all carry `500`. The rest carry none, so
 * they measure 400 here and 600 in the wild: every h1 on the site pairs its
 * step with `font-semibold tracking-tight`, which is a decision of the heading
 * rather than of the token.
 */
export const Scale: Story = {
  render: (_args, { globals }) => (
    <Stack gap={12}>
      {SCALE.map((step) => (
        <TypeRow
          key={step.className}
          className={step.className}
          sample={step.sample}
          use={step.use}
          tokenKey={keyOf(globals)}
        />
      ))}
    </Stack>
  ),
};

const MONO_SAMPLES = [
  {
    className: 'font-mono text-body-medium',
    sample: 'curl -fsSL https://aria2t.c0nn3ct.info/install.sh | bash',
    use: 'The install command block',
  },
  {
    className: 'font-mono text-body-small',
    sample: 'aria2t --url ws://seedbox.local:6800/jsonrpc --secret …',
    use: 'FAQ answers that quote a flag',
  },
  {
    className: 'font-mono text-label-small',
    sample: '~/.config/aria2t/daemon/session.txt',
    use: 'Paths and the diagram captions',
  },
  {
    className: 'font-mono text-label-small',
    sample: 'magnet:?xt=urn:btih:8c4adbf9ebe66f1d804fb6a4fb9b74966c3ab609',
    use: 'Badge variant="mono", which also tightens tracking',
  },
] as const;

/**
 * Monospace is not a scale of its own: `font-mono` swaps the face and keeps a
 * body or label size. Install commands, RPC URLs, config paths and info hashes
 * use it so that a command can be read character by character and lookalikes
 * (`Il1`, `O0`) stay apart — which matters more here than on most sites, since
 * a mistyped magnet link or `--secret` is a download that silently never
 * starts.
 *
 * Nothing overrides `fontFamily` in `tailwind.config.ts`, so this is Tailwind's
 * own stack rather than an aria2t one; the face below is whatever the platform
 * offers first from it.
 */
export const Mono: Story = {
  render: (_args, { globals }) => (
    <Stack gap={12}>
      {MONO_SAMPLES.map((step) => (
        <TypeRow
          key={step.sample}
          className={step.className}
          sample={step.sample}
          use={step.use}
          tokenKey={keyOf(globals)}
        />
      ))}
    </Stack>
  ),
};

/** Reports the face an element actually resolved to, plus the stack it was offered. */
function StackProbe({
  label,
  className,
  tokenKey,
  children,
}: {
  label: string;
  className: string;
  tokenKey: string;
  children: string;
}) {
  const probe = useRef<HTMLDivElement>(null);
  const [family, setFamily] = useState('');
  useEffect(() => {
    const element = probe.current;
    if (!element) return;
    setFamily(getComputedStyle(element).fontFamily);
  }, [tokenKey]);
  return (
    <Stack gap={6}>
      <div className="text-label-large text-on-surface-variant">{label}</div>
      <div ref={probe} className={className}>
        <Stack gap={4}>
          <div className="text-title-medium">{children}</div>
          <div className="text-body-medium">0123456789 — Il1 O0 · {'{}[]()<>'} · 4.7 GiB</div>
        </Stack>
      </div>
      <code className="text-label-small text-on-surface-variant">{family || 'measuring…'}</code>
    </Stack>
  );
}

// The site's own `home.hero.lede`, one locale each, so the coverage sample is
// text the site actually ships rather than a pangram invented for the page.
// Switching the toolbar's Locale re-renders these in place; they stay in their
// own script either way, which is the point.
const SCRIPTS = [
  { label: 'Latin', text: 'Manage aria2 from the terminal or browser.' },
  { label: 'Cyrillic', text: 'Управляйте aria2 из терминала или браузера.' },
  { label: 'Simplified Chinese', text: '从终端或浏览器管理 aria2。' },
  {
    label: 'Persian',
    text: '‏aria2 را از ترمینال یا مرورگر مدیریت کنید.',
  },
  { label: 'Arabic', text: 'أدِر aria2 من الطرفية أو المتصفح.' },
] as const;

/**
 * No web font ships with the site: nothing is fetched from a font host and
 * there is no `@font-face`, so the UI face is whatever the platform offers
 * first from the stack on `html, body` — San Francisco on macOS, Segoe on
 * Windows. `Inter` used to lead that stack without a single font file behind
 * it, so it rendered only for people who happened to have it installed; the
 * stack stated now is the one that was always rendering.
 *
 * The Arabic and CJK fallbacks are in it on purpose: four of the six locales
 * the site ships need them, and the samples below fall through to them rather
 * than to a default with no coverage. `font-feature-settings: 'cv11', 'ss01'`
 * rides along on the same rule — a stylistic set that only some faces honour,
 * which is why the digits and the `l` can change shape between platforms
 * without any class changing.
 *
 * Every one of these samples keeps the ASCII word `aria2` inside it, which is
 * the case that actually bites: a bidi run in the Arabic and Persian lines, and
 * a Latin island in the Chinese one that the CJK face has to hand back.
 */
export const Stacks: Story = {
  render: (_args, { globals }) => (
    <Stack gap={24}>
      <StackProbe label="System UI stack" className="" tokenKey={keyOf(globals)}>
        The quick brown fox downloads over the lazy dog
      </StackProbe>
      <StackProbe label="Monospace stack" className="font-mono" tokenKey={keyOf(globals)}>
        The quick brown fox downloads over the lazy dog
      </StackProbe>
      <Section title="Coverage">
        {SCRIPTS.map((script) => (
          <div
            key={script.label}
            style={{
              display: 'grid',
              gridTemplateColumns: 'minmax(0, 1fr) auto',
              alignItems: 'baseline',
              gap: 24,
            }}
          >
            <span className="text-title-medium">{script.text}</span>
            <code className="text-label-small text-on-surface-variant">{script.label}</code>
          </div>
        ))}
      </Section>
    </Stack>
  ),
};
