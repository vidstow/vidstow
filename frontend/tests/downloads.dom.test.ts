import { render, screen } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';

import Downloads from '../src/pages/Downloads.svelte';
import { history, modal } from '../src/lib/stores.js';
import type { HistoryEntry } from '../src/lib/types.js';

function daysAgo(days: number): string {
  const date = new Date();
  date.setDate(date.getDate() - days);
  date.setHours(15, 0, 0, 0);
  return date.toISOString();
}

function entry(overrides: Partial<HistoryEntry> & { id: string; title: string }): HistoryEntry {
  return {
    videoId: 'vid',
    channel: 'Channel',
    quality: '1080p',
    filename: 'file.mp4',
    absolutePath: `/tmp/${overrides.id}.mp4`,
    sizeBytes: 22 * 1024 * 1024,
    completedAt: daysAgo(0),
    durationLabel: '12:00',
    thumbnail: '',
    ...overrides,
  };
}

describe('Downloads page', () => {
  beforeEach(() => {
    history.set([]);
    modal.set(null);
    (window as any).go = { main: { App: {
      OpenFile: vi.fn(async () => {}),
      RevealInFinder: vi.fn(async () => {}),
      RemoveDownload: vi.fn(async () => {}),
      DeleteDownloadFile: vi.fn(async () => {}),
    } } };
  });

  test('empty history shows No downloads yet', () => {
    render(Downloads);
    expect(screen.getByRole('heading', { name: 'Downloads' })).toBeInTheDocument();
    expect(screen.getByLabelText('Search downloads')).toBeInTheDocument();
    expect(screen.getByText('No downloads yet')).toBeInTheDocument();
  });

  test('date-groups individual rows and does not invent playlist groups', () => {
    history.set([
      entry({ id: 'today', title: 'Go 2026', completedAt: daysAgo(0) }),
      entry({ id: 'yesterday', title: 'Tracing', channel: 'TechCraft', completedAt: daysAgo(1) }),
      entry({ id: 'old', title: 'Chaos Engineering', completedAt: daysAgo(8) }),
    ]);
    render(Downloads);
    expect(screen.getByText('Today')).toBeInTheDocument();
    expect(screen.getByText('Yesterday')).toBeInTheDocument();
    expect(screen.getByText('Earlier')).toBeInTheDocument();
    expect(screen.getByText('Go 2026')).toBeInTheDocument();
    expect(screen.getByText('Tracing')).toBeInTheDocument();
    expect(screen.queryByText(/Playlist ·/)).not.toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Show in Finder' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('button', { name: 'Open downloaded file' }).length).toBeGreaterThan(0);
  });

  test('expands a playlist group only when history carries collection identity', async () => {
    const user = userEvent.setup();
    history.set([
      entry({
        id: 'ep1', title: 'Introduction', playlistId: 'PLgo', playlistTitle: 'Concurrency in Go',
        collectionIndex: 1, completedAt: daysAgo(1),
      }),
      entry({
        id: 'ep2', title: 'WaitGroups', playlistId: 'PLgo', playlistTitle: 'Concurrency in Go',
        collectionIndex: 2, completedAt: daysAgo(1),
      }),
    ]);
    render(Downloads);
    expect(screen.getByText('Concurrency in Go')).toBeInTheDocument();
    expect(screen.getByText(/Playlist · 2 episodes · 1080p/)).toBeInTheDocument();
    expect(screen.queryByText('Introduction')).not.toBeInTheDocument();

    await user.click(screen.getByText('Concurrency in Go'));
    expect(screen.getByText('Introduction')).toBeInTheDocument();
    expect(screen.getByText('EP 01')).toBeInTheDocument();
    expect(screen.getByText('WaitGroups')).toBeInTheDocument();
  });

  test('search filters rows and Open calls the existing file API', async () => {
    const user = userEvent.setup();
    history.set([
      entry({ id: 'keep', title: 'Go 2026', channel: 'Gopher Talks' }),
      entry({ id: 'hide', title: 'Chaos Engineering' }),
    ]);
    render(Downloads);
    await user.type(screen.getByLabelText('Search downloads'), 'gopher');
    expect(screen.getByText('Go 2026')).toBeInTheDocument();
    expect(screen.queryByText('Chaos Engineering')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Open downloaded file' }));
    expect((window as any).go.main.App.OpenFile).toHaveBeenCalledWith('/tmp/keep.mp4');
  });

  test('missing files disable Open and Reveal and still allow remove from history', async () => {
    const user = userEvent.setup();
    history.set([
      entry({ id: 'gone', title: 'Missing clip', fileMissing: true }),
    ]);
    render(Downloads);
    expect(screen.getByText(/File missing/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Open downloaded file' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Show in Finder' })).toBeDisabled();

    await user.click(screen.getByLabelText('Show details for Missing clip'));
    await user.click(screen.getByRole('button', { name: 'Remove from history' }));
    expect(get(modal)).toMatchObject({ kind: 'confirm', title: 'Remove from history?' });
  });
});
