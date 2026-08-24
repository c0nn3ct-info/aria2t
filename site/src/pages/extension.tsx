import { useState } from 'react';
import {
  Apple,
  AppWindow,
  Check,
  Chrome,
  Copy,
  ExternalLink,
  FolderTree,
  Globe,
  Info,
  Magnet,
  Puzzle,
  RefreshCw,
  Server,
  Terminal,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { IconButton } from '@/components/ui/icon-button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Section } from '@/components/m3/section';
import { t } from '../i18n';
import { Layout } from '../layout';

const WEB_STORE_URL = 'https://chromewebstore.google.com/';
const SITE_BASE = 'https://aria2t.c0nn3ct.info';

function CodeBlock({ children }: { children: string }) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      // clipboard blocked — silently no-op
    }
  };

  return (
    <div className="group relative rounded-md bg-surface-container-highest">
      <pre className="overflow-x-auto px-3 py-3 pe-12 text-body-small font-mono text-on-surface">
        <code>{children}</code>
      </pre>
      <IconButton
        type="button"
        variant="standard"
        size="xs"
        onClick={() => void copy()}
        aria-label={copied ? t('code.copied') : t('code.copy')}
        title={copied ? t('code.copied') : t('code.copy_short')}
        className="absolute end-1.5 top-1.5 text-on-surface-variant"
      >
        {copied ? <Check /> : <Copy />}
      </IconButton>
      {/* Swapping the button's aria-label does not announce anything; a live
          region is the only thing a screen reader reports on copy. */}
      <span role="status" aria-live="polite" className="sr-only">
        {copied ? t('code.copied') : ''}
      </span>
    </div>
  );
}

export function ExtensionPage() {
  return (
    <Layout current="extension">
      <section className="space-y-3 pb-8">
        <h1 className="text-headline-large font-semibold tracking-tight">{t('extension.h1')}</h1>
        <p className="text-body-large text-on-surface-variant">{t('extension.lede')}</p>
      </section>

      <section className="pb-8">
        <Card variant="filled" padding="md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Info className="h-4 w-4 text-on-surface-variant" />
              {t('extension.what.title')}
            </CardTitle>
          </CardHeader>
          <ul className="mt-3 space-y-2 text-body-medium text-on-surface-variant">
            <li className="flex items-start gap-2">
              <Magnet className="mt-0.5 h-4 w-4 shrink-0" />
              {t('extension.what.capture')}
            </li>
            <li className="flex items-start gap-2">
              <FolderTree className="mt-0.5 h-4 w-4 shrink-0" />
              {t('extension.what.files')}
            </li>
            <li className="flex items-start gap-2">
              <Server className="mt-0.5 h-4 w-4 shrink-0" />
              {t('extension.what.external')}
            </li>
          </ul>
        </Card>
      </section>

      <div className="space-y-4 pb-8">
        <Section header={t('extension.step1.title')} icon={Puzzle}>
          <div className="space-y-5 px-2 pb-3 pt-2 text-body-large text-on-surface-variant">
            <p>{t('extension.step1.body')}</p>
            <div>
              <Button asChild variant="outlined" size="s">
                <a href={WEB_STORE_URL} target="_blank" rel="noreferrer noopener">
                  <Chrome />
                  {t('extension.step1.cta')}
                  <ExternalLink />
                </a>
              </Button>
            </div>
          </div>
        </Section>

        <Section header={t('extension.step2.title')} icon={Terminal}>
          <div className="space-y-5 px-2 pb-3 pt-2 text-body-large text-on-surface-variant">
            <p>{t('extension.step2.body1')}</p>

            <div className="space-y-2">
              <h3 className="flex items-center gap-2 text-title-small text-on-surface">
                <Apple className="h-4 w-4" />
                macOS / <Terminal className="h-4 w-4" /> Linux
              </h3>
              <CodeBlock>{`curl -fsSL ${SITE_BASE}/install.sh | sh -s -- <extension-id>`}</CodeBlock>
            </div>

            <div className="space-y-2">
              <h3 className="flex items-center gap-2 text-title-small text-on-surface">
                <AppWindow className="h-4 w-4" />
                Windows (PowerShell)
              </h3>
              <CodeBlock>{`$env:ARIA2T_EXT_ID='<extension-id>'; iwr -useb ${SITE_BASE}/windows.ps1 | iex`}</CodeBlock>
            </div>

            <p>{t('extension.step2.id_note')}</p>
          </div>
        </Section>

        <Section header={t('extension.step3.title')} icon={RefreshCw}>
          <div className="space-y-5 px-2 pb-3 pt-2 text-body-large text-on-surface-variant">
            <p>{t('extension.step3.body')}</p>
          </div>
        </Section>
      </div>

      <div className="grid gap-3 pb-8 sm:grid-cols-2">
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <RefreshCw className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('extension.updating.title')}</CardTitle>
            <CardDescription>{t('extension.updating.body')}</CardDescription>
          </CardHeader>
        </Card>
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <Globe className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('extension.browsers.title')}</CardTitle>
            <CardDescription>{t('extension.browsers.body')}</CardDescription>
          </CardHeader>
        </Card>
      </div>
    </Layout>
  );
}
