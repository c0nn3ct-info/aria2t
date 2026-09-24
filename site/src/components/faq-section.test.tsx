import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, userEvent } from '@/test/render';
import { LOCALES, setLocale } from '@/i18n';
import { FaqList, FaqSection } from './faq-section';

afterEach(() => setLocale('en'));

describe('FaqList', () => {
  it('draws the same entries bare when the page frames them itself', () => {
    const card = render(<FaqList />);
    const boxed = card.container.firstElementChild!;
    expect(boxed.className).toMatch(/border/);
    expect(boxed.querySelectorAll('details')).toHaveLength(8);
    card.unmount();

    const { container } = render(<FaqList variant="flush" />);
    const flush = container.firstElementChild!;
    expect(flush.className).not.toMatch(/border-outline/);
    expect(flush.querySelectorAll('details')).toHaveLength(8);
    // the same questions, whichever frame
    expect(flush.textContent).toBe(boxed.textContent);
  });

  it('gives the bare rows hover alone, and no radius', () => {
    const card = render(<FaqList />);
    const boxedRow = card.container.querySelector('summary')!;
    expect(boxedRow.className).toMatch(/m3-state-layer/);
    card.unmount();

    const { container } = render(<FaqList variant="flush" />);
    for (const row of container.querySelectorAll('summary')) {
      expect(row.className).toMatch(/m3-hover-layer/);
      expect(row.className).not.toMatch(/m3-state-layer/);
      expect(row.className).not.toMatch(/rounded/);
    }
  });
});

describe('FaqSection', () => {
  it('renders one collapsible entry per question, all closed', () => {
    const { container } = render(<FaqSection />);
    const entries = container.querySelectorAll('details');
    expect(entries).toHaveLength(8);
    for (const d of entries) expect(d.open).toBe(false);
    expect(screen.getByRole('heading')).toBeInTheDocument();
  });

  it('opens an entry to reveal its answer', async () => {
    const { container } = render(<FaqSection />);
    const first = container.querySelector('details')!;
    await userEvent.click(first.querySelector('summary')!);
    expect(first.open).toBe(true);
  });

  it('is anchorable, so the nav can link to it', () => {
    const { container } = render(<FaqSection />);
    expect(container.querySelector('#faq')).not.toBeNull();
  });

  it('has real text in every locale — no leaked keys', () => {
    for (const locale of LOCALES) {
      setLocale(locale);
      const { container, unmount } = render(<FaqSection />);
      expect(container.textContent, locale).not.toMatch(/home\.faq\./);
      unmount();
    }
  });
});

describe('FaqList motion', () => {
  const summaries = (c: HTMLElement) => [...c.querySelectorAll('summary')] as HTMLElement[];
  const click = (el: HTMLElement) =>
    el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }));

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    // @ts-expect-error jsdom has no Web Animations; tests add and remove it
    delete Element.prototype.animate;
    // @ts-expect-error see above
    delete Element.prototype.getAnimations;
  });

  function withAnimations() {
    const made: { keyframes: Keyframe[]; opts: KeyframeAnimationOptions; anim: { onfinish: null | (() => void); cancel: ReturnType<typeof vi.fn> } }[] = [];
    const running = { cancel: vi.fn() };
    Element.prototype.animate = function (keyframes: Keyframe[], opts: KeyframeAnimationOptions) {
      const anim = { onfinish: null as null | (() => void), cancel: vi.fn() };
      made.push({ keyframes, opts, anim });
      return anim as unknown as Animation;
    } as typeof Element.prototype.animate;
    Element.prototype.getAnimations = () => [running as unknown as Animation];
    return { made, running };
  }

  it('leaves the toggle to the browser where CSS can animate it', () => {
    vi.spyOn(CSS, 'supports').mockReturnValue(true);
    withAnimations();
    const { container } = render(<FaqList />);
    expect(click(summaries(container)[1])).toBe(true);
  });

  it('leaves it native under reduced motion, and without Web Animations', () => {
    const { container } = render(<FaqList />);
    // jsdom: no Element.animate
    expect(click(summaries(container)[1])).toBe(true);

    withAnimations();
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: true } as MediaQueryList);
    expect(click(summaries(container)[1])).toBe(true);
  });

  it('opens an answer from nothing, and folds it before it closes', () => {
    const { made, running } = withAnimations();
    const { container } = render(<FaqList />);
    const details = container.querySelectorAll('details')[1];

    expect(click(summaries(container)[1])).toBe(false);
    expect(running.cancel).toHaveBeenCalled();
    expect(details.open).toBe(true);
    expect(details).toHaveAttribute('data-expanded');
    expect(made[0].keyframes[0]).toMatchObject({ height: '0px', paddingBottom: '0px' });

    click(summaries(container)[1]);
    expect(details).toHaveAttribute('data-closing');
    expect(details).not.toHaveAttribute('data-expanded');
    // Still open while it folds, so the answer is visible going away.
    expect(details.open).toBe(true);
    expect(made[1].keyframes[1]).toMatchObject({ height: '0px' });

    made[1].anim.onfinish!();
    expect(details.open).toBe(false);
    expect(details).not.toHaveAttribute('data-closing');
    expect(made[1].anim.cancel).toHaveBeenCalled();
  });

  it('reopens a folding answer from where it is', () => {
    const { made } = withAnimations();
    const { container } = render(<FaqList openFirst />);
    const details = container.querySelectorAll('details')[0];

    click(summaries(container)[0]);
    expect(details).toHaveAttribute('data-closing');
    click(summaries(container)[0]);
    expect(details).not.toHaveAttribute('data-closing');
    expect(details).toHaveAttribute('data-expanded');
    // From the panel's own height and padding, not from nothing.
    expect(made[1].keyframes[0].paddingBottom).not.toBe('0px');
  });

  it('keeps the chevron on the native toggle, unless a fold owns it', () => {
    const { container } = render(<FaqList />);
    const details = container.querySelectorAll('details')[2];

    details.open = true;
    details.dispatchEvent(new Event('toggle'));
    expect(details).toHaveAttribute('data-expanded');

    details.setAttribute('data-closing', '');
    details.open = false;
    details.dispatchEvent(new Event('toggle'));
    expect(details).toHaveAttribute('data-expanded');
  });
});
