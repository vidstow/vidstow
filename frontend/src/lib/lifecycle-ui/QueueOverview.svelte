<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import LifecycleJobRow, { type LifecycleJobActionEvent } from './LifecycleJobRow.svelte';
  import CollectionRow from './CollectionRow.svelte';
  import { queueDisplayItems } from '../queue-view.js';
  import { isValidCommandToken, lifecycleLabel } from './types.js';
  import type { LifecycleJobEventDetail, QueueCollectionActionEvent, QueueOverviewViewModel } from './types.js';

  export interface QueueOverviewEvents {
    'pause-all': void;
    'clear-completed': void;
    pause: LifecycleJobEventDetail;
    cancel: LifecycleJobEventDetail;
    'collection-action': QueueCollectionActionEvent;
  }

  interface Props {
    model: QueueOverviewViewModel;
    title?: string;
    onPauseAll?: () => void;
    onClearCompleted?: () => void;
    onAction?: (event: LifecycleJobActionEvent) => void;
    onCollectionAction?: (event: QueueCollectionActionEvent) => void;
  }

  let { model, title = 'Queue', onPauseAll, onClearCompleted, onAction, onCollectionAction }: Props = $props();
  const dispatch = createEventDispatcher<QueueOverviewEvents>();
  let selectedJobId = $state<string | undefined>();
  const hasQueueAuthority = $derived(isValidCommandToken(model.commandToken));
  const collections = $derived(model.collections ?? []);
  const jobsById = $derived(new Map(model.jobs.map((job) => [job.id, job])));
  const activeJobs = $derived(model.jobs.filter((job) => job.lifecycle !== 'completed'));
  const completedJobs = $derived(model.jobs.filter((job) => job.lifecycle === 'completed'));
  const displayItems = $derived(queueDisplayItems(activeJobs, collections));
  const selectedJob = $derived(jobsById.get(selectedJobId ?? '') ?? activeJobs[0]);

  function forward(event: LifecycleJobActionEvent): void {
    dispatch(event.action, { jobId: event.jobId, commandToken: event.commandToken });
    onAction?.(event);
  }

  function forwardCollection(event: QueueCollectionActionEvent): void {
    dispatch('collection-action', event);
    onCollectionAction?.(event);
  }

  function pauseAll(): void {
    if (!(model.canPauseAll === true && hasQueueAuthority)) return;
    dispatch('pause-all');
    onPauseAll?.();
  }

  function clearCompleted(): void {
    if (!(model.canClearCompleted === true && hasQueueAuthority)) return;
    dispatch('clear-completed');
    onClearCompleted?.();
  }

  function inspectorAction(action: 'pause' | 'cancel'): void {
    if (!selectedJob || !isValidCommandToken(selectedJob.commandToken) || selectedJob.capabilities?.[action] !== true) return;
    forward({ action, jobId: selectedJob.id, commandToken: selectedJob.commandToken });
  }
</script>

