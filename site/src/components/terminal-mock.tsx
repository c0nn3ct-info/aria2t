import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface TerminalMockProps {
  title?: string;
  children: ReactNode;
  className?: string;
}

// Fake terminal window chrome framing the live list mock. Colors are the
// app's own Tokyo Night palette, not the site theme — the terminal looks the
// same in light and dark mode, like a real screenshot would.
export function TerminalMock({ title = 'aria2t — ~/Downloads', children, className }: TerminalMockProps) {
  return (
    <div
      dir="ltr"
      className={cn(
        'pointer-events-auto w-full overflow-hidden rounded-lg border border-[#3b4261]/60 bg-[#16161e] shadow-e4',
        className,
      )}
    >
      <div className="relative flex h-9 shrink-0 items-center gap-1.5 border-b border-[#24283b] bg-[#1f2335] px-3">
        <span className="h-2.5 w-2.5 rounded-full bg-[#f7768e]" aria-hidden />
        <span className="h-2.5 w-2.5 rounded-full bg-[#e0af68]" aria-hidden />
        <span className="h-2.5 w-2.5 rounded-full bg-[#9ece6a]" aria-hidden />
        <span className="pointer-events-none absolute inset-x-12 text-center font-mono text-[11px] text-[#565f89]">
          <span className="truncate">{title}</span>
        </span>
      </div>
      <div className="p-3 sm:p-4">{children}</div>
    </div>
  );
}
