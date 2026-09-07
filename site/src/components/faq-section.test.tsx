import { afterEach, describe, expect, it } from 'vitest';
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
