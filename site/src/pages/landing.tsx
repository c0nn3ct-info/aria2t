// The home page: one queue, told as a sequence of bands rather than as a
// column of prose. It replaced a page of prose with the same `Layout`, the same
// footer, the same install and store CTAs and the same eight FAQ entries.
//
// Every band lays itself out edge to edge, so the layout is asked for its
// chrome without its reading column (`bleed`). Blue is the site's permanent
// accent (`main.tsx` sets it once at mount), so this page needs no wrapper of
// its own for it — and follows the real system light/dark preference like
// every other page.
import { FaqBand } from '@/components/landing/faq-band';
import { FilesSection } from '@/components/landing/files-section';
import { LandingHero } from '@/components/landing/hero';
import { LimitsSection } from '@/components/landing/limits-section';
import { PiecesSection } from '@/components/landing/pieces-section';
import { QueueSection } from '@/components/landing/queue-section';
import { StatsSection } from '@/components/landing/stats-section';
import { SurfacesSection } from '@/components/landing/surfaces';
import { useSectionEntrance } from '@/lib/use-enter';
import { Layout } from '../layout';

export function LandingPage() {
  // Bands arrive as they come up. Where the browser has scroll-driven
  // timelines this is pure CSS and the hook does nothing; see
  // `src/lib/use-enter.ts` for which path runs where.
  useSectionEntrance();
  return (
    <Layout current="home" bleed>
      <LandingHero />
      <SurfacesSection />
      <QueueSection />
      <StatsSection />
      <FilesSection />
      <PiecesSection />
      <LimitsSection />
      <FaqBand />
    </Layout>
  );
}
