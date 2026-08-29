// The hero demonstrates one interface at a time, and its primary button has to
// point at the one on screen — a visitor looking at the extension should not be
// sent to the terminal install guide.
import { describe, expect, it } from 'vitest';
import { render, screen, userEvent, within } from '@/test/render';
import { HomePage } from './home';

const hero = () => screen.getAllByRole('group', { name: /interface|интерфейс/i })[0];

describe('HomePage hero', () => {
  it('opens on the extension, showing its popup', () => {
    const { container } = render(<HomePage />);
    expect(within(hero()).getByRole('button', { name: /extension|افزونه|الإضافة/i })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    // the popup mock, not the terminal one
    expect(container.textContent).toContain('Transfer status');
    expect(container.textContent).not.toContain('connected');
  });

  it('offers Install as a link and the store as a button that cannot be pressed yet', () => {
    render(<HomePage />);
    const install = screen
      .getAllByRole('link')
      .find((a) => a.getAttribute('href')?.includes('/install/'));
    expect(install, 'no Install link').toBeDefined();

    // The listing does not exist yet, so the store CTA is present but disabled
    // rather than pointing somewhere that 404s.
    const store = screen.getByRole('button', { name: /chrome web store|应用商店|متجر|Store/i });
    expect(store).toBeDisabled();
    expect(store.tagName).toBe('BUTTON');
    expect(store.getAttribute('href')).toBeNull();
  });

  it('swaps to the terminal mock and repoints the primary', async () => {
    const { container } = render(<HomePage />);
    await userEvent.click(within(hero()).getByRole('button', { name: /terminal|терминал/i }));

    // the TUI list, with the app header the extension popup does not have
    expect(container.textContent).toContain('connected');
    expect(container.textContent).toContain('Aria2t');
    expect(container.textContent).not.toContain('Transfer status');

    const install = screen
      .getAllByRole('link')
      .find((a) => a.getAttribute('href')?.includes('/install/'));
    expect(install).toBeDefined();
  });
});
