// Picks which interface the hero demonstrates. Nothing it renders controls a
// tabpanel in the ARIA sense (it swaps a decorative mock and the primary CTA),
// so it is a group of toggle buttons with aria-pressed, matching the
// extension's own Segmented control rather than a tablist.
import { Puzzle, TerminalSquare } from 'lucide-react';
import { cn } from '@/lib/utils';
import { t } from '../i18n';

export type Surface = 'terminal' | 'extension';

// Extension first: it is the surface both pages open on, and a group of toggle
// buttons should lead with the one that is pressed.
export const SURFACES: readonly Surface[] = ['extension', 'terminal'];

const ICON = {
  terminal: TerminalSquare,
  extension: Puzzle,
} as const;

interface Props {
  value: Surface;
  onChange: (s: Surface) => void;
  className?: string;
}

export function SurfaceSwitch({ value, onChange, className }: Props) {
  return (
    <div
      role="group"
      aria-label={t('home.hero.surface_aria')}
      className={cn(
        'inline-flex items-center gap-1 rounded-pill border border-outline-variant bg-surface-container-low p-1',
        className,
      )}
    >
      {SURFACES.map((s) => {
        const Icon = ICON[s];
        const active = s === value;
        return (
          <button
            key={s}
            type="button"
            aria-pressed={active}
            onClick={() => onChange(s)}
            className={cn(
              // 44px tall: the comp draws this control at 46, and anything under 44
              // is below every touch-target floor.
              'inline-flex min-h-[44px] items-center gap-1.5 rounded-pill px-4 text-label-medium transition-colors',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
              active
                ? 'bg-secondary-container text-secondary-on-container'
                : 'text-on-surface-variant hover:bg-surface-container-high',
            )}
          >
            <Icon className="h-4 w-4" aria-hidden />
            {t(`home.hero.surface_${s}`)}
          </button>
        );
      })}
    </div>
  );
}
