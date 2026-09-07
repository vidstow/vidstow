import { fireEvent, render, screen } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { describe, expect, test, vi } from 'vitest';

import ActionRequiredReviewDialog from '../src/lib/lifecycle-ui/ActionRequiredReviewDialog.svelte';
import DestinationConflictDialog from '../src/lib/lifecycle-ui/DestinationConflictDialog.svelte';
import LifecycleJobRow, { type LifecycleJobActionEvent } from '../src/lib/lifecycle-ui/LifecycleJobRow.svelte';
import QueueOverview from '../src/lib/lifecycle-ui/QueueOverview.svelte';
import QuitConfirmationDialog from '../src/lib/lifecycle-ui/QuitConfirmationDialog.svelte';
import type {
  ActionRequiredReviewViewModel,
  DestinationConflictEventDetail,
  DestinationConflictViewModel,
  LifecycleJobEventDetail,
  LifecycleJobViewModel,
  QueueOverviewViewModel,
} from '../src/lib/lifecycle-ui/types.js';

function queueModel(overrides: Partial<QueueOverviewViewModel> = {}): QueueOverviewViewModel {
  return {
    summary: {
      totalJobs: 0,
      runningJobs: 0,
      occupiedSlots: 0,
      slotLimit: 2,
      waitingJobs: 0,
      pausedJobs: 0,
    },
    jobs: [],
    canPauseAll: false,
    canClearCompleted: false,
    commandToken: 'queue-token-7',
    ...overrides,
  };
}

function conflictModel(overrides: Partial<DestinationConflictViewModel> = {}): DestinationConflictViewModel {
  return {
    conflictToken: 'conflict-token-7',
    unavailableName: 'Video.mp4',
    proposedName: 'Video (2).mp4',
    proposedNameAvailable: true,
    ...overrides,
  };
}

