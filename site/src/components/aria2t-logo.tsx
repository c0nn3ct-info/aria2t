import { cn } from '@/lib/utils';

interface Aria2tLogoProps {
  className?: string;
}

// Download-arrow mark; the same path is inlined in public/favicon.svg — keep them in sync.
export function Aria2tLogo({ className }: Aria2tLogoProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      fill="currentColor"
      className={cn('shrink-0', className)}
    >
      <path
        fillRule="evenodd"
        d="M12 2.75c.69 0 1.25.56 1.25 1.25v8.19l3.09-3.09a1.25 1.25 0 1 1 1.77 1.77l-5.22 5.22c-.49.49-1.29.49-1.78 0L5.89 10.87a1.25 1.25 0 1 1 1.77-1.77l3.09 3.09V4c0-.69.56-1.25 1.25-1.25ZM4.5 19c0-.69.56-1.25 1.25-1.25h12.5a1.25 1.25 0 1 1 0 2.5H5.75c-.69 0-1.25-.56-1.25-1.25Z"
      />
    </svg>
  );
}
