// The hero's WebGL backdrop. The React side is thin by design: a canvas in
// the tree (so the prerendered markup and the first client render agree), and
// an effect that pulls the three.js scene in as its own chunk and boots it.
// Everything that paints lives in `hero-scene-three.ts`.
import { useEffect, useRef } from 'react';
import type { HeroSceneHandle } from './hero-scene-three';
import { cn } from '@/lib/utils';

/**
 * Whether the scene should run here at all: no WebGL means the hero's gradient
 * backdrop stands alone.
 *
 * There is no `navigator.webdriver` guard here, unlike the two live
 * mocks. Those tick React state, so a prerender that let them run would capture
 * a different frame than the one the visitor's first render produces. This
 * boots after mount and paints into a canvas, which the prerender's DOM capture
 * cannot see either way, so freezing it would only hide the hero from every
 * screenshot without buying any hydration safety.
 */
export function canRunScene(): boolean {
  return typeof WebGLRenderingContext !== 'undefined';
}

interface Props {
  /**
   * What the illustration shows. It carries the page's metaphor rather than
   * decorating a paragraph, so it is announced as an image instead of hidden.
   */
  'aria-label': string;
  /**
   * Where the pylon tips should stand, in normalised device coordinates. The
   * hero measures its own heading and passes that line's height; the scene
   * solves its aim against it on every resize.
   */
  alignTipsNdc?: () => number | null;
  className?: string;
}

export function HeroScene({ 'aria-label': label, alignTipsNdc, className }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const canvas = useRef<HTMLCanvasElement>(null);
  // Held in a ref so the effect can boot once and still read the latest
  // measurement on every resize.
  const align = useRef(alignTipsNdc);
  align.current = alignTipsNdc;

  useEffect(() => {
    if (!canRunScene()) return;
    let alive = true;
    let handle: HeroSceneHandle | undefined;
    import('./hero-scene-three')
      .then((m) => {
        // Unmounted while the chunk was loading: never boot into a dead node.
        if (!alive) return;
        // Both refs are attached by the time an effect runs.
        // The hero mirrors this canvas on a right-to-left page; tell the
        // scene so the text it bakes into its textures is flipped back.
        const rtl = document.documentElement.dir === 'rtl';
        handle = m.bootHeroScene(host.current!, canvas.current!, {
          mirrorText: rtl,
          alignTipsNdc: align.current,
        });
      })
      // A chunk that fails to load (offline, an ad blocker) costs the
      // animation, not the page: the backdrop's gradients stay.
      .catch(() => undefined);
    return () => {
      alive = false;
      handle?.dispose();
    };
  }, []);

  return (
    <div ref={host} role="img" aria-label={label} className={cn('relative h-full w-full', className)}>
      <canvas ref={canvas} className="block h-full w-full" />
    </div>
  );
}
