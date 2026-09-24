import { ChevronDown, HelpCircle } from 'lucide-react';
import { motionAllowed } from '@/lib/mock-motion';
import { cn } from '@/lib/utils';
import { t } from '../i18n';

export const FAQ_KEYS = [
  'what',
  'aria2c',
  'external',
  'persist',
  'files',
  'seed',
  'mouse',
  'telemetry',
] as const;

interface FaqListProps {
  /**
   * `card` is the boxed list the home page has always shown; `flush` draws the
   * same entries as bare rows separated by rules, for a page that frames them
   * itself (the landing's two-column FAQ).
   */
  variant?: 'card' | 'flush';
  /**
   * Opens the first answer. Eight identical closed rows give no sign that there
   * is anything behind them; one open answer shows the shape of the rest.
   */
  openFirst?: boolean;
  className?: string;
}

/** The emphasized curve and the medium step, as globals.css spells them. */
const EMPH = 'cubic-bezier(0.2, 0, 0, 1)';
const DUR = 250;

/**
 * Opens and closes an answer on the chevron's clock where CSS cannot.
 *
 * globals.css animates `::details-content` to `block-size: auto`, which needs
 * `interpolate-size`; a browser without it (Safari and Firefox, today) would
 * snap the panel while the chevron turns. There the click is taken over: the
 * panel's height and bottom padding are tweened from measured pixels, and a
 * closing answer stays open until it has folded. Everywhere else, and under
 * reduced motion, the native toggle runs untouched.
 */
export function tweenToggle(e: React.MouseEvent<HTMLElement>): void {
  const details = e.currentTarget.parentElement as HTMLDetailsElement;
  const panel = details.querySelector<HTMLElement>('[data-faq-panel]')!;
  if (CSS.supports('interpolate-size', 'allow-keywords')) return;
  if (!motionAllowed() || typeof panel.animate !== 'function') return;
  e.preventDefault();
  panel.getAnimations().forEach((a) => a.cancel());
  const pad = getComputedStyle(panel).paddingBottom;
  // A click while it is still folding opens it again from where it is.
  const closing = details.hasAttribute('data-closing');
  if (!details.open || closing) {
    const from = closing ? panel.offsetHeight : 0;
    details.removeAttribute('data-closing');
    details.open = true;
    details.setAttribute('data-expanded', '');
    panel.animate(
      [
        { height: `${from}px`, paddingBottom: closing ? pad : '0px', overflow: 'hidden' },
        { height: `${panel.scrollHeight}px`, paddingBottom: pad, overflow: 'hidden' },
      ],
      { duration: DUR, easing: EMPH },
    );
    return;
  }
  details.setAttribute('data-closing', '');
  details.removeAttribute('data-expanded');
  const fold = panel.animate(
    [
      { height: `${panel.offsetHeight}px`, paddingBottom: pad, overflow: 'hidden' },
      { height: '0px', paddingBottom: '0px', overflow: 'hidden' },
    ],
    { duration: DUR, easing: EMPH, fill: 'forwards' },
  );
  fold.onfinish = () => {
    details.open = false;
    details.removeAttribute('data-closing');
    fold.cancel();
  };
}

/** Keeps the chevron's attribute on the native toggle; a tweened close owns it. */
export function syncExpanded(e: React.SyntheticEvent<HTMLDetailsElement>): void {
  const d = e.currentTarget;
  if (!d.hasAttribute('data-closing')) d.toggleAttribute('data-expanded', d.open);
}

// The questions and answers on their own, one collapsible entry each. Two
// pages show them and only differ in the frame around them, so the frame is a
// variant rather than a second copy of the list.
export function FaqList({ variant = 'card', openFirst = false, className }: FaqListProps) {
  return (
    <div
      className={cn(
        'divide-y divide-outline-variant',
        variant === 'card' &&
          'overflow-hidden rounded-md border border-outline-variant bg-surface-container-low',
        className,
      )}
    >
      {FAQ_KEYS.map((k, i) => {
        const q = t(`home.faq.${k}.q`);
        const a = t(`home.faq.${k}.a`);
        return (
          <details
            key={k}
            open={openFirst && i === 0}
            // The chevron turns off this attribute rather than off `[open]`:
            // a tweened close keeps the answer open until it has folded, and
            // the chevron has to start turning on the click, with the panel.
            data-expanded={(openFirst && i === 0) || undefined}
            onToggle={syncExpanded}
            className="faq-details group"
          >
            <summary
              onClick={tweenToggle}
              className={cn(
                'flex cursor-pointer list-none items-start gap-3 text-on-surface marker:hidden',
                variant === 'card'
                  // The boxed rows are the card's own surface, so they take the
                  // M3 hover/press layer.
                  ? 'm3-state-layer px-4 py-3 text-title-small'
                  // The flush rows take the hover fill alone — the press and
                  // focus fills linger on a bare background and read as a
                  // selected band — and no radius, so the hover spans the full
                  // row and the focus ring (which inherits the radius) is
                  // square with it.
                  : 'm3-hover-layer px-4 py-5 text-title-medium',
              )}
            >
              {variant === 'card' && (
                <ChevronDown className="mt-0.5 h-4 w-4 shrink-0 text-on-surface-variant transition-transform duration-med ease-emph group-data-[expanded]:rotate-180" />
              )}
              <span className="flex-1">{q}</span>
              {variant === 'flush' && (
                <ChevronDown className="mt-0.5 h-[18px] w-[18px] shrink-0 text-on-surface-variant transition-transform duration-med ease-emph group-data-[expanded]:rotate-180 group-data-[expanded]:text-primary" />
              )}
            </summary>
            <div
              data-faq-panel
              className={cn(
                'text-body-medium text-on-surface-variant',
                variant === 'card' ? 'px-4 pb-4 ps-11' : 'px-4 pb-5 pe-12',
              )}
            >
              {a}
            </div>
          </details>
        );
      })}
    </div>
  );
}

export function FaqSection() {
  return (
    <section className="scroll-mt-24 space-y-4 pb-12" id="faq">
      <h2 className="flex items-center gap-2 text-headline-small font-medium tracking-tight">
        <HelpCircle className="h-5 w-5 text-on-surface-variant" />
        {t('home.faq.h2')}
      </h2>
      <FaqList />
    </section>
  );
}
