import { FileText, Github, Package } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Section } from '@/components/m3/section';
import { GITHUB_URL } from '../constants';
import { t } from '../i18n';
import { Layout } from '../layout';

const LICENSE_FILE_URL = `${GITHUB_URL}/blob/main/LICENSE`;

export function LicensePage() {
  return (
    <Layout current="license">
      <section className="space-y-3 pb-8">
        <h1 className="text-headline-large font-semibold tracking-tight">{t('license.h1')}</h1>
        <p className="text-label-medium uppercase tracking-[0.16em] text-on-surface-variant">
          {t('license.last_updated')}
        </p>
        <p className="text-body-large text-on-surface-variant">{t('license.intro')}</p>
        <ol className="space-y-1 ps-5 text-body-large text-on-surface-variant list-decimal">
          <li>
            <b className="text-on-surface">{t('license.item1.b')}</b>
            {t('license.item1.body')}
          </li>
          <li>
            <b className="text-on-surface">{t('license.item2.b')}</b>
            {t('license.item2.body')}
          </li>
          <li>
            <b className="text-on-surface">{t('license.item3.b')}</b>
            {t('license.item3.body')}
          </li>
        </ol>
      </section>

      <Section header={t('license.apache.h2')} icon={FileText}>
        <div className="space-y-4 px-2 py-2 text-body-large text-on-surface-variant">
          <p>{t('license.apache.copyright')}</p>
          <p>{t('license.apache.preamble')}</p>

          <div className="space-y-2">
            <h3 className="text-title-small text-on-surface">{t('license.apache.grant.h3')}</h3>
            <p>{t('license.apache.grant.body')}</p>
          </div>

          <div className="space-y-2">
            <h3 className="text-title-small text-on-surface">{t('license.apache.conditions.h3')}</h3>
            <p>{t('license.apache.conditions.intro')}</p>
            <ul className="space-y-1.5 ps-5 list-disc">
              <li>{t('license.apache.conditions.item1')}</li>
              <li>{t('license.apache.conditions.item2')}</li>
              <li>{t('license.apache.conditions.item3')}</li>
            </ul>
          </div>

          <div className="space-y-2">
            <h3 className="text-title-small text-on-surface">{t('license.apache.warranty.h3')}</h3>
            <p>{t('license.apache.warranty.body')}</p>
          </div>

          <div className="space-y-2">
            <h3 className="text-title-small text-on-surface">{t('license.apache.liability.h3')}</h3>
            <p>{t('license.apache.liability.body')}</p>
          </div>

          <p>
            {t('license.apache.full.body_before')}
            <a
              className="text-on-surface underline underline-offset-4 hover:text-primary"
              href={LICENSE_FILE_URL}
            >
              {t('license.apache.full.body_link')}
            </a>
            {t('license.apache.full.body_after')}
          </p>
        </div>
      </Section>

      <div className="grid gap-3 pt-6 pb-8 sm:grid-cols-2">
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <Github className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('license.app.h3')}</CardTitle>
          </CardHeader>
          <p className="mt-2 text-body-medium text-on-surface-variant">
            {t('license.app.body_before')}
            <a
              className="text-on-surface underline underline-offset-4 hover:text-primary"
              href={GITHUB_URL}
            >
              {t('license.app.body_link')}
            </a>
            {t('license.app.body_after')}
          </p>
        </Card>
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <Package className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('license.deps.h3')}</CardTitle>
          </CardHeader>
          <p className="mt-2 text-body-medium text-on-surface-variant">
            {t('license.deps.intro')}
          </p>
          <ul className="mt-2 space-y-1 text-body-medium text-on-surface-variant">
            <li>
              <a
                className="text-on-surface underline underline-offset-4 hover:text-primary"
                href="https://github.com/aria2/aria2"
              >
                aria2
              </a>
              {' — GPL-2.0'}
            </li>
            <li>
              <a
                className="text-on-surface underline underline-offset-4 hover:text-primary"
                href="https://github.com/charmbracelet/bubbletea"
              >
                Bubble Tea
              </a>
              {', '}
              <a
                className="text-on-surface underline underline-offset-4 hover:text-primary"
                href="https://github.com/charmbracelet/bubbles"
              >
                Bubbles
              </a>
              {', '}
              <a
                className="text-on-surface underline underline-offset-4 hover:text-primary"
                href="https://github.com/charmbracelet/lipgloss"
              >
                Lip Gloss
              </a>
              {' — MIT'}
            </li>
            <li>
              <a
                className="text-on-surface underline underline-offset-4 hover:text-primary"
                href="https://github.com/folke/tokyonight.nvim"
              >
                Tokyo Night
              </a>
              {' — ' + t('license.deps.theme')}
            </li>
          </ul>
        </Card>
      </div>
    </Layout>
  );
}
