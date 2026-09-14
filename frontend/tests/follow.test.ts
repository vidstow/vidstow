import { describe, expect, test } from 'vitest';

import { followOutputSummary, pendingCount, word } from '../src/lib/follow.js';
import type { FollowRecord } from '../src/lib/types.js';

describe('follow helpers', () => {
  test('pendingCount ignores empty review lists', () => {
    expect(pendingCount({
      follows: [
        { id: 'a', playlistId: 'PL1', sourceUrl: 'https://www.youtube.com/playlist?list=PL1', title: 'One', videoCount: 1, output: { quality: '1080p', folder: '/tmp/a' }, knownVideoIds: [], createdAt: '', updatedAt: '' },
        { id: 'b', playlistId: 'PL2', sourceUrl: 'https://www.youtube.com/playlist?list=PL2', title: 'Two', videoCount: 2, output: { quality: '1080p', folder: '/tmp/b' }, knownVideoIds: [], pending: [{ videoId: 'aaaaaaaaaaa', url: 'https://www.youtube.com/watch?v=aaaaaaaaaaa', title: 'New', available: true, index: 2 }], createdAt: '', updatedAt: '' },
      ],
      checking: false, checkDone: 0, checkTotal: 0,
    })).toBe(1);
  });

  test('followOutputSummary names the saved policy', () => {
    const follow: FollowRecord = {
      id: 'fol', playlistId: 'PL', sourceUrl: 'https://www.youtube.com/playlist?list=PL', title: 'Course',
      videoCount: 1, output: { quality: 'audio', audioBitrate: 192, folder: '/tmp/Course' },
      knownVideoIds: [], createdAt: '', updatedAt: '',
    };
    expect(followOutputSummary(follow)).toBe('MP3 192');
    expect(followOutputSummary({ ...follow, output: { quality: '1080p', folder: '/tmp/Course' } })).toBe('Up to 1080p');
    expect(word(1, 'video', 'videos')).toBe('1 video');
    expect(word(2, 'video', 'videos')).toBe('2 videos');
  });
});
