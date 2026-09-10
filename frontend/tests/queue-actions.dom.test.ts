import { act, render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';
import Queue from '../src/pages/Queue.svelte';
import { queueView, modal, banner } from '../src/lib/stores.js';
import type { QueueView, LifecycleJobViewModel } from '../src/lib/lifecycle-ui/types.js';

const row = (id: string): LifecycleJobViewModel => ({ id, title: id, lifecycle: 'paused', occupiesSlot: false, commandToken: `token-${id}`, capabilities: { resume: true } });
function view(rows: LifecycleJobViewModel[]): QueueView {
  return { revision: 1, rows, collections: [], summary: { totalJobs: rows.length, runningJobs: 0, occupiedSlots: 0, slotLimit: 2, waitingJobs: 0, pausedJobs: rows.length }, capabilities: { commandToken: 'queue-token' }, persistence: { available: true, healthy: true } };
}
beforeEach(() => { modal.set(null); banner.set(null); });

test('retry is locked until completion and refresh errors do not imply rejection', async () => {
  const data = view([{ ...row('failed'), lifecycle: 'failed', capabilities: { retry: true } }]);
  queueView.set(data);
  let finish!: () => void;
  const RetryQueueJob = vi.fn(() => new Promise<void>((resolve) => { finish = resolve; }));
  (window as any).go = { main: { App: { RetryQueueJob, GetQueueView: vi.fn().mockRejectedValue(new Error('offline')) } } };
  render(Queue);
  await userEvent.setup().dblClick(screen.getByRole('button', { name: 'Retry failed' }));
  expect(RetryQueueJob).toHaveBeenCalledOnce();
  expect(screen.getByRole('status')).toHaveTextContent('Retrying…');
  expect(screen.getByRole('button', { name: 'Retry' })).toBeDisabled();
  await act(() => finish());
  await waitFor(() => expect(get(banner)?.message).toContain('Action accepted'));
  expect(get(modal)).toBeNull();
  expect(screen.getByRole('button', { name: 'Retry' })).toBeEnabled();
});

test('resume all continues after failure and submits collection children only once', async () => {
  const data = view([row('first'), row('second'), { ...row('child'), collectionId: 'list' }]);
  data.collections = [{ id: 'list', kind: 'playlist', title: 'Playlist', policy: 'video:1080p', childJobIds: ['child'], total: 1, completed: 0, failed: 0, canceled: 0, active: 0, pending: 0, paused: 1, progress: 0, progressLabel: '', commandToken: 'list-token', capabilities: { resume: true } }];
  queueView.set(data);
  const ResumeQueueJob = vi.fn().mockRejectedValueOnce(new Error('stale')).mockResolvedValue(undefined);
  const ResumeQueueCollection = vi.fn().mockResolvedValue(1);
  (window as any).go = { main: { App: { ResumeQueueJob, ResumeQueueCollection, GetQueueView: vi.fn().mockResolvedValue(data) } } };
  render(Queue);
  await userEvent.setup().click(screen.getByRole('button', { name: 'Resume all' }));
  await waitFor(() => expect(get(modal)?.title).toBe('Resume all results'));
  expect(ResumeQueueJob.mock.calls.map((args) => args[0])).toEqual(['first', 'second']);
  expect(ResumeQueueCollection).toHaveBeenCalledExactlyOnceWith('list', 'list-token');
  expect(get(modal)?.message).toBe('Resume requested for 2 of 3 items. 1 could not resume.');
  expect(get(modal)?.detail).toBe('first: stale');
});

test.each([
  ['network_interrupted', 'Retry problem'],
  ['rate_limited', 'Retry problem'],
  ['folder_unavailable', 'Change folder for problem'],
  ['resource_unavailable', 'Open source for problem'],
])('failure %s highlights its recovery action', async (category, label) => {
  queueView.set(view([{ ...row('problem'), lifecycle: 'failed', capabilities: { retry: true, changeFolder: true, openSource: true }, failure: { category, messageKey: '', heading: 'Problem', message: 'Could not finish.', recommendedAction: 'Recovery guidance', retryable: true, partialOutput: false } }]));
  render(Queue);
  expect(screen.getByRole('button', { name: label })).toHaveClass('primary');
  expect(screen.getByText('Recovery guidance')).toBeInTheDocument();
});
