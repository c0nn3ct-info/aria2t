// A browser window with the extension's popup hanging off its toolbar button,
// so the extension mock reads as something living in a browser rather than a
// floating card. The page behind it stays abstract: a real-looking
// page would invite reading, and the popup is the subject.
import type { ReactNode } from 'react';
import { Lock, Puzzle, RotateCw } from 'lucide-react';
import { Aria2tLogo } from '@/components/aria2t-logo';
import { cn } from '@/lib/utils';

interface Props {
  children: ReactNode;
  className?: string;
}

export function BrowserMock({ children, className }: Props) {
  return (
    // Browser chrome reads left to right whatever the page's direction, so the
    // whole frame is pinned and every offset inside it is physical.
    <div dir="ltr" className={cn('relative isolate', className)}>
      <div className="overflow-hidden rounded-xl border border-outline-variant bg-surface-container-low shadow-e4">
        <div className="flex items-center gap-3 border-b border-outline-variant bg-surface-container px-3 py-2">
          <div className="flex shrink-0 items-center gap-1.5" aria-hidden>
            <span className="h-3 w-3 rounded-full bg-[#ff5f57]" />
            <span className="h-3 w-3 rounded-full bg-[#febc2e]" />
            <span className="h-3 w-3 rounded-full bg-[#28c840]" />
          </div>
          <RotateCw className="ml-1 h-3.5 w-3.5 shrink-0 text-on-surface-variant" aria-hidden />
          {/* `min-w-0` is what lets the URL truncate. Without it the address
              bar's automatic minimum is the whole URL laid out on one line, and
              a window narrower than that - the home hero's, on a phone - has its
              toolbar pushed out past its own edge and cropped. */}
          <div className="flex h-7 min-w-0 flex-1 items-center gap-2 rounded-pill bg-surface-container-highest px-3 text-label-small text-on-surface-variant">
            <Lock className="h-3 w-3 shrink-0" aria-hidden />
            <span className="truncate font-mono">releases.example.org/ubuntu</span>
          </div>
          <div className="flex shrink-0 items-center gap-1.5 pl-1 text-on-surface-variant">
            <Puzzle className="h-4 w-4" aria-hidden />
            <span
              className="relative grid h-7 w-7 place-items-center rounded-md bg-primary-container text-primary-on-container ring-2 ring-primary/40"
              aria-hidden
            >
              <Aria2tLogo className="h-4 w-4" />
              <span className="absolute -bottom-0.5 -right-0.5 h-1.5 w-1.5 rounded-full bg-success ring-2 ring-surface-container" />
            </span>
          </div>
        </div>

        {/* The page area is a grid and the popup is the one item in its flow:
            the page behind is positioned out of it. So the window's height comes
            from the popup instead of a number somebody has to keep above it,
            which is what lets the popup show at its real size: no translation
            of a longer string can grow it into a crop, and it never has to be
            scaled down to fit. The floor is for a frame drawn with no popup in
            it, which should still look like a browser. */}
        <div className="relative grid min-h-[620px] overflow-hidden bg-surface-container-lowest">
          <div
            className="absolute inset-0 opacity-60"
            aria-hidden
            style={{
              backgroundImage:
                'radial-gradient(circle at 18% 0%, color-mix(in srgb, hsl(var(--primary)) 16%, transparent), transparent 55%), radial-gradient(circle at 85% 75%, color-mix(in srgb, hsl(var(--tertiary)) 12%, transparent), transparent 60%)',
            }}
          />
          {/* The page behind, at the only detail a 110px strip beside the
              popup can carry. Anything with a word in it gets sliced by the
              popup's left edge instead of read. */}
          <div className="absolute left-8 top-10 max-w-[52%] space-y-3" aria-hidden>
            <div className="h-3 w-36 rounded bg-outline-variant" />
            <div className="space-y-2">
              <div className="h-2 w-full rounded bg-outline-variant/60" />
              <div className="h-2 w-11/12 rounded bg-outline-variant/60" />
              <div className="h-2 w-8/12 rounded bg-outline-variant/60" />
            </div>
            <div className="inline-flex items-center gap-2 rounded-pill border border-outline-variant px-3 py-1.5">
              <span className="h-2 w-2 rounded-full bg-primary" />
              <span className="h-2 w-24 rounded bg-outline-variant/70" />
            </div>
          </div>

          {/* Under the toolbar button, with the pointer Chrome draws at its
              corner. The slot spans the window and the popup is pushed to its
              end, rather than the slot being shrink-wrapped: that gives the
              popup's `max-w-full` a width to measure, so a window narrower than
              380 narrows the popup instead of clipping it. Both `min-w-0`s are
              load-bearing: a percentage max-width counts as none while the
              browser works out intrinsic sizes, so without them the popup's 380
              becomes the slot's automatic minimum and holds the window open at a
              width the window does not have. */}
          <div className="pointer-events-none relative z-10 flex min-w-0 justify-end pb-4 pr-2 pt-2">
            <div className="relative min-w-0">
              <span
                aria-hidden
                className="absolute -top-1.5 right-[10px] h-3 w-3 rotate-45 border-l border-t border-outline-variant bg-background"
              />
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
