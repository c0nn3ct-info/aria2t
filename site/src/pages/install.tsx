import { useState } from 'react';
import {
  Apple,
  Check,
  Copy,
  Download,
  ExternalLink,
  Github,
  Info,
  MonitorCheck,
  PlayCircle,
  RefreshCw,
  Terminal,
  Trash2,
  Wrench,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { IconButton } from '@/components/ui/icon-button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Section } from '@/components/m3/section';
import { GITHUB_URL } from '../constants';
import { t } from '../i18n';
import { Layout } from '../layout';

const RELEASES_URL = `${GITHUB_URL}/releases`;

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
        aria-label={copied ? 'Copied' : 'Copy command'}
        title={copied ? 'Copied' : 'Copy'}
        className="absolute end-1.5 top-1.5 text-on-surface-variant"
      >
        {copied ? <Check /> : <Copy />}
      </IconButton>
    </div>
  );
}

export function InstallPage() {
  return (
    <Layout current="install">
      <section className="space-y-3 pb-8">
        <h1 className="text-headline-large font-semibold tracking-tight">{t('install.h1')}</h1>
        <p className="text-body-large text-on-surface-variant">{t('install.lede')}</p>
      </section>

      <section className="pb-8">
        <Card variant="filled" padding="md">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Info className="h-4 w-4 text-on-surface-variant" />
              {t('install.before.title')}
            </CardTitle>
          </CardHeader>
          <ul className="mt-3 space-y-2 text-body-medium text-on-surface-variant">
            <li className="flex items-start gap-2">
              <Download className="mt-0.5 h-4 w-4 shrink-0" />
              {t('install.before.aria2')}
            </li>
            <li className="flex items-start gap-2">
              <MonitorCheck className="mt-0.5 h-4 w-4 shrink-0" />
              {t('install.before.terminal')}
            </li>
            <li className="flex items-start gap-2">
              <Wrench className="mt-0.5 h-4 w-4 shrink-0" />
              {t('install.before.go')}
            </li>
          </ul>
        </Card>
      </section>

      <div className="space-y-4 pb-8">
        <Section header={t('install.step1.title')} icon={Download}>
          <div className="space-y-5 px-2 pb-3 pt-2 text-body-large text-on-surface-variant">
            <p>{t('install.step1.body')}</p>

            <div className="space-y-2">
              <h3 className="flex items-center gap-2 text-title-small text-on-surface">
                <Apple className="h-4 w-4" />
                macOS
              </h3>
              <CodeBlock>brew install aria2</CodeBlock>
            </div>

            <div className="space-y-2">
              <h3 className="flex items-center gap-2 text-title-small text-on-surface">
                <Terminal className="h-4 w-4" />
                Linux
              </h3>
              <CodeBlock>sudo apt install aria2</CodeBlock>
            </div>
          </div>
        </Section>

        <Section header={t('install.step2.title')} icon={Terminal}>
          <div className="space-y-5 px-2 pb-3 pt-2 text-body-large text-on-surface-variant">
            <p>{t('install.step2.body1')}</p>

            <div>
              <Button asChild variant="outlined" size="s">
                <a href={RELEASES_URL} target="_blank" rel="noreferrer noopener">
                  <Github />
                  {t('install.step2.releases_cta')}
                  <ExternalLink />
                </a>
              </Button>
            </div>

            <p>{t('install.step2.body2')}</p>
            <CodeBlock>{`git clone ${GITHUB_URL}.git
cd aria2t/tui && go build -o aria2t ./cmd/aria2t`}</CodeBlock>
          </div>
        </Section>

        <Section header={t('install.step3.title')} icon={PlayCircle}>
          <div className="space-y-5 px-2 pb-3 pt-2 text-body-large text-on-surface-variant">
            <p>{t('install.step3.body1')}</p>
            <CodeBlock>./aria2t</CodeBlock>
            <p>{t('install.step3.body2')}</p>
            <CodeBlock>./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret</CodeBlock>
            <p>{t('install.step3.body3')}</p>
          </div>
        </Section>
      </div>

      <div className="grid gap-3 pb-8 sm:grid-cols-2">
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <RefreshCw className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('install.updating.title')}</CardTitle>
            <CardDescription>{t('install.updating.body')}</CardDescription>
          </CardHeader>
        </Card>
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <Trash2 className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('install.uninstalling.title')}</CardTitle>
          </CardHeader>
          <ol className="mt-3 space-y-2 ps-5 text-body-medium text-on-surface-variant list-decimal">
            <li>{t('install.uninstalling.step1')}</li>
            <li>
              {t('install.uninstalling.step2') + ' '}
              <code className="rounded bg-surface-container-highest px-1 py-0.5 font-mono text-body-small">
                ~/.config/aria2t/
              </code>
            </li>
          </ol>
        </Card>
      </div>
    </Layout>
  );
}
