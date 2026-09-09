// The landing hero. Every wash over the scene is the page's own `--background`
// at an alpha (see `bg`), so the band follows the real light/dark preference
// with the rest of the page instead of carrying colours invented for it.
//
// The scene sits behind the copy on a wide screen and under it on a phone.
// That is the whole mobile story here: at 390px the pylons would stand
// directly where the sentences are, and no amount of dimming makes a lit tower
// behind a paragraph read well. So the copy takes the top of the band and the
// machinery gets the strip beneath it - the crowns land on the row of source
// chips and the belt runs out under it. Which is also why the washes below
// are not the same at the two layouts: stacked, there is only the one that
// keeps the copy readable, and it has to be spent by the time it reaches that
// strip. The three that contain the scene - the column wash, the far-edge
// wash and the corner pocket - belong to the wide layout, where the scene is
// a backdrop behind a column of copy. Stacked it is the subject, and the
// conveyor is meant to run out of the screen's own edges.
import { useCallback, useRef } from 'react';
import { ArrowRight, Chrome, ExternalLink } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { HeroScene } from './hero-scene';
import { Eyebrow } from './shell';
import { localePath, t } from '@/i18n';

/** Every source aria2 accepts, in the order the add overlay lists them. */
const SOURCES = ['HTTP(S)', 'FTP', 'SFTP', 'BitTorrent', 'Metalink', 'magnet'] as const;

/** Whether the band is in its column layout - the `lg:` breakpoint, in one
 * place, since the scene has to aim by the same rule the copy lays out by.
 *
 * Tried at `md` too, with the column narrowed to 380 to make room: measured
 * against the stacked reading at 768 the machines come out half the size,
 * because stacked they have the band's whole width and a strip 380 deep, and
 * beside a column they have 350px of width for both of them. Narrowing the
 * column pays from 1024 up, where there is width to spare; below that the
 * copy belongs above the scene. */
const wide = () => window.matchMedia('(min-width: 1024px)').matches;

/** The page's ground, at an alpha. Every wash below is that colour fading.
 *
 * Used raw for the band's edge and seam fades, which dissolve the scene where
 * it runs off the band and are the same job in either theme. The two washes
 * that exist to hold the scene *down* behind the copy go through `wash` and
 * `pocket` instead. */
const bg = (alpha?: number) =>
  alpha === undefined ? 'hsl(var(--background))' : `hsl(var(--background) / ${alpha})`;

/**
 * The copy's wash, scaled by the stage it is on (`--hero-wash`, globals.css).
 *
 * The alphas below are authored for the dark stage, where the copy is light on
 * dark over lit machines and the scene has to be pushed a long way down for
 * the text to win. Light inverts the problem: the copy is dark on light and
 * the machines are near-white, so the text wins unaided and the identical wash
 * is a white fog over white machinery — every crate greys out and every trim
 * line goes pastel. One variable holds the difference so both stages read off
 * the same authored shape.
 */
const wash = (alpha: number) => `hsl(var(--background) / calc(${alpha} * var(--hero-wash)))`;

/** The corner pocket, which is scaled harder still (`--hero-pocket`): it is
 * there to sit on the glare of the nearest crates, and a white stage has no
 * glare to sit on — only fog to add. */
const pocket = (alpha: number) => `hsl(var(--background) / calc(${alpha} * var(--hero-pocket)))`;

