<script lang="ts">
  import { history, modal, showBanner, showError } from '../lib/stores.js';
  import { api } from '../lib/api.js';
  import { formatBytes, formatRelative, qualityLabel } from '../lib/format.js';
  import { Tabs, EmptyState } from '../lib/components/ui/index.js';
  import type { HistoryEntry } from '../lib/types.js';

  const RECENT_LIMIT = 10;

  let query = '';
  let view: 'recent' | 'all' = 'recent';
  let selected: HistoryEntry | null = null;

  $: recentCount = Math.min(RECENT_LIMIT, $history.length);
  $: showRange = $history.length > RECENT_LIMIT;
  $: source = view === 'recent' && showRange ? $history.slice(0, RECENT_LIMIT) : $history;
  $: needle = query.trim().toLowerCase();
  $: filtered = source.filter((entry) =>
    [entry.title, entry.channel, entry.filename, entry.quality, entry.container || '', entry.deliveryNote || '']
      .some((value) => value.toLowerCase().includes(needle)),
  );
  $: if (selected && !filtered.some((entry) => entry.id === selected?.id)) selected = null;

  $: rangeTabs = [
    { value: 'recent', label: 'Recent', count: recentCount },
    { value: 'all', label: 'All', count: $history.length },
  ];

  function actualContainer(entry: HistoryEntry): string {
    const filename = entry.filename || entry.absolutePath.split(/[\\/]/).pop() || '';
    const match = filename.match(/\.([a-z0-9]+)$/i);
    return match?.[1]?.toLowerCase() || entry.container || '';
  }

  function formatLabel(entry: HistoryEntry): string {
    const quality = qualityLabel(entry.quality);
    const container = actualContainer(entry);
    return container ? `${quality} · ${container}` : quality;
  }

  function codecSummary(entry: HistoryEntry): string {
    return [entry.videoCodec, entry.audioCodec].filter(Boolean).join(' · ');
  }

  function thumbnailFor(entry: HistoryEntry): string {
    if (entry.thumbnail) return entry.thumbnail;
    return entry.videoId
      ? `https://i.ytimg.com/vi/${encodeURIComponent(entry.videoId)}/hqdefault.jpg`
      : '';
  }

  function toggle(entry: HistoryEntry) {
    selected = selected?.id === entry.id ? null : entry;
  }

  const open = async (entry: HistoryEntry) => {
    if (entry.fileMissing) {
      showBanner('warning', 'That downloaded file is no longer on disk.');
      return;
    }
    try {
      await api.fs.open(entry.absolutePath);
    } catch (err) {
      showError(err, 'Could not open the downloaded file');
    }
  };

  const reveal = async (entry: HistoryEntry) => {
    if (entry.fileMissing) {
      showBanner('warning', 'That downloaded file is no longer on disk.');
      return;
    }
    try {
      await api.fs.reveal(entry.absolutePath);
    } catch (err) {
      showError(err, 'Could not show the downloaded file');
    }
  };

  function confirmRemoveHistory(entry: HistoryEntry) {
    modal.set({
      kind: 'confirm',
      title: 'Remove from history?',
      message: `Remove “${entry.title}” from Downloads. The media file on disk stays untouched.`,
      actions: [
        {
          label: 'Remove from history',
          primary: true,
          action: async () => {
            try {
              await api.downloads.remove(entry.id);
              showBanner('info', 'Removed from history');
            } catch (err) {
              showError(err, 'Could not remove the history entry');
            }
          },
        },
      ],
    });
  }

  function confirmDeleteFile(entry: HistoryEntry) {
    modal.set({
      kind: 'confirm',
      title: 'Delete downloaded file?',
      message: `Permanently delete “${entry.filename || entry.title}” from disk and remove it from Downloads and the queue.`,
      actions: [
        {
          label: 'Delete file',
          primary: true,
          action: async () => {
            try {
              await api.downloads.deleteFile(entry.id);
              showBanner('info', 'Deleted downloaded file');
            } catch (err) {
              showError(err, 'Could not delete the downloaded file');
            }
          },
        },
      ],
    });
  }
</script>

