import type { Meta, StoryObj } from '@storybook/react-vite';
import { HeroScene } from './hero-scene';

// The React side of the hero's backdrop, on its own.
//
// It is deliberately thin: a canvas in the tree, so the prerendered markup and
// the first client render agree, and an effect that pulls three.js in as its
// own chunk and boots it. Everything that paints lives in
// `hero-scene-three.ts`, which never loads where the browser has no WebGL.
//
// Given no `alignTipsNdc` it keeps its own aim, which is what this story shows:
// the framing the scene solves for an aspect ratio, without a heading to
// answer to.
const meta = {
  title: 'Landing/HeroScene',
  component: HeroScene,
  args: { 'aria-label': 'A production line packing downloaded files into cubes.' },
  parameters: { layout: 'fullscreen' },
  globals: { theme: 'dark', accent: 'blue' },
  tags: ['autodocs'],
} satisfies Meta<typeof HeroScene>;

export default meta;

type Story = StoryObj<typeof meta>;

/**
 * The scene filling a wide frame. Two pylons feed a blueprint, the finished
 * cube drops onto the belt, and the belt indexes one slot every five seconds -
 * a packing line rather than a conveyor that glides, which is the metaphor.
 */
export const Wide: Story = {
  render: (args) => (
    <div className="h-[520px] w-full bg-background">
      <HeroScene {...args} />
    </div>
  ),
};

/**
 * The same scene in a portrait frame. The camera interpolates between two
 * solved anchors by aspect ratio, so the crowns and the halo stay inside the
 * frame instead of the whole thing being scaled down.
 */
export const Portrait: Story = {
  render: (args) => (
    <div className="h-[760px] w-[390px] bg-background">
      <HeroScene {...args} />
    </div>
  ),
};

/**
 * Aimed at a line: `alignTipsNdc` hands the scene a height in normalised device
 * coordinates and it bisects its camera aim until the higher pylon tip lands
 * there. +0.5 is a quarter of the way down the frame.
 */
export const AimedHigh: Story = {
  render: (args) => (
    <div className="h-[520px] w-full bg-background">
      <HeroScene {...args} alignTipsNdc={() => 0.5} />
    </div>
  ),
};
