import { render, screen } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';

import Downloads from '../src/pages/Downloads.svelte';
import { history, modal, banner, route } from '../src/lib/stores.js';
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
    banner.set(null);
    route.set('downloads');
    (window as any).go = { main: { App: {
      OpenFile: vi.fn(async () => {}),
      RevealInFinder: vi.fn(async () => {}),
      RemoveDownload: vi.fn(async () => {}),
      DeleteDownloadFile: vi.fn(async () => {}),
    } } };
  });

  test('empty history shows a Queue-style empty state', async () => {
    const user = userEvent.setup();
    render(Downloads);
    expect(screen.getByRole('heading', { name: 'Downloads' })).toBeInTheDocument();
    expect(screen.getByLabelText('Search downloads')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Nothing here yet' })).toBeInTheDocument();
    expect(screen.getByText(/Paste a link on Home/)).toBeInTheDocument();
    expect(screen.getByText(/Finished files show up here/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Go to Home' }));
    expect(get(route)).toBe('home');
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
    const reveal = screen.getAllByRole('button', { name: 'Show in Finder' })[0];
    const open = screen.getAllByRole('button', { name: 'Open downloaded file' })[0];
    expect(reveal).toHaveClass('ghost');
    expect(open).toHaveClass('pri');
    expect(reveal.querySelector('svg')).not.toBeNull();
    expect(open.querySelector('svg')).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Show details for Go 2026' }).querySelector('.chev')?.textContent).toBe('▸');
    expect(screen.getByText(/1080p · 22.0 MB · Channel · (just now|\d+[mhd] ago)/)).toBeInTheDocument();
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
    const groupReveal = screen.getByRole('button', { name: 'Reveal' });
    const groupOpen = screen.getByRole('button', { name: 'Open' });
    expect(groupReveal).toHaveClass('ghost');
    expect(groupOpen).toHaveClass('pri');
    expect(groupReveal.querySelector('svg')).not.toBeNull();
    expect(groupOpen.querySelector('svg')).not.toBeNull();

    await user.click(screen.getByText('Concurrency in Go'));
    expect(screen.getByText('Introduction')).toBeInTheDocument();
    expect(screen.getByText('EP 01')).toBeInTheDocument();
    expect(screen.getByText('WaitGroups')).toBeInTheDocument();
    expect(screen.getAllByText(/12:00 · 22.0 MB · (just now|\d+[mhd] ago)/)).toHaveLength(2);
    expect(screen.queryByText(/12:00 · 1080p/)).not.toBeInTheDocument();
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

  test('search with no matches offers Clear search', async () => {
    const user = userEvent.setup();
    history.set([
      entry({ id: 'keep', title: 'Go 2026' }),
    ]);
    render(Downloads);
    await user.type(screen.getByLabelText('Search downloads'), 'zzzz');
    expect(screen.getByText('No matching downloads')).toBeInTheDocument();
    expect(screen.getByText('Try a different title, channel, or filename.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Go to Home' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Clear search' }));
    expect(screen.getByText('Go 2026')).toBeInTheDocument();
    expect(screen.queryByText('No matching downloads')).not.toBeInTheDocument();
  });

  test('keeps a receipt without a File missing chip and says the path is gone on Open', async () => {
    const user = userEvent.setup();
    (window as any).go.main.App.OpenFile = vi.fn(async () => {
      throw new Error('That file is no longer at this path');
    });
    history.set([
      entry({ id: 'gone', title: 'Finished video' }),
    ]);
    render(Downloads);
    expect(screen.getByText('Finished video')).toBeInTheDocument();
    expect(screen.queryByText(/File missing/i)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Open downloaded file' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Show in Finder' })).toBeEnabled();
    await user.click(screen.getByRole('button', { name: 'Open downloaded file' }));
    expect(get(banner)).toMatchObject({ kind: 'danger', message: 'That file is no longer at this path' });
  });

  test('a row chevron shows that details can expand', async () => {
    const user = userEvent.setup();
    history.set([
      entry({ id: 'today', title: 'Go 2026', completedAt: daysAgo(0) }),
    ]);
    render(Downloads);
    const row = screen.getByRole('button', { name: 'Show details for Go 2026' });
    expect(row).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('/tmp/today.mp4')).not.toBeInTheDocument();
    await user.click(row);
    expect(screen.getByRole('button', { name: 'Hide details for Go 2026' })).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('/tmp/today.mp4')).toBeInTheDocument();
  });
});
