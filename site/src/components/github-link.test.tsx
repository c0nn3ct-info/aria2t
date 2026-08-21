import { describe, expect, it } from 'vitest';
import { render, screen } from '@/test/render';
import { GITHUB_URL } from '@/constants';
import { GithubLink } from './github-link';

describe('GithubLink', () => {
  it('links out to the repository in a new tab', () => {
    render(<GithubLink />);
    const link = screen.getByRole('link', { name: 'GitHub' });
    expect(link).toHaveAttribute('href', GITHUB_URL);
    expect(link).toHaveAttribute('target', '_blank');
    // noreferrer as well as noopener — the tab must not see this page.
    expect(link.getAttribute('rel')).toContain('noreferrer');
  });
});