describe('backend-authored capabilities', () => {
  test('empty Queue still shows how many slots can run', () => {
    render(QueueOverview, { props: { model: queueModel() } });
    expect(screen.getByRole('heading', { name: 'Queue' })).toBeInTheDocument();
    expect(screen.getByText('No active downloads')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Nothing here yet' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Pause all' })).not.toBeInTheDocument();
  });

  test('queue-wide actions fail closed when capabilities are absent at runtime', async () => {
    const onPauseAll = vi.fn();
    const onResumeAll = vi.fn();
    const user = userEvent.setup();

    render(QueueOverview, {
      props: {
        model: {
          ...queueModel(),
          canPauseAll: undefined as unknown as boolean,
          jobs: [{
            id: 'active-1', title: 'Active', lifecycle: 'active', occupiesSlot: true,
            capabilities: {}, commandToken: 'job-token',
          }],
        } as QueueOverviewViewModel,
      },
      events: { 'pause-all': onPauseAll, 'resume-all': onResumeAll },
    });

    expect(screen.queryByRole('button', { name: 'Pause all' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Resume all' })).not.toBeInTheDocument();
    expect(onPauseAll).not.toHaveBeenCalled();
    expect(onResumeAll).not.toHaveBeenCalled();
  });

  test('playlist collections show children in the inspector and emit only backend-authorized parent actions', async () => {
    const onCollectionAction = vi.fn();
    const user = userEvent.setup();
    const { container } = render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'child-1', collectionId: 'collection-1', collectionIndex: 1,
            title: 'Child video', lifecycle: 'pending', desired: 'running', occupiesSlot: false,
            thumbnailUrl: 'https://i.ytimg.com/vi/child/hqdefault.jpg',
            capabilities: { pause: true }, commandToken: 'child-token',
          }],
          collections: [{
            id: 'collection-1', kind: 'playlist', title: 'Fixture playlist', metadata: 'Creator', policy: 'video:1080p',
            thumbnailUrl: 'https://i.ytimg.com/vi/playlist/hqdefault.jpg',
            childJobIds: ['child-1'], total: 1, completed: 0, failed: 0, canceled: 0,
            active: 0, pending: 1, paused: 0, progress: 0, progressLabel: '0 of 1 complete',
            capabilities: { pause: true }, commandToken: 'collection-token',
          }],
        }),
        onCollectionAction,
      },
    });

    expect(screen.getAllByText('Fixture playlist').length).toBeGreaterThan(0);
    expect(screen.getByText('Playlist')).toBeInTheDocument();
    expect(screen.getByText('Child video')).toBeInTheDocument();
    const thumbs = container.querySelectorAll('.qth img, .kth img, .ibig');
    expect(thumbs.length).toBeGreaterThanOrEqual(3);
    expect(container.querySelector('.qth img')).toHaveAttribute('src', 'https://i.ytimg.com/vi/playlist/hqdefault.jpg');
    expect(container.querySelector('.kth img')).toHaveAttribute('src', 'https://i.ytimg.com/vi/child/hqdefault.jpg');
    await user.click(screen.getByRole('button', { name: 'Pause collection' }));
    expect(onCollectionAction).toHaveBeenCalledWith({ collectionId: 'collection-1', commandToken: 'collection-token', action: 'pause' });
    expect(screen.queryByRole('button', { name: 'Collapse Fixture playlist' })).not.toBeInTheDocument();
  });

  test('playlist inspector shows a finished count chip instead of a muted row', () => {
    const { container } = render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [
            {
              id: 'done-1', collectionId: 'collection-1', collectionIndex: 1,
              title: 'Done one', lifecycle: 'completed', occupiesSlot: false,
              capabilities: {}, commandToken: 'done-token',
            },
            {
              id: 'live-1', collectionId: 'collection-1', collectionIndex: 2,
              title: 'Live child', lifecycle: 'active', occupiesSlot: true, speedLabel: '2.9 MiB/s',
              capabilities: { pause: true }, commandToken: 'live-token',
            },
          ],
          collections: [{
            id: 'collection-1', kind: 'playlist', title: 'Mixed playlist', policy: 'video:1080p',
            childJobIds: ['done-1', 'live-1'], total: 12, completed: 4, failed: 0, canceled: 0,
            active: 1, pending: 7, paused: 0, progress: 0.33, progressLabel: '4 of 12 complete',
            capabilities: { pause: true }, commandToken: 'collection-token',
          }],
        }),
      },
    });

    expect(screen.getByText('4 finished')).toBeInTheDocument();
    expect(container.querySelector('.kdone')).not.toBeNull();
    expect(container.querySelector('.kid.done')).toBeNull();
    expect(screen.getByText('Live child')).toBeInTheDocument();
    expect(screen.queryByText('Done one')).not.toBeInTheDocument();
  });

  test('interleaves collections and standalone videos using backend priority order', () => {
    const { container } = render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [
            {
              id: 'active-video', title: 'Active standalone video', lifecycle: 'active',
              occupiesSlot: true, capabilities: {},
            },
            {
              id: 'playlist-child', collectionId: 'live-collection', collectionIndex: 1,
              title: 'Playlist episode', lifecycle: 'pending',
              occupiesSlot: false, capabilities: {},
            },
          ],
          collections: [{
            id: 'live-collection', kind: 'playlist', title: 'Live collection', policy: 'video:1080p',
            childJobIds: ['playlist-child'], total: 1, completed: 0, failed: 0, canceled: 0,
            active: 0, pending: 1, paused: 0, progress: 0, progressLabel: '0 of 1 complete',
            capabilities: {},
          }],
        }),
      },
    });

    const entries = Array.from(container.querySelectorAll('.qrow'));
    expect(entries).toHaveLength(2);
    expect(entries[0]).toHaveAttribute('aria-label', 'Active standalone video');
    expect(entries[1]).toHaveAttribute('aria-label', 'Live collection');
  });

  test('batch collection parents omit synthetic thumbnails', () => {
    const { container } = render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [],
          collections: [{
            id: 'batch-1', kind: 'batch', title: 'Batch download · 2 videos', thumbnailUrl: 'https://example.invalid/batch.jpg',
            policy: 'video:720p', childJobIds: [], total: 2, completed: 0, failed: 0, canceled: 0,
            active: 2, pending: 0, paused: 0, progress: 0.5, progressLabel: '0 of 2 complete',
            capabilities: {}, commandToken: 'batch-token',
          }],
        }),
      },
    });

    expect(screen.getAllByText('Batch download · 2 videos').length).toBeGreaterThan(0);
    expect(container.querySelector('.qrow[data-policy="video:720p"]')).toBeTruthy();
    expect(container.querySelector('.qth img')).not.toBeInTheDocument();
    expect(container.querySelector('.qth .cnt')).toHaveTextContent('2');
    expect(screen.getByText('Batch')).toBeInTheDocument();
  });

  test('playlist rows fall back to a child still when the parent has none', () => {
    const { container } = render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'ep-1', collectionId: 'pl-1', collectionIndex: 1,
            title: 'Episode one', lifecycle: 'active', occupiesSlot: true,
            thumbnailUrl: 'https://i.ytimg.com/vi/episode/hqdefault.jpg',
            capabilities: {},
          }],
          collections: [{
            id: 'pl-1', kind: 'playlist', title: 'No parent art', policy: 'video:1080p',
            childJobIds: ['ep-1'], total: 1, completed: 0, failed: 0, canceled: 0,
            active: 1, pending: 0, paused: 0, progress: 0.2, progressLabel: '0 of 1 complete',
            capabilities: {},
          }],
        }),
      },
    });

    expect(container.querySelector('.qth img')).toHaveAttribute('src', 'https://i.ytimg.com/vi/episode/hqdefault.jpg');
    expect(screen.getByText('Playlist')).toBeInTheDocument();
  });

  test('playlist collection actions fail closed without a valid token', async () => {
    const onCollectionAction = vi.fn();
    const user = userEvent.setup();
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [],
          collections: [{
            id: 'collection-1', kind: 'playlist', title: 'Fixture playlist', policy: 'audio:original', childJobIds: [],
            total: 0, completed: 0, failed: 0, canceled: 0, active: 0, pending: 0, paused: 0,
            progress: 0, progressLabel: '0 of 0 complete', capabilities: { remove: true }, commandToken: '',
          }],
        }),
        onCollectionAction,
      },
    });
    const remove = screen.getByRole('button', { name: 'Remove' });
    expect(remove).toBeDisabled();
    await user.click(remove);
    expect(onCollectionAction).not.toHaveBeenCalled();
  });

  test('queue-wide actions emit only when explicitly enabled', async () => {
    const onPauseAll = vi.fn();
    const onResumeAll = vi.fn();
    const user = userEvent.setup();

    render(QueueOverview, {
      props: {
        model: queueModel({
          canPauseAll: true,
          jobs: [{
            id: 'paused-1', title: 'Paused video', lifecycle: 'paused', occupiesSlot: false,
            capabilities: { resume: true }, commandToken: 'job-token',
          }],
        }),
      },
      events: { 'pause-all': onPauseAll, 'resume-all': onResumeAll },
    });

    await user.click(screen.getByRole('button', { name: 'Pause all' }));
    await user.click(screen.getByRole('button', { name: 'Resume all' }));
    expect(onPauseAll).toHaveBeenCalledOnce();
    expect(onResumeAll).toHaveBeenCalledOnce();
  });

  test('disabled row capabilities do not emit actions', async () => {
    const onPause = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const onCancel = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const job: LifecycleJobViewModel = {
      id: 'job-1',
      title: 'Example video',
      lifecycle: 'active',
      phase: 'downloading',
      occupiesSlot: true,
      capabilities: { pause: false, cancel: false },
      commandToken: 'job-token-7',
    };
    const user = userEvent.setup();

    render(LifecycleJobRow, {
      props: { job },
      events: { pause: onPause, cancel: onCancel },
    });

    const pause = screen.getByRole('button', { name: 'Pause download' });
    const cancel = screen.getByRole('button', { name: 'Cancel download' });
    expect(pause).toBeDisabled();
    expect(cancel).toBeDisabled();
    await user.click(pause);
    await user.click(cancel);
    expect(onPause).not.toHaveBeenCalled();
    expect(onCancel).not.toHaveBeenCalled();
    expect(screen.getByLabelText('Downloading, occupies an active slot')).toBeInTheDocument();
  });

  test('row actions require an opaque token and echo it without deriving authority', async () => {
    const onResume = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const user = userEvent.setup();
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'paused-1', title: 'Paused video', lifecycle: 'paused', occupiesSlot: false,
          capabilities: { resume: true }, commandToken: 'backend-command-token',
        },
      },
      events: { resume: onResume },
    });
    await user.click(screen.getByRole('button', { name: 'Resume download' }));
    expect(onResume.mock.calls[0][0].detail).toEqual({ jobId: 'paused-1', commandToken: 'backend-command-token' });
  });

  test('failed rows render backend-authored recovery copy and capability-backed start again', async () => {
    const onStartAgain = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const user = userEvent.setup();
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'disk-1', title: 'Interrupted video', lifecycle: 'failed', occupiesSlot: false,
          failure: {
            category: 'disk_full', messageKey: 'queue.failure.disk_full', heading: 'Not enough disk space',
            message: 'VidStow could not finish writing this download.',
            recommendedAction: 'Free space or change the default folder, then start this item again.',
            retryable: false, partialOutput: true,
          },
          capabilities: { startAgain: true, remove: true }, commandToken: 'disk-command-token',
        },
      },
      events: { 'start-again': onStartAgain },
    });
    expect(screen.getByRole('alert')).toHaveTextContent('Not enough disk space');
    expect(screen.getByRole('alert')).toHaveTextContent('Free space or change the default folder');
    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Start again' }));
    expect(onStartAgain.mock.calls[0][0].detail).toEqual({ jobId: 'disk-1', commandToken: 'disk-command-token' });
  });

  test('action-required rows expose independently authorized Review and queue-only Remove', async () => {
    const onReview = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const onRemove = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const user = userEvent.setup();
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'action-1', title: 'Saved video', lifecycle: 'action-required', occupiesSlot: false,
          capabilities: { review: true, remove: true }, commandToken: 'action-command-token',
        },
      },
      events: { review: onReview, remove: onRemove },
    });
    await user.click(screen.getByRole('button', { name: 'Review' }));
    await user.click(screen.getByRole('button', { name: 'Remove download' }));
    expect(onReview.mock.calls[0][0].detail).toEqual({ jobId: 'action-1', commandToken: 'action-command-token' });
    expect(onRemove.mock.calls[0][0].detail).toEqual({ jobId: 'action-1', commandToken: 'action-command-token' });
  });

  test('cleanup-phase terminal rows can expose backend-authorized Review without Remove', async () => {
    const onReview = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const user = userEvent.setup();
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'cleanup-1', title: 'Preserved cleanup', lifecycle: 'canceled', phase: 'cleaning-up', occupiesSlot: false,
          capabilities: { review: true, remove: false }, commandToken: 'cleanup-command-token',
        },
      },
      events: { review: onReview },
    });
    await user.click(screen.getByRole('button', { name: 'Review' }));
    expect(onReview.mock.calls[0][0].detail).toEqual({ jobId: 'cleanup-1', commandToken: 'cleanup-command-token' });
    expect(screen.queryByRole('button', { name: 'Remove download' })).not.toBeInTheDocument();
  });

  test('cleanup-phase terminal rows expose Remove after backend cleanup settles', () => {
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'cleanup-settled', title: 'Settled cleanup', lifecycle: 'canceled', phase: 'cleaning-up', occupiesSlot: false,
          capabilities: { review: false, remove: true }, commandToken: 'cleanup-settled-token',
        },
      },
    });
    expect(screen.getByRole('button', { name: 'Remove download' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Review' })).not.toBeInTheDocument();
    expect(screen.getByLabelText('Canceled')).toBeInTheDocument();
    expect(screen.getByText('Canceled. Resumable data was removed.')).toBeInTheDocument();
  });

  test('persistence-revoked queue data can render but cannot authorize controls', () => {
    render(QueueOverview, {
      props: { model: queueModel({ commandToken: undefined, canPauseAll: false, canClearCompleted: false, jobs: [{ id: 'active-1', title: 'Active', lifecycle: 'active', occupiesSlot: true, capabilities: {}, commandToken: undefined }] }) },
    });
    expect(screen.queryByRole('button', { name: 'Pause all' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Resume all' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Pause' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
  });

  test('missing-folder errors offer Change from the inspector', async () => {
    const onAction = vi.fn<(event: LifecycleJobActionEvent) => void>();
    const user = userEvent.setup();
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'missing-folder', title: 'Missing folder video', lifecycle: 'failed', occupiesSlot: false,
            failure: {
              category: 'folder_unavailable', messageKey: 'queue.failure.folder_unavailable',
              heading: 'Save folder is missing',
              message: 'The folder for this download is gone. Plug the drive back in, or Change to a different folder.',
              recommendedAction: 'Bring the original path back, or Change the folder.',
              retryable: true, partialOutput: true,
            },
            capabilities: { retry: true, changeFolder: true, remove: true }, commandToken: 'folder-token',
          }],
        }),
        onAction,
      },
    });

    expect(screen.getAllByText('Save folder is missing').length).toBeGreaterThan(0);
    await user.click(screen.getByRole('button', { name: 'Change' }));
    expect(onAction.mock.calls[0][0]).toEqual({
      jobId: 'missing-folder', commandToken: 'folder-token', action: 'change-folder',
    });
  });

  test('failed inspector keeps Retry and Remove without Cancel or Open source', () => {
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'start-failed', title: 'Could not start', lifecycle: 'failed', occupiesSlot: false,
            failure: {
              category: 'could_not_start', messageKey: 'queue.failure.could_not_start',
              heading: 'Download could not start',
              message: 'Nothing was saved.',
              recommendedAction: 'Retry this item.',
              retryable: true, partialOutput: false,
            },
            capabilities: { retry: true, remove: true }, commandToken: 'start-token',
          }],
        }),
      },
    });
    expect(screen.getAllByText('Download could not start').length).toBeGreaterThan(0);
    expect(screen.getByText('Nothing was saved.')).toBeInTheDocument();
    expect(screen.queryByText(/hint:/)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Copy details' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Open source' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Copy link' })).not.toBeInTheDocument();
  });

  test('canceled inspector drops Pause and does not stay on Cleaning up after Remove is authorized', async () => {
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'canceled', title: 'Canceled video', lifecycle: 'canceled', phase: 'cleaning-up', occupiesSlot: false,
            capabilities: { remove: true }, commandToken: 'canceled-token',
          }],
        }),
      },
    });
    await userEvent.click(screen.getByText('Canceled (1)'));
    expect(screen.getAllByText('Canceled').length).toBeGreaterThan(0);
    expect(screen.queryByText('Cleaning up')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Pause' })).not.toBeInTheDocument();
  });

  test('refused download inspector offers Retry, Open source, Copy link, and Remove', () => {
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'refused', title: 'Public video', lifecycle: 'failed', occupiesSlot: false,
            failure: {
              category: 'authentication_required', messageKey: 'queue.failure.authentication_required',
              heading: 'Download was refused',
              message: 'The page may still play in a browser. Try again.', recommendedAction: 'Retry this item.',
              retryable: true, partialOutput: false,
            },
            capabilities: { retry: true, remove: true, openSource: true, copyLink: true }, commandToken: 'auth-token',
          }],
        }),
      },
    });
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Open source' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copy link' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove' })).toBeInTheDocument();
  });

  test('unusable leftover paused rows offer Resume and Discard', async () => {
    const onAction = vi.fn<(event: LifecycleJobActionEvent) => void>();
    const user = userEvent.setup();
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'leftover', title: 'Unusable leftover', lifecycle: 'paused', occupiesSlot: false,
            savedBytes: 843 * 1024 * 1024,
            capabilities: { resume: true, discard: true, cancel: true }, commandToken: 'leftover-token',
          }],
        }),
        onAction,
      },
    });
    expect(screen.getByRole('button', { name: 'Resume' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Discard' }));
    expect(onAction.mock.calls[0][0]).toEqual({
      jobId: 'leftover', commandToken: 'leftover-token', action: 'discard',
    });
  });
});

