import type { Meta, StoryObj } from '@storybook/react-vite';
import { Row, Stack } from '@/storybook/layout';
import { GITHUB_URL } from '@/constants';
import { GithubLink } from './github-link';
import { LanguageSwitcher } from './language-switcher';

const meta = {
  title: 'Blocks/GithubLink',
  component: GithubLink,
  tags: ['autodocs'],
} satisfies Meta<typeof GithubLink>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Caption under a sample. */
function Label({ children }: { children: string }) {
  return <div className="text-label-medium text-on-surface-variant">{children}</div>;
}

/**
 * The whole component: a 40px `standard` icon button that is really an anchor
 * (`asChild`), pointing at the public repository in a new tab with
 * `rel="noreferrer noopener"` so that tab never gets a handle on this page.
 * Icon-only, so the name lives in `aria-label` — and in `title`, which is what
 * a mouse user gets on hover. Both stay "GitHub" in every locale: it is the
 * name of the site being linked, not a word to translate.
 */
export const Default: Story = {
  render: () => (
    <Stack gap={8} align="start">
      <GithubLink />
      <Label>{GITHUB_URL}</Label>
    </Stack>
  ),
};

/**
 * The one composition the site renders: the end of the header bar, where the
 * link sits beside the language switcher. The `standard` variant paints no
 * background of its own — it is transparent over the bar's
 * `bg-surface-container-low`, and only its hover state layer
 * (`bg-on-surface/[0.08]`) draws a shape — so this is the pairing worth
 * checking. Both actions are the same `size="s"` icon button, which is what
 * keeps their 40px hit targets level across the 4px gap.
 */
export const InTheHeaderBar: Story = {
  render: () => (
    <div
      className="bg-surface-container-low"
      style={{
        borderBottom: '1px solid hsl(var(--outline-variant))',
        padding: '0 16px',
        height: 64,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-end',
      }}
    >
      <Row gap={4}>
        <GithubLink />
        <LanguageSwitcher />
      </Row>
    </div>
  ),
};
