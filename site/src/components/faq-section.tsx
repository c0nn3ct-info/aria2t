import { ChevronDown, HelpCircle } from 'lucide-react';
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
          <details key={k} open={openFirst && i === 0} className="group">
            <summary
              className={cn(
                'm3-state-layer flex cursor-pointer list-none items-start gap-3 text-on-surface marker:hidden',
                variant === 'card'
                  ? 'px-4 py-3 text-title-small'
                  // Padded and rounded: the hover surface spans the row, so without
                  // it the question sits flush against the highlight's edge.
                  : 'rounded-md px-4 py-5 text-title-medium',
              )}
            >
              {variant === 'card' && (
                <ChevronDown className="mt-0.5 h-4 w-4 shrink-0 text-on-surface-variant transition-transform duration-short ease-emph group-open:rotate-180" />
              )}
              <span className="flex-1">{q}</span>
              {variant === 'flush' && (
                <ChevronDown className="mt-0.5 h-[18px] w-[18px] shrink-0 text-on-surface-variant transition-transform duration-short ease-emph group-open:rotate-180 group-open:text-primary" />
              )}
            </summary>
            <div
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
