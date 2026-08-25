<script lang="ts">
  import LifecycleJobRow, { type LifecycleJobActionEvent } from './LifecycleJobRow.svelte';
  import { isValidCommandToken } from './types.js';
  import type { LifecycleJobViewModel, QueueCollectionActionEvent, QueueCollectionViewModel } from './types.js';

  interface Props {
    collection: QueueCollectionViewModel;
    children: LifecycleJobViewModel[];
    selectedJobId?: string;
    onSelect?: (jobId: string) => void;
    onAction?: (event: LifecycleJobActionEvent) => void;
    onCollectionAction?: (event: QueueCollectionActionEvent) => void;
  }

  let { collection, children, selectedJobId, onSelect, onAction, onCollectionAction }: Props = $props();
  let expanded = $state(true);
  const panelId = $derived(`collection-children-${collection.id.replace(/[^A-Za-z0-9_-]/g, '-')}`);
  const progress = $derived(Math.max(0, Math.min(100, Math.round(collection.progress * 100))));
  const collectionLabel = $derived(collection.kind === 'batch' ? 'batch' : 'playlist');

  function enabled(action: 'pause' | 'cancel'): boolean {
    return isValidCommandToken(collection.commandToken) && collection.capabilities?.[action] === true;
  }

  function trigger(action: 'pause' | 'cancel'): void {
    if (!enabled(action) || !collection.commandToken) return;
    onCollectionAction?.({ collectionId: collection.id, commandToken: collection.commandToken, action });
  }
</script>

<section class="collection" aria-labelledby={`${panelId}-title`}>
  <div class="parent-row">
    <button type="button" class="toggle" aria-expanded={expanded} aria-controls={panelId} aria-label={`${expanded ? 'Collapse' : 'Expand'} ${collection.title}`} onclick={() => expanded = !expanded}>
      <svg viewBox="0 0 12 12" aria-hidden="true" class:expanded><path d="m4 2.5 3.5 3.5L4 9.5" /></svg>
    </button>
    <div class="identity">
      <strong id={`${panelId}-title`}>{collection.title}</strong>
      <span>{collection.policy} · {collection.total} items</span>
    </div>
    <div class="progress-line">
      <div class="track" role="progressbar" aria-label={`${collection.title} progress`} aria-valuemin="0" aria-valuemax="100" aria-valuenow={progress}><span style={`width: ${progress}%`}></span></div>
      <b>{collection.progressLabel}</b>
    </div>
    <div class="actions" aria-label={`${collectionLabel} actions`}>
      {#if collection.capabilities?.pause}<button type="button" class="app-btn" disabled={!enabled('pause')} onclick={() => trigger('pause')}>Pause</button>{/if}
      {#if collection.capabilities?.cancel}<button type="button" class="app-btn" disabled={!enabled('cancel')} onclick={() => trigger('cancel')}>Cancel</button>{/if}
    </div>
  </div>
  {#if expanded}
    <ul class="children" id={panelId} aria-label={`${collection.title} items`}>
      {#each children as child, index (child.id)}
        <LifecycleJobRow job={child} index={index + 1} nested selected={selectedJobId === child.id} {onSelect} {onAction} />
      {/each}
    </ul>
  {/if}
</section>

<style>
  .collection { border-bottom: 1px solid var(--border-subtle); }
  .parent-row { display: grid; grid-template-columns: 20px minmax(0, 1fr) minmax(150px, 35%) auto; min-height: 44px; padding: 5px 10px; align-items: center; gap: 8px; background: var(--surface-base); }
  .toggle { display: grid; width: 20px; height: 20px; place-items: center; color: var(--text-muted); }
  .toggle svg { width: 11px; height: 11px; fill: none; stroke: currentColor; stroke-width: 1.5; transition: transform 120ms ease; }
  .toggle svg.expanded { transform: rotate(90deg); }
  .identity { min-width: 0; }
  .identity strong, .identity span { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .identity strong { font-size: var(--fs-sm); }
  .identity span { margin-top: 2px; color: var(--text-muted); font-size: var(--fs-xs); }
  .progress-line { display: grid; grid-template-columns: minmax(60px, 1fr) auto; align-items: center; gap: 8px; color: var(--text-muted); font-size: var(--fs-xs); }
  .progress-line b { font-weight: 550; }
  .track { height: 3px; overflow: hidden; background: var(--surface-active); }
  .track span { display: block; height: 100%; background: var(--accent-500); }
  .actions { display: flex; gap: 4px; }
  .actions :global(.app-btn) { min-height: 24px; padding: 0 7px; }
  .children { margin: 0; padding: 0; border-top: 1px solid var(--border-subtle); list-style: none; }
  @media (max-width: 760px) { .parent-row { grid-template-columns: 20px minmax(0, 1fr) auto; } .progress-line { grid-column: 2 / -1; } }
</style>
