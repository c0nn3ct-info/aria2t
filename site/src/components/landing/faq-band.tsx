// The closing band: the eight questions the site already answers, beside the
// two ways to ask a ninth. The questions themselves are `FaqList`, the same
// component and the same strings the current home page shows, in its flush
// frame rather than its boxed one.
import { ArrowRight, Github, Mail } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { FaqList } from '@/components/faq-section';
import { CONTACT_MAILTO, GITHUB_URL } from '@/constants';
import { LandingSection } from './shell';
import { localePath, t } from '@/i18n';

/** `mailto:help@c0nn3ct.info` → `help@c0nn3ct.info`, for the visible label. */
export function mailtoAddress(mailto: string): string {
  return mailto.replace(/^mailto:/, '');
}

/** `https://github.com/c0nn3ct-info/aria2t` → `github.com/c0nn3ct-info/aria2t`. */
export function repoLabel(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/$/, '');
}

export function FaqBand() {
  return (
    <LandingSection id="faq">
      <div className="grid gap-10 lg:grid-cols-[360px_minmax(0,1fr)] lg:gap-20">
        <div>
          <h2 data-enter className="text-balance text-[clamp(28px,4.4vw,44px)] font-semibold leading-[1.08] tracking-[-0.035em]">
            {t('landing.faq.h2')}
          </h2>
          <p data-enter className="mt-4 text-pretty text-body-large leading-[1.7] text-on-surface-variant">
            {t('landing.faq.body')}
          </p>

          <div data-enter className="mt-7 flex flex-col gap-3 border-t border-outline-variant pt-6">
            <span className="text-label-small uppercase tracking-[0.14em] text-on-surface-variant">
              {t('landing.faq.no_answer')}
            </span>
            <a
              className="inline-flex min-h-[44px] items-center gap-2.5 text-body-medium font-medium underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              href={GITHUB_URL}
              target="_blank"
              rel="noreferrer noopener"
            >
              <Github className="h-4 w-4 shrink-0" aria-hidden />
              <span dir="ltr" className="min-w-0 truncate">
                {repoLabel(GITHUB_URL)}
              </span>
            </a>
            <a
              className="inline-flex min-h-[44px] items-center gap-2.5 text-body-medium font-medium underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              href={CONTACT_MAILTO}
            >
              <Mail className="h-4 w-4 shrink-0" aria-hidden />
              <span dir="ltr" className="min-w-0 truncate">
                {mailtoAddress(CONTACT_MAILTO)}
              </span>
            </a>
          </div>

          <Button asChild variant="outlined" size="s" className="mt-6">
            <a href={localePath('/install/')}>
              {t('home.start.cta')}
              <ArrowRight className="rtl:-scale-x-100" />
            </a>
          </Button>
        </div>

        {/* Wrapped, and as one object rather than a stagger: `FaqList` is the
            current home page's list too, and the entrance rules are global, so
            an attribute inside it would animate a page that never asked. */}
        <div data-enter>
          <FaqList variant="flush" openFirst />
        </div>
      </div>
    </LandingSection>
  );
}
