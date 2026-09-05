import type { ReactNode } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  CalendarClock,
  Check,
  Download,
  Gauge,
  Puzzle,
  Server,
  SlidersHorizontal,
  Terminal,
  UploadCloud,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Stack } from '@/storybook/layout';
import { Section, SectionLink } from './section';

/**
 * The body wrapper the install page puts inside every section — the card
 * supplies the surface, the content supplies its own inset and type scale.
 */
function Prose({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-5 px-2 pb-3 pt-2 text-body-large text-on-surface-variant">
      {children}
    </div>
  );
}

/** The install page's copy-able command block, inlined (it is local to that page). */
function Command({ children }: { children: string }) {
  return (
    <pre className="overflow-x-auto rounded-md bg-surface-container-highest px-3 py-3 font-mono text-body-small text-on-surface">
      <code>{children}</code>
    </pre>
  );
}

const meta = {
  title: 'M3/Section',
  component: Section,
  args: {
    header: 'Install aria2',
    icon: Download,
    children: (
      <Prose>
        <p>
          aria2t drives an aria2c daemon, so the engine goes on first — Homebrew on macOS, your
          package manager on Linux, aria2&apos;s own static build on Windows.
        </p>
        <Command>brew install aria2</Command>
      </Prose>
    ),
  },
  tags: ['autodocs'],
} satisfies Meta<typeof Section>;

export default meta;

type Story = StoryObj<typeof meta>;

/** One step of the install page: circled icon, title, body in the well below. */
export const Default: Story = {};

/**
 * `count` renders a pill after the title — the home page passes the number of
 * feature rows it is about to list. The check is `typeof count === 'number'`,
 * so a count of `0` still shows a pill and only an omitted count hides it.
 */
export const WithCount: Story = {
  args: {
    header: 'What you get',
    icon: Check,
    count: 3,
    children: (
      <ul className="space-y-2 px-2 pb-2 pt-1">
        {[
          ['Every source aria2 speaks', 'HTTP, FTP, SFTP, BitTorrent, Magnet, Metalink'],
          ['Pick files before the first byte', 'Torrents and magnets open a tree picker while paused'],
          ['A daemon that outlives the app', 'Session, speed caps and schedule survive a quit'],
        ].map(([title, body]) => (
          <li key={title} className="flex items-start gap-3 rounded-md px-2 py-2">
            <span className="mt-1 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <Check className="h-3.5 w-3.5" />
            </span>
            <div className="min-w-0">
              <div className="text-title-small">{title}</div>
              <div className="text-body-medium text-on-surface-variant">{body}</div>
            </div>
          </li>
        ))}
      </ul>
    ),
  },
};

/**
 * The `action` slot is pinned to the end of the header and never shrinks — a
 * control scoped to this section rather than to the page.
 */
export const WithAction: Story = {
  args: {
    header: 'Active downloads',
    icon: Download,
    count: 3,
    action: (
      <Button variant="text" size="xs">
        Pause all
      </Button>
    ),
    children: (
      <Prose>
        <p>ubuntu-24.04.1-desktop-amd64.iso — 4.7 GiB, 12.4 MiB/s, 6 min left</p>
        <p>debian-12.7.0-amd64-netinst.iso — 631 MiB, 8.1 MiB/s, 1 min left</p>
        <p>archlinux-2026.09.01-x86_64.iso — seeding, ratio 1.42</p>
      </Prose>
    ),
  },
};

/**
 * A section titles itself with an `h2`, because it sits directly under the page
 * `h1`. `headingLevel={3}` is for the one case that is genuinely nested —
 * hard-coding `h3` skipped a level on every page that used a section (WCAG
 * 1.3.1).
 */
export const NestedHeading: Story = {
  render: () => (
    <Stack gap={16}>
      <Section header="Set up the browser extension" icon={Puzzle}>
        <Prose>
          <p>Two steps: install the native host, then pin the extension and let it connect.</p>
          <Section header="Point the host at your binary" icon={Terminal} headingLevel={3}>
            <Prose>
              <p>
                The installer writes a com.aria2t.host manifest for the extension id you paste in.
              </p>
              <Command>./install.sh &lt;extension-id&gt;</Command>
            </Prose>
          </Section>
        </Prose>
      </Section>
    </Stack>
  ),
};

/**
 * `SectionLink` is the row form of the same card: a full-width button with the
 * circled icon, an optional supporting line carrying the current value, and a
 * trailing chevron that mirrors under RTL. Give it `supporting` for a setting,
 * omit it for a bare destination (the last row).
 */
export const Links: Story = {
  render: () => (
    <Section header="Settings" icon={SlidersHorizontal}>
      <SectionLink
        title="Global speed caps"
        icon={Gauge}
        supporting="5 MiB/s down · 1 MiB/s up"
        onClick={() => undefined}
      />
      <SectionLink
        title="Scheduler"
        icon={CalendarClock}
        supporting="Weeknights 23:00–07:00, unlimited"
        onClick={() => undefined}
      />
      <SectionLink
        title="Seeding defaults"
        icon={UploadCloud}
        supporting="Stop at ratio 1.5 or 60 minutes"
        onClick={() => undefined}
      />
      <SectionLink title="Servers" icon={Server} onClick={() => undefined} />
    </Section>
  ),
};

/** The controls panel drives the header, the count and the heading level. */
export const Playground: Story = {
  args: {
    header: 'Run aria2t',
    icon: Terminal,
    count: 2,
    action: <Badge variant="success">daemon up</Badge>,
  },
};
