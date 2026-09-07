// Everything on the landing page that moves is a pure function of one tick, so
// the frame at t = 0 is the prerender and the frame at t = n is what the visitor
// sees n ticks later. These pin the shapes those functions promise.
import { describe, expect, it } from 'vitest';
import {
  CHART_H,
  CHART_W,
  connections,
  fillPath,
  HISTORY_N,
  history,
  linePath,
  mib,
  mirrorRows,
  mirrorTotal,
  peerRows,
  peers,
  pieceCells,
  piecePct,
  sample,
  wob,
} from './landing-motion';

describe('wob', () => {
  it('is deterministic and bounded', () => {
    expect(wob(3, 1)).toBe(wob(3, 1));
    for (let t = 0; t < 500; t += 0.7) {
      const v = wob(t, 2.2);
      expect(Math.abs(v)).toBeLessThanOrEqual(1.18);
    }
  });

  it('separates two figures by their phase', () => {
    expect(wob(10, 1)).not.toBe(wob(10, 2));
  });
});

describe('sample and history', () => {
  it('keeps the global speeds near their targets', () => {
    for (let t = -300; t < 300; t += 3) {
      const s = sample(t);
      expect(s.d).toBeGreaterThan(20);
      expect(s.d).toBeLessThan(37);
      expect(s.u).toBeGreaterThan(3);
      expect(s.u).toBeLessThan(6.5);
    }
  });

  it('spans a fixed window ending at the tick, oldest first', () => {
    const h = history(10);
    expect(h.d).toHaveLength(HISTORY_N);
    expect(h.u).toHaveLength(HISTORY_N);
    expect(h.d[HISTORY_N - 1]).toBe(sample(10).d);
    expect(h.u[0]).toBe(sample(10 - (HISTORY_N - 1)).u);
    // one tick later the window has slid by one sample
    expect(history(11).d.slice(0, -1)).toEqual(h.d.slice(1));
  });
});

describe('chart paths', () => {
  it('draws a polyline across the viewBox, clamped to the scale', () => {
    const d = linePath([0, 20, 40, 80], 40);
    expect(d).toBe(`M0.0 ${CHART_H.toFixed(1)} L333.3 110.0 L666.7 0.0 L${CHART_W.toFixed(1)} 0.0`);
    // a negative sample sits on the baseline rather than below it
    expect(linePath([-5, 40], 40)).toBe(`M0.0 220.0 L${CHART_W.toFixed(1)} 0.0`);
  });

  it('closes the fill down to the baseline', () => {
    const line = linePath([10, 20], 40);
    expect(fillPath([10, 20], 40)).toBe(`${line} L1000.0 220.0 L0.0 220.0 Z`);
  });
});

describe('the live rows', () => {
  it('formats MiB/s with one decimal', () => {
    expect(mib(4)).toBe('4.0');
    expect(mib(12.345)).toBe('12.3');
  });

  it('gives each mirror a speed and a bar, and totals them', () => {
    const rows = mirrorRows(0);
    expect(rows.map((r) => r.name)).toEqual(['mirror.one', 'mirror.two', 'mirror.three']);
    for (const r of rows) {
      expect(r.speed).toMatch(/^\d+\.\d MiB\/s$/);
      expect(parseFloat(r.width)).toBeGreaterThan(0);
      expect(parseFloat(r.width)).toBeLessThanOrEqual(100);
    }
    const total = rows.reduce((a, r) => a + parseFloat(r.speed), 0);
    expect(mirrorTotal(rows)).toBe(total.toFixed(1));
    expect(mirrorRows(7)).not.toEqual(rows);
  });

  it('marks the one peer being uploaded to, without widening its cell', () => {
    const rows = peerRows(0);
    expect(rows).toHaveLength(3);
    expect(rows.filter((r) => r.up)).toHaveLength(1);
    // The direction lives in the flag, and the row colours it. A glyph in the
    // string made that one cell wider than the others and broke the column.
    for (const r of rows) {
      expect(r.speed).toMatch(/^\d+\.\d MiB\/s$/);
      expect(parseFloat(r.width)).toBeLessThanOrEqual(100);
    }
  });

  it('keeps connections and peers as whole numbers near their targets', () => {
    for (let t = 0; t < 400; t += 5) {
      const c = connections(t);
      const p = peers(t);
      expect(Number.isInteger(c)).toBe(true);
      expect(Number.isInteger(p)).toBe(true);
      expect(c).toBeGreaterThan(35);
      expect(c).toBeLessThan(49);
      expect(p).toBeGreaterThan(26);
      expect(p).toBeLessThan(42);
    }
  });
});

describe('the piece map', () => {
  it('starts at 63%, only ever creeps up, and parks at 100', () => {
    expect(piecePct(0)).toBeCloseTo(63);
    expect(piecePct(100)).toBeCloseTo(66.1);
    // 0.031 a tick covers the last 37 points in about six minutes at 300ms
    expect(piecePct(1193)).toBeLessThan(100);
    expect(piecePct(1194)).toBe(100);

    // The map states bytes on disk, so it can never fall: this is the property
    // the old wrapping version broke, dropping 255 pieces to 13 every 33s.
    let prev = -Infinity;
    for (let t = 0; t < 4000; t++) {
      const p = piecePct(t);
      expect(p).toBeGreaterThanOrEqual(prev);
      expect(p).toBeLessThanOrEqual(100);
      prev = p;
    }
    expect(prev).toBe(100);
  });

  it('draws the pieces we have, a six-cell frontier, then nothing', () => {
    const cells = pieceCells(128, 50);
    expect(cells).toHaveLength(128);
    expect(cells.slice(0, 64).every((c) => c === 'have')).toBe(true);
    expect(cells.slice(64, 70).every((c) => c === 'edge')).toBe(true);
    expect(cells.slice(70).every((c) => c === 'none')).toBe(true);
    // the frontier is clipped at the end of the map, never overflowing it
    expect(pieceCells(16, 100).every((c) => c === 'have')).toBe(true);
    expect(pieceCells(16, 0).filter((c) => c === 'edge')).toHaveLength(6);
  });
});
