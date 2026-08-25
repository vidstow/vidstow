<script lang="ts">
  import { api } from '../lib/api.js';
  import { modal, queueView, showBanner, showError } from '../lib/stores.js';
  import QueueOverview from '../lib/lifecycle-ui/QueueOverview.svelte';
  import { isValidCommandToken } from '../lib/lifecycle-ui/types.js';
  import type { LifecycleJobEventDetail, QueueCollectionActionEvent, QueueOverviewViewModel, QueueView } from '../lib/lifecycle-ui/types.js';
  import { newestQueueView } from '../lib/queue-view.js';

  function modelFrom(view: QueueView | null): QueueOverviewViewModel {
    const persistence = view?.persistence;
    const writable = persistence?.available === true && persistence.healthy === true;
    return {
      summary: view?.summary ?? { totalJobs: 0, runningJobs: 0, occupiedSlots: 0, slotLimit: 2, processingOccupied: 0, processingLimit: 3, waitingJobs: 0, pausedJobs: 0 },
      jobs: writable ? (view?.rows ?? []) : (view?.rows ?? []).map((row) => ({ ...row, capabilities: {}, commandToken: undefined })),
      collections: writable ? (view?.collections ?? []) : (view?.collections ?? []).map((collection) => ({ ...collection, capabilities: {}, commandToken: undefined })),
      canPauseAll: writable && view?.capabilities?.pauseAll === true,
      canClearCompleted: writable && view?.capabilities?.clearCompleted === true,
      commandToken: writable ? view?.capabilities?.commandToken : undefined,
      notice: !persistence?.available
        ? (persistence?.message || 'Download state is unavailable. Queue actions are disabled.')
        : !persistence.healthy
          ? (persistence.message || 'VidStow could not save the download queue.')
          : undefined,
      noticeTone: 'warning',
      footerText: persistence?.available && persistence.healthy ? 'Jobs are saved automatically.' : 'Queue changes require healthy saved state.',
    };
  }

  $: model = modelFrom($queueView);

  async function refresh(): Promise<QueueView> {
    const next = await api.queue.get();
    queueView.update((current) => newestQueueView(current, next));
    return next;
  }

  function authorizedJob(detail: LifecycleJobEventDetail, action: 'pause' | 'cancel'): boolean {
    const row = model.jobs.find((candidate) => candidate.id === detail.jobId);
    return row?.capabilities?.[action] === true && isValidCommandToken(row.commandToken) && row.commandToken === detail.commandToken;
  }

  async function jobAction(detail: LifecycleJobEventDetail, action: 'pause' | 'cancel') {
    if (!authorizedJob(detail, action)) return;
    try {
      await (action === 'pause' ? api.queue.pause : api.queue.cancel)(detail.jobId, detail.commandToken);
      await refresh();
    } catch (err) {
      await refresh().catch(() => undefined);
      showError(err, action === 'pause' ? 'Could not pause the download' : 'Could not cancel the download');
    }
  }

  async function collectionAction(detail: QueueCollectionActionEvent) {
    if (detail.action !== 'pause' && detail.action !== 'cancel') return;
    const collection = model.collections?.find((candidate) => candidate.id === detail.collectionId);
    if (collection?.capabilities?.[detail.action] !== true || !isValidCommandToken(collection.commandToken) || collection.commandToken !== detail.commandToken) return;
    const collectionLabel = collection?.kind === 'batch' ? 'batch' : 'playlist';
    const execute = async () => {
      const current = model.collections?.find((candidate) => candidate.id === detail.collectionId);
      if (current?.capabilities?.[detail.action] !== true || current.commandToken !== detail.commandToken || !isValidCommandToken(current.commandToken)) {
        await refresh().catch(() => undefined);
        showBanner('warning', 'The queue changed. Review the playlist or batch and try again.');
        return;
      }
      try {
        const operation = detail.action === 'pause' ? api.queue.pauseCollection : api.queue.cancelCollection;
        const count = await operation(detail.collectionId, detail.commandToken);
        await refresh();
        showBanner('info', `Updated ${count} ${collectionLabel} item${count === 1 ? '' : 's'}.`);
      } catch (err) {
        await refresh().catch(() => undefined);
        showError(err, `Could not ${detail.action} the ${collectionLabel}`);
      }
    };
    if (detail.action === 'cancel') {
      modal.set({
        kind: 'confirm',
        title: `Cancel this ${collectionLabel}?`,
        message: `${collection.title} contains ${collection.total} queue item${collection.total === 1 ? '' : 's'}. Completed files remain on disk.`,
        actions: [{ label: `Cancel ${collectionLabel === 'batch' ? 'Batch' : 'Playlist'}`, primary: true, action: execute }],
      });
      return;
    }
    await execute();
  }

  async function pauseAll() {
    if (model.canPauseAll !== true || !isValidCommandToken(model.commandToken)) return;
    try {
      const count = await api.queue.pauseAll(model.commandToken);
      await refresh();
      showBanner('info', count ? `Pause requested for ${count} job${count === 1 ? '' : 's'}.` : 'No jobs can be paused right now.');
    } catch (err) {
      await refresh().catch(() => undefined);
      showError(err, 'Could not pause the queue');
    }
  }

  async function clearCompleted() {
    if (model.canClearCompleted !== true || !isValidCommandToken(model.commandToken)) return;
    try {
      await api.queue.clearCompleted(model.commandToken);
      await refresh();
    } catch (err) {
      await refresh().catch(() => undefined);
      showError(err, 'Could not clear completed downloads');
    }
  }
</script>

<QueueOverview
  {model}
  onPauseAll={pauseAll}
  onClearCompleted={clearCompleted}
  onCollectionAction={collectionAction}
  onAction={(event) => jobAction(event, event.action)}
/>