describe('action-required recovery', () => {
  const review: ActionRequiredReviewViewModel = {
    jobId: 'action-1', title: 'Saved video', heading: 'This download needs your decision',
    message: 'The saved session could not be inspected safely.',
    preservationNotice: 'Removing the row leaves saved data on disk.', canStartOver: true,
    canRetryRecovery: true, canRetryFreshLink: true, canDiscard: true, canRemove: true, canRetryCleanup: false,
  };

  test('offers a fresh Home analysis while clearly preserving the original row', async () => {
    const onStartOver = vi.fn();
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(ActionRequiredReviewDialog, { props: { open: true, review, onStartOver, onClose } });

    expect(screen.getByRole('dialog', { name: review.heading })).toBeInTheDocument();
    expect(screen.getByText(review.preservationNotice)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Keep for now' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Start over from Home' }));
    expect(onStartOver).toHaveBeenCalledOnce();
    expect(onClose).not.toHaveBeenCalled();
  });

  test('offers queue-only removal only when the backend authorizes it', async () => {
    const onRemove = vi.fn();
    const user = userEvent.setup();
    const { rerender } = render(ActionRequiredReviewDialog, { props: { open: true, review, onRemove } });
    await user.click(screen.getByRole('button', { name: 'Remove from queue' }));
    expect(onRemove).toHaveBeenCalledOnce();

    await rerender({ open: true, review: { ...review, canRemove: false }, onRemove });
    expect(screen.queryByRole('button', { name: 'Remove from queue' })).not.toBeInTheDocument();
  });

  test('does not advertise start-over when the backend withholds it', () => {
    render(ActionRequiredReviewDialog, { props: { open: true, review: { ...review, canStartOver: false } } });
    expect(screen.queryByRole('button', { name: 'Start over from Home' })).not.toBeInTheDocument();
    expect(screen.getByText(/Starting over is unavailable/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Remove from queue' })).toBeInTheDocument();
  });

  test('offers explicit recovery, discard, and quarantined-cleanup actions only when authorized', async () => {
    const onRetryRecovery = vi.fn();
    const onRetryFreshLink = vi.fn();
    const onDiscard = vi.fn();
    const onRetryCleanup = vi.fn();
    const user = userEvent.setup();
    const { rerender } = render(ActionRequiredReviewDialog, { props: { open: true, review, onRetryRecovery, onRetryFreshLink, onDiscard, onRetryCleanup } });
    await user.click(screen.getByRole('button', { name: 'Try recovery again' }));
    await user.click(screen.getByRole('button', { name: 'Retry with fresh link' }));
    await user.click(screen.getByRole('button', { name: 'Discard saved data' }));
    expect(onRetryRecovery).toHaveBeenCalledOnce();
    expect(onRetryFreshLink).toHaveBeenCalledOnce();
    expect(onDiscard).toHaveBeenCalledOnce();

    await rerender({ open: true, review: { ...review, canStartOver: false, canRetryRecovery: false, canRetryFreshLink: false, canDiscard: false, canRemove: false, canRetryCleanup: true }, onRetryRecovery, onRetryFreshLink, onDiscard, onRetryCleanup });
    await user.click(screen.getByRole('button', { name: 'Retry cleanup' }));
    expect(onRetryCleanup).toHaveBeenCalledOnce();
    expect(screen.queryByRole('button', { name: 'Discard saved data' })).not.toBeInTheDocument();
  });

  test('empty inspect review offers Retry this download and Remove only', async () => {
    const onRetryFreshLink = vi.fn();
    const onRemove = vi.fn();
    const user = userEvent.setup();
    render(ActionRequiredReviewDialog, {
      props: {
        open: true,
        review: {
          ...review,
          heading: 'Download could not start',
          message: 'Nothing was saved.',
          preservationNotice: '',
          canStartOver: false,
          canRetryRecovery: false,
          canRetryFreshLink: true,
          canDiscard: false,
          canRemove: true,
          canRetryCleanup: false,
        },
        onRetryFreshLink,
        onRemove,
      },
    });
    expect(screen.queryByText('What happens to the saved data?')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Start over from Home' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Try recovery again' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Discard saved data' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Retry with fresh link' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(onRetryFreshLink).toHaveBeenCalledOnce();
    await user.click(screen.getByRole('button', { name: 'Remove' }));
    expect(onRemove).toHaveBeenCalledOnce();
  });
});

describe('destination conflict authority', () => {
  test('emits only the opaque backend token for both choices', async () => {
    const onUseNewName = vi.fn<(event: CustomEvent<DestinationConflictEventDetail>) => void>();
    const onCancelDownload = vi.fn<(event: CustomEvent<DestinationConflictEventDetail>) => void>();
    const user = userEvent.setup();

    render(DestinationConflictDialog, {
      props: {
        open: true,
        conflict: conflictModel(),
      },
      events: { 'use-new-name': onUseNewName, 'cancel-download': onCancelDownload },
    });

    expect(screen.getByRole('dialog', { name: 'Choose a new filename' })).toBeInTheDocument();
    expect(screen.getByDisplayValue('Video (2).mp4')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Use new name' }));
    await user.click(screen.getByRole('button', { name: 'Cancel download' }));

    expect(onUseNewName).toHaveBeenCalledOnce();
    expect(onUseNewName.mock.calls[0][0].detail).toEqual({ conflictToken: 'conflict-token-7' });
    expect(onUseNewName.mock.calls[0][0].detail).not.toHaveProperty('name');
    expect(onCancelDownload.mock.calls[0][0].detail).toEqual({ conflictToken: 'conflict-token-7' });
  });

  test('disables authoritative choices when the backend token is absent', () => {
    const conflict = conflictModel() as Partial<DestinationConflictViewModel>;
    delete conflict.conflictToken;
    render(DestinationConflictDialog, {
      props: { open: true, conflict: conflict as DestinationConflictViewModel },
    });

    expect(screen.getByRole('button', { name: 'Use new name' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Cancel download' })).toBeDisabled();
  });

  test.each([
    ['null', null],
    ['non-string', 7],
    ['empty', ''],
    ['oversized UTF-8', 'é'.repeat(129)],
    ['C0 control', 'token\u0000value'],
    ['C1 control', 'token\u0085value'],
    ['lone surrogate', 'token\ud800value'],
  ])('fails closed for %s conflict authority', async (_name, token) => {
    const onUseNewName = vi.fn();
    const onCancelDownload = vi.fn();
    const conflict = { ...conflictModel(), conflictToken: token } as unknown as DestinationConflictViewModel;
    const user = userEvent.setup();
    render(DestinationConflictDialog, {
      props: { open: true, conflict },
      events: { 'use-new-name': onUseNewName, 'cancel-download': onCancelDownload },
    });

    const useNewName = screen.getByRole('button', { name: 'Use new name' });
    const cancelDownload = screen.getByRole('button', { name: 'Cancel download' });
    expect(useNewName).toBeDisabled();
    expect(cancelDownload).toBeDisabled();
    await user.click(useNewName);
    await user.click(cancelDownload);
    expect(onUseNewName).not.toHaveBeenCalled();
    expect(onCancelDownload).not.toHaveBeenCalled();
  });

  test('Escape and overlay clicks emit close while dialog clicks do not', async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    const { container } = render(DestinationConflictDialog, {
      props: { open: true, conflict: conflictModel() },
      events: { close: onClose },
    });

    await user.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalledTimes(1);

    const dialog = screen.getByRole('dialog');
    await fireEvent.click(dialog);
    expect(onClose).toHaveBeenCalledTimes(1);

    const overlay = container.querySelector('.overlay');
    expect(overlay).not.toBeNull();
    await fireEvent.click(overlay!);
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});

describe('modal keyboard focus', () => {
  test('contains focus and restores the previously focused control on unmount', async () => {
    const outside = document.createElement('button');
    outside.textContent = 'Outside';
    document.body.append(outside);
    outside.focus();
    const user = userEvent.setup();

    const rendered = render(QuitConfirmationDialog, {
      props: {
        open: true,
        model: { activeDownloads: 2, waitingOrPausedDownloads: 1 },
      },
    });
    await Promise.resolve();

    const keepWorking = screen.getByRole('button', { name: 'Keep working' });
    const close = screen.getByRole('button', { name: 'Close' });
    const quit = screen.getByRole('button', { name: 'Quit' });
    expect(keepWorking).toHaveFocus();

    outside.focus();
    expect(keepWorking).toHaveFocus();

    close.focus();
    await user.tab({ shift: true });
    expect(quit).toHaveFocus();
    await user.tab();
    expect(close).toHaveFocus();

    rendered.unmount();
    expect(outside).toHaveFocus();
    outside.remove();
  });
});

describe('restored historical attempts', () => {
  test('separates failed and canceled resolutions, hides idle controls, and preserves action authority', async () => {
    const onAction = vi.fn();
    const user = userEvent.setup();
    const { container, rerender } = render(QueueOverview, { props: {
      model: queueModel({ jobs: [
        { id: 'failed', title: 'Same video', qualityLabel: '480p', lifecycle: 'failed', occupiesSlot: false, capabilities: { retry: true }, commandToken: 'retry-token', failure: { category: 'authentication_required', messageKey: 'queue.failure.authentication_required', heading: 'Download was refused', message: 'Try again.', recommendedAction: 'Retry.', retryable: true, partialOutput: false, evidence: { stage: 'extraction', code: 'authentication', at: '2026-09-07T20:00:00Z' } } },
        { id: 'canceled', title: 'Same video', qualityLabel: '1080p60', lifecycle: 'canceled', phase: 'cleaning-up', occupiesSlot: false, capabilities: { remove: true }, commandToken: 'remove-token' },
        { id: 'done', title: 'Completed video', qualityLabel: '360p', lifecycle: 'completed', occupiesSlot: false },
      ] }), onAction,
    } });
    expect(screen.getByText('No active downloads · 1 failed · 1 canceled')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /^Needs attention/ })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^In progress/ })).not.toBeInTheDocument();
    expect(screen.getByText('480p · Failed')).toBeVisible();
    expect(screen.queryByRole('button', { name: 'Remove Same video' })).not.toBeInTheDocument();
    expect(screen.queryByText('Completed video')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Resume all' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry Same video' }));
    expect(onAction).toHaveBeenLastCalledWith({ action: 'retry', jobId: 'failed', commandToken: 'retry-token' });
    await user.click(screen.getByText('Failure details'));
    expect(screen.getByText('extraction · authentication')).toBeVisible();
    await user.click(screen.getByText('Canceled (1)'));
    expect(screen.getByText('1080p60 · Canceled')).toBeVisible();
    const canceledRow = screen.getByText('1080p60 · Canceled').closest('.qrow')!;
    expect(canceledRow.querySelector('.qbar')).toBeNull();
    await user.click(canceledRow);
    expect(screen.getByText('Temporary data removed.')).toBeVisible();
    expect(container.querySelector('.qinsp .istats')).toBeNull();
    expect(container.querySelector('.qinsp .ibar')).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Remove Same video' }));
    expect(onAction).toHaveBeenLastCalledWith({ action: 'remove', jobId: 'canceled', commandToken: 'remove-token' });
    await rerender({ model: queueModel({ jobs: [{ id: 'failed', title: 'Same video', qualityLabel: '480p', lifecycle: 'pending', occupiesSlot: false, capabilities: { pause: true }, commandToken: 'new-token' }] }), onAction });
    expect(screen.getByRole('heading', { name: /^In progress/ })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /^Needs attention/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Retry Same video' })).not.toBeInTheDocument();
  });
});
