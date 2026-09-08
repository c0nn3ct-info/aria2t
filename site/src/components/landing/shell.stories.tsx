import type { Meta, StoryObj } from '@storybook/react-vite';
import { ArrowUp, Check, Clock } from 'lucide-react';
import { Stack } from '@/storybook/layout';
import {
  Eyebrow,
  LandingSection,
  MockCard,
  MockHeader,
  MockStage,
  PointList,
  SectionHeading,
} from './shell';

// The furniture every landing band is built from. Five pieces, no colour of
// their own: every value is an M3 token, so the page follows the site theme and
// the accent switch like the rest of the site does.
//
// `LandingSection` is also where a band becomes something that arrives:
// `data-enter-section` is the attribute `useSectionEntrance` observes and the
// stylesheet's `view()` rules key off. The hero is the one section on the page
// that does not carry it, because it is already on screen when the page opens.
//
// There is no single component here, so the stories render the pieces rather
// than one subject.
const meta = {
  title: 'Landing/Shell',
  parameters: { layout: 'fullscreen' },
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

/**
 * The band itself: the page's horizontal rhythm (`px-5 sm:px-8 lg:px-10`), its
 * vertical one (`py-12 sm:py-16 lg:py-24`) and the 1160px reading column. A
 * band given an id also gets `scroll-mt-20`, so a jump lands below the sticky
 * header rather than under it.
 */
export const Band: Story = {
  render: () => (
    <>
      <LandingSection id="contained">
        <div className="rounded-md border border-dashed border-outline-variant p-6 text-body-medium text-on-surface-variant">
          Contained: the children sit in the 1160px column.
        </div>
      </LandingSection>
      <LandingSection contained={false}>
        <div className="border-y border-dashed border-outline-variant p-6 text-body-medium text-on-surface-variant">
          Edge to edge: no column, for a band that lays out its own.
        </div>
      </LandingSection>
    </>
  ),
};

/**
 * The heading block, with and without the two optional parts. The size is off
 * the viewport rather than at breakpoints and the wrap is balanced, because
 * these headings are two or three words in English and can be half again as
 * long in Spanish or Persian.
 *
 * An eyebrow goes over a list or a figure, never over a heading: a heading
 * carries its own weight, and a kicker above one is the thing this page had
 * until the craft floor took it out.
 */
export const Headings: Story = {
  render: () => (
    <LandingSection>
      <Stack gap={40} align="start">
        <SectionHeading title="A heading on its own" />
        <SectionHeading
          title="A heading with a body"
          body="The paragraph under it, at the reading measure the bands share."
        />
        <SectionHeading
          eyebrow="over a figure"
          title="An eyebrow, then a heading"
          body="The eyebrow is the one place a kicker belongs."
          level={3}
        />
        <Stack gap={8} align="start">
          <Eyebrow>primary</Eyebrow>
          <Eyebrow tone="muted">muted</Eyebrow>
        </Stack>
      </Stack>
    </LandingSection>
  ),
};

/**
 * The tick list that sits under a heading. It staggers: each item arrives along
 * the line it is read on, and the badges are start-aligned so a point that
 * wraps to two lines does not leave its badge floating between them.
 */
export const Points: Story = {
  render: () => (
    <LandingSection>
      <PointList
        points={[
          { icon: Clock, tone: 'primary', text: 'A point in the primary tone' },
          { icon: Check, text: 'A point in the default tone, which is success' },
          {
            icon: ArrowUp,
            tone: 'tertiary',
            text: 'A point long enough to wrap onto a second line, which is what the start alignment is for',
          },
        ]}
      />
    </LandingSection>
  ),
};

/**
 * The frame a product mock stands in. `MockStage` centres it in its column and
 * paints nothing: it used to carry two radial washes, which lit the page rather
 * than the card - the wrong surface for them, since a mock carries its own
 * light. `MockCard` is the card, `MockHeader` its opening strip, with or
 * without something beside the title.
 */
export const Mocks: Story = {
  render: () => (
    <LandingSection>
      <Stack gap={24}>
        <MockStage>
          <MockCard>
            <MockHeader title="With an aside" aside={<span className="font-mono text-[11px] text-on-surface-variant">ubuntu…iso</span>} />
            <div className="p-4 text-body-medium text-on-surface-variant">The card's body.</div>
          </MockCard>
        </MockStage>
        <MockStage>
          <MockCard>
            <MockHeader title="Title only" />
            <div className="p-4 text-body-medium text-on-surface-variant">The card's body.</div>
          </MockCard>
        </MockStage>
      </Stack>
    </LandingSection>
  ),
};
