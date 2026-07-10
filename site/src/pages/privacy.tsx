import { Ban, Database, Github, Mail, Network } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Section } from '@/components/m3/section';
import { GITHUB_URL } from '../constants';
import { t } from '../i18n';
import { Layout } from '../layout';

const STORE_ITEMS = [
  'privacy.stores.item1',
  'privacy.stores.item2',
  'privacy.stores.item3',
  'privacy.stores.item4',
];

const NOTHING_ITEMS = [
  'privacy.nothing.item1',
  'privacy.nothing.item2',
  'privacy.nothing.item3',
  'privacy.nothing.item4',
];

export function PrivacyPage() {
  return (
    <Layout current="privacy">
      <section className="space-y-3 pb-8">
        <h1 className="text-headline-large font-semibold tracking-tight">{t('privacy.h1')}</h1>
        <p className="text-label-medium uppercase tracking-[0.16em] text-on-surface-variant">
          {t('privacy.last_updated')}
        </p>
        <p className="text-body-large text-on-surface-variant">{t('privacy.lede')}</p>
      </section>

      <div className="space-y-4 pb-8">
        <Section header={t('privacy.stores.h2')} icon={Database}>
          <div className="space-y-3 px-2 py-2 text-body-large text-on-surface-variant">
            <p>{t('privacy.stores.intro')}</p>
            <ul className="space-y-1.5">
              {STORE_ITEMS.map((k) => (
                <li key={k} className="flex items-start gap-2">
                  <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-on-surface-variant" />
                  {t(k)}
                </li>
              ))}
            </ul>
            <p>{t('privacy.stores.outro')}</p>
          </div>
        </Section>

        <Section header={t('privacy.network.h2')} icon={Network}>
          <div className="space-y-3 px-2 py-2 text-body-large text-on-surface-variant">
            <p>{t('privacy.network.intro')}</p>
            <ol className="space-y-1.5 ps-5 list-decimal">
              <li>
                <b className="text-on-surface">{t('privacy.network.downloads.b')}</b>
                {t('privacy.network.downloads.body')}
              </li>
              <li>
                <b className="text-on-surface">{t('privacy.network.rpc.b')}</b>
                {t('privacy.network.rpc.body')}
              </li>
              <li>
                <b className="text-on-surface">{t('privacy.network.site.b')}</b>
                {t('privacy.network.site.body')}
              </li>
            </ol>
            <p>{t('privacy.network.outro')}</p>
          </div>
        </Section>

        <Section header={t('privacy.nothing.h2')} icon={Ban}>
          <ul className="space-y-2 px-2 py-2 text-body-large text-on-surface-variant">
            {NOTHING_ITEMS.map((k) => (
              <li key={k} className="flex items-start gap-2">
                <Ban className="mt-1 h-4 w-4 shrink-0 text-on-surface-variant" />
                {t(k)}
              </li>
            ))}
          </ul>
        </Section>
      </div>

      <div className="grid gap-3 pb-8 sm:grid-cols-2">
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <Github className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('privacy.oss.h3')}</CardTitle>
          </CardHeader>
          <p className="mt-2 text-body-medium text-on-surface-variant">
            {t('privacy.oss.body_before')}
            <a
              className="text-on-surface underline underline-offset-4 hover:text-primary"
              href={GITHUB_URL}
            >
              {t('privacy.oss.body_link')}
            </a>
            {t('privacy.oss.body_after')}
          </p>
        </Card>
        <Card variant="outlined" padding="md">
          <CardHeader>
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-secondary-container text-secondary-on-container">
              <Mail className="h-5 w-5" />
            </span>
            <CardTitle className="mt-2">{t('privacy.contact.h3')}</CardTitle>
          </CardHeader>
          <p className="mt-2 text-body-medium text-on-surface-variant">
            {t('privacy.contact.body_before')}
            <a
              className="text-on-surface underline underline-offset-4 hover:text-primary"
              href="mailto:help@c0nn3ct.info"
            >
              help@c0nn3ct.info
            </a>
            {t('privacy.contact.body_after')}
          </p>
        </Card>
      </div>

      <section className="pb-4 text-body-medium text-on-surface-variant">
        <h2 className="text-title-medium text-on-surface">{t('privacy.changes.h2')}</h2>
        <p className="mt-2">{t('privacy.changes.body')}</p>
      </section>
    </Layout>
  );
}
