<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { LifecycleJobActionEvent } from './LifecycleJobRow.svelte';
  import PageEmpty from '../components/PageEmpty.svelte';
  import { queueDisplayItems, type QueueDisplayItem } from '../queue-view.js';
  import {
    isValidCommandToken,
    lifecycleLabel,
    lifecycleMessage,
    type LifecycleJobAction,
    type LifecycleJobEventDetail,
    type LifecycleJobViewModel,
    type QueueCollectionAction,
    type QueueCollectionActionEvent,
    type QueueCollectionViewModel,
    type QueueOverviewViewModel,
  } from './types.js';

  export interface QueueOverviewEvents {
    'pause-all': void;
    'resume-all': void;
    'clear-completed': void;
    'go-home': void;
    pause: LifecycleJobEventDetail;
    cancel: LifecycleJobEventDetail;
    resume: LifecycleJobEventDetail;
    retry: LifecycleJobEventDetail;
    'download-again': LifecycleJobEventDetail;
    'start-again': LifecycleJobEventDetail;
    'open-source': LifecycleJobEventDetail;
    'copy-link': LifecycleJobEventDetail;
    review: LifecycleJobEventDetail;
    open: LifecycleJobEventDetail;
    remove: LifecycleJobEventDetail;
    discard: LifecycleJobEventDetail;
    'change-folder': LifecycleJobEventDetail;
    'collection-action': QueueCollectionActionEvent;
  }

  type Sel = { type: 'job'; id: string } | { type: 'col'; id: string };

  interface Props {
    model: QueueOverviewViewModel;
    title?: string;
    onPauseAll?: () => void;
    onResumeAll?: () => void;
    onClearCompleted?: () => void;
    onGoHome?: () => void;
    onAction?: (event: LifecycleJobActionEvent) => void;
    onCollectionAction?: (event: QueueCollectionActionEvent) => void;
  }

  let {
    model,
    title = 'Queue',
    onPauseAll,
    onResumeAll,
    onGoHome,
    onAction,
    onCollectionAction,
  }: Props = $props();

  const dispatch = createEventDispatcher<QueueOverviewEvents>();
  const hasQueueAuthority = $derived(isValidCommandToken(model.commandToken));
  const collections = $derived(model.collections ?? []);
  const displayItems = $derived.by(() => {
    const items = queueDisplayItems(model.jobs, collections).filter(isVisibleQueueItem);
    return [...items].sort((left, right) => queueRank(left) - queueRank(right));
  });
  const canResumeAll = $derived(
    hasQueueAuthority &&
      (model.jobs.some((job) => jobEnabled(job, 'resume')) ||
        collections.some((collection) => collectionEnabled(collection, 'resume'))),
  );

  let selectedOverride = $state<Sel | null>(null);
  const selected = $derived.by((): Sel | null => {
    const items = displayItems;
    const current = selectedOverride;
    const still = current && items.some((item) =>
      current.type === 'col'
        ? item.kind === 'collection' && item.collection.id === current.id
        : item.kind === 'job' && item.job.id === current.id,
    );
    if (still) return current;
    const first = items[0];
    if (!first) return null;
    return first.kind === 'collection'
      ? { type: 'col', id: first.collection.id }
      : { type: 'job', id: first.job.id };
  });

  const selectedJob = $derived(
    selected?.type === 'job'
      ? model.jobs.find((job) => job.id === selected.id)
      : undefined,
  );
  const selectedCollection = $derived(
    selected?.type === 'col'
      ? collections.find((collection) => collection.id === selected.id)
      : undefined,
  );
  const selectedChildren = $derived(
    selectedCollection ? childrenForCollection(displayItems, selectedCollection.id) : [],
  );

  function childrenForCollection(items: QueueDisplayItem[], id: string): LifecycleJobViewModel[] {
    for (const item of items) {
      if (item.kind === 'collection' && item.collection.id === id) {
        return item.children;
      }
    }
    return [];
  }

  function queueRank(item: ReturnType<typeof queueDisplayItems>[number]): number {
    if (item.kind === 'collection') return 50;
    const job = item.job;
    const phase = job.phase;
    if (job.lifecycle === 'active' && (phase === 'finalizing' || phase === 'publishing' || phase === 'ready-to-publish')) return 1;
    if (job.lifecycle === 'active' || job.lifecycle === 'canceling') return 0;
    if (job.lifecycle === 'failed' || job.lifecycle === 'action-required') return 2;
    if (job.lifecycle === 'paused' || job.lifecycle === 'pausing') return 3;
    return 4;
  }

  function isVisibleQueueItem(item: ReturnType<typeof queueDisplayItems>[number]): boolean {
    if (item.kind === 'job') return item.job.lifecycle !== 'completed';
    const collection = item.collection;
    if (item.children.some((child) => child.lifecycle !== 'completed')) return true;
    if (collection.total === 0) return true;
    return collection.completed < collection.total || collection.failed > 0 || collection.paused > 0 || collection.active > 0 || collection.pending > 0;
  }

  function jobEnabled(job: LifecycleJobViewModel, action: LifecycleJobAction | LifecycleJobActionEvent['action']): boolean {
    if (!isValidCommandToken(job.commandToken)) return false;
    const capabilities = job.capabilities;
    if (!capabilities) return false;
    if (action === 'download-again') return capabilities.downloadAgain === true;
    if (action === 'start-again') return capabilities.startAgain === true;
    if (action === 'open-source') return capabilities.openSource === true;
    if (action === 'copy-link') return capabilities.copyLink === true;
    if (action === 'change-folder') return capabilities.changeFolder === true;
    if (action === 'discard') return capabilities.discard === true;
    return capabilities[action] === true;
  }

  function collectionEnabled(collection: QueueCollectionViewModel, action: QueueCollectionAction): boolean {
    return isValidCommandToken(collection.commandToken) && collection.capabilities?.[action] === true;
  }

  function progressPct(progress?: number): number {
    if (progress === undefined) return 0;
    const value = progress > 1 ? progress : progress * 100;
    return Math.max(0, Math.min(100, Math.round(value)));
  }

  function jobTail(job: LifecycleJobViewModel): { text: string; err: boolean } {
    if (job.lifecycle === 'failed' || job.lifecycle === 'action-required') {
      return { text: job.failure?.heading ?? lifecycleLabel(job.lifecycle, job.phase), err: true };
    }
    if (job.lifecycle === 'paused' || job.lifecycle === 'pausing') return { text: 'paused', err: false };
    if (job.phase === 'finalizing' || job.phase === 'publishing') return { text: 'merging', err: false };
    if (job.lifecycle === 'pending') return { text: job.queueLabel ?? 'waiting · slot', err: false };
    const live = [job.speedLabel, job.etaLabel].filter(Boolean).join(' · ');
    if (live) return { text: live, err: false };
    if (job.progressLabel) return { text: job.progressLabel, err: false };
    return { text: lifecycleLabel(job.lifecycle, job.phase), err: false };
  }

  function collectionTail(collection: QueueCollectionViewModel, children: LifecycleJobViewModel[]): string {
    const anyActive = collection.active > 0 || children.some((child) => child.lifecycle === 'active' || child.lifecycle === 'pending');
    const speed = children.find((child) => child.speedLabel)?.speedLabel;
    const counts = `${collection.completed}/${collection.total}`;
    if (anyActive) return speed ? `${counts} · ${speed}` : counts;
    return `${counts} · paused`;
  }

  function collectionArtwork(collection: QueueCollectionViewModel, children: LifecycleJobViewModel[]): string {
    if (collection.kind !== 'playlist') return '';
    const own = collection.thumbnailUrl?.trim() ?? '';
    if (own) return own;
    return children.find((child) => child.thumbnailUrl)?.thumbnailUrl?.trim() ?? '';
  }

  function selectJob(job: LifecycleJobViewModel): void {
    selectedOverride = { type: 'job', id: job.id };
  }

  function selectCollection(collection: QueueCollectionViewModel): void {
    selectedOverride = { type: 'col', id: collection.id };
  }

  function trigger(job: LifecycleJobViewModel, action: LifecycleJobActionEvent['action']): void {
    if (!jobEnabled(job, action) || !job.commandToken) return;
    const detail = { jobId: job.id, commandToken: job.commandToken };
    dispatch(action, detail);
    onAction?.({ ...detail, action });
  }

  function triggerCollection(collection: QueueCollectionViewModel, action: QueueCollectionAction): void {
    if (!collectionEnabled(collection, action) || !collection.commandToken) return;
    const event = { collectionId: collection.id, commandToken: collection.commandToken, action };
    dispatch('collection-action', event);
    onCollectionAction?.(event);
  }

  function pauseAll(): void {
    if (!(model.canPauseAll === true && hasQueueAuthority)) return;
    dispatch('pause-all');
    onPauseAll?.();
  }

  function resumeAll(): void {
    if (!canResumeAll) return;
    dispatch('resume-all');
    onResumeAll?.();
  }

  const slotCopy = $derived(`${model.summary.occupiedSlots} of ${model.summary.slotLimit} slots in use`);

  function goHome(): void {
    dispatch('go-home');
    onGoHome?.();
  }

  function copyFailure(job: LifecycleJobViewModel): void {
    const lines = [job.failure?.heading, job.failure?.message, job.failure?.recommendedAction].filter(Boolean);
    navigator.clipboard?.writeText(lines.join('\n')).catch(() => undefined);
  }

  function onRowKey(event: KeyboardEvent, select: () => void, space?: () => void): void {
    if (event.key === 'Enter') {
      event.preventDefault();
      select();
    }
    if (event.key === ' ' && space) {
      event.preventDefault();
      space();
    }
  }

  function jobSpace(job: LifecycleJobViewModel): void {
    if (job.lifecycle === 'paused') trigger(job, 'resume');
    else trigger(job, 'pause');
  }
