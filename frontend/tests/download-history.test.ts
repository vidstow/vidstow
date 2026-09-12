import { describe, expect, test } from 'vitest';

import {
  buildHistorySections,
  collectionIdentity,
  episodeLabel,
  historyDayKey,
  historySubtitle,
  episodeSubtitle,
} from '../src/lib/download-history.js';
import type { HistoryEntry } from '../src/lib/types.js';

function entry(overrides: Partial<HistoryEntry> & { id: string; title: string }): HistoryEntry {
  return {
    videoId: 'vid',
    channel: 'Channel',
    quality: '1080p',
    filename: 'file.mp4',
    absolutePath: '/tmp/file.mp4',
    sizeBytes: 22 * 1024 * 1024,
    completedAt: '2026-08-31T12:00:00',
    durationLabel: '12:00',
    thumbnail: '',
    ...overrides,
  };
}

const noon = new Date('2026-08-31T12:00:00');

describe('historyDayKey', () => {
  test('splits local calendar days into Today, Yesterday, and Earlier', () => {
    expect(historyDayKey('2026-08-31T23:50:00', noon)).toBe('today');
    expect(historyDayKey('2026-08-30T08:00:00', noon)).toBe('yesterday');
    expect(historyDayKey('2026-08-29T18:00:00', noon)).toBe('earlier');
    expect(historyDayKey('', noon)).toBe('earlier');
  });
});

describe('buildHistorySections', () => {
  test('date-groups individual rows when history has no collection identity', () => {
    const sections = buildHistorySections([
      entry({ id: 'today', title: 'Go 2026', completedAt: '2026-08-31T10:00:00' }),
      entry({ id: 'yesterday', title: 'Tracing', completedAt: '2026-08-30T10:00:00', quality: '720p' }),
      entry({ id: 'old', title: 'Chaos', completedAt: '2026-07-01T10:00:00' }),
    ], '', noon);

    expect(sections.map((section) => section.label)).toEqual(['Today', 'Yesterday', 'Earlier']);
    expect(sections.every((section) => section.rows.every((row) => row.kind === 'item'))).toBe(true);
    expect(collectionIdentity(entry({ id: 'x', title: 'x' }))).toBe('');
  });

  test('groups rows that share a real playlist or collection id', () => {
    const sections = buildHistorySections([
      entry({
        id: 'ep1', title: 'Introduction', playlistId: 'PLgo', playlistTitle: 'Concurrency in Go',
        collectionIndex: 1, completedAt: '2026-08-30T10:00:00',
      }),
      entry({
        id: 'ep2', title: 'WaitGroups', playlistId: 'PLgo', playlistTitle: 'Concurrency in Go',
        collectionIndex: 2, completedAt: '2026-08-30T11:00:00',
      }),
      entry({ id: 'solo', title: 'Rust vs Go', completedAt: '2026-08-30T12:00:00' }),
    ], '', noon);

    expect(sections).toHaveLength(1);
    expect(sections[0].rows).toHaveLength(2);
    const group = sections[0].rows[0];
    expect(group.kind).toBe('collection');
    if (group.kind !== 'collection') return;
    expect(group.title).toBe('Concurrency in Go');
    expect(group.entries.map((item) => item.id)).toEqual(['ep1', 'ep2']);
    expect(group.format).toBe('1080p');
    expect(episodeLabel(1)).toBe('EP 01');
    expect(sections[0].rows[1]).toMatchObject({ kind: 'item', entry: { id: 'solo' } });
  });

  test('does not invent groups from matching titles or channels', () => {
    const sections = buildHistorySections([
      entry({ id: 'a', title: 'Concurrency 1', channel: 'Steve Hook', completedAt: '2026-08-31T10:00:00' }),
      entry({ id: 'b', title: 'Concurrency 2', channel: 'Steve Hook', completedAt: '2026-08-31T11:00:00' }),
    ], '', noon);
    expect(sections[0].rows.map((row) => row.kind)).toEqual(['item', 'item']);
  });

  test('search matches titles and keeps a playlist group when the collection title hits', () => {
    const entries = [
      entry({
        id: 'ep1', title: 'Introduction', playlistId: 'PLgo', playlistTitle: 'Concurrency in Go',
        collectionIndex: 1, completedAt: '2026-08-31T10:00:00',
      }),
      entry({
        id: 'ep2', title: 'WaitGroups', playlistId: 'PLgo', playlistTitle: 'Concurrency in Go',
        collectionIndex: 2, completedAt: '2026-08-31T11:00:00',
      }),
      entry({ id: 'other', title: 'Clock Synchronization', completedAt: '2026-08-31T12:00:00' }),
    ];
    const byTitle = buildHistorySections(entries, 'clock', noon);
    expect(byTitle).toHaveLength(1);
    expect(byTitle[0].rows).toHaveLength(1);
    expect(byTitle[0].rows[0].kind).toBe('item');

    const byPlaylist = buildHistorySections(entries, 'concurrency', noon);
    expect(byPlaylist[0].rows).toHaveLength(1);
    expect(byPlaylist[0].rows[0].kind).toBe('collection');
  });
});

describe('historySubtitle', () => {
  test('joins format, size, channel, and relative time', () => {
    expect(historySubtitle(entry({
      id: 'a', title: 'Go', quality: '2160p', sizeBytes: 1_690_000_000, channel: 'Gopher Talks',
      completedAt: '2026-08-31T12:00:00',
    }), Date.parse('2026-08-31T14:00:00'))).toBe('2160p · 1.6 GB · Gopher Talks · 2h ago');
  });

  test('does not append a File missing chip', () => {
    expect(historySubtitle(entry({
      id: 'gone', title: 'Finished video', fileMissing: true, channel: 'Creator',
      completedAt: '2026-08-31T12:00:00',
    }), Date.parse('2026-08-31T12:00:30'))).toBe('1080p · 22.0 MB · Creator · just now');
  });
});

describe('episodeSubtitle', () => {
  test('joins duration, size, and relative time', () => {
    expect(episodeSubtitle(entry({
      id: 'ep', title: 'Introduction', durationLabel: '12:04', sizeBytes: 22 * 1024 * 1024, channel: 'Steve Hook',
      completedAt: '2026-08-31T12:00:00',
    }), Date.parse('2026-08-31T14:00:00'))).toBe('12:04 · 22.0 MB · 2h ago');
  });
});
