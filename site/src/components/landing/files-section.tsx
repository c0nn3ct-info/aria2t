// "Download only the files you want": the file picker the extension opens
// before a torrent starts moving data, standing on its lit panel.
//
// The rows are real checkboxes. The section's claim is that the choice is made
// before any data moves, and a picture of three ticks cannot make that claim
// the way three ticks you can change does. Everything below the list is
// computed from the selection, so the count, the total and the button can never
// disagree with the boxes above them.
import { Check, Clock, Folder } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button, buttonVariants } from '@/components/ui/button';
import { fmtSize } from '@/lib/queue-scene';
import { FILES, TOTAL_BYTES, pickedBytes, togglePick, usePick } from '@/lib/pick-scene';
import {
  MockStage,
  LandingSection,
  MockCard,
  MockHeader,
  PointList,
  SectionHeading,
  type Point,
} from './shell';
import { t } from '@/i18n';

/**
 * The box a row draws. It is inside the row's button rather than being one
 * itself: the whole row is the target, which is both a bigger one to hit and
 * the only way the name and the size become the checkbox's accessible name.
 */
function Box({ on }: { on: boolean }) {
  // 6px, not the 8px shape token: at 20px that token rounds a checkbox far
  // enough to read as a radio, and these are multi-select. The comp draws 6
  // for the same reason.
  return on ? (
    <span className="grid h-5 w-5 shrink-0 place-items-center rounded-[6px] bg-primary text-primary-foreground">
      <Check className="h-3.5 w-3.5" strokeWidth={2.6} aria-hidden />
    </span>
  ) : (
    <span aria-hidden className="h-5 w-5 shrink-0 rounded-[6px] border-[1.5px] border-outline" />
  );
}

/**
 * `Download 1.4 GiB`, with the size in its own left-to-right island.
 *
 * Substituting it into the sentence leaves the number and its unit as two runs
 * inside Arabic or Persian, and the bidi algorithm reorders them: the button
 * read "تنزيل GiB 1.4".
 */
function DownloadLabel({ bytes }: { bytes: number }) {
  const [before, after] = t('landing.files.download').split('{{size}}');
  return (
    <>
      {before}
      <span dir="ltr">{fmtSize(bytes)}</span>
      {after}
    </>
  );
}

export function FilesSection() {
  const picked = usePick();
  const count = picked.filter(Boolean).length;
  const bytes = pickedBytes(picked);

  const points: readonly Point[] = [
    { icon: Check, tone: 'success', text: t('landing.files.p1') },
    { icon: Check, tone: 'success', text: t('landing.files.p2') },
    { icon: Clock, tone: 'primary', text: t('landing.files.p3') },
  ];

  return (
    <LandingSection id="file-selection">
      <div className="grid items-center gap-10 lg:grid-cols-[1fr_420px] lg:gap-14">
        <div className="min-w-0">
          {/* Wrapped rather than given the attribute: `SectionHeading` is
              shared, and where a band arrives is the page's business. */}
          <div data-enter>
            <SectionHeading
              title={t('landing.files.h2')}
              body={t('landing.files.body')}
              className="max-w-[520px]"
            />
          </div>
          <PointList points={points} className="mt-7" />
        </div>

        <div data-enter className="min-w-0">
          <MockStage>
            <MockCard>
              <MockHeader title={t('landing.files.card')} />
              <div className="flex items-center gap-2.5 border-b border-outline-variant px-4 py-3.5">
                <Folder className="h-4 w-4 shrink-0 text-on-surface-variant" aria-hidden />
                <span className="flex-1 truncate text-[13px] font-semibold">
                  {t('landing.demo.torrent')}
                </span>
                <span className="text-label-small text-on-surface-variant">
                  {t('landing.queue.in2.files').replace('{{count}}', String(FILES.length))}
                </span>
              </div>

              {FILES.map((f, i) => {
                const on = picked[i];
                return (
                  <button
                    key={f.name}
                    type="button"
                    role="checkbox"
                    aria-checked={on}
                    onClick={() => togglePick(i)}
                    className="flex w-full items-center gap-3 border-b border-outline-variant px-4 py-3.5 text-start transition-colors hover:bg-surface-container-low/60 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-primary"
                  >
                    <Box on={on} />
                    <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span
                        dir="ltr"
                        className={cn('truncate text-[13px]', !on && 'text-on-surface-variant')}
                      >
                        {f.name}
                      </span>
                      <span
                        dir="ltr"
                        className={cn(
                          'font-mono text-[10px]',
                          on ? 'text-on-surface-variant' : 'text-on-surface-variant/80',
                        )}
                      >
                        {f.type}
                      </span>
                    </span>
                    <span
                      dir="ltr"
                      className={cn(
                        'font-mono text-[11px]',
                        on ? 'text-on-surface-variant' : 'text-on-surface-variant/80',
                      )}
                    >
                      {fmtSize(f.bytes)}
                    </span>
                  </button>
                );
              })}

              <div className="flex items-center gap-2.5 px-4 py-3.5 text-label-medium text-on-surface-variant">
                <span>
                  {t('landing.files.selected')
                    .replace('{{n}}', String(count))
                    .replace('{{total}}', String(FILES.length))}
                </span>
                <span dir="ltr" className="ms-auto font-mono text-[11px]">
                  {`${fmtSize(bytes)} / ${fmtSize(TOTAL_BYTES)}`}
                </span>
              </div>
              {/* Later is a picture of a control - it has nowhere to go - so it
                  stays unfocusable. Download is the one the selection drives, and
                  an empty selection is the one thing aria2 will not start, so it
                  says so instead of pretending. */}
              <div className="flex gap-2.5 px-4 pb-4">
                <span
                  className={cn(
                    buttonVariants({ variant: 'filled-tonal', size: 's' }),
                    'pointer-events-none flex-1',
                  )}
                >
                  {t('landing.files.later')}
                </span>
                <Button
                  type="button"
                  variant="filled"
                  size="s"
                  disabled={count === 0}
                  className="flex-1 px-4 text-[13px]"
                >
                  <span className="truncate">
                    {count === 0 ? t('landing.files.nothing') : <DownloadLabel bytes={bytes} />}
                  </span>
                </Button>
              </div>
            </MockCard>
          </MockStage>
        </div>
      </div>
    </LandingSection>
  );
}
