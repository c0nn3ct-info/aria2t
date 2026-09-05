import { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Row, Stack } from '@/storybook/layout';
import { BrowserMock } from './browser-mock';
import { ListMock } from './list-mock';
import { PopupMock } from './popup-mock';
import { SURFACES, SurfaceSwitch, type Surface } from './surface-switch';
import { TerminalMock } from './terminal-mock';

const meta = {
  title: 'Blocks/SurfaceSwitch',
  component: SurfaceSwitch,
  // Controlled, with no internal state of its own: the home hero holds
  // `surface` and hands it back down. These args make the switch renderable in
  // isolation; `Interactive` is the story where the value is actually wired.
  args: { value: 'extension', onChange: () => {} },
  argTypes: { value: { control: 'inline-radio', options: [...SURFACES] } },
  tags: ['autodocs'],
} satisfies Meta<typeof SurfaceSwitch>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Caption under a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/**
 * Both positions of the two-state switch. The chosen one is filled
 * (`aria-pressed="true"`, secondary container); the other is a quiet label that
 * only takes a background on hover — the contrast is what tells you which
 * interface the hero is showing.
 */
export const Selected: Story = {
  render: () => (
    <Row gap={32} align="start">
      <Stack gap={8} align="start">
        <SurfaceSwitch value="extension" onChange={() => {}} />
        <Label>value=&quot;extension&quot; — how the hero opens</Label>
      </Stack>
      <Stack gap={8} align="start">
        <SurfaceSwitch value="terminal" onChange={() => {}} />
        <Label>value=&quot;terminal&quot;</Label>
      </Stack>
    </Row>
  ),
};

/** The hero's own state, so the buttons move: click one, `onChange` sets it. */
function Wired() {
  const [surface, setSurface] = useState<Surface>('extension');
  return (
    <Stack gap={8} align="start">
      <SurfaceSwitch value={surface} onChange={setSurface} />
      <Label>{`showing the ${surface}`}</Label>
    </Stack>
  );
}

/**
 * The switch owns nothing: it reports a click and re-renders from the `value`
 * it is handed back. Nothing it renders controls a tabpanel — it swaps a
 * decorative mock and nudges which CTA to read first — so it is a group of
 * toggle buttons with `aria-pressed`, matching the extension's own Segmented
 * control rather than a tablist.
 */
export const Interactive: Story = {
  render: () => <Wired />,
};

/** The two showcases the hero swaps between, wired to a real switch. */
function Hero() {
  const [surface, setSurface] = useState<Surface>('extension');
  return (
    <Stack gap={12} style={{ maxWidth: 560 }}>
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <SurfaceSwitch value={surface} onChange={setSurface} />
      </div>
      {surface === 'terminal' ? (
        <TerminalMock>
          <ListMock />
        </TerminalMock>
      ) : (
        <BrowserMock>
          <PopupMock />
        </BrowserMock>
      )}
    </Stack>
  );
}

/**
 * What the home page renders around it: the switch pinned to the top-right of
 * the showcase column, swapping the TUI window for the extension popup. Both
 * install CTAs stay visible on the real page either way — the two surfaces are
 * two things to get, not two views of one, and the switch only picks which one
 * is being demonstrated.
 */
export const InTheHero: Story = {
  render: () => <Hero />,
};

/**
 * The controls panel drives one switch. Because the value is a prop and this
 * story hands back a no-op `onChange`, clicking a button here changes nothing —
 * set `value` in the controls, or use `Interactive` for the wired version.
 */
export const Playground: Story = {};
