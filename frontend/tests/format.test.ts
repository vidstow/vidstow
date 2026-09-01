import { describe, expect, test } from 'vitest';

import { formatDiscardConfirm, formatEngineVersion, formatPlanSize, formatViewCount, youtubeUrlFromText } from '../src/lib/format.js';

describe('formatPlanSize', () => {
  test('keeps approximate and exact sizes as plain labels', () => {
    expect(formatPlanSize(65.3 * 1024 * 1024)).toBe('65.3 MB');
    expect(formatPlanSize(478.6 * 1024 * 1024, true)).toBe('~478.6 MB');
  });
});

describe('formatViewCount', () => {
  test('promotes 1000 of a unit to the next unit', () => {
    expect(formatViewCount(999_999)).toBe('1M');
    expect(formatViewCount(999_999_999)).toBe('1B');
  });

  test('keeps ordinary compact values', () => {
    expect(formatViewCount(405_000_000)).toBe('405M');
    expect(formatViewCount(12_000_000)).toBe('12M');
    expect(formatViewCount(1500)).toBe('1.5K');
    expect(formatViewCount(999)).toBe('999');
  });
});

describe('youtubeUrlFromText', () => {
  test('extracts Shorts URLs from pasted text', () => {
    expect(youtubeUrlFromText('check this https://www.youtube.com/shorts/dQw4w9WgXcQ out')).toBe(
      'https://www.youtube.com/shorts/dQw4w9WgXcQ',
    );
  });
});

describe('formatDiscardConfirm', () => {
  test('names the leftover size in the confirm', () => {
    expect(formatDiscardConfirm(843 * 1024 * 1024)).toBe('Delete 843 MB of saved data?');
  });

  test('asks without a size when leftover bytes are unknown', () => {
    expect(formatDiscardConfirm(0)).toBe('Delete the saved data for this download?');
  });
});

describe('formatEngineVersion', () => {
  test('uses the commit hash from a Go pseudo-version, not the timestamp', () => {
    expect(formatEngineVersion('v0.1.1-0.20260807091708-09a8354be2be')).toBe('v0.1.1 (09a8354)');
  });

  test('keeps stable and prerelease tags without inventing a hash', () => {
    expect(formatEngineVersion('v0.2.1')).toBe('v0.2.1');
    expect(formatEngineVersion('v0.2.1-beta.1')).toBe('v0.2.1-beta.1');
  });
});
