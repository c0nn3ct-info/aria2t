// A browser window with the extension's popup hanging off its toolbar button,
// so the extension mock reads as something living in a browser rather than a
// floating card. The page behind it is deliberately abstract: a real-looking
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
    <div className={cn('relative isolate', className)}>
      <div className="overflow-hidden rounded-xl border border-outline-variant bg-surface-container-low shadow-e4">
        <div className="flex items-center gap-2 border-b border-outline-variant bg-surface-container px-3 py-2">
          <div className="flex items-center gap-1.5" aria-hidden>
            <span className="h-2.5 w-2.5 rounded-full bg-[#ff5f57]" />
            <span className="h-2.5 w-2.5 rounded-full bg-[#febc2e]" />
            <span className="h-2.5 w-2.5 rounded-full bg-[#28c840]" />
          </div>
          <RotateCw className="ms-1 h-3.5 w-3.5 text-on-surface-variant" aria-hidden />
          <div className="flex h-6 flex-1 items-center gap-1.5 rounded-pill bg-surface-container-highest px-2.5 text-[11px] text-on-surface-variant">
            <Lock className="h-2.5 w-2.5" aria-hidden />
            <span dir="ltr" className="truncate font-mono">releases.example.org/ubuntu</span>
          </div>
          <div className="flex items-center gap-1.5 ps-1 text-on-surface-variant">
            <Puzzle className="h-3.5 w-3.5" aria-hidden />
            <span
              className="relative grid h-6 w-6 place-items-center rounded-md bg-primary-container text-primary-on-container ring-2 ring-primary/40"
              aria-hidden
            >
              <Aria2tLogo className="h-3.5 w-3.5" />
              <span className="absolute -bottom-0.5 -end-0.5 h-1.5 w-1.5 rounded-full bg-success ring-2 ring-surface-container" />
            </span>
          </div>
        </div>

        {/* the page under the popup: a download link and some paragraph
            furniture, enough to say "you were on a page" and no more */}
        <div className="relative h-[640px] overflow-hidden bg-surface-container-lowest">
          <div
            className="absolute inset-0 opacity-60"
            aria-hidden
            style={{
              backgroundImage:
                'radial-gradient(circle at 18% 0%, color-mix(in srgb, hsl(var(--primary)) 16%, transparent), transparent 55%), radial-gradient(circle at 85% 75%, color-mix(in srgb, hsl(var(--tertiary)) 12%, transparent), transparent 60%)',
            }}
          />
          <div className="absolute start-8 top-10 max-w-[52%] space-y-3" aria-hidden>
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
        </div>
      </div>

      {/* anchored under the toolbar button, with the little pointer Chrome draws */}
      <div className="pointer-events-none absolute end-2 top-[40px] z-10">
        <span
          aria-hidden
          className="absolute -top-1.5 end-[10px] h-3 w-3 rotate-45 border-s border-t border-outline-variant bg-surface-container-lowest"
        />
        <div className="origin-top-right">{children}</div>
      </div>
    </div>
  );
}