<section class="queue-shell" aria-labelledby="lifecycle-queue-title">
  <div class="queue-master">
    <header class="queue-header">
      <h1 id="lifecycle-queue-title">{title}</h1>
      <div class="header-actions">
        <button type="button" class="tool-button" aria-label="Pause all" disabled={!(model.canPauseAll === true && hasQueueAuthority)} onclick={pauseAll}>
          <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M4.5 3v10M11.5 3v10" /></svg><span>Pause all</span>
        </button>
        <button type="button" class="tool-button" aria-label="Clear completed" disabled={!(model.canClearCompleted === true && hasQueueAuthority)} onclick={clearCompleted}>
          <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M2 4h1M5.5 4h8M2 8h1M5.5 8h4.5M2 12h1M5.5 12h8M11.5 8h2.5" /></svg><span>Clear completed</span>
        </button>
      </div>
    </header>

    {#if model.notice}
      <div class="notice" data-tone={model.noticeTone ?? 'info'} role="status"><span aria-hidden="true">i</span>{model.notice}</div>
    {/if}

    <div class="queue-list">
      {#if displayItems.length}
        {#each displayItems as item, index (item.key)}
          {#if item.kind === 'collection'}
            <CollectionRow collection={item.collection} children={item.children} selectedJobId={selectedJob?.id} onSelect={(id) => selectedJobId = id} onAction={forward} onCollectionAction={forwardCollection} />
          {:else}
            <ul class="standalone-list" aria-label="Queue items">
              <LifecycleJobRow job={item.job} index={index + 1} selected={selectedJob?.id === item.job.id} onSelect={(id) => selectedJobId = id} onAction={forward} />
            </ul>
          {/if}
        {/each}
      {:else if completedJobs.length === 0}
        <div class="empty" role="status">Nothing in the queue.</div>
      {/if}

      {#if completedJobs.length > 0}
        <div class="completed-strip">Completed · {completedJobs.length}</div>
      {/if}
    </div>
  </div>

  <aside class="inspector" aria-label="Queue item details">
    {#if selectedJob}
      <div class="inspector-content">
        <h2>{selectedJob.title}</h2>
        <dl>
          <div><dt>Status</dt><dd>{lifecycleLabel(selectedJob.lifecycle, selectedJob.phase)}</dd></div>
          {#if selectedJob.progress !== undefined}<div><dt>Progress</dt><dd>{selectedJob.progressLabel ?? `${Math.max(0, Math.min(100, Math.round(selectedJob.progress * 100)))}%`}</dd></div>{/if}
          {#if selectedJob.speedLabel}<div><dt>Speed</dt><dd>{selectedJob.speedLabel}</dd></div>{/if}
          {#if selectedJob.etaLabel}<div><dt>ETA</dt><dd>{selectedJob.etaLabel}</dd></div>{/if}
          {#if selectedJob.metadata}<div><dt>Format</dt><dd>{selectedJob.metadata}</dd></div>{/if}
        </dl>
      </div>
      <div class="inspector-actions">
        {#if selectedJob.capabilities?.pause}
          <button type="button" class="app-btn" disabled={!isValidCommandToken(selectedJob.commandToken)} onclick={() => inspectorAction('pause')}>
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M4.5 3v10M11.5 3v10" /></svg>Pause
          </button>
        {/if}
        {#if selectedJob.capabilities?.cancel}
          <button type="button" class="app-btn" disabled={!isValidCommandToken(selectedJob.commandToken)} onclick={() => inspectorAction('cancel')}>
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m4 4 8 8m0-8-8 8" /></svg>Cancel
          </button>
        {/if}
      </div>
    {:else}
      <p class="inspector-empty">Select a queue item to see its details.</p>
    {/if}
  </aside>
</section>

<style>
  .queue-shell { display: grid; width: 100%; height: 100%; min-height: 0; grid-template-columns: minmax(430px, 2fr) minmax(230px, 1fr); overflow: hidden; background: var(--surface-bg); }
  .queue-master { min-width: 0; min-height: 0; display: flex; flex-direction: column; border-right: 1px solid var(--border-default); }
  .queue-header { display: flex; height: 32px; padding: 0 10px 0 14px; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--border-default); background: var(--surface-sunken); }
  h1 { margin: 0; font-size: var(--fs-sm); font-weight: 650; }
  .header-actions { display: grid; width: max-content; grid-template-columns: 1fr 1fr; }
  .tool-button { display: flex; height: 26px; padding: 0 10px; align-items: center; justify-content: center; gap: 7px; border: 1px solid var(--border-default); background: var(--surface-sunken); color: var(--text-secondary); font-size: var(--fs-sm); white-space: nowrap; }
  .tool-button:first-child { border-radius: var(--r-sm) 0 0 var(--r-sm); }
  .tool-button:last-child { margin-left: -1px; border-radius: 0 var(--r-sm) var(--r-sm) 0; }
  .tool-button:hover:not(:disabled) { background: var(--surface-hover); color: var(--text-primary); }
  .tool-button svg, .inspector-actions svg { width: 13px; height: 13px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
  .notice { display: flex; min-height: 30px; padding: 6px 10px; align-items: center; gap: 8px; border-bottom: 1px solid rgba(245, 158, 11, .35); background: var(--status-warning-soft); color: var(--status-warning); font-size: var(--fs-xs); }
  .notice > span { display: grid; width: 14px; height: 14px; place-items: center; border: 1px solid currentColor; border-radius: 50%; }
  .queue-list { min-height: 0; flex: 1; overflow: auto; }
  .standalone-list { margin: 0; padding: 0; list-style: none; }
  .completed-strip { height: 24px; margin: 4px 10px; padding: 0 10px; display: flex; align-items: center; border-radius: 2px; background: var(--surface-raised); color: var(--text-muted); font-size: var(--fs-xs); }
  .empty { display: grid; min-height: 120px; place-items: center; color: var(--text-muted); font-size: var(--fs-sm); }
  .inspector { position: sticky; top: 0; display: flex; min-width: 0; height: 100%; min-height: 360px; flex-direction: column; background: var(--surface-inspector); }
  .inspector-content { padding: 13px 14px; }
  .inspector h2 { margin: 0; padding-bottom: 10px; border-bottom: 1px solid var(--border-default); overflow: hidden; font-size: var(--fs-sm); font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
  dl { margin: 10px 0 0; }
  dl div { display: grid; grid-template-columns: 78px minmax(0, 1fr); min-height: 20px; align-items: start; font-size: var(--fs-xs); }
  dt { color: var(--text-muted); }
  dd { margin: 0; overflow: hidden; color: var(--text-primary); font-family: var(--font-mono); font-weight: 550; text-overflow: ellipsis; white-space: nowrap; }
  .inspector-actions { display: flex; margin-top: auto; padding: 10px 14px 14px; gap: 5px; }
  .inspector-actions :global(.app-btn) { gap: 6px; min-height: 26px; }
  .inspector-empty { margin: 14px; color: var(--text-muted); font-size: var(--fs-xs); }
  @media (max-width: 760px) {
    .queue-shell { grid-template-columns: 1fr; }
    .queue-master { border-right: 0; }
    .inspector { position: static; min-height: 180px; border-top: 1px solid var(--border-default); }
  }
</style>
