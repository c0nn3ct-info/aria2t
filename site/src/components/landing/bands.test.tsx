// The page's bands: what each one renders, and what the controls in them do.
// The two file pickers and the limits panel are real controls, so these drive
// them rather than reading their markup.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, userEvent, within } from '@/test/render';
import { LOCALES, setLocale } from '@/i18n';
import { FILES } from '@/lib/pick-scene';
import { FilesSection } from './files-section';
import { LimitsSection } from './limits-section';
import { PiecesSection } from './pieces-section';
import { QueueB2Section } from './queue-b2-section';
import { QueueSection } from './queue-section';
import { StatsSection } from './stats-section';
import { SurfacesSection } from './surfaces';
import { FaqBand } from './faq-band';
import { Eyebrow, LandingSection, MockCard, MockHeader, MockStage, PointList, SectionHeading } from './shell';
import { Check } from 'lucide-react';

const webdriver = { value: true };

beforeEach(() => {
  // Frozen: these suites are about structure and interaction, not the walk.
  Object.defineProperty(navigator, 'webdriver', { configurable: true, get: () => webdriver.value });
  webdriver.value = true;
  setLocale('en');
});

afterEach(() => {
  setLocale('en');
  vi.restoreAllMocks();
});

/** The selection is a page-wide store, so each test puts it back. */
async function resetPicks(): Promise<void> {
  const { togglePick } = await import('@/lib/pick-scene');
  const want = [true, true, false];
  const probe = render(<FilesSection />);
  const boxes = within(probe.container).getAllByRole('checkbox');
  boxes.forEach((box, i) => {
    if ((box.getAttribute('aria-checked') === 'true') !== want[i]) act(() => togglePick(i));
  });
  probe.unmount();
}

describe('the shell', () => {
  it('marks a band as something that arrives, and holds the reading width', () => {
    const { container } = render(
      <LandingSection id="x" className="extra">
        <span>body</span>
      </LandingSection>,
    );
    const section = container.querySelector('section')!;
    expect(section).toHaveAttribute('data-enter-section');
    expect(section).toHaveAttribute('id', 'x');
    expect(section.className).toContain('extra');
    expect(section.className).toContain('scroll-mt-20');
    expect(container.querySelector('.max-w-\\[1160px\\]')).not.toBeNull();
  });

  it('lets a band go edge to edge, and stay unnamed', () => {
    const { container } = render(<LandingSection contained={false}>edge</LandingSection>);
    expect(container.querySelector('.max-w-\\[1160px\\]')).toBeNull();
    expect(container.querySelector('section')).not.toHaveAttribute('id');
    expect(container.querySelector('section')!.className).not.toContain('scroll-mt-20');
  });

  it('draws a heading with an eyebrow, a body and a level of its own', () => {
    const { container } = render(
      <SectionHeading eyebrow="kicker" title="Title" body="Body" level={3} className="w" />,
    );
    expect(container.querySelector('h3')!.textContent).toBe('Title');
    expect(container.querySelector('h3')!.className).toContain('mt-3.5');
    expect(screen.getByText('kicker')).toBeInTheDocument();
    expect(screen.getByText('Body')).toBeInTheDocument();
  });

  it('draws a heading with neither', () => {
    const { container } = render(<SectionHeading title="Bare" />);
    expect(container.querySelector('h2')!.textContent).toBe('Bare');
    expect(container.querySelector('p')).toBeNull();
  });

  it('tones an eyebrow two ways', () => {
    const a = render(<Eyebrow>primary</Eyebrow>).container.firstElementChild!;
    const b = render(<Eyebrow tone="muted">muted</Eyebrow>).container.firstElementChild!;
    expect(a.className).toContain('text-primary');
    expect(b.className).toContain('text-on-surface-variant');
  });

  it('staggers a point list, and defaults a point to the success tone', () => {
    const { container } = render(
      <PointList
        points={[
          { icon: Check, text: 'plain' },
          { icon: Check, tone: 'primary', text: 'primary' },
        ]}
      />,
    );
    expect(container.querySelector('ul')).toHaveAttribute('data-enter-stagger', 'wipe');
    const badges = container.querySelectorAll('li > span:first-child');
    expect(badges[0].className).toContain('bg-success-container');
    expect(badges[1].className).toContain('bg-primary-container');
  });

  it('frames a mock, with and without something beside its title', () => {
    const withAside = render(
      <MockStage className="c">
        <MockCard className="d">
          <MockHeader title="T" aside={<em>aside</em>} />
        </MockCard>
      </MockStage>,
    );
    expect(withAside.getByText('aside')).toBeInTheDocument();
    const bare = render(<MockHeader title="Only" />);
    expect(bare.getByText('Only')).toBeInTheDocument();
  });
});

