import type { ComponentType, SVGProps } from 'react';
import {
  ArrowDown,
  ArrowRight,
  Cpu,
  Globe,
  TerminalSquare,
} from 'lucide-react';
import { ConnectionVisual } from '@/components/m3/connection-visual';
import { cn } from '@/lib/utils';

export function ArchitectureDiagram() {
  return (
    <div dir="ltr" className="relative flex flex-col gap-3 pt-4 lg:flex-row lg:items-stretch lg:gap-2">
      <Node
        context="Terminal"
        icon={TerminalSquare}
        title="aria2t"
        subtitle="List, details, pickers, scheduler"
      />
      <Connector label="JSON-RPC · push + poll" featured />
      <Node
        context="Your machine"
        icon={Cpu}
        title="aria2c daemon"
        subtitle="Managed: spawned, saved, reaped"
      />
      <Connector label="downloads · seeds" />
      <Node
        context="Internet"
        icon={Globe}
        title="Mirrors & peers"
        subtitle="HTTP(S) · FTP · BitTorrent · DHT · Metalink"
        muted
      />
    </div>
  );
}

interface NodeProps {
  context: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  title: string;
  subtitle: string;
  muted?: boolean;
}

function Node({ context, icon: Icon, title, subtitle, muted }: NodeProps) {
  return (
    <div
      className={cn(
        'relative flex min-w-0 flex-col gap-1.5 rounded-lg border bg-surface p-3 lg:flex-1',
        muted
          ? 'border-outline-variant bg-surface-container-low/60'
          : 'border-outline-variant',
      )}
    >
      <span
        className={cn(
          'text-[10px] uppercase tracking-[0.16em]',
          muted ? 'text-on-surface-variant/70' : 'text-on-surface-variant',
        )}
      >
        {context}
      </span>
      <span
        className={cn(
          'grid h-8 w-8 shrink-0 place-items-center rounded-md',
          muted
            ? 'bg-surface-container-high text-on-surface-variant'
            : 'bg-secondary-container text-secondary-on-container',
        )}
        aria-hidden
      >
        <Icon className="h-4 w-4" />
      </span>
      <div
        className={cn(
          'text-title-small leading-tight',
          muted && 'text-on-surface-variant',
        )}
      >
        {title}
      </div>
      <div className="line-clamp-2 text-label-small text-on-surface-variant">{subtitle}</div>
    </div>
  );
}

interface ConnectorProps {
  label: string;
  featured?: boolean;
}

function Connector({ label, featured }: ConnectorProps) {
  return (
    <div className="relative flex shrink-0 flex-col items-center justify-center self-stretch py-1 lg:py-0">
      <div className="relative z-[1] flex items-center gap-1">
        {featured && (
          <ConnectionVisual state="connected" size={20} className="shrink-0" />
        )}
        <span
          className={cn(
            'whitespace-nowrap rounded-pill border bg-background px-2 py-0.5 text-label-small',
            featured
              ? 'border-success/40 text-on-surface'
              : 'border-outline-variant text-on-surface-variant',
          )}
        >
          {label}
        </span>
        <ArrowRight
          className="hidden h-3.5 w-3.5 text-on-surface-variant lg:inline-block"
          aria-hidden
        />
        <ArrowDown className="h-3.5 w-3.5 text-on-surface-variant lg:hidden" aria-hidden />
      </div>
    </div>
  );
}