export function LandingHero() {
  const section = useRef<HTMLElement>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const chips = useRef<HTMLUListElement>(null);
  const copy = useRef<HTMLDivElement>(null);

  /**
   * The vertical centre of the heading's first line, in viewport px. A Range
   * over the heading's contents gives one rect per rendered line, so this
   * follows the real wrap at any width and in any language.
   */
  const headingLine = (): number | null => {
    const range = document.createRange();
    range.selectNodeContents(heading.current!);
    const first = range.getClientRects()[0];
    return first ? (first.top + first.bottom) / 2 : null;
  };

  /**
   * Where the pylon crowns should stand, in the normalised device coordinates
   * the scene projects into.
   *
   * On the wide layout the copy is a column on the left and the machines stand
   * level with the heading's first line, one of them behind its end. Stacked,
   * that line runs *through* the copy: the machines would hide behind the
   * sentences and leave the band's lower half empty, which is what the strip
   * below the copy is for. So they aim just under the copy's last row - the
   * source chips - and the strip the band carries below it (see the container
   * `pb`) is what the machines, the volume they feed and the crates crossing
   * between them have to themselves. Standing the crowns in the gap *above*
   * the chips instead buys a bigger composition and was tried: it puts the
   * volume they feed behind a chip at every shape a phone or a small tablet
   * wraps to, which is worse than a smaller picture.
   */
  const alignTipsNdc = useCallback((): number | null => {
    // Both refs are attached, so there is no null branch to guard: the scene
    // is handed this callback from an effect, calls it from there and from its
    // own resize observer, and disconnects that observer before it lets go.
    const box = section.current!.getBoundingClientRect();
    if (box.height === 0) return null;
    const line = wide() ? headingLine() : chips.current!.getBoundingClientRect().bottom + 26;
    if (line === null) return null;
    // Reported as measured, however low that label sits: the scene holds the
    // machines' feet inside the frame itself, so it is the one that decides
    // when the crowns stop following the line.
    return 1 - 2 * ((line - box.top) / box.height);
  }, []);

  /**
   * The far edge of the copy column, in the same coordinates: the line the
   * machines have to stand clear of, so they never end up standing in the
   * sentences however narrow that column gets. Null while the copy is stacked
   * above the scene, which tells the scene it has the band's whole width.
   *
   * Mirrored for a right-to-left page: the canvas is flipped there (see the
   * wrapper below), so the column's near edge in page coordinates is its far
   * edge in the scene's.
   */
  const copyEdgeNdc = useCallback((): number | null => {
    if (!wide()) return null;
    const box = section.current!.getBoundingClientRect();
    if (box.width === 0) return null;
    const rect = copy.current!.getBoundingClientRect();
    const rtl = document.documentElement.dir === 'rtl';
    const edge = 2 * (((rtl ? rect.left : rect.right) - box.left) / box.width) - 1;
    return rtl ? -edge : edge;
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
        <HeroScene
          aria-label={t('landing.hero.scene_alt')}
          alignTipsNdc={alignTipsNdc}
          copyEdgeNdc={copyEdgeNdc}
        />
      </div>

      {/* The wash that keeps the copy readable, turned to match where the copy
          is. On a phone it runs the full width, so the wash runs top to bottom:
          it holds through the heading and the lede and is spent by the time it
          reaches the strip the machines stand in. The buttons and the source
          chips cross the top of that strip on grounds of their own. It stops
          short of opaque on purpose: at full strength the crates nearest the
          camera go out entirely, and half the picture is then black. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 z-10 lg:hidden"
        style={{
          background: `linear-gradient(to bottom, ${wash(0.88)} 0%, ${wash(0.86)} 16%, ${wash(0.78)} 26%, ${wash(0.58)} 36%, ${wash(0.34)} 44%, ${wash(0.14)} 52%, ${wash(0.04)} 58%, ${bg(0)} 64%)`,
        }}
      />
      {/* On the wide layout the copy is a column, and the wash is cut against
          that column rather than against the band: the column stops growing at
          1160 and sits centred, so `50% - 580px` is its left edge and
          `50% + 60px` its right one, whatever the band does around it. It
          holds through the copy and releases across the machines, one of which
          stands behind the end of the heading - and it also *stops* a little
          past the column's left edge rather than running on to the band's,
          because on a wide window the belt's near end is out there and the
          only thing a wash over it can do is black out live scene. What tidies
          that edge is the frame fade below, at the edge, where the scene
          actually ends. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 z-10 hidden rtl:-scale-x-100 lg:block"
        style={{
          background:
            `linear-gradient(100deg, ${bg(0)} calc(50% - 820px), ${wash(0.86)} calc(50% - 560px),`
            + ` ${wash(0.86)} calc(50% - 240px), ${wash(0.8)} calc(50% - 60px),`
            + ` ${wash(0.62)} calc(50% + 60px), ${wash(0.3)} calc(50% + 180px),`
            + ` ${wash(0.07)} calc(50% + 320px), ${bg(0)} calc(50% + 440px))`,
        }}
      />
      {/* A pocket of shadow in the band's bottom-left corner, where the
          conveyor arrives at full brightness, being the nearest thing to the
          camera, and where the column's own last rows sit. Sized to the copy
          again (`50% + 60px` is the column's far edge) and, like the wash
          above it, part of the wide layout only - stacked, the copy is nowhere
          near that corner. */}
      <div
        aria-hidden
        className="pointer-events-none absolute bottom-0 start-0 z-10 hidden h-[62%] w-[calc(50%+60px)] rtl:-scale-x-100 lg:block"
        style={{
          background: `radial-gradient(100% 100% at 0% 100%, ${pocket(0.92)} 0%, ${pocket(0.55)} 45%, ${bg(0)} 78%)`,
        }}
      />
      {/* The frame fades: the scene runs off all three of the band's free
          edges and every one of them has to dissolve rather than cut, the
          bottom one most of all - the next section starts there, and a
          conveyor sliced by that seam reads as a mistake rather than as a
          picture. They sit at the edges, which is where the scene really ends,
          and they are the only fades a stacked layout gets besides the copy's
          own wash.

          The near side is softer, and that is the point: at half strength it
          reads as the line carrying on past the screen instead of stopping at
          it. The far side is not an edge fade at all but a tail fade - past
          the machines there is only empty belt, and it dissolves from where
          they stop (`--hero-tail`, which the scene measures and hands over)
          out to the window's edge. The bottom reaches the page's own colour at
          both layouts, since a seam is the one edge that has to disappear. */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-y-0 start-0 z-10 w-[14%] lg:w-[9%]"
        style={{ background: `linear-gradient(to right, ${bg(0.58)} 0%, ${bg(0.2)} 45%, ${bg(0)} 100%)` }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-y-0 end-0 z-10 w-[var(--hero-tail,14%)] min-w-[6%] rtl:-scale-x-100"
        style={{ background: `linear-gradient(to left, ${bg(0.92)} 0%, ${bg(0.5)} 40%, ${bg(0)} 100%)` }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 bottom-0 z-10 h-[20%] lg:h-[210px]"
        style={{
          background: `linear-gradient(to bottom, ${bg(0)} 0%, ${bg(0.05)} 22%, ${bg(0.2)} 44%, ${bg(0.5)} 64%, ${bg(0.8)} 82%, ${bg()} 100%)`,
        }}
      />
      {/* The band's own height, and below `lg` that includes a strip for the
          machinery: the scene is the subject there, not a backdrop, and it can
          only be drawn at a size worth looking at if the band carries room for
          it under the copy. Capped in vh as well as px so a short window does
          not end up with a hero two screens tall. */}
      <div className="relative z-20 mx-auto flex min-h-[620px] w-full max-w-[1160px] flex-col px-5 pb-[min(46vh,340px)] pt-14 sm:min-h-[720px] sm:px-8 sm:pb-[min(42vh,380px)] sm:pt-20 lg:min-h-[720px] lg:justify-center lg:px-10 lg:py-24">
        {/* The column is what the machines beside it have to work around, so
            from `lg` it is only as wide as the reading takes: 460 on a small
            laptop, the drawn 600 from 1280 up. That is the difference between
            a composition on a third of a 1024 band and one on half of it -
            the heading takes a third line there and nothing else moves. */}
        <div ref={copy} className="max-w-[600px] lg:max-w-[460px] xl:max-w-[600px]">
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
          <ul ref={chips} className="flex max-w-[600px] flex-wrap gap-1.5 sm:gap-2">
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