<section class="page" aria-labelledby="downloads-title">
  <header class="page-header">
    <h1 id="downloads-title">Downloads</h1>
    <p>View your recently downloaded items.</p>
  </header>
  <div class="toolbar">
    <label class="search">
      <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
        <circle cx="11" cy="11" r="6.5" /><path d="M16 16l4 4" />
      </svg>
      <input type="search" bind:value={query} placeholder="Search downloads…" aria-label="Search downloads" />
    </label>
    {#if showRange}
      <Tabs options={rangeTabs} value={view} onChange={(value) => (view = value === 'all' ? 'all' : 'recent')} ariaLabel="Download history range">
        <span class="visually-hidden">Showing {view === 'recent' ? 'recent' : 'all'} downloads</span>
      </Tabs>
    {/if}
    {#if filtered.length}
        <ul class="library" aria-label="Downloaded videos">
          {#each filtered as entry (entry.id)}
            <li class="item" class:selected={selected?.id === entry.id} class:missing={entry.fileMissing}>
              <button
                class="item-main"
                type="button"
                aria-label={`${selected?.id === entry.id ? 'Hide' : 'Show'} details for ${entry.title}`}
                aria-expanded={selected?.id === entry.id}
                on:click={() => toggle(entry)}
              >
                <span class="thumb">
                  {#if thumbnailFor(entry)}
                    <img src={thumbnailFor(entry)} alt="" referrerpolicy="no-referrer" />
                  {/if}
                  {#if entry.durationLabel}
                    <span class="duration">{entry.durationLabel}</span>
                  {/if}
                </span>
                <span class="copy">
                  <strong title={entry.title}>{entry.title}</strong>
                  <span class="meta">
                    <span class="channel">{entry.channel || 'YouTube'}</span>
                    <span aria-hidden="true">·</span>
                    <span>{formatLabel(entry)}</span>
                    {#if entry.sizeBytes}
                      <span aria-hidden="true">·</span>
                      <span>{formatBytes(entry.sizeBytes)}</span>
                    {/if}
                    {#if entry.completedAt}
                      <span aria-hidden="true">·</span>
                      <span>{formatRelative(entry.completedAt)}</span>
                    {/if}
                    {#if entry.fileMissing}
                      <span class="missing-flag">File missing</span>
                    {/if}
                  </span>
                </span>
              </button>

              <div class="item-actions">
                <button
                  type="button"
                  class="app-btn primary"
                  aria-label="Open downloaded file"
                  disabled={entry.fileMissing}
                  on:click={() => open(entry)}
                >Open</button>
                <button
                  type="button"
                  class="app-btn"
                  aria-label="Show in Finder"
                  disabled={entry.fileMissing}
                  on:click={() => reveal(entry)}
                >Show in Finder</button>
              </div>

              {#if selected?.id === entry.id}
                <div class="item-detail">
                  {#if entry.fileMissing}
                    <p class="missing-note">This file is no longer on disk. You can still remove the history entry.</p>
                  {/if}
                  {#if entry.deliveryNote}
                    <p class="delivery-note">{entry.deliveryNote}</p>
                  {/if}
                  <p class="path-full" title={entry.absolutePath}>
                    {#if codecSummary(entry)}{codecSummary(entry)} · {/if}{entry.absolutePath}
                  </p>
                  <div class="detail-actions">
                    <button type="button" class="app-btn" aria-label="Remove from history" on:click={() => confirmRemoveHistory(entry)}>
                      Remove
                    </button>
                    <button
                      type="button"
                      class="app-btn danger"
                      aria-label="Delete downloaded file"
                      disabled={entry.fileMissing}
                      on:click={() => confirmDeleteFile(entry)}
                    >Delete file</button>
                  </div>
                </div>
              {/if}
            </li>
          {/each}
        </ul>
        {#if view === 'recent' && $history.length > RECENT_LIMIT && !needle}
          <button type="button" class="more" on:click={() => (view = 'all')}>
            Show all {$history.length} downloads
          </button>
        {/if}
    {:else}
      <EmptyState
        icon={needle ? 'search' : 'inbox'}
        title={needle ? 'No downloads match your search.' : 'No downloads yet.'}
        message={needle ? 'Try a title, channel, or format.' : 'Finished downloads will show up here.'}
      />
    {/if}
  </div>
</section>

<style>
  .page { min-height: 100%; padding: 0 var(--page-pad-x) var(--page-pad-bottom); gap: var(--sp-3); }
  .page-header {
    display: flex;
    height: var(--topbar-h);
    min-height: var(--topbar-h);
    margin: 0 calc(-1 * var(--page-pad-x));
    padding: 0 12px;
    align-items: center;
    gap: var(--sp-3);
    border-bottom: 1px solid var(--border-default);
    background: var(--surface-sunken);
  }
  .page-header h1 { font-size: var(--fs-lg); font-weight: 650; line-height: 1; letter-spacing: -0.015em; }
  .page-header p { margin: 0; color: var(--text-muted); }

  .toolbar { display: flex; flex-direction: column; align-items: stretch; gap: var(--sp-2); }
  .search { position: relative; display: block; width: 100%; }
  .search svg {
    position: absolute;
    left: 10px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-muted);
    pointer-events: none;
  }
  .search input { height: 32px; padding-left: 32px; background: var(--surface-sunken); }
  .toolbar :global(.tabs) { align-self: flex-start; }

  .library {
    overflow: hidden;
    list-style: none;
    margin: 0;
    padding: 0;
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    background: var(--surface-base);
  }
  .item {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: var(--sp-2) var(--sp-3);
    padding: var(--sp-2);
    border-bottom: 1px solid var(--border-default);
    background: var(--surface-base);
  }
  .item:last-child { border-bottom: 0; }
  .item:hover, .item.selected { background: var(--surface-hover); }
  .item.selected { box-shadow: inset 2px 0 var(--accent-500); }
  .item.missing { opacity: 0.82; }

  .item-main {
    display: grid;
    grid-template-columns: 80px minmax(0, 1fr);
    align-items: center;
    gap: var(--sp-3);
    min-width: 0;
    text-align: left;
  }
  .thumb {
    position: relative;
    width: 80px;
    aspect-ratio: 16 / 9;
    overflow: hidden;
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-sm);
    background: var(--surface-sunken);
  }
  .thumb img { display: block; width: 100%; height: 100%; object-fit: cover; }
  .duration {
    position: absolute;
    right: 3px;
    bottom: 3px;
    padding: 0 3px;
    border-radius: 2px;
    background: rgba(9, 9, 11, 0.88);
    color: #fff;
    font-size: 9px;
    font-weight: 600;
  }
  .copy { min-width: 0; }
  .copy strong {
    display: block;
    overflow: hidden;
    color: var(--text-primary);
    font-size: var(--fs-sm);
    font-weight: 600;
    line-height: 1.35;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 5px;
    margin-top: 3px;
    color: var(--text-secondary);
    font-size: var(--fs-xs);
  }
  .channel { color: var(--text-muted); }
  .missing-flag {
    padding: 0 5px;
    border: 1px solid rgba(245, 158, 11, 0.3);
    border-radius: var(--r-sm);
    color: var(--status-warning);
    font-size: 9px;
    font-weight: 600;
  }
  .item-actions { display: flex; align-items: center; align-self: center; gap: var(--sp-1); }
  .item-detail {
    grid-column: 1 / -1;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: var(--sp-2);
    padding: var(--sp-2) 0 0 92px;
    border-top: 1px solid var(--border-subtle);
  }
  .missing-note { grid-column: 1 / -1; margin: 0; color: var(--status-warning); font-size: var(--fs-xs); }
  .delivery-note { grid-column: 1 / -1; margin: 0; color: var(--text-secondary); font-size: var(--fs-xs); font-weight: 650; }
  .path-full {
    min-width: 0;
    margin: 0;
    overflow: hidden;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: var(--fs-xs);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .detail-actions { display: flex; flex-wrap: wrap; gap: var(--sp-1); }
  .more {
    display: block;
    width: 100%;
    min-height: 28px;
    color: var(--accent-primary);
    font-size: var(--fs-xs);
    font-weight: 600;
  }
  .more:hover { background: var(--accent-soft); border-radius: var(--r-sm); }

  @media (max-width: 700px) {
    .item { grid-template-columns: 1fr; }
    .item-actions { padding-left: 92px; }
    .item-detail { grid-template-columns: 1fr; padding-left: 0; }
  }
</style>
