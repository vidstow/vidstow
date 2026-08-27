import { render, screen, within } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';

import Following from '../src/pages/Following.svelte';
import Queue from '../src/pages/Queue.svelte';
import Sidebar from '../src/lib/components/Sidebar.svelte';
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

describe('Following sidebar page', () => {
  test('Following sits under Queue in the sidebar', async () => {
    const user = userEvent.setup();
    const GetQueueView = vi.fn();
    (window as any).go = { main: { App: { GetQueueView } } };
    const storage = vi.spyOn(Storage.prototype, 'setItem');

    render(Sidebar);
    const primary = screen.getByRole('navigation', { name: 'Primary navigation' });
    expect(within(primary).getAllByRole('button').map((button) => button.textContent?.replace(/\s+/g, ' ').trim())).toEqual([
      'Home',
      'Queue',
      'Following',
      'Downloads',
    ]);

    await user.click(within(primary).getByRole('button', { name: 'Following' }));
    expect(get(route)).toBe('following');
    expect(GetQueueView).not.toHaveBeenCalled();
    expect(storage).not.toHaveBeenCalled();
    storage.mockRestore();
  });

  test('Following is a local empty list', () => {
    const GetQueueView = vi.fn();
    (window as any).go = { main: { App: { GetQueueView } } };
    const storage = vi.spyOn(Storage.prototype, 'setItem');

    render(Following);
    expect(screen.getByRole('heading', { level: 1, name: 'Following' })).toBeInTheDocument();
    expect(screen.getByText('You are not following any playlists.')).toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Queue' })).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Following' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Pause All' })).not.toBeInTheDocument();
    expect(GetQueueView).not.toHaveBeenCalled();
    expect(storage).not.toHaveBeenCalled();
    storage.mockRestore();
  });

  test('Queue has no Following switcher', () => {
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
    expect(screen.getByText('Queued fixture video')).toBeInTheDocument();
    expect(screen.queryByRole('tablist', { name: 'Queue sections' })).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: 'Following' })).not.toBeInTheDocument();
    expect(screen.queryByText('You are not following any playlists.')).not.toBeInTheDocument();
  });
});
