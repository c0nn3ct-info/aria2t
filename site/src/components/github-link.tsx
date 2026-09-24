import { Github } from 'lucide-react';
import { IconButton } from '@/components/ui/icon-button';
import { GITHUB_URL } from '@/constants';

export function GithubLink() {
  return (
    <IconButton
      asChild
      variant="standard"
      size="s"
      // 44px to tap; the `s` tier is 40.
      className="h-11 w-11"
      aria-label="GitHub"
      title="GitHub"
    >
      <a href={GITHUB_URL} target="_blank" rel="noreferrer noopener">
        <Github />
      </a>
    </IconButton>
  );
}