</script>

<section class="page queue-page" aria-labelledby="lifecycle-queue-title">
  {#if displayItems.length === 0}
    <header class="qhead">
      <div class="qident">
        <h1 id="lifecycle-queue-title">{title}</h1>
        <p class="qslots">{slotCopy}</p>
      </div>
    </header>
    <PageEmpty
      icon="queue"
      title="Nothing here yet"
      message={"Paste a link on Home.\nIn-progress downloads show up here."}
      action="Go to Home"
      onAction={goHome}
    />
  {:else}
    <div class="qwrap">
      <div class="qmaster">
        <header class="qhead">
          <div class="qident">
            <h1 id="lifecycle-queue-title">{title}</h1>
            <p class="qslots">{slotCopy}</p>
          </div>
          <div class="acts">
            <button
              type="button"
              class="qbtn sm ghost"
              disabled={!(model.canPauseAll === true && hasQueueAuthority)}
              onclick={pauseAll}
            >Pause all</button>
            <button
              type="button"
              class="qbtn sm ghost"
              disabled={!canResumeAll}
              onclick={resumeAll}
            >Resume all</button>
          </div>
        </header>

        {#if model.notice}
          <p class="notice" data-tone={model.noticeTone ?? 'info'} role="status" aria-live="polite">
            <span class="notice-icon" aria-hidden="true">i</span>
            <span>{model.notice}</span>
          </p>
        {/if}

        <div class="qlist" role="listbox" aria-label="Queue">
          {#each displayItems as item (item.key)}
            {#if item.kind === 'collection'}
              {@const sel = selected?.type === 'col' && selected.id === item.collection.id}
              {@const tail = collectionTail(item.collection, item.children)}
              {@const art = collectionArtwork(item.collection, item.children)}
              <div
                class="qrow"
                class:sel
                role="option"
                tabindex="0"
                aria-selected={sel}
                aria-label={item.collection.title}
                data-policy={item.collection.policy}
                onclick={() => selectCollection(item.collection)}
                onkeydown={(event) => onRowKey(event, () => selectCollection(item.collection))}
              >
                <div class="qth">
                  {#if art}
                    <img src={art} alt="" referrerpolicy="no-referrer" />
                  {:else}
                    <span class="cnt">{item.collection.total}</span>
                  {/if}
                </div>
                <div class="qmid">
                  <span class="qt">
                    <span class="qtitle">{item.collection.title}</span>
                    <em class="qkind">{item.collection.kind === 'batch' ? 'Batch' : 'Playlist'}</em>
                  </span>
                  <span class="qbar"><i style={`width:${progressPct(item.collection.progress)}%`}></i></span>
                </div>
                <span class="qtail">{tail}</span>
              </div>
            {:else}
              {@const sel = selected?.type === 'job' && selected.id === item.job.id}
              {@const tail = jobTail(item.job)}
              {@const failed = item.job.lifecycle === 'failed' || item.job.lifecycle === 'action-required'}
              <div
                class="qrow"
                class:sel
                class:fail={failed}
                role="option"
                tabindex="0"
                aria-selected={sel}
                aria-label={item.job.title}
                onclick={() => selectJob(item.job)}
                onkeydown={(event) => onRowKey(event, () => selectJob(item.job), () => jobSpace(item.job))}
              >
                <div class="qth">
                  {#if item.job.thumbnailUrl}
                    <img src={item.job.thumbnailUrl} alt="" referrerpolicy="no-referrer" />
                  {/if}
                </div>
                <div class="qmid">
                  <span class="qt"><span class="qtitle">{item.job.title}</span></span>
                  <span class="qbar"><i style={`width:${progressPct(item.job.progress)}%`}></i></span>
                </div>
                <span class="qtail" class:err={tail.err}>{tail.text}</span>
              </div>
            {/if}
          {/each}
        </div>
      </div>

      <aside class="qinsp">
        {#if selectedJob}
          {@const job = selectedJob}
          {@const fail = job.lifecycle === 'failed'}
          {@const paused = job.lifecycle === 'paused'}
          {@const waiting = job.lifecycle === 'pending'}
          <h3>{job.title}</h3>
          {#if job.thumbnailUrl}
            <img class="ibig" src={job.thumbnailUrl} alt="" referrerpolicy="no-referrer" />
          {/if}
          {#if job.metadata}
            <div class="imeta">{job.metadata}</div>
          {/if}
          <dl class="istats">
            <div><dt>Speed</dt><dd>{fail || !job.speedLabel ? '—' : job.speedLabel}</dd></div>
            <div><dt>ETA</dt><dd>{fail || !job.etaLabel ? '—' : job.etaLabel}</dd></div>
            <div><dt>Progress</dt><dd>{job.progressLabel ?? `${progressPct(job.progress)}%`}</dd></div>
            <div><dt>Size</dt><dd>—</dd></div>
          </dl>
          <div class="ibar"><i style={`width:${progressPct(job.progress)}%`}></i></div>
          {#if fail && (job.failure || lifecycleMessage(job))}
            <div class="ierr">
              {job.failure?.heading ?? 'Download failed'}<br>
              {job.failure?.message ?? lifecycleMessage(job)}
              {#if job.failure?.recommendedAction}<br>hint: {job.failure.recommendedAction}{/if}
            </div>
          {/if}
          <div class="ibtns">
            {#if fail}
              {#if job.capabilities?.retry !== undefined}
                <button type="button" class="qbtn pri" disabled={!jobEnabled(job, 'retry')} onclick={() => trigger(job, 'retry')}>Retry</button>
              {/if}
              {#if job.capabilities?.changeFolder}
                <button type="button" class="qbtn" disabled={!jobEnabled(job, 'change-folder')} onclick={() => trigger(job, 'change-folder')}>Change</button>
              {/if}
              {#if job.capabilities?.startAgain}
                <button type="button" class="qbtn pri" disabled={!jobEnabled(job, 'start-again')} onclick={() => trigger(job, 'start-again')}>Start again</button>
              {/if}
            {:else if paused}
              <button type="button" class="qbtn pri" disabled={!jobEnabled(job, 'resume')} onclick={() => trigger(job, 'resume')}>Resume</button>
              {#if job.capabilities?.discard}
                <button type="button" class="qbtn ghost danger" disabled={!jobEnabled(job, 'discard')} onclick={() => trigger(job, 'discard')}>Discard</button>
              {/if}
            {:else if job.lifecycle === 'action-required'}
              {#if job.capabilities?.review}
                <button type="button" class="qbtn pri" disabled={!jobEnabled(job, 'review')} onclick={() => trigger(job, 'review')}>Review</button>
              {/if}
            {:else}
              <button
                type="button"
                class="qbtn"
                class:pri={!waiting}
                disabled={!jobEnabled(job, 'pause')}
                onclick={() => trigger(job, 'pause')}
              >Pause</button>
            {/if}
            {#if job.capabilities?.cancel !== undefined || ['pending', 'active', 'pausing', 'paused', 'canceling', 'failed'].includes(job.lifecycle)}
              <button type="button" class="qbtn ghost danger" disabled={!jobEnabled(job, 'cancel')} onclick={() => trigger(job, 'cancel')}>Cancel</button>
            {/if}
            {#if fail}
              <button type="button" class="qbtn sm ghost" onclick={() => copyFailure(job)}>Copy details</button>
            {/if}
            {#if job.capabilities?.review && job.lifecycle !== 'action-required'}
              <button type="button" class="qbtn ghost" disabled={!jobEnabled(job, 'review')} onclick={() => trigger(job, 'review')}>Review</button>
            {/if}
            {#if job.capabilities?.downloadAgain}
              <button type="button" class="qbtn ghost" disabled={!jobEnabled(job, 'download-again')} onclick={() => trigger(job, 'download-again')}>Download again</button>
            {/if}
            {#if job.capabilities?.openSource}
              <button type="button" class="qbtn ghost" disabled={!jobEnabled(job, 'open-source')} onclick={() => trigger(job, 'open-source')}>Open source</button>
            {/if}
            {#if job.capabilities?.copyLink}
              <button type="button" class="qbtn ghost" disabled={!jobEnabled(job, 'copy-link')} onclick={() => trigger(job, 'copy-link')}>Copy link</button>
            {/if}
            {#if job.capabilities?.open}
              <button type="button" class="qbtn pri" disabled={!jobEnabled(job, 'open')} onclick={() => trigger(job, 'open')}>Open</button>
            {/if}
            {#if job.capabilities?.remove}
              <button type="button" class="qbtn ghost" disabled={!jobEnabled(job, 'remove')} onclick={() => trigger(job, 'remove')}>Remove</button>
            {/if}
          </div>
        {:else if selectedCollection}
          {@const collection = selectedCollection}
          {@const children = selectedChildren}
          {@const anyActive = collection.active > 0 || children.some((child) => child.lifecycle === 'active' || child.lifecycle === 'pending')}
          {@const live = children.filter((child) => child.lifecycle === 'active' || child.lifecycle === 'pausing' || child.phase === 'finalizing').slice(0, 2)}
          {@const waitingKids = children.filter((child) => child.lifecycle === 'pending').slice(0, 3)}
          {@const done = children.filter((child) => child.lifecycle === 'completed').length}
          {@const speed = children.find((child) => child.speedLabel)?.speedLabel}
          {@const art = collectionArtwork(collection, children)}
          <h3>{collection.title}</h3>
          {#if art}
            <img class="ibig" src={art} alt="" referrerpolicy="no-referrer" />
          {/if}
          <div class="imeta" id="c-meta">
            {collection.kind === 'batch' ? 'Batch' : 'Playlist'} · {collection.completed}/{collection.total} finished · {anyActive ? (speed ?? 'working') : 'paused'}
          </div>
          <div class="ibar"><i style={`width:${progressPct(collection.progress)}%`}></i></div>
          <div class="kids">
            {#if done}
              <div class="kid done"><span></span><span class="kt">✓ {done} finished</span><span></span></div>
            {/if}
            {#each live as child (child.id)}
              <div class="kid live">
                <span class="kth">
                  {#if child.thumbnailUrl}<img src={child.thumbnailUrl} alt="" referrerpolicy="no-referrer" />{/if}
                </span>
                <span class="kt">{child.title}<span class="kbar"><i style={`width:${progressPct(child.progress)}%`}></i></span></span>
                <span class="kd">{child.speedLabel ?? ''}</span>
              </div>
            {/each}
            {#each waitingKids as child (child.id)}
              <div class="kid">
                <span class="kth">
                  {#if child.thumbnailUrl}<img src={child.thumbnailUrl} alt="" referrerpolicy="no-referrer" />{/if}
                </span>
                <span class="kt">{child.title}</span>
                <span class="kd">waiting</span>
              </div>
            {/each}
          </div>
          <div class="ibtns">
            <button
              type="button"
              class="qbtn"
              class:pri={!anyActive}
              disabled={!collectionEnabled(collection, anyActive ? 'pause' : 'resume')}
              onclick={() => triggerCollection(collection, anyActive ? 'pause' : 'resume')}
            >{anyActive ? 'Pause collection' : 'Resume collection'}</button>
            {#if collection.capabilities?.retry}
              <button type="button" class="qbtn ghost" disabled={!collectionEnabled(collection, 'retry')} onclick={() => triggerCollection(collection, 'retry')}>Retry failed</button>
            {/if}
            <button type="button" class="qbtn ghost danger" disabled={!collectionEnabled(collection, 'cancel')} onclick={() => triggerCollection(collection, 'cancel')}>Cancel</button>
            {#if collection.capabilities?.remove}
              <button type="button" class="qbtn ghost" disabled={!collectionEnabled(collection, 'remove')} onclick={() => triggerCollection(collection, 'remove')}>Remove</button>
            {/if}
          </div>
        {:else}
          <div class="imeta">Select a job to see its details.</div>
        {/if}
      </aside>
    </div>
  {/if}
</section>

<style>
  .queue-page {
    position: absolute;
    inset: 0;
    padding: 0;
    gap: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .qwrap {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 300px;
    height: 100%;
    min-height: 0;
  }
  .qmaster {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
  }
  .qhead {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    flex-shrink: 0;
    min-height: calc(var(--page-pad-y) + 28px + 16px);
    padding: var(--page-pad-y) var(--page-pad-x) 16px;
  }
  .qident {
    display: flex;
    min-width: 0;
    align-items: baseline;
    gap: 10px;
  }
  .qhead h1 {
    margin: 0;
    font-size: 17px;
    font-weight: 650;
    letter-spacing: -0.02em;
  }
  .qslots {
    margin: 0;
    color: var(--text-secondary);
    font-size: 12px;
    font-weight: 500;
    font-variant-numeric: tabular-nums;
  }
  .acts { display: flex; gap: 6px; }
  .qlist {
    flex: 1;
    overflow-y: auto;
    padding: 0 calc(var(--page-pad-x) - 8px) 16px;
  }
  .qrow {
    display: grid;
    grid-template-columns: 34px minmax(0, 1fr) 150px;
    gap: 12px;
    align-items: center;
    height: 36px;
    padding: 0 8px;
    border-radius: 7px;
    cursor: pointer;
    transition: background-color 120ms ease;
  }
  .qrow:hover { background: #131316; }
  .qrow.sel { background: var(--surface-raised); }
  .qrow.fail { box-shadow: inset 2px 0 0 var(--status-danger); }
  .qth {
    width: 34px;
    height: 22px;
    border-radius: 3px;
    overflow: hidden;
    background: #152033;
    display: grid;
    place-items: center;
  }
  .qth img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .cnt { color: #93C5FD; font-size: 10px; font-weight: 700; }
  .qmid { min-width: 0; }
  .qt {
    display: flex;
    align-items: baseline;
    gap: 6px;
    min-width: 0;
    font-size: 12.5px;
    font-weight: 500;
  }
  .qtitle {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .qkind {
    flex-shrink: 0;
    color: #93C5FD;
    font-size: 10px;
    font-style: normal;
    font-weight: 650;
    letter-spacing: 0.04em;
  }
  .qbar {
    display: block;
    height: 3px;
    margin-top: 4px;
    border-radius: 99px;
    background: #26262C;
    overflow: hidden;
  }
  .qbar i { display: block; height: 100%; background: var(--accent-500); }
  .qtail {
    text-align: right;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--text-secondary);
    white-space: nowrap;
  }
  .qtail.err { color: #FCA5A5; }

  .qinsp {
    border-left: 1px solid var(--border-default);
    background: #0D0D0F;
    padding: var(--page-pad-y) var(--page-pad-x) 18px;
    overflow-y: auto;
  }
  .qinsp h3 { margin: 0; font-size: 14px; font-weight: 600; line-height: 1.4; }
  .ibig {
    width: 100%;
    aspect-ratio: 16 / 9;
    border-radius: 8px;
    object-fit: cover;
    display: block;
    margin: 12px 0;
  }
  .imeta { color: var(--text-secondary); font-size: 12px; margin-bottom: 12px; }
  .istats { display: grid; grid-template-columns: 1fr 1fr; gap: 10px 14px; margin: 0 0 12px; }
  .istats dt { color: var(--text-muted); font-size: 11px; }
  .istats dd { margin: 2px 0 0; font-size: 12.5px; font-family: var(--font-mono); }
  .ibar {
    height: 5px;
    border-radius: 99px;
    background: #26262C;
    overflow: hidden;
    margin-bottom: 12px;
  }
  .ibar i { display: block; height: 100%; background: var(--accent-500); }
  .ierr {
    margin: 12px 0;
    padding: 10px 12px;
    border: 1px solid rgba(239, 68, 68, 0.4);
    border-radius: 8px;
    background: rgba(239, 68, 68, 0.05);
    font-family: var(--font-mono);
    font-size: 11px;
    color: #FCA5A5;
    line-height: 1.7;
  }
  .ibtns { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 12px; }
  .kids { display: flex; flex-direction: column; gap: 2px; margin: 8px 0 4px; }
  .kid {
    display: grid;
    grid-template-columns: 34px minmax(0, 1fr) auto;
    gap: 9px;
    align-items: center;
    padding: 5px 8px;
    border-radius: 6px;
    font-size: 12px;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .kt { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .kth {
    width: 34px;
    height: 22px;
    border-radius: 3px;
    overflow: hidden;
    background: #152033;
  }
  .kth img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .kd { font-family: var(--font-mono); font-size: 10.5px; color: var(--text-muted); }
  .kid.done { color: #52525B; cursor: default; }
  .kid.done:hover { background: none; }
  .kid.live { background: var(--surface-raised); color: var(--text-primary); cursor: default; }
  .kid.live:hover { background: var(--surface-raised); }
  .kid:hover { background: var(--surface-base); }
  .kbar {
    display: block;
    height: 3px;
    margin-top: 4px;
    border-radius: 99px;
    background: #26262C;
    overflow: hidden;
  }
  .kbar i { display: block; height: 100%; background: var(--accent-500); }

  .qbtn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 28px;
    padding: 0 12px;
    border-radius: 7px;
    border: 1px solid var(--border-default);
    background: var(--surface-raised);
    color: var(--text-primary);
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 120ms ease, border-color 120ms ease;
  }
  .qbtn:hover:not(:disabled) { background: var(--surface-hover); }
  .qbtn:disabled { opacity: 0.4; cursor: default; }
  .qbtn.pri { background: var(--accent-600); border-color: var(--accent-600); color: #fff; }
  .qbtn.pri:hover:not(:disabled) { background: #1D4ED8; }
  .qbtn.ghost { background: transparent; }
  .qbtn.danger { color: #FCA5A5; }
  .qbtn.sm { height: 24px; padding: 0 9px; font-size: 11px; border-radius: 6px; }

  .notice {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    margin: 0 18px 8px;
    padding: var(--sp-3) var(--sp-4);
    border: 1px solid var(--accent-400);
    border-radius: var(--r-md);
    color: var(--accent-600);
    background: var(--accent-soft);
    font-size: var(--fs-sm);
  }
  .notice[data-tone='warning'] {
    border-color: rgba(176, 118, 7, 0.4);
    color: var(--status-warning);
    background: var(--status-warning-soft);
  }
  .notice-icon {
    display: grid;
    place-content: center;
    width: 18px;
    height: 18px;
    flex: 0 0 auto;
    border: 1.5px solid currentColor;
    border-radius: 50%;
    font-size: var(--fs-xs);
    font-weight: 700;
  }

  @media (max-width: 760px) {
    .qwrap { grid-template-columns: 1fr; grid-template-rows: minmax(0, 1fr) minmax(220px, 40%); }
    .qinsp { border-left: 0; border-top: 1px solid var(--border-default); }
  }
</style>