describe('the surfaces band', () => {
  it('opens on the extension and swaps to the terminal', async () => {
    const user = userEvent.setup();
    const { container } = render(<SurfacesSection />);
    // the popup, twice: alone below `sm` and inside the browser frame above it
    expect(container.querySelectorAll('.h-\\[600px\\]').length).toBeGreaterThan(0);
    expect(screen.getByText(/Straight from the page/)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /terminal/i }));
    expect(screen.getByText(/No mouse required/)).toBeInTheDocument();
    expect(screen.queryByText(/Straight from the page/)).toBeNull();
    // the terminal reads left to right whatever the page does
    expect(container.querySelector('[dir="ltr"]')).not.toBeNull();
  });
});

describe('the queue band', () => {
  it('ticks a file and recomputes what the selection costs', async () => {
    await resetPicks();
    const user = userEvent.setup();
    const { container } = render(<QueueSection />);
    const boxes = within(container).getAllByRole('checkbox');
    expect(boxes).toHaveLength(FILES.length);
    expect(boxes.map((b) => b.getAttribute('aria-checked'))).toEqual(['true', 'true', 'false']);
    // The torrent card is the second of the three inputs, and its footer is the
    // strip under its boxes; the mirrors card has one too.
    const footer = () =>
      container.querySelectorAll('li')[1].querySelector('.mt-auto')!.textContent;
    expect(within(container).getByText('2 of 3 selected')).toBeInTheDocument();
    expect(footer()).toContain('1.4 GiB');
    expect(footer()).toContain('−320 MiB');

    await user.click(boxes[2]);
    expect(within(container).getByText('3 of 3 selected')).toBeInTheDocument();
    expect(footer()).toContain('1.7 GiB');
    // nothing is being skipped, so no saving is claimed
    expect(footer()).not.toContain('−');
    await user.click(boxes[2]);
  });
});

describe('the queue band, split spine', () => {
  it('gives every row the route it arrived by, and the routes their legend', () => {
    const { container } = render(<QueueB2Section />);
    const rows = container.querySelectorAll('li');
    // three legend entries in the rail, five downloads in the queue
    expect(rows).toHaveLength(8);

    // the mark is an icon, so the route reaches a screen reader as the word
    const queue = [...rows].slice(3);
    expect(queue.map((r) => r.querySelector('.sr-only')!.textContent)).toEqual([
      'link',
      'torrent',
      'magnet',
      'torrent',
      'input',
    ]);
    // and the rail's three points wear the same three marks
    expect([...rows].slice(0, 3).every((r) => r.querySelector('svg') !== null)).toBe(true);
    // the same headline and the same list title as the band above it
    expect(container.querySelector('h2')!.textContent).toBe('One queue for every download');
    expect(within(container).getByText('The aria2 queue')).toBeInTheDocument();
  });

  it('keeps figures left to right and lets translated notes follow the page', () => {
    setLocale('ar');
    const { container } = render(<QueueB2Section />);
    const eta = (i: number) => [...container.querySelectorAll('li')][3 + i].lastElementChild!.lastElementChild!;
    // "4m 12s" would be reordered by the bidi algorithm; a ratio is a phrase
    expect(eta(0)).toHaveAttribute('dir', 'ltr');
    expect(eta(1)).not.toHaveAttribute('dir');
  });

  it('draws the paused row as the one that is not moving', () => {
    const { container } = render(<QueueB2Section />);
    const paused = [...container.querySelectorAll('li')].at(-1)!;
    expect(paused.textContent).toContain('raspios-arm64.img.xz');
    expect(paused.textContent).toContain('paused');
    // no speed colour on a row with no speed, and the faintest of the bars
    expect(paused.querySelector('.bg-surface-container-highest')).not.toBeNull();
    expect(paused.lastElementChild!.querySelector('.text-primary')).toBeNull();
  });

  it('leaks no raw key in any locale', () => {
    for (const locale of LOCALES) {
      setLocale(locale);
      const { container, unmount } = render(<QueueB2Section />);
      expect(container.textContent, locale).not.toMatch(/landing\.queue\./);
      unmount();
    }
  });
});

