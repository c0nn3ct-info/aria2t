// "Leave bandwidth for everything else": the per-download presets, the global
// cap, and the scheduler that moves that cap by time of day.
//
// All three are real controls. The section is about choosing a number, and a
// picture of a chosen number does not let the visitor choose one - the old
// slider was, in its own comment, "a picture of the slider at its ceiling, not
// a slider". Nothing here animates on its own: a limit that drifted by itself
// would say the opposite of what the section claims.
import { useId, useState } from 'react';
import { ArrowUp, Check, Clock } from 'lucide-react';
import { cn } from '@/lib/utils';
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

/** The presets the throttle overlay offers, in MiB/s, plus "no limit". */
const PRESETS = ['1', '5', '10'] as const;
type Preset = (typeof PRESETS)[number] | 'none';

/**
 * The global slider's range, in MiB/s, with the top step standing for no limit
 * at all - the empty string aria2 takes for "unlimited". Higher means more
 * bandwidth allowed, so infinity belongs at the far end rather than off the
 * scale.
 */
const MAX_STEP = 21;

/**
 * The scheduler strip: one bar per two hours, with the height each two hours
 * would reach unthrottled and whether it falls inside the active window. The
 * window's bars are drawn at the capped height only while the schedule is on,
 * which is what the switch beside them changes.
 */
const HOURS: ReadonlyArray<[height: number, inWindow: boolean]> = [
  [80, false], [88, false], [74, false], [92, false],
  [72, true], [84, true], [90, true], [78, true], [66, true],
  [56, false], [68, false], [84, false],
];

/** Where the window's bars sit while the schedule holds them down. */
const CAPPED = 25;

