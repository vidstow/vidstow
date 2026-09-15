import { describe, expect, test } from 'vitest';

import { MOTION_CLOSE, MOTION_HOVER, MOTION_OPEN, MOTION_PROGRESS, motionMs, prefersReducedMotion } from '../src/lib/motion.js';

describe('motion tokens', () => {
  test('keeps the named durations', () => {
    expect(MOTION_HOVER).toBe(120);
    expect(MOTION_CLOSE).toBe(150);
    expect(MOTION_OPEN).toBe(250);
    expect(MOTION_PROGRESS).toBe(180);
  });

  test('is instant in Vitest so outros do not stall the DOM tests', () => {
    expect(prefersReducedMotion()).toBe(true);
    expect(motionMs(MOTION_OPEN)).toBe(0);
  });
});