describe('the files band', () => {
  it('drives the picker and refuses to start on an empty selection', async () => {
    await resetPicks();
    const user = userEvent.setup();
    const { container } = render(<FilesSection />);
    const commit = () => within(container).getAllByRole('button').at(-1)!;
    expect(commit()).toHaveTextContent('Download 1.4 GiB');
    expect(commit()).not.toBeDisabled();

    const boxes = within(container).getAllByRole('checkbox');
    await user.click(boxes[0]);
    await user.click(boxes[1]);
    expect(within(container).getByText('0 of 3 files selected')).toBeInTheDocument();
    expect(commit()).toHaveTextContent('Nothing selected');
    expect(commit()).toBeDisabled();

    await user.click(boxes[0]);
    await user.click(boxes[1]);
    expect(commit()).toHaveTextContent('Download 1.4 GiB');
  });
});

describe('the limits band', () => {
  it('picks a preset, and no limit is one of them', async () => {
    const user = userEvent.setup();
    const { container } = render(<LimitsSection />);
    const chips = within(container).getAllByRole('button').filter((b) => b.hasAttribute('aria-pressed'));
    expect(chips.map((c) => c.getAttribute('aria-pressed'))).toEqual(['false', 'true', 'false', 'false']);

    await user.click(chips[0]);
    expect(chips[0]).toHaveAttribute('aria-pressed', 'true');
    expect(chips[1]).toHaveAttribute('aria-pressed', 'false');

    await user.click(chips[3]);
    expect(chips[3]).toHaveAttribute('aria-pressed', 'true');
    expect(chips[0]).toHaveAttribute('aria-pressed', 'false');
  });

  it('moves the global cap, and its far end is no limit at all', () => {
    const { container } = render(<LimitsSection />);
    const range = container.querySelector('input[type="range"]') as HTMLInputElement;
    expect(range.value).toBe('20');
    expect(range).toHaveAttribute('aria-valuetext', '20 MiB/s');
    expect(within(container).getByText('20 MiB/s')).toBeInTheDocument();

    // `fireEvent.change`, not an assignment: React tracks the value it set and
    // swallows the event when a direct write leaves its tracker untouched.
    fireEvent.change(range, { target: { value: '8' } });
    expect(within(container).getByText('8 MiB/s')).toBeInTheDocument();

    fireEvent.change(range, { target: { value: '21' } });
    expect(range).toHaveAttribute('aria-valuetext', 'no limit');
  });

  it('releases the scheduler window when the switch goes off', async () => {
    const user = userEvent.setup();
    const { container } = render(<LimitsSection />);
    const sw = within(container).getByRole('switch');
    const bars = () => [...container.querySelectorAll('i')].map((i) => i.style.height);
    expect(sw).toHaveAttribute('aria-checked', 'true');
    expect(bars()).toContain('25%');

    await user.click(sw);
    expect(sw).toHaveAttribute('aria-checked', 'false');
    expect(bars()).not.toContain('25%');

    await user.click(sw);
    expect(bars()).toContain('25%');
  });
});

describe('the live bands', () => {
  it('draws a minute of throughput and the figures beside it', () => {
    const { container } = render(<StatsSection />);
    expect(container.querySelector('svg path')).not.toBeNull();
    expect(container.querySelector('linearGradient')).not.toBeNull();
    expect(within(container).getAllByText(/MiB\/s/).length).toBeGreaterThan(0);
  });

  it('draws the piece map and its peers', () => {
    const { container } = render(<PiecesSection />);
    expect(container.querySelectorAll('.aspect-square')).toHaveLength(128);
    expect(within(container).getByText(/161 \/ 256|\d+ \/ 256/)).toBeInTheDocument();
  });
});

describe('the closing band', () => {
  it('answers the questions and offers the two ways to ask a ninth', () => {
    const { container } = render(<FaqBand />);
    expect(container.querySelectorAll('details').length).toBeGreaterThan(0);
    expect(within(container).getByText(/github\.com\//)).toBeInTheDocument();
    expect(within(container).getByText(/@/)).toBeInTheDocument();
  });
});

describe('every locale', () => {
  it.each(LOCALES)('leaks no raw key in %s', (loc) => {
    setLocale(loc);
    const { container } = render(
      <>
        <SurfacesSection />
        <QueueSection />
        <StatsSection />
        <FilesSection />
        <PiecesSection />
        <LimitsSection />
        <FaqBand />
      </>,
    );
    expect(container.textContent).not.toMatch(/landing\.|popup\.|home\.|\{\{/);
  });
});
