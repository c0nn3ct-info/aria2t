// The five pages are compositions: what matters is that each renders inside its
// layout, in every locale, with no raw i18n key and no missing landmark.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, render, screen, userEvent, waitFor } from '@/test/render';
import { LOCALES, setLocale } from '@/i18n';
import en from '@/i18n/en.json';
import { HomePage } from './home';
import { InstallPage } from './install';
import { ExtensionPage } from './extension';
import { PrivacyPage } from './privacy';
import { LicensePage } from './license';

const PAGES = [
  ['home', HomePage],
  ['install', InstallPage],
  ['extension', ExtensionPage],
  ['privacy', PrivacyPage],
  ['license', LicensePage],
] as const;

afterEach(() => setLocale('en'));

describe.each(PAGES)('%s page', (name, Page) => {
  it('renders a heading inside the site layout', () => {
    render(<Page />);
    // Pages may carry their own <header>, so the site chrome is "at least one".
    expect(screen.getAllByRole('banner').length).toBeGreaterThan(0);
    expect(screen.getByRole('main')).toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toBeInTheDocument();
    expect(screen.getAllByRole('heading').length).toBeGreaterThan(0);
  });

  it('leaks no raw i18n key in any locale', () => {
    // t() returns the key itself when a string is missing, so the check is
    // "no dictionary key appears verbatim in the rendered text" — precise
    // enough to ignore real content like "install.sh".
    const keys = Object.keys(en as Record<string, string>);
    for (const locale of LOCALES) {
      setLocale(locale);
      const { container, unmount } = render(<Page />);
      const text = container.textContent ?? '';
      const leaked = keys.filter((k) => text.includes(k));
      expect(leaked, `${name}/${locale}`).toEqual([]);
      unmount();
    }
  });
});

describe.each([
  ['install', InstallPage],
  ['extension', ExtensionPage],
] as const)('%s page code blocks', (name, Page) => {
  it('copies a command and says so, then goes back', async () => {
    const writeText = vi.fn(() => Promise.resolve());
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
    vi.useFakeTimers({ shouldAdvanceTime: true });

    render(<Page />);
    const [copy] = screen.getAllByRole('button', { name: /copy/i });
    const command = copy.closest('div')?.querySelector('code')?.textContent ?? '';
    expect(command, name).toBeTruthy();

    await userEvent.click(copy);
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(command));
    expect(await screen.findByRole('button', { name: /copied/i })).toBeInTheDocument();

    // The confirmation is temporary.
    await act(async () => {
      vi.advanceTimersByTime(1_600);
    });
    expect(screen.getAllByRole('button', { name: /copy/i }).length).toBeGreaterThan(0);
    vi.useRealTimers();
  });

  it('says nothing when the clipboard is blocked', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: vi.fn(() => Promise.reject(new Error('denied'))) },
      configurable: true,
    });
    render(<Page />);
    const [copy] = screen.getAllByRole('button', { name: /copy/i });
    await userEvent.click(copy);
    expect(screen.queryByRole('button', { name: /copied/i })).toBeNull();
  });
});
