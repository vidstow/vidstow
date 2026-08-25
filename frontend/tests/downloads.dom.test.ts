import { render, screen } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import { afterEach, describe, expect, test } from 'vitest';

import Downloads from '../src/pages/Downloads.svelte';
import { history } from '../src/lib/stores.js';

afterEach(() => history.set([]));

describe('Downloads format truthfulness', () => {
  test('prefers the completed filename extension over the planned container', () => {
    history.set([{
      id: 'history-1', videoId: 'fixture0001', title: 'Fallback file', channel: 'Fixture channel',
      quality: '1080p', container: 'mp4', filename: 'Fallback file.mkv', absolutePath: '/tmp/Fallback file.mkv',
      sizeBytes: 100, completedAt: '2026-01-01T00:00:00Z', durationLabel: '1:00', thumbnail: '',
    }]);

    render(Downloads);
    expect(screen.getByText('1080p · mkv')).toBeInTheDocument();
    expect(screen.queryByText('1080p · mp4')).not.toBeInTheDocument();
  });
});
