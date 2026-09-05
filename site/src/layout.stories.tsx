import { useEffect, useRef, type ReactNode } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Button } from '@/components/ui/button';
import { Stack } from '@/storybook/layout';
import { Layout } from './layout';

// The frame every page renders inside: the skip link, the sticky header
// (wordmark, GitHub link, language switcher), the `<main>` measure and the
// three-column footer. Only `children` varies between these stories — the
// chrome is the subject, so the bodies below stay deliberately plain.
const meta = {
  title: 'Blocks/Layout',
  component: Layout,
  args: { current: 'home', children: <ShortPage /> },
  // `fullscreen`, not the preview's `padded`: the frame is `min-h-screen` and
  // paints `bg-background` itself, so an inset around it only reads as a seam.
  parameters: { layout: 'fullscreen' },
  argTypes: {
    current: {
      description:
        'Which page is being rendered. Nothing in the nav marks it active today, so the control changes nothing on screen — it exists so callers keep naming where they are.',
    },
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Layout>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Stand-in page body: heading, lede, one CTA. */
function ShortPage() {
  return (
    <Stack gap={12} align="start">
      <h1 className="text-headline-small font-medium tracking-tight">Install Aria2t</h1>
      <p className="text-body-large text-on-surface-variant">
        One command installs the TUI, points a native-messaging manifest at it and offers to install
        aria2 itself when the machine has none.
      </p>
      <Button asChild variant="filled" size="s">
        <a href="#install">Copy the install command</a>
      </Button>
    </Stack>
  );
}

const SECTIONS = [
  [
    'Every source aria2 speaks',
    'HTTP(S), FTP, SFTP, BitTorrent, magnet links, .torrent and Metalink files, and aria2 input files — one queue for all of them, added from the same overlay.',
  ],
  [
    'No daemon to set up',
    'aria2t starts its own aria2c on a free port with a generated RPC secret, and shuts it down when you quit. Nothing to configure, no secret on the command line.',
  ],
  [
    'Downloads survive a restart',
    'The session file lives in ~/.config/aria2t/daemon/, so an interrupted 4.7 GiB ISO resumes where it stopped instead of starting over.',
  ],
  [
    'Pick files before the bytes move',
    'A torrent or magnet is added paused; the file tree opens once metadata resolves, and only what you check is fetched.',
  ],
  [
    'Speed limits on a schedule',
    'Cap the global rate to 500 KiB/s during the working day and let it off the leash at 23:00 — the rules live in the config, not in your head.',
  ],
  [
    'Verify what landed',
    'Streaming sha-256 over a finished download, with a progress counter, so a mirror that served you a truncated file cannot pass unnoticed.',
  ],
  [
    'Seeding you can see',
    'A torrent that finished downloading stays visible as seeding with its ratio and upload rate, instead of vanishing into the stopped list.',
  ],
  [
    'The browser hands off to it',
    'The extension captures a link and passes it to the same managed daemon the TUI drives, so both surfaces show one queue.',
  ],
] as const;

/** A page body long enough to scroll. */
function LongPageBody() {
  return (
    <Stack gap={28}>
      <ShortPage />
      {SECTIONS.map(([title, body]) => (
        <section key={title}>
          <Stack gap={8}>
            <h2 className="text-title-small">{title}</h2>
            <p className="text-body-medium text-on-surface-variant">{body}</p>
          </Stack>
        </section>
      ))}
    </Stack>
  );
}

/**
 * The frame around a short page. `main` is `flex-1` inside a `min-h-screen`
 * column, so the footer is pushed to the bottom edge rather than floating up
 * under the content.
 */
export const Default: Story = {};

/**
 * A page long enough to scroll: the header is `sticky top-0` with a blurred
 * translucent surface, and the footer arrives after the content instead of at
 * the viewport's bottom edge.
 */
export const LongPage: Story = {
  args: { current: 'install', children: <LongPageBody /> },
};

/**
 * The skip link — the first focusable thing on every page and `sr-only` until
 * it takes focus, so a keyboard user reaches the content without tabbing
 * through the wordmark, the GitHub link and the language menu first. This
 * story focuses it, which is the only way to see it at all.
 */
export const SkipLink: Story = {
  render: (args) => (
    <FocusFirstLink>
      <Layout {...args} />
    </FocusFirstLink>
  ),
};

/**
 * Focuses the first link of whatever it wraps, which is what Tab does on a real
 * page load. `preventScroll` because autodocs renders every story on one page:
 * pulling focus must not scroll the reader to this one.
 */
function FocusFirstLink({ children }: { children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.querySelector('a')?.focus({ preventScroll: true });
  }, []);
  return <div ref={ref}>{children}</div>;
}
