import { describe, expect, it } from 'vitest';
import { createRef } from 'react';
import { render, screen } from '@/test/render';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  cardVariants,
} from './card';

describe('Card', () => {
  it('renders the whole composition', () => {
    render(
      <Card variant="outlined" padding="lg">
        <CardHeader>
          <CardTitle>Seedbox</CardTitle>
          <CardDescription>ws://sb:6800</CardDescription>
        </CardHeader>
        <CardContent>body</CardContent>
        <CardFooter>footer</CardFooter>
      </Card>,
    );
    expect(screen.getByText('Seedbox')).toBeInTheDocument();
    expect(screen.getByText('ws://sb:6800')).toBeInTheDocument();
    expect(screen.getByText('body')).toBeInTheDocument();
    expect(screen.getByText('footer')).toBeInTheDocument();
  });

  it('gives every variant and padding its own classes', () => {
    const variants = ['elevated', 'filled', 'outlined', 'tonal', 'accent'] as const;
    expect(new Set(variants.map((variant) => cardVariants({ variant }))).size).toBe(variants.length);
    const paddings = ['none', 'sm', 'md', 'lg'] as const;
    expect(new Set(paddings.map((padding) => cardVariants({ padding }))).size).toBe(paddings.length);
  });

  it('forwards refs and merges classNames on every part', () => {
    const refs = {
      card: createRef<HTMLDivElement>(),
      header: createRef<HTMLDivElement>(),
      title: createRef<HTMLDivElement>(),
      description: createRef<HTMLDivElement>(),
      content: createRef<HTMLDivElement>(),
      footer: createRef<HTMLDivElement>(),
    };
    render(
      <Card ref={refs.card} className="c">
        <CardHeader ref={refs.header} className="h">
          <CardTitle ref={refs.title} className="t">
            t
          </CardTitle>
          <CardDescription ref={refs.description} className="d">
            d
          </CardDescription>
        </CardHeader>
        <CardContent ref={refs.content} className="ct">
          ct
        </CardContent>
        <CardFooter ref={refs.footer} className="f">
          f
        </CardFooter>
      </Card>,
    );
    for (const [name, ref] of Object.entries(refs)) {
      expect(ref.current, name).toBeInstanceOf(HTMLDivElement);
    }
    expect(refs.card.current).toHaveClass('c');
    expect(refs.footer.current).toHaveClass('f');
  });
});
