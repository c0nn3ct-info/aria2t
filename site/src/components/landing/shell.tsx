// The furniture every landing band is built from: the band itself, its heading
// block, the tick list that sits under a heading, and the glowing panel that
// frames a product mock. Nothing here carries a colour of its own: every value
// is an M3 token from `globals.css`, so the page follows the site theme and the
// accent switch like the rest of the site does.
import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

interface LandingSectionProps {
  id?: string;
  className?: string;
  /** Wraps the children in the page's reading width. Off for edge-to-edge bands. */
  contained?: boolean;
  children: ReactNode;
}

/**
 * The side gutter, and it sits *inside* the reading width rather than outside
 * it. Both spellings centre a 1160 column on a wide window, but they put the
 * content at different places: padding on the section insets the band and then
 * centres 1160 inside what is left, so the text starts at the column's own
 * edge; padding inside the column insets the text within it. The hero and the
 * footer are written the second way, so a band written the first way stands 40
 * px wider than the page it is in the middle of.
 */
export const GUTTER = 'px-5 sm:px-8 lg:px-10';

export function LandingSection({ id, className, contained = true, children }: LandingSectionProps) {
  return (
    // `data-enter-section` is what `useSectionEntrance` observes, and every
    // band wants it: the hero is the one section on the page that does not,
    // because it is already on screen when the page opens.
    <section
      id={id}
      data-enter-section
      className={cn('py-12 sm:py-16 lg:py-24', !contained && GUTTER, id && 'scroll-mt-20', className)}
    >
      {contained ? (
        <div className={cn('mx-auto w-full max-w-[1160px]', GUTTER)}>{children}</div>
      ) : (
        children
      )}
    </section>
  );
}

interface EyebrowProps {
  tone?: 'primary' | 'muted';
  className?: string;
  children: ReactNode;
}

/**
 * A small label over a list or a figure. Not over a heading: a heading carries
 * its own weight, and a kicker above one is the thing this page had until the
 * craft floor took it out.
 */
export function Eyebrow({ tone = 'primary', className, children }: EyebrowProps) {
  return (
    <span
      className={cn(
        'inline-block text-label-small uppercase tracking-[0.16em]',
        tone === 'primary' ? 'text-primary' : 'text-on-surface-variant',
        className,
      )}
    >
      {children}
    </span>
  );
}

interface SectionHeadingProps {
  eyebrow?: string;
  title: string;
  body?: string;
  /** Heading level; a band under the page h1 is an h2, which is the default. */
  level?: 2 | 3;
  className?: string;
}

export function SectionHeading({
  eyebrow,
  title,
  body,
  level = 2,
  className,
}: SectionHeadingProps) {
  const Heading = `h${level}` as const;
  return (
    <div className={cn('max-w-[600px]', className)}>
      {eyebrow && <Eyebrow>{eyebrow}</Eyebrow>}
      {/* Sized off the viewport rather than at breakpoints, and balanced: these
          headings are two or three words in English and can be half again as
          long in Spanish or Persian, so the wrap has to be the browser's call. */}
      <Heading
        className={cn(
          'text-balance text-[clamp(28px,4.4vw,44px)] font-semibold leading-[1.1] tracking-[-0.035em]',
          eyebrow && 'mt-3.5',
        )}
      >
        {title}
      </Heading>
      {body && (
        <p className="mt-4 text-pretty text-body-large leading-[1.7] text-on-surface-variant">
          {body}
        </p>
      )}
    </div>
  );
}

export type PointTone = 'primary' | 'tertiary' | 'success';

const POINT_TONE: Record<PointTone, string> = {
  primary: 'bg-primary-container text-primary-on-container',
  tertiary: 'bg-tertiary-container text-tertiary-on-container',
  success: 'bg-success-container text-success-on-container',
};

export interface Point {
  icon: LucideIcon;
  tone?: PointTone;
  text: string;
}

/** The three supporting facts under a section heading. */
export function PointList({ points, className }: { points: readonly Point[]; className?: string }) {
  return (
    // A list arrives item by item along the line it is read on.
    <ul data-enter-stagger="wipe" className={cn('flex flex-col gap-3', className)}>
      {points.map((p) => {
        const Icon = p.icon;
        // Start-aligned, not centred: a point that wraps to two lines would
        // otherwise float its badge between them. `text-body-large` is 16/24
        // and the badge is 24, so a one-line point looks the same either way.
        return (
          <li key={p.text} className="flex items-start gap-3">
            <span
              className={cn(
                'grid h-6 w-6 shrink-0 place-items-center rounded-full',
                POINT_TONE[p.tone ?? 'success'],
              )}
            >
              <Icon className="h-3.5 w-3.5" aria-hidden />
            </span>
            <span className="text-body-large">{p.text}</span>
          </li>
        );
      })}
    </ul>
  );
}

/**
 * Centres a product mock in its column.
 *
 * It used to paint two radial washes behind the card. They lit the page rather
 * than the card, which is the wrong surface for them: a mock carries its own
 * light, and a glow on the background is decoration with nothing under it.
 */
export function MockStage({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn('flex justify-center px-2 py-6 sm:p-6', className)}>
      <div className="w-full max-w-[364px]">{children}</div>
    </div>
  );
}

/** A popup-sized card standing on a `MockStage`. */
export function MockCard({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        'overflow-hidden rounded-md border border-outline-variant bg-background text-on-surface shadow-e4',
        className,
      )}
    >
      {children}
    </div>
  );
}

/** The header strip a `MockCard` opens with: a title and an optional aside. */
export function MockHeader({ title, aside }: { title: string; aside?: ReactNode }) {
  return (
    <div className="flex items-center gap-2.5 border-b border-outline-variant px-4 py-3.5">
      <span className="flex-1 text-title-small font-semibold">{title}</span>
      {aside}
    </div>
  );
}
