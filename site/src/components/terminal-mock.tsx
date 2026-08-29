import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface TerminalMockProps {
  title?: string;
  children: ReactNode;
  className?: string;
}

// Fake terminal window chrome framing the live list mock. Colors are the
// app's own palettes (--tui-* vars in globals.css): Tokyo Night Day in light
// mode, Tokyo Night in dark — the mock follows the site theme the way the
// real app follows its theme setting.
export function TerminalMock({ title = 'Aria2t — ~/Downloads', children, className }: TerminalMockProps) {
  return (
    <div
      dir="ltr"
      className={cn(
        'pointer-events-auto w-full overflow-hidden rounded-lg border border-[var(--tui-frame-border)] bg-[var(--tui-bg)] shadow-e4',
        className,
      )}
    >
      <div className="relative flex h-9 shrink-0 items-center gap-1.5 border-b border-[var(--tui-border-dim)] bg-[var(--tui-surface)] px-3">
        <span className="h-2.5 w-2.5 rounded-full bg-[var(--tui-red)]" aria-hidden />
        <span className="h-2.5 w-2.5 rounded-full bg-[var(--tui-yellow)]" aria-hidden />
        <span className="h-2.5 w-2.5 rounded-full bg-[var(--tui-green)]" aria-hidden />
        <span className="pointer-events-none absolute inset-x-12 text-center font-mono text-[11px] text-[var(--tui-fg-dim)]">
          <span className="truncate">{title}</span>
        </span>
      </div>
      <div className="p-3 sm:p-4">{children}</div>
    </div>
  );
}
