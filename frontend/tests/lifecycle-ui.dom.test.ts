import { fireEvent, render, screen, within } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { describe, expect, test, vi } from 'vitest';

import ActionRequiredReviewDialog from '../src/lib/lifecycle-ui/ActionRequiredReviewDialog.svelte';
import DestinationConflictDialog from '../src/lib/lifecycle-ui/DestinationConflictDialog.svelte';
import LifecycleJobRow from '../src/lib/lifecycle-ui/LifecycleJobRow.svelte';
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
  test('queue-wide actions fail closed when capabilities are absent at runtime', async () => {
    const onPauseAll = vi.fn();
    const onClearCompleted = vi.fn();
    const user = userEvent.setup();
    const model = queueModel() as Partial<QueueOverviewViewModel>;
    delete model.canPauseAll;
    delete model.canClearCompleted;

    render(QueueOverview, {
      props: { model: model as QueueOverviewViewModel },
      events: { 'pause-all': onPauseAll, 'clear-completed': onClearCompleted },
    });

    const pauseAll = screen.getByRole('button', { name: 'Pause all' });
    const clearCompleted = screen.getByRole('button', { name: 'Clear completed' });
    expect(pauseAll).toBeDisabled();
    expect(clearCompleted).toBeDisabled();
    await user.click(pauseAll);
    await user.click(clearCompleted);
    expect(onPauseAll).not.toHaveBeenCalled();
    expect(onClearCompleted).not.toHaveBeenCalled();
  });

  test('playlist collections expand children and emit only authorized Pause', async () => {
    const onCollectionAction = vi.fn();
    const user = userEvent.setup();
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{
            id: 'child-1', collectionId: 'collection-1', collectionIndex: 1,
            title: 'Child video', lifecycle: 'pending', desired: 'running', occupiesSlot: false,
            capabilities: { pause: true, cancel: true }, commandToken: 'child-token',
          }],
          collections: [{
            id: 'collection-1', kind: 'playlist', title: 'Fixture playlist', metadata: 'Creator', policy: 'video:1080p',
            childJobIds: ['child-1'], total: 1, completed: 0, failed: 0, canceled: 0,
            active: 0, pending: 1, paused: 0, progress: 0, progressLabel: '0 of 1 complete',
            capabilities: { pause: true }, commandToken: 'collection-token',
          }],
        }),
        onCollectionAction,
      },
    });

    expect(screen.getByText('Fixture playlist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Child video' })).toBeInTheDocument();
    const collection = screen.getByRole('region', { name: 'Fixture playlist' });
    await user.click(within(collection).getByRole('button', { name: 'Pause' }));
    expect(onCollectionAction).toHaveBeenCalledWith({ collectionId: 'collection-1', commandToken: 'collection-token', action: 'pause' });
    expect(screen.queryByRole('button', { name: /Resume|Retry|Remove/ })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Collapse Fixture playlist' }));
    expect(screen.queryByRole('button', { name: 'Child video' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Expand Fixture playlist' })).toHaveAttribute('aria-expanded', 'false');
  });

  test('batch collection parents stay compact and omit synthetic thumbnails', () => {
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

    expect(screen.getByText('Batch download · 2 videos')).toBeInTheDocument();
    expect(screen.getByText(/video:720p · 2 items/)).toBeInTheDocument();
    expect(container.querySelector('.parent-row > .thumbnail')).not.toBeInTheDocument();
  });

  test('collection controls fail closed without a valid token', async () => {
    const onCollectionAction = vi.fn();
    const user = userEvent.setup();
    render(QueueOverview, {
      props: {
        model: queueModel({
          collections: [{
            id: 'collection-1', kind: 'playlist', title: 'Fixture playlist', policy: 'audio:original', childJobIds: [],
            total: 0, completed: 0, failed: 0, canceled: 0, active: 0, pending: 0, paused: 0,
            progress: 0, progressLabel: '0 of 0 complete', capabilities: { pause: true, cancel: true }, commandToken: '',
          }],
        }),
        onCollectionAction,
      },
    });
    const pause = screen.getByRole('button', { name: 'Pause' });
    const cancel = screen.getByRole('button', { name: 'Cancel' });
    expect(pause).toBeDisabled();
    expect(cancel).toBeDisabled();
    await user.click(pause);
    await user.click(cancel);
    expect(onCollectionAction).not.toHaveBeenCalled();
  });

  test('queue-wide actions emit only when explicitly enabled', async () => {
    const onPauseAll = vi.fn();
    const onClearCompleted = vi.fn();
    const user = userEvent.setup();
    render(QueueOverview, {
      props: { model: queueModel({ canPauseAll: true, canClearCompleted: true }) },
      events: { 'pause-all': onPauseAll, 'clear-completed': onClearCompleted },
    });

    await user.click(screen.getByRole('button', { name: 'Pause all' }));
    await user.click(screen.getByRole('button', { name: 'Clear completed' }));
    expect(onPauseAll).toHaveBeenCalledOnce();
    expect(onClearCompleted).toHaveBeenCalledOnce();
  });

  test('row Pause and Cancel fail closed without positive capabilities', async () => {
    const onPause = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const onCancel = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const user = userEvent.setup();
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'job-1', title: 'Example video', lifecycle: 'active', phase: 'downloading', occupiesSlot: true,
          capabilities: { pause: false, cancel: false }, commandToken: 'job-token-7',
        },
      },
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
  });

  test('row actions echo the opaque token without deriving authority', async () => {
    const onPause = vi.fn<(event: CustomEvent<LifecycleJobEventDetail>) => void>();
    const user = userEvent.setup();
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'active-1', title: 'Active video', lifecycle: 'active', phase: 'downloading', occupiesSlot: true,
          capabilities: { pause: true, cancel: true }, commandToken: 'backend-command-token',
        },
      },
      events: { pause: onPause },
    });
    await user.click(screen.getByRole('button', { name: 'Pause download' }));
    expect(onPause.mock.calls[0][0].detail).toEqual({ jobId: 'active-1', commandToken: 'backend-command-token' });
  });

  test('non-active rows show truthful state and never synthesize recovery buttons', () => {
    render(LifecycleJobRow, {
      props: {
        job: {
          id: 'disk-1', title: 'Interrupted video', lifecycle: 'failed', progress: 0.7, occupiesSlot: false,
          failure: {
            category: 'disk_full', messageKey: 'queue.failure.disk_full', heading: 'Not enough disk space',
            message: 'VidStow could not finish writing this download.',
            recommendedAction: 'Free space, then try again.', retryable: false, partialOutput: true,
          },
          capabilities: { resume: true, retry: true, startAgain: true, review: true, remove: true },
          commandToken: 'disk-command-token',
        },
      },
    });
    expect(screen.getByText('Not enough disk space')).toBeInTheDocument();
    for (const name of ['Resume', 'Retry', 'Start again', 'Review', 'Remove download']) {
      expect(screen.queryByRole('button', { name })).not.toBeInTheDocument();
    }
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
  });

  test('completed rows collapse into the queue-only completed strip', () => {
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [{ id: 'done-1', title: 'Finished video', lifecycle: 'completed', progress: 1, occupiesSlot: false, capabilities: { open: true }, commandToken: 'done-token' }],
          canClearCompleted: true,
        }),
      },
    });
    expect(screen.getByText('Completed · 1')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Finished video' })).not.toBeInTheDocument();
  });

  test('selecting a row updates the inspector and keeps only Pause and Cancel', async () => {
    const user = userEvent.setup();
    render(QueueOverview, {
      props: {
        model: queueModel({
          jobs: [
            { id: 'first', title: 'First video', lifecycle: 'active', phase: 'downloading', progress: 0.2, occupiesSlot: true, capabilities: { pause: true, cancel: true }, commandToken: 'first-token' },
            { id: 'second', title: 'Second video', metadata: 'MP4 · 1080p', lifecycle: 'active', phase: 'downloading', progress: 0.4, speedLabel: '2 MB/s', etaLabel: '00:20', occupiesSlot: true, capabilities: { pause: true, cancel: true }, commandToken: 'second-token' },
          ],
        }),
      },
    });
    const inspector = screen.getByRole('complementary', { name: 'Queue item details' });
    expect(within(inspector).getByRole('heading', { name: 'First video' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Second video' }));
    expect(within(inspector).getByRole('heading', { name: 'Second video' })).toBeInTheDocument();
    expect(within(inspector).getByText('2 MB/s')).toBeInTheDocument();
    expect(within(inspector).getByRole('button', { name: 'Pause' })).toBeInTheDocument();
    expect(within(inspector).getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    expect(within(inspector).queryByRole('button', { name: /Resume|Retry|Remove/ })).not.toBeInTheDocument();
  });

  test('persistence-revoked queue data renders but cannot authorize controls', () => {
    render(QueueOverview, {
      props: {
        model: queueModel({
          commandToken: undefined, canPauseAll: false, canClearCompleted: false,
          jobs: [{ id: 'active-1', title: 'Active', lifecycle: 'active', occupiesSlot: true, capabilities: {}, commandToken: undefined }],
        }),
      },
    });
    expect(screen.getByRole('button', { name: 'Pause all' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Clear completed' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Pause download' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Cancel download' })).toBeDisabled();
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
    const quit = screen.getByRole('button', { name: 'Pause downloads and quit' });
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
