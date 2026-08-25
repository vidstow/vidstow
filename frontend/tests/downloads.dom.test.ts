import { render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/vitest';
import { afterEach, describe, expect, test } from 'vitest';

import Downloads from '../src/pages/Downloads.svelte';
import { history } from '../src/lib/stores.js';

afterEach(() => history.set([]));

describe('Downloads format truthfulness', () => {
  test('prefers the completed filename extension and reports fallback delivery', async () => {
    history.set([{
      id: 'history-1', videoId: 'fixture0001', title: 'Fallback file', channel: 'Fixture channel',
      quality: '1080p', container: 'mp4', filename: 'Fallback file.mkv', absolutePath: '/tmp/Fallback file.mkv',
      sizeBytes: 100, completedAt: '2026-01-01T00:00:00Z', durationLabel: '1:00', thumbnail: '',
      deliveryNote: 'Saved as MKV with an SRT subtitle sidecar after embedding fallback; artwork and chapters could not be embedded.',
    }]);

    render(Downloads);
    expect(screen.getByText('1080p · mkv')).toBeInTheDocument();
    expect(screen.queryByText('1080p · mp4')).not.toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole('button', { name: 'Show details for Fallback file' }));
    expect(screen.getByText('Saved as MKV with an SRT subtitle sidecar after embedding fallback; artwork and chapters could not be embedded.')).toBeInTheDocument();
  });
});
