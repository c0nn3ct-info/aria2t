import { describe, expect, it, vi } from 'vitest';
import { Download, Settings } from 'lucide-react';
import { render, screen, userEvent } from '@/test/render';
import { Section, SectionLink } from './section';

describe('Section', () => {
  it('renders a header with its icon and children', () => {
    const { container } = render(
      <Section header="Install" icon={Download}>
        <p>body</p>
      </Section>,
    );
    expect(screen.getByRole('heading', { name: 'Install' })).toBeInTheDocument();
    expect(screen.getByText('body')).toBeInTheDocument();
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('shows a count badge only for a numeric count', () => {
    const { unmount } = render(
      <Section header="Install" icon={Download} count={0}>
        <p>body</p>
      </Section>,
    );
    expect(screen.getByText('0')).toBeInTheDocument();
    unmount();

    render(
      <Section header="Install" icon={Download}>
        <p>body</p>
      </Section>,
    );
    expect(screen.queryByText('0')).toBeNull();
  });

  it('renders an action slot when given one', () => {
    const { unmount } = render(
      <Section header="Install" icon={Download} action={<button type="button">act</button>}>
        <p>body</p>
      </Section>,
    );
    expect(screen.getByRole('button', { name: 'act' })).toBeInTheDocument();
    unmount();

    render(
      <Section header="Install" icon={Download}>
        <p>body</p>
      </Section>,
    );
    expect(screen.queryByRole('button')).toBeNull();
  });
});

describe('SectionLink', () => {
  it('reports clicks and shows supporting text only when given', async () => {
    const onClick = vi.fn();
    const { unmount } = render(
      <SectionLink
        title="Settings"
        icon={Settings}
        supporting="managed daemon"
        onClick={onClick}
        className="mt-2"
      />,
    );
    expect(screen.getByText('managed daemon')).toBeInTheDocument();
    expect(screen.getByRole('button')).toHaveClass('mt-2');
    await userEvent.click(screen.getByRole('button'));
    expect(onClick).toHaveBeenCalled();
    unmount();

    render(<SectionLink title="Settings" icon={Settings} onClick={() => undefined} />);
    expect(screen.queryByText('managed daemon')).toBeNull();
  });
});
