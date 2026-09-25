// "Two interfaces, one daemon" - the band that shows the product itself. Both
// mocks are the ones the site already ships: the extension popup inside its
// browser frame, and the terminal list inside its window chrome. The switch is
// the site's own `SurfaceSwitch`, so this band and the rest of the site cannot
// drift apart by a change to one of them.
import { useState, type CSSProperties } from 'react';
import {
  Keyboard,
  MousePointerClick,
  Package,
  PanelsTopLeft,
  Power,
  Server,
  type LucideIcon,
} from 'lucide-react';
import { BrowserMock } from '@/components/browser-mock';
import { ListMock } from '@/components/list-mock';
import { PopupMock } from '@/components/popup-mock';
import { SurfaceSwitch, type Surface } from '@/components/surface-switch';
import { TerminalMock } from '@/components/terminal-mock';
import { LandingSection, SectionHeading } from './shell';
import { t } from '@/i18n';
import { cn } from '@/lib/utils';

/**
 * The extension: its popup alone on a phone, and inside a browser frame once
 * there is room for both.
 *
 * The popup renders at its real 380px in both, with no transform. The window is
 * 500 wide, so the popup nearly fills it - the proportion the home hero
 * already draws, and the one a visitor sees on their own screen. Handing the
 * frame the whole column instead would keep the popup at 380 and still
 * misreport its size, by making the browser around it enormous.
 *
 * Below `sm` the frame is the wrong container: wider than the viewport, it
 * would put the subject of the section off the right edge and open on the page
 * behind instead. The popup goes alone there, laid out at its own 380 and
 * scaled down to the column as one picture, so a 360px phone sees all of it. A
 * popup reflowed to 280 would be a different popup; one scrolled sideways hid
 * the half with the downloads in it.
 *
 * What shows of the page behind the popup is a 110px strip, so it is the
 * frame's own grey bars. A fuller page drawn there gets sliced by the popup's
 * left edge mid-label, which reads as a rendering bug rather than as a browser.
 */
function Extension() {
  return (
    <>
      <div className="[container-type:inline-size] sm:hidden">
        {/* `--s` is the column over 382 — the surface plus a pixel of framing
            on each edge — as a bare number: `atan2` of two lengths is an
            angle, and its tangent is their ratio. Never above 1, so a wide
            phone shows the popup at its own size. The height follows the
            scale, so nothing is left standing under it. */}
        <div
          data-fit
          className="flex justify-center"
          style={
            {
              '--s': 'min(1, tan(atan2(100cqw, 382px)))',
              height: 'calc(602px * var(--s))',
            } as CSSProperties
          }
        >
          <div className="shrink-0 origin-top [transform:scale(var(--s))]">
            <PopupMock />
          </div>
        </div>
      </div>
      <div className="hidden sm:flex sm:justify-center">
        <BrowserMock className="w-full max-w-[500px]">
          <PopupMock />
        </BrowserMock>
      </div>
    </>
  );
}

/**
 * The terminal client, at the 100 columns it prints.
 *
 * It reads left to right from its first column, so a sideways scroll on a
 * phone still opens on the part that matters. Letting it reflow instead would
 * shrink its type to about 5px, because the list sizes itself off a container
 * query — and the real list stops giving up columns at 72, so a narrower
 * terminal is not one the product draws.
 *
 * On a phone the strip is wide enough for the list's own 12px: `min(12px,
 * 1.64cqw)` reaches it at 732px of list, which is 760 with the window's
 * padding and border. At 620 it scrolled and was still 9.7px, which is the
 * worst of both. A fade on the right edge says the screen goes on, and goes
 * once there is nothing further to scroll to.
 */
function Terminal() {
  const [end, setEnd] = useState(false);
  return (
    <div className="relative -mx-5 sm:mx-0">
      <div
        data-terminal-strip
        dir="ltr"
        onScroll={(e) => {
          const el = e.currentTarget;
          setEnd(el.scrollLeft + el.clientWidth >= el.scrollWidth - 1);
        }}
        className="overflow-x-auto px-5 sm:overflow-visible sm:px-0"
      >
        <div className="min-w-[760px] sm:min-w-0">
          <TerminalMock>
            <ListMock />
          </TerminalMock>
        </div>
      </div>
      <div
        aria-hidden
        data-more
        data-state={end ? 'end' : 'more'}
        className={cn(
          'pointer-events-none absolute inset-y-0 right-0 w-14 bg-gradient-to-l from-background to-transparent',
          'transition-opacity duration-short ease-emph sm:hidden',
          end ? 'opacity-0' : 'opacity-100',
        )}
      />
    </div>
  );
}

