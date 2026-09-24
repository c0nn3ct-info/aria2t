// One clock for every live figure on the landing page. Each section derives its
// numbers from the tick with a pure function (`src/lib/landing-motion.ts`), so
// they all agree about what "now" looks like and none of them needs state of
// its own.
//
// It starts at 0 and stays there until after mount, which is what makes the
// prerendered frame and the first client frame identical.
import { useEffect, useState, type RefObject } from 'react';
import { motionAllowed } from '@/lib/mock-motion';

/** The design's cadence: fast enough to read as live, slow enough to follow. */
export const TICK_MS = 300;

/**
 * `target` is the figure the tick drives. While it is off screen the clock
 * holds, for the same reason as a background tab: a section three screens
 * down was re-rendering itself three times a second for nobody.
 */
export function useTick(
  intervalMs: number = TICK_MS,
  target?: RefObject<Element | null>,
): number {
  const [tick, setTick] = useState(0);

  useEffect(() => {
    // Prerender capture and reduced motion, the same test every live figure on
    // the site applies.
    if (!motionAllowed()) return;
    let onScreen = true;
    const el = target?.current;
    const io =
      el && 'IntersectionObserver' in window
        ? new IntersectionObserver(([e]) => void (onScreen = e.isIntersecting))
        : undefined;
    io?.observe(el!);
    const id = setInterval(() => {
      // A background tab still runs timers on some engines; advancing the clock
      // there only burns battery redrawing something nobody is looking at.
      if (document.hidden || !onScreen) return;
      setTick((t) => t + 1);
    }, intervalMs);
    return () => {
      clearInterval(id);
      io?.disconnect();
    };
  }, [intervalMs, target]);

  return tick;
}
