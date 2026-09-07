// The home page: one queue, told as a sequence of bands rather than as a
// column of prose. It replaced a page of prose with the same `Layout`, the same
// footer, the same install and store CTAs and the same eight FAQ entries.
//
// Every band lays itself out edge to edge, so the layout is asked for its
// chrome without its reading column (`bleed`).
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
    // The comp is a dark stage in one palette from the header to the footer, so
    // the page declares that palette once and every band inherits it. Both
    // attributes are the site's own token hooks, not a private stylesheet:
    // `data-theme="dark"` re-resolves the M3 tokens to their dark values for
    // this subtree, and `data-accent="blue"` swaps in the comp's blue and
    // violet (see the `[data-accent='blue']` block in globals.css). Everything
    // inside, from Button and Badge to the popup and terminal mocks and the
    // site header and footer, picks both up without knowing this page exists.
    <div data-theme="dark" data-accent="blue">
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
    </div>
  );
}
