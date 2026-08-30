// Port of the extension's FAB (extension/src/components/ui/fab.tsx), so the
// popup mock's hero action is the same control the real popup draws rather than
// a hand-rolled square that happens to be green. Only the plain icon FAB is
// here — the extended one has no call site on the site.
import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const fabVariants = cva(
  'relative inline-flex shrink-0 items-center justify-center font-medium shadow-e3 transition-all duration-med ease-emph focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background hover:shadow-e4 active:scale-[0.97] disabled:pointer-events-none disabled:opacity-50 [&_svg]:shrink-0 select-none',
  {
    variants: {
      color: {
        primary: 'bg-primary-container text-primary-on-container',
        surface: 'bg-surface-container-high text-primary',
        secondary: 'bg-secondary-container text-secondary-on-container',
        tertiary: 'bg-tertiary-container text-tertiary-on-container',
        success: 'bg-success-container text-success-on-container',
        error: 'bg-error-container text-error-on-container',
      },
      size: {
        small: 'h-10 w-10 rounded-xl [&_svg]:size-5',
        regular: 'h-14 w-14 rounded-2xl [&_svg]:size-6',
        large: 'h-24 w-24 rounded-3xl [&_svg]:size-9',
      },
    },
    defaultVariants: { color: 'primary', size: 'regular' },
  },
);

export interface FabProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'color'>,
    VariantProps<typeof fabVariants> {
  asChild?: boolean;
}

const Fab = React.forwardRef<HTMLButtonElement, FabProps>(
  ({ className, color, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button';
    return <Comp className={cn(fabVariants({ color, size }), className)} ref={ref} {...props} />;
  },
);
Fab.displayName = 'Fab';

export { Fab, fabVariants };
