// The torrent both file pickers on the page pick from. One selection, held in a
// module store, so ticking a box in the queue band and scrolling to the picker
// cannot show two answers to the same question.
import { describe, expect, it } from 'vitest';
import { act, renderHook } from '@/test/render';
import { FILES, TOTAL_BYTES, pickedBytes, togglePick, usePick } from './pick-scene';

const GiB = 1073741824;
const MiB = 1048576;

describe('the torrent', () => {
  it('holds three files whose sizes add up to the total', () => {
    expect(FILES.map((f) => f.name)).toEqual(['film.mp4', 'subtitles.srt', 'extras.zip']);
    expect(TOTAL_BYTES).toBe(1.4 * GiB + 84 * 1024 + 320 * MiB);
  });

  it('sums only what is ticked', () => {
    expect(pickedBytes([true, true, false])).toBe(1.4 * GiB + 84 * 1024);
    expect(pickedBytes([false, false, false])).toBe(0);
    expect(pickedBytes([true, true, true])).toBe(TOTAL_BYTES);
  });
});

describe('the shared selection', () => {
  it('opens on the two files worth having and lets every box move', () => {
    const { result } = renderHook(() => usePick());
    expect(result.current).toEqual([true, true, false]);

    act(() => togglePick(2));
    expect(result.current).toEqual([true, true, true]);

    // Emptying it is allowed; refusing to start is the button's business.
    act(() => {
      togglePick(0);
      togglePick(1);
      togglePick(2);
    });
    expect(result.current).toEqual([false, false, false]);

    act(() => {
      togglePick(0);
      togglePick(1);
    });
    expect(result.current).toEqual([true, true, false]);
  });

  it('tells every subscriber, and stops telling one that has gone', () => {
    const a = renderHook(() => usePick());
    const b = renderHook(() => usePick());
    act(() => togglePick(2));
    expect(a.result.current).toEqual([true, true, true]);
    expect(b.result.current).toEqual([true, true, true]);

    b.unmount();
    act(() => togglePick(2));
    expect(a.result.current).toEqual([true, true, false]);
    a.unmount();
  });
});
