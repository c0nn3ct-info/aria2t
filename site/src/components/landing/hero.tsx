// The landing hero. The band forces the dark tokens on itself (see
// `LandingSection`), so the type and the washes over the scene are the site's
// own dark palette rather than a second set of colours invented for one page.
//
// The scene sits behind the copy on a wide screen and under it on a phone.
// That is the whole mobile story here: at 390px the pylons stand directly
// where the sentences are, and no amount of dimming makes a lit tower behind a
// paragraph read well. So the copy takes the top of the band and the machinery
// gets a strip of its own beneath it, wide enough to still look like a scene.
import { useCallback, useRef } from 'react';
import { ArrowRight, Chrome, ExternalLink } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { HeroScene } from './hero-scene';
import { Eyebrow } from './shell';
import { localePath, t } from '@/i18n';

/** Every source aria2 accepts, in the order the add overlay lists them. */
const SOURCES = ['HTTP(S)', 'FTP', 'SFTP', 'BitTorrent', 'Metalink', 'magnet'] as const;

/** The page's ground, at an alpha. Every wash below is that colour fading. */
const bg = (alpha?: number) =>
  alpha === undefined ? 'hsl(var(--background))' : `hsl(var(--background) / ${alpha})`;

export function LandingHero() {
  const section = useRef<HTMLElement>(null);
  const heading = useRef<HTMLHeadingElement>(null);

  /**
   * The vertical centre of the heading's first line, as a fraction of the
   * band, in the normalised device coordinates the scene projects into. A
   * Range over the heading's contents gives one rect per rendered line, so
   * this follows the real wrap at any width and in any language.
   */
  const alignTipsNdc = useCallback((): number | null => {
    // Both refs are attached, so there is no null branch to guard: the scene
    // is handed this callback from an effect, calls it from there and from its
    // own resize observer, and disconnects that observer before it lets go.
    const range = document.createRange();
    range.selectNodeContents(heading.current!);
    const first = range.getClientRects()[0];
    const box = section.current!.getBoundingClientRect();
    if (!first || box.height === 0) return null;
    const mid = (first.top + first.bottom) / 2 - box.top;
    return 1 - 2 * (mid / box.height);
  }, []);

  return (
    <section
      ref={section}
      className="relative isolate overflow-hidden bg-background text-on-surface"
    >
      {/* One composition at every size, the way the wide one reads: the scene
          fills the band and the copy sits on it. Mirrored for a right-to-left
          page so the machinery stays on the side the copy is not. */}
      <div className="absolute inset-0 z-0 rtl:-scale-x-100">
        <HeroScene aria-label={t('landing.hero.scene_alt')} alignTipsNdc={alignTipsNdc} />
      </div>

      {/* The wash that keeps the copy readable, turned to match where the copy
          is. On a phone it runs the full width, so the wash runs top to bottom.
          It holds through the heading and the lede, then releases across the
          band the machines occupy, which the buttons and the source chips
          cross with grounds of their own. On a wide screen the copy is a
          column on the left, so the wash runs across instead, holding through
          that column: with the pylons' crowns aligned to the heading line, one
          of them stands behind the end of it. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 z-10 lg:hidden"
        style={{
          background: `linear-gradient(to bottom, ${bg(0.9)} 0%, ${bg(0.88)} 20%, ${bg(0.8)} 30%, ${bg(0.62)} 40%, ${bg(0.42)} 50%, ${bg(0.24)} 60%, ${bg(0.11)} 70%, ${bg(0.04)} 80%, ${bg(0)} 92%)`,
        }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 z-10 hidden rtl:-scale-x-100 lg:block"
        style={{
          background: `linear-gradient(100deg, ${bg(0.99)} 0%, ${bg(0.97)} 28%, ${bg(0.92)} 42%, ${bg(0.7)} 52%, ${bg(0.34)} 62%, ${bg(0.08)} 74%, ${bg(0)} 86%)`,
        }}
      />
      {/* Reaches full opacity, which is what makes the seam with the next band
          disappear instead of ending on a cut edge. Proportional below `lg`,
          where the band's height varies with how the copy wraps and a fixed
          90px was too short to swallow the belt on its way out. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 bottom-0 z-10 h-[26%] lg:h-[210px]"
        style={{
          background: `linear-gradient(to bottom, ${bg(0)} 0%, ${bg(0.12)} 24%, ${bg(0.38)} 48%, ${bg(0.68)} 70%, ${bg(0.9)} 86%, ${bg()} 100%)`,
        }}
      />
      {/* The conveyor runs diagonally out of the bottom-left corner, straight
          through where the sources sit, and arrives there at full brightness
          because it is the nearest thing to the camera. This pocket of shadow
          puts it back behind the type. Needed at every size: the vertical wash
          above has already released by the time the belt reaches the corner. */}
      <div
        aria-hidden
        className="pointer-events-none absolute bottom-0 start-0 z-10 h-[60%] w-[58%] rtl:-scale-x-100 sm:w-[50%] lg:h-[52%] lg:w-[44%]"
        style={{
          background: `radial-gradient(110% 85% at 0% 100%, ${bg(0.92)} 0%, ${bg(0.55)} 45%, ${bg(0)} 78%)`,
        }}
      />

      <div className="relative z-20 mx-auto flex min-h-[620px] w-full max-w-[1160px] flex-col px-5 pb-16 pt-14 sm:min-h-[720px] sm:px-8 sm:pt-20 lg:min-h-[720px] lg:justify-center lg:px-10 lg:py-24">
        <div className="max-w-[600px]">
          <h1
            ref={heading}
            className="text-balance text-[clamp(34px,7vw,64px)] font-semibold leading-[1.04] tracking-[-0.042em]"
          >
            {t('landing.hero.h1')}
          </h1>
          <p className="mt-4 max-w-[520px] text-pretty text-[17px] leading-[1.6] sm:mt-5 sm:text-[18px] sm:leading-[1.65]">
            {t('landing.hero.lede')}
          </p>
        </div>

        {/* Full width while they stack, their own width once they sit in a row. */}
        <div className="mt-7 flex flex-col items-stretch gap-3 sm:mt-8 sm:flex-row sm:flex-wrap sm:items-center">
          <Button asChild variant="filled" size="s" className="h-12 px-7 text-[15px]">
            <a href={localePath('/install/')}>
              {t('home.hero.cta_install')}
              <ArrowRight className="rtl:-scale-x-100" />
            </a>
          </Button>
          {/* Disabled until the listing is live, so it is announced as
              unavailable, but not at the variant's 50% opacity: over the scene
              that was unreadable. It carries its own blurred ground instead,
              and the badge says why it cannot be pressed. */}
          <Button
            variant="outlined"
            size="s"
            className="h-12 border-outline-variant bg-surface-container-low/80 px-5 text-on-surface-variant backdrop-blur-sm disabled:opacity-100"
            disabled
            title={t('home.hero.cta_webstore_soon')}
          >
            <Chrome />
            {t('home.hero.cta_webstore')}
            <ExternalLink />
            <Badge variant="mono" size="sm" className="uppercase tracking-[0.08em]">
              {t('landing.hero.soon')}
            </Badge>
          </Button>
        </div>

        <div className="mt-8 flex flex-col gap-3 lg:mt-10">
          {/* The chips below carry their own grounds and the buttons above are
              filled, which leaves this label as the only copy directly over
              the belt. Measured against the rendered scene it came out at
              1.4:1, so it gets a ground too. */}
          <Eyebrow
            tone="muted"
            className="w-fit rounded-sm bg-background/95 px-1.5 py-0.5 text-[10px] backdrop-blur-sm"
          >
            {t('home.works_with')}
          </Eyebrow>
          <ul className="flex max-w-[600px] flex-wrap gap-1.5 sm:gap-2">
            {SOURCES.map((s) => (
              <li key={s}>
                <span className="inline-flex h-9 items-center rounded-pill border border-outline-variant bg-surface-container-low/80 px-3.5 font-mono text-xs text-on-surface backdrop-blur-sm sm:h-10 sm:px-4">
                  {s}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}

export { SOURCES };