/**
 * What each surface is good at, under the switch that selects it. The claims
 * are the ones only that surface can make - the extension's are about being
 * inside the browser, the terminal's about being a single binary you can run
 * over SSH - so the list is a reason to try the other one rather than the same
 * three features twice.
 *
 * A claim and how it works, not one sentence trying to be both: the title is
 * what the visitor gets, the line under it is the mechanism. Written as pairs
 * rather than a heading bolted onto an existing sentence, which is what makes
 * the two halves say different things instead of one repeating the other.
 */
const ICONS: Record<Surface, readonly LucideIcon[]> = {
  extension: [MousePointerClick, PanelsTopLeft, Power],
  terminal: [Keyboard, Server, Package],
};

/** The key prefix each surface's copy is stored under. */
const PREFIX: Record<Surface, string> = { extension: 'ext', terminal: 'term' };

/**
 * The rows under the switch.
 *
 * One muted icon each, at the title's size rather than in a filled badge. The
 * three coloured badges this replaced coded nothing, and the plain version with
 * no icon at all read as one grey block: the icon is what gives the eye a place
 * to start on each row, and the indent under it holds the pair together.
 */
function Rows({ surface }: { surface: Surface }) {
  // Resolved here rather than in a module constant. `t` reads the locale that
  // is current when it is called, and a constant at module scope is evaluated
  // on import - before the entry point has set the page's locale - so every
  // language got the English string.
  const rows = ICONS[surface].map((icon, i) => ({
    icon,
    title: t(`landing.surfaces.${PREFIX[surface]}_t${i + 1}`),
    body: t(`landing.surfaces.${PREFIX[surface]}_p${i + 1}`),
  }));
  return (
    <ul data-enter-stagger="wipe" className="flex w-full flex-col gap-6">
      {rows.map(({ icon: Icon, title, body }) => (
        <li key={title} className="flex gap-3.5">
          <Icon className="mt-0.5 h-4 w-4 shrink-0 text-on-surface-variant" aria-hidden />
          <div className="min-w-0">
            <div className="text-title-dense font-semibold leading-[1.4]">{title}</div>
            <p className="mt-1 text-caption leading-[1.55] text-on-surface-variant">{body}</p>
          </div>
        </li>
      ))}
    </ul>
  );
}

export function SurfacesSection() {
  // Opens on the extension: it is the surface most visitors can use today.
  const [surface, setSurface] = useState<Surface>('extension');
  return (
    <LandingSection id="interfaces">
      {/* What is being switched sits beside the switch rather than under it:
          the heading and the control read as one column, the mock as the thing
          they act on. The right column is the wider of the two because it holds
          a 100-column terminal; the left only has to hold a three-word heading
          and a two-button group. Both start at the top of the row: the heading
          is what the visitor reads first, and centring it against a 675px mock
          pushed it down the band past the point where reading starts. */}
      <div className="grid items-start gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.6fr)] lg:gap-14">
        <div className="flex flex-col items-start gap-6">
          {/* Wrapped rather than given a `data-enter` prop: the attribute is a
              page concern and these two are shared components. */}
          <div data-enter>
            <SectionHeading title={t('landing.surfaces.h2')} body={t('landing.surfaces.body')} />
          </div>
          <div data-enter>
            <SurfaceSwitch value={surface} onChange={setSurface} />
          </div>
          {/* Keyed like the mock, so the switch changes the reasons and the
              picture together. */}
          <div key={surface} className="w-full animate-swap-in">
            <Rows surface={surface} />
          </div>
        </div>

        {/* Keyed on the surface so React remounts the branch and the entrance
            replays. Reduced motion shortens it to nothing through globals.css. */}
        <div key={surface} data-enter className="min-w-0 animate-swap-in">
          {surface === 'extension' ? <Extension /> : <Terminal />}
        </div>
      </div>
    </LandingSection>
  );
}
