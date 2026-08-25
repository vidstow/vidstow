<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import {
    lifecycleLabel,
    lifecycleMessage,
    queuePositionLabel,
    type LifecycleJobEventDetail,
    type LifecycleJobViewModel,
    isValidCommandToken,
  } from './types.js';

  export interface LifecycleJobRowEvents {
    pause: LifecycleJobEventDetail;
    cancel: LifecycleJobEventDetail;
  }

  export interface LifecycleJobActionEvent extends LifecycleJobEventDetail {
    action: 'pause' | 'cancel';
  }

  interface Props {
    job: LifecycleJobViewModel;
    index?: number;
    selected?: boolean;
    nested?: boolean;
    onSelect?: (jobId: string) => void;
    onAction?: (event: LifecycleJobActionEvent) => void;
  }

  let { job, index = 1, selected = false, nested = false, onSelect, onAction }: Props = $props();
  const dispatch = createEventDispatcher<LifecycleJobRowEvents>();
  const progress = $derived(job.progress === undefined ? undefined : Math.max(0, Math.min(100, Math.round(job.progress * 100))));
  const message = $derived(job.failure?.heading ?? lifecycleMessage(job) ?? job.queueLabel ?? queuePositionLabel(job.queuePosition));
  const showsProgress = $derived(
    progress !== undefined && ['active', 'pausing', 'canceling'].includes(job.lifecycle) && job.phase !== 'cleaning-up',
  );
  const showsControls = $derived(
    ['pending', 'active', 'pausing', 'paused', 'canceling'].includes(job.lifecycle) && job.phase !== 'cleaning-up',
  );
  const hasAuthority = $derived(isValidCommandToken(job.commandToken));

  function enabled(action: 'pause' | 'cancel'): boolean {
    return hasAuthority && job.capabilities?.[action] === true;
  }

  function trigger(event: MouseEvent, action: 'pause' | 'cancel'): void {
    event.stopPropagation();
    if (!enabled(action) || !job.commandToken) return;
    const detail = { jobId: job.id, commandToken: job.commandToken };
    dispatch(action, detail);
    onAction?.({ ...detail, action });
  }

  function select(): void { onSelect?.(job.id); }
</script>

<li class="job-row" class:selected class:nested>
  <button
    type="button"
    class="row-main"
    aria-label={job.title || `Queue item ${index}`}
    aria-current={selected ? 'true' : undefined}
    onclick={select}
  >
    {#if job.thumbnailUrl}
      <img class="thumbnail" src={job.thumbnailUrl} alt="" referrerpolicy="no-referrer" />
    {:else}
      <span class="thumbnail placeholder" aria-hidden="true"></span>
    {/if}

    <span class="identity">
      <strong title={job.title}>{job.title}</strong>
      <span>{job.metadata ?? lifecycleLabel(job.lifecycle, job.phase)}</span>
    </span>

    <span class="progress-area">
      {#if showsProgress}
        <span class="progress-top">
          <span class="track" role="progressbar" aria-label={`${job.title} progress`} aria-valuemin="0" aria-valuemax="100" aria-valuenow={progress}>
            <span style={`width: ${progress}%`}></span>
          </span>
          <b>{job.progressLabel ?? `${progress}%`}</b>
        </span>
        <span class="transfer">
          <span>{job.speedLabel ?? message ?? ''}</span>
          {#if job.etaLabel}<span>·</span><span>{job.etaLabel}</span>{/if}
        </span>
      {:else}
        <span class="state-label" title={job.failure?.message ?? message ?? ''}>{message ?? lifecycleLabel(job.lifecycle, job.phase)}</span>
      {/if}
    </span>
  </button>

  <div class="row-actions" aria-label="Job actions">
    {#if showsControls}
      <button type="button" class="icon-button" aria-label="Pause download" disabled={!enabled('pause')} onclick={(event) => trigger(event, 'pause')}>
        <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M4.5 3.25v9.5M11.5 3.25v9.5" /></svg>
      </button>
      <button type="button" class="icon-button" aria-label="Cancel download" disabled={!enabled('cancel')} onclick={(event) => trigger(event, 'cancel')}>
        <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m4 4 8 8m0-8-8 8" /></svg>
      </button>
    {/if}
  </div>
</li>

<style>
  .job-row {
    position: relative;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    min-height: 48px;
    padding: 6px 12px 6px 10px;
    align-items: center;
    gap: 10px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-bg);
  }
  .row-main {
    display: grid;
    min-width: 0;
    grid-template-columns: 36px minmax(0, 1fr) minmax(190px, 40%);
    align-items: center;
    gap: 10px;
    text-align: left;
  }
  .job-row:hover, .job-row.selected { background: var(--surface-base); }
  .job-row.selected::before { position: absolute; z-index: 3; inset: 0 auto 0 0; width: 2px; background: var(--accent-500); content: ''; }
  .job-row.nested { padding-left: 24px; background: var(--surface-sunken); }
  .row-main:focus-visible { outline-offset: 4px; }
  .job-row.nested.selected { background: var(--surface-base); }
  .thumbnail { width: 36px; height: 28px; object-fit: cover; border-radius: 2px; background: var(--surface-active); }
  .placeholder { background: var(--surface-active); }
  .identity { min-width: 0; }
  .identity strong, .identity span { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .identity strong { color: var(--text-primary); font-size: var(--fs-sm); font-weight: 650; }
  .identity span { margin-top: 2px; color: var(--text-muted); font-size: var(--fs-xs); }
  .progress-area { min-width: 0; }
  .progress-top { display: grid; grid-template-columns: minmax(70px, 1fr) 34px; align-items: center; gap: 9px; }
  .progress-top b { color: var(--text-secondary); font-size: var(--fs-xs); font-weight: 650; text-align: right; font-variant-numeric: tabular-nums; }
  .track { height: 3px; overflow: hidden; background: var(--surface-active); }
  .track span { display: block; height: 100%; background: var(--accent-500); }
  .transfer { display: flex; min-height: 15px; margin-top: 3px; justify-content: flex-end; gap: 7px; overflow: hidden; color: var(--text-muted); font-family: var(--font-mono); font-size: 9px; white-space: nowrap; }
  .state-label { overflow: hidden; color: var(--text-muted); font-size: var(--fs-xs); text-align: right; text-overflow: ellipsis; white-space: nowrap; }
  .row-actions { display: flex; gap: 4px; }
  .icon-button { display: grid; width: 24px; height: 24px; place-items: center; border: 1px solid var(--border-default); border-radius: var(--r-sm); color: var(--text-secondary); }
  .icon-button:hover:not(:disabled) { background: var(--surface-hover); color: var(--text-primary); }
  .icon-button svg { width: 13px; height: 13px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; }
  @media (max-width: 760px) {
    .row-main { grid-template-columns: 36px minmax(0, 1fr); }
    .progress-area { grid-column: 2; width: 100%; }
  }
</style>
