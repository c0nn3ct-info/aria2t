import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { render, screen, userEvent } from '@/test/render';
import { LOCALES, setLocale } from '@/i18n';
import { LanguageSwitcher } from './language-switcher';

beforeEach(() => {
  window.history.replaceState({}, '', '/install/');
});

afterEach(() => setLocale('en'));

const trigger = () => screen.getByRole('button');

describe('LanguageSwitcher', () => {
  it('starts closed and opens on click', async () => {
    render(<LanguageSwitcher className="ms-2" />);
    expect(trigger()).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('menu')).toBeNull();

    await userEvent.click(trigger());
    expect(trigger()).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('menu')).toBeInTheDocument();
  });

  it('offers every shipped language, paired to the current page', async () => {
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());

    const items = screen.getAllByRole('menuitem');
    expect(items).toHaveLength(LOCALES.length);
    expect(screen.getByRole('menuitem', { name: 'English' })).toHaveAttribute('href', '/install/');
    expect(screen.getByRole('menuitem', { name: 'Русский' })).toHaveAttribute('href', '/ru/install/');
    // Each carries its own hreflang for crawlers.
    for (const item of items) expect(item).toHaveAttribute('hreflang');
  });

  it('pairs from a prefixed path back to the same page', async () => {
    window.history.replaceState({}, '', '/fa/extension/');
    setLocale('fa');
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());

    expect(screen.getByRole('menuitem', { name: 'English' })).toHaveAttribute('href', '/extension/');
    expect(screen.getByRole('menuitem', { name: '中文' })).toHaveAttribute('href', '/zh-CN/extension/');
  });

  it('marks the current language', async () => {
    setLocale('ar');
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());
    expect(screen.getByRole('menuitem', { name: 'العربية' })).toHaveAttribute('aria-current', 'true');
    expect(screen.getByRole('menuitem', { name: 'English' })).not.toHaveAttribute('aria-current');
  });

  it('remembers the choice so the auto-redirect stops fighting it', async () => {
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());
    await userEvent.click(screen.getByRole('menuitem', { name: 'Русский' }));
    expect(window.localStorage.getItem('aria2t-locale')).toBe('ru');
  });

  it('still navigates when storage is blocked', async () => {
    const setItem = window.localStorage.setItem;
    window.localStorage.setItem = () => {
      throw new Error('blocked');
    };
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());
    await userEvent.click(screen.getByRole('menuitem', { name: 'Русский' }));
    window.localStorage.setItem = setItem;
  });

  it('closes on a click outside and on Escape', async () => {
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());
    await userEvent.click(document.body);
    expect(screen.queryByRole('menu')).toBeNull();

    await userEvent.click(trigger());
    await userEvent.keyboard('{Escape}');
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('ignores other keys and clicks inside itself', async () => {
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());
    await userEvent.keyboard('a');
    await userEvent.click(screen.getByRole('menu'));
    expect(screen.getByRole('menu')).toBeInTheDocument();
  });

  it('closes again from its own trigger', async () => {
    render(<LanguageSwitcher />);
    await userEvent.click(trigger());
    await userEvent.click(trigger());
    expect(screen.queryByRole('menu')).toBeNull();
  });
});
