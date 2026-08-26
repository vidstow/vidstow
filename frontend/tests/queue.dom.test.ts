import { render, screen } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';

import Queue from '../src/pages/Queue.svelte';
import { queueView, route } from '../src/lib/stores.js';
import type { QueueView } from '../src/lib/lifecycle-ui/types.js';

function queueSnapshot(overrides: Partial<QueueView> = {}): QueueView {
  return {
    revision: 1,
    rows: [],
    summary: {
      totalJobs: 0,
      runningJobs: 0,
      occupiedSlots: 0,
      slotLimit: 2,
      waitingJobs: 0,
      pausedJobs: 0,
    },
    capabilities: {},
    persistence: { available: true, healthy: true },
    ...overrides,
  };
}

afterEach(() => {
  queueView.set(null);
  route.set('home');
});

describe('Following empty rail under Queue', () => {
  test('Queue is the default pane and Following is a local empty list', async () => {
    const user = userEvent.setup();
    const GetQueueView = vi.fn();
    (window as any).go = { main: { App: { GetQueueView } } };
    const storage = vi.spyOn(Storage.prototype, 'setItem');
    route.set('queue');
    queueView.set(queueSnapshot({
      rows: [{
        id: 'job-1',
        title: 'Queued fixture video',
        lifecycle: 'pending',
        occupiesSlot: false,
        capabilities: {},
      }],
      summary: {
        totalJobs: 1,
        runningJobs: 0,
        occupiedSlots: 0,
        slotLimit: 2,
        waitingJobs: 1,
        pausedJobs: 0,
      },
    }));

    render(Queue);

    expect(screen.getByRole('tab', { name: 'Queue' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: 'Following' })).toHaveAttribute('aria-selected', 'false');
    expect(screen.getByText('Queued fixture video')).toBeInTheDocument();
    expect(screen.queryByText('No playlists followed')).not.toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Following' }));

    expect(screen.getByRole('tab', { name: 'Following' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('heading', { level: 1, name: 'Following' })).toBeInTheDocument();
    expect(screen.getByText('No playlists followed')).toBeInTheDocument();
    expect(screen.getByText('Follow a public playlist from Home. New videos can show up here later.')).toBeInTheDocument();
    expect(screen.queryByText('Queued fixture video')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Pause All' })).not.toBeInTheDocument();
    expect(get(route)).toBe('queue');
    expect(GetQueueView).not.toHaveBeenCalled();
    expect(storage).not.toHaveBeenCalled();

    await user.click(screen.getByRole('tab', { name: 'Queue' }));
    expect(screen.getByText('Queued fixture video')).toBeInTheDocument();
    expect(screen.queryByText('No playlists followed')).not.toBeInTheDocument();

    storage.mockRestore();
  });
});