export function LimitsSection() {
  const [preset, setPreset] = useState<Preset>('5');
  const [step, setStep] = useState(20);
  const [scheduled, setScheduled] = useState(true);

  const globalId = useId();
  const scheduleId = useId();
  const presetId = useId();

  const unlimited = t('landing.limits.unlimited');
  const perDownload = t('landing.limits.per_download');
  const globalText = step === MAX_STEP ? unlimited : `${step} MiB/s`;
  // The filled part of the track, and the thumb centred on its end.
  const fill = `${((step - 1) / (MAX_STEP - 1)) * 100}%`;

  const points: readonly Point[] = [
    { icon: Clock, tone: 'primary', text: t('landing.limits.p1') },
    { icon: Check, tone: 'success', text: t('landing.limits.p2') },
    { icon: ArrowUp, tone: 'tertiary', text: t('landing.limits.p3') },
  ];

  return (
    <LandingSection id="limits">
      <div className="grid items-center gap-10 lg:grid-cols-[1fr_420px] lg:gap-14">
        <div className="min-w-0">
          {/* Wrapped rather than given the attribute: `SectionHeading` is
              shared, and where a band arrives is the page's business. */}
          <div data-enter>
            <SectionHeading
              title={t('landing.limits.h2')}
              body={t('landing.limits.body')}
              className="max-w-[520px]"
            />
          </div>
          <PointList points={points} className="mt-7" />
        </div>

        <div data-enter className="min-w-0">
          <MockStage>
            <MockCard>
              <MockHeader
                title={t('landing.limits.card')}
                aside={
                  <span dir="ltr" className="font-mono text-[11px] text-on-surface-variant">
                    ubuntu…iso
                  </span>
                }
              />

              {/* One choice out of four, so the buttons carry `aria-pressed` and
                  the caption under them names the group: on its own a button
                  called "5" says nothing. */}
              <div role="group" aria-labelledby={presetId} className="flex flex-wrap gap-2 p-4">
                {PRESETS.map((p) => {
                  const active = p === preset;
                  return (
                    <button
                      key={p}
                      type="button"
                      aria-pressed={active}
                      aria-label={`${p} ${perDownload}`}
                      onClick={() => setPreset(p)}
                      dir="ltr"
                      className={cn(
                        'inline-flex h-[34px] items-center gap-1.5 rounded-pill px-4 font-mono text-xs transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
                        active
                          ? 'bg-primary text-primary-foreground'
                          : 'border border-outline-variant text-on-surface-variant hover:border-outline hover:text-on-surface',
                      )}
                    >
                      {active && <Check className="h-3.5 w-3.5" strokeWidth={2.4} aria-hidden />}
                      {p}
                    </button>
                  );
                })}
                <button
                  type="button"
                  aria-pressed={preset === 'none'}
                  onClick={() => setPreset('none')}
                  className={cn(
                    'inline-flex h-[34px] items-center gap-1.5 rounded-pill px-4 font-mono text-xs transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
                    preset === 'none'
                      ? 'bg-primary text-primary-foreground'
                      : 'border border-outline-variant text-on-surface-variant hover:border-outline hover:text-on-surface',
                  )}
                >
                  {preset === 'none' && (
                    <Check className="h-3.5 w-3.5" strokeWidth={2.4} aria-hidden />
                  )}
                  {unlimited}
                </button>
                <span id={presetId} className="w-full text-label-small text-on-surface-variant">
                  {perDownload}
                </span>
              </div>

              <div className="border-t border-outline-variant px-4 py-3.5">
                <div className="flex items-center gap-2.5">
                  <span id={globalId} className="flex-1 text-[13px] font-semibold">
                    {t('landing.limits.global')}
                  </span>
                  <span dir="ltr" className="font-mono text-xs text-primary tabular-nums">
                    {globalText}
                  </span>
                </div>
                {/* The range input carries the behaviour and the keyboard, and the
                    three spans over it carry the look. Painting a native track
                    instead would need vendor pseudo-elements and a mirrored
                    gradient for the two right-to-left locales; logical properties
                    on plain spans get both for free. They are click-through, so
                    the input underneath still takes the drag. */}
                <div className="relative mt-2.5 flex h-[26px] items-center">
                  <input
                    type="range"
                    min={1}
                    max={MAX_STEP}
                    step={1}
                    value={step}
                    onChange={(e) => setStep(Number(e.target.value))}
                    aria-labelledby={globalId}
                    aria-valuetext={globalText}
                    className="peer absolute inset-0 h-full w-full cursor-pointer appearance-none bg-transparent opacity-0"
                  />
                  <span
                    aria-hidden
                    className="pointer-events-none absolute inset-x-0 h-1.5 rounded-sm bg-surface-container-high"
                  />
                  <span
                    aria-hidden
                    className="pointer-events-none absolute h-1.5 rounded-sm bg-primary"
                    style={{ insetInlineStart: 0, width: fill }}
                  />
                  <span
                    aria-hidden
                    className="pointer-events-none absolute h-[26px] w-2 rounded-xs bg-primary ring-4 ring-background peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-primary"
                    style={{ insetInlineStart: `calc(${fill} - 4px)` }}
                  />
                </div>
              </div>

              <div className="border-t border-outline-variant px-4 py-3.5">
                <div className="flex items-center gap-2.5">
                  <span id={scheduleId} className="flex-1 text-[13px] font-semibold">
                    {t('landing.limits.schedule')}
                  </span>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={scheduled}
                    aria-labelledby={scheduleId}
                    onClick={() => setScheduled((s) => !s)}
                    className={cn(
                      'inline-flex h-[26px] w-11 shrink-0 items-center rounded-pill p-[3px] transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
                      scheduled ? 'bg-primary' : 'bg-surface-container-highest',
                    )}
                  >
                    <span
                      aria-hidden
                      className={cn(
                        'block h-5 w-5 rounded-full transition-transform',
                        scheduled
                          ? 'translate-x-[18px] bg-primary-foreground rtl:-translate-x-[18px]'
                          : 'translate-x-0 bg-outline',
                      )}
                    />
                  </button>
                </div>
                <div aria-hidden className="mt-3 flex h-11 items-end gap-0.5">
                  {HOURS.map(([height, inWindow], i) => {
                    const held = inWindow && scheduled;
                    return (
                      <i
                        key={i}
                        className={cn(
                          'flex-1 rounded-t-[2px] transition-[height,background-color] duration-200',
                          held ? 'bg-primary' : 'bg-surface-container-high',
                        )}
                        style={{ height: `${held ? CAPPED : height}%` }}
                      />
                    );
                  })}
                </div>
                <div className="mt-2 flex justify-between font-mono text-[10px] text-on-surface-variant">
                  <span dir="ltr">00:00</span>
                  <span
                    dir="ltr"
                    className={cn(scheduled ? 'text-primary' : 'text-on-surface-variant/70')}
                  >
                    {t('landing.limits.window')}
                  </span>
                  <span dir="ltr">24:00</span>
                </div>
              </div>
            </MockCard>
          </MockStage>
        </div>
      </div>
    </LandingSection>
  );
}
