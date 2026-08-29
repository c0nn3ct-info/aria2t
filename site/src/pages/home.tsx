import { useState } from 'react';
import { ArrowRight, Check, Chrome, ExternalLink, PackageOpen } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Section } from '@/components/m3/section';
import { ArchitectureDiagram } from '../components/architecture-diagram';
import { FaqSection } from '../components/faq-section';
import { ListMock } from '../components/list-mock';
import { TerminalMock } from '../components/terminal-mock';
import { BrowserMock } from '../components/browser-mock';
import { PopupMock } from '../components/popup-mock';
import { SurfaceSwitch, type Surface } from '../components/surface-switch';
import { localePath, t } from '../i18n';
import { Layout } from '../layout';

const SOURCES = [
  'HTTP(S)',
  'FTP',
  'SFTP',
  'BitTorrent',
  'magnet:',
  '.torrent',
  'Metalink',
  'aria2 input file',
] as const;

const FEATURE_KEYS = [
  'sources',
  'nosetup',
  'capture',
  'manage',
  'detail',
  'limits',
  'integrity',
  'persist',
  'mouse',
] as const;

function Showcase({ surface }: { surface: Surface }) {
  return surface === 'terminal' ? (
    <TerminalMock>
      <ListMock />
    </TerminalMock>
  ) : (
    <BrowserMock>
      <PopupMock />
    </BrowserMock>
  );
}

export function HomePage() {
  const installHref = localePath('/install/');
  // The extension is the surface most visitors can use, so the hero opens on
  // it; the switch swaps the mock, while both CTAs stay visible because they
  // are two different things to get, not two views of one.
  const [surface, setSurface] = useState<Surface>('extension');
  return (
    <Layout current="home">
      <section className="grid items-start gap-8 pb-10 lg:grid-cols-[1.05fr_1fr] lg:gap-10">
        <div className="flex min-w-0 flex-col items-start gap-6">
          <h1 className="text-display-small font-semibold leading-[1.05] tracking-tight sm:text-[44px] lg:text-[52px]">
            {t('home.hero.h1')}
            <br />
            <span className="text-on-surface-variant">{t('home.hero.h1_sub')}</span>
          </h1>
          <div className="flex w-full flex-col items-center gap-3 lg:hidden">
            <SurfaceSwitch value={surface} onChange={setSurface} className="self-start" />
            <div className="w-full max-w-[560px]">
              <Showcase surface={surface} />
            </div>
          </div>
          <p className="max-w-xl text-body-large text-on-surface-variant">{t('home.hero.lede')}</p>
          <div className="flex flex-wrap items-center gap-2">
            <Button asChild variant="filled" size="s">
              <a href={installHref}>
                {t('home.hero.cta_install')}
                <ArrowRight className="rtl:-scale-x-100" />
              </a>
            </Button>
            {/* Disabled until the listing is live: a store button that 404s is
                worse than one that plainly cannot be pressed yet. */}
            <Button variant="outlined" size="s" disabled title={t('home.hero.cta_webstore_soon')}>
              <Chrome />
              {t('home.hero.cta_webstore')}
              <ExternalLink />
            </Button>
          </div>
        </div>
        <div className="hidden min-w-0 lg:block lg:pt-2">
          <div className="mb-3 flex justify-end">
            <SurfaceSwitch value={surface} onChange={setSurface} />
          </div>
          <Showcase surface={surface} />
        </div>
      </section>

      <div className="pb-12">
        <div className="flex items-baseline gap-3 border-b border-outline-variant pb-2">
          <span className="text-label-medium uppercase tracking-[0.12em] text-on-surface-variant">
            {t('home.works_with')}
          </span>
        </div>
        <ul className="mt-3 flex flex-wrap gap-2">
          {SOURCES.map((p) => (
            <li key={p}>
              <Badge variant="outline" size="md" className="font-mono">
                {p}
              </Badge>
            </li>
          ))}
        </ul>
      </div>

      <div className="space-y-4 pb-12">
        <Section header={t('home.what_you_get')} icon={Check} count={FEATURE_KEYS.length}>
          <ul className="space-y-2 px-2 pb-2 pt-1">
            {FEATURE_KEYS.map((k) => (
              <li
                key={k}
                className="flex items-start gap-3 rounded-md px-2 py-2"
              >
                <span className="mt-1 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
                  <Check className="h-3.5 w-3.5" />
                </span>
                <div className="min-w-0">
                  <div className="text-title-small">{t(`home.feat.${k}.title`)}</div>
                  <div className="text-body-medium text-on-surface-variant">
                    {t(`home.feat.${k}.body`)}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </Section>
      </div>

      <section className="space-y-3 pb-12" id="sources">
        <h2 className="flex items-center gap-2 text-headline-small font-medium tracking-tight">
          <PackageOpen className="h-5 w-5 text-on-surface-variant" />
          {t('home.sources.h2')}
        </h2>
        <p className="max-w-3xl text-body-large text-on-surface-variant">
          {t('home.sources.body')}
        </p>
      </section>

      <section id="how-it-works" className="scroll-mt-24 space-y-4 pb-12">
        <h2 className="text-headline-small font-medium tracking-tight">{t('home.how.h2')}</h2>
        <p className="max-w-2xl text-body-medium text-on-surface-variant">{t('home.how.body')}</p>
        <ArchitectureDiagram />
      </section>

      <section className="space-y-3 pb-12" id="engine">
        <h2 className="text-headline-small font-medium tracking-tight">{t('home.engine.h2')}</h2>
        <p className="max-w-3xl text-body-large text-on-surface-variant">
          {t('home.engine.body')}
        </p>
      </section>

      <FaqSection />

      <section className="space-y-4 pb-4">
        <h2 className="text-headline-small font-medium tracking-tight">{t('home.start.h2')}</h2>
        <p className="text-body-large text-on-surface-variant">{t('home.start.body')}</p>
        <Button asChild variant="filled-tonal" size="s">
          <a href={installHref}>
            {t('home.start.cta')}
            <ArrowRight className="rtl:-scale-x-100" />
          </a>
        </Button>
      </section>
    </Layout>
  );
}
