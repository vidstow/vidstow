<script lang="ts">
  import { history, modal, showBanner, showError } from '../lib/stores.js';
  import { api } from '../lib/api.js';
  import {
    buildHistorySections,
    episodeIndex,
    episodeLabel,
    historySubtitle,
    thumbnailFor,
    type HistoryCollectionRow,
    type HistoryRecord,
  } from '../lib/download-history.js';

  const isMac = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
  const searchPlaceholder = isMac ? 'Search · ⌘F' : 'Search · Ctrl+F';

  let query = '';
  let searchField: HTMLInputElement | null = null;
  let selectedId: string | null = null;
  let openGroups = new Set<string>();

  $: sections = buildHistorySections($history, query);
  $: needle = query.trim();
  $: openGroupList = [...openGroups];

  $: if (selectedId && !$history.some((entry) => entry.id === selectedId)) selectedId = null;

  function toggleGroup(id: string) {
    if (needle) return;
    const next = new Set(openGroups);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    openGroups = next;
  }

  function toggleDetails(entry: HistoryRecord) {
    selectedId = selectedId === entry.id ? null : entry.id;
  }

  const open = async (entry: HistoryRecord) => {
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

  const reveal = async (entry: HistoryRecord) => {
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

  function firstPresent(entries: HistoryRecord[]): HistoryRecord | undefined {
    return entries.find((entry) => !entry.fileMissing) ?? entries[0];
  }

  async function openGroup(group: HistoryCollectionRow) {
    const entry = firstPresent(group.entries);
    if (entry) await open(entry);
  }

  async function revealGroup(group: HistoryCollectionRow) {
    const entry = firstPresent(group.entries);
    if (entry) await reveal(entry);
  }

  function confirmRemoveHistory(entry: HistoryRecord) {
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

  function confirmDeleteFile(entry: HistoryRecord) {
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

  function onWindowKeydown(event: KeyboardEvent) {
    if (!(event.metaKey || event.ctrlKey) || event.altKey || event.repeat) return;
    if (event.key.toLowerCase() !== 'f') return;
    event.preventDefault();
    searchField?.focus();
    searchField?.select();
  }
</script>

<svelte:window on:keydown={onWindowKeydown} />

<section class="page downloads-page" aria-labelledby="downloads-title">
  <header class="dhead">
    <h1 id="downloads-title">Downloads</h1>
    <input
      bind:this={searchField}
      class="dsearch"
      type="search"
      bind:value={query}
      placeholder={searchPlaceholder}
      aria-label="Search downloads"
    />
  </header>

  <div class="dlist">
    {#if sections.length}
      {#each sections as section (section.key)}
        <div class="dgroup">{section.label}</div>
        {#each section.rows as row (row.kind === 'collection' ? row.id : row.entry.id)}
          {#if row.kind === 'collection'}
            {@const expanded = !!needle || openGroupList.includes(row.id)}
            <div
              class="drow grp"
              class:open={expanded}
              tabindex="0"
              role="button"
              title={expanded ? 'Hide episodes' : 'Show episodes'}
              aria-expanded={expanded}
              on:click={() => toggleGroup(row.id)}
              on:keydown={(event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); toggleGroup(row.id); } }}
            >
              <span class="chev" aria-hidden="true">{expanded ? '▾' : '▸'}</span>
              {#if row.thumbnail}
                <img src={row.thumbnail} alt="" referrerpolicy="no-referrer" />
              {:else}
                <span class="thumb-fallback" aria-hidden="true"></span>
              {/if}
              <div class="copy">
                <b title={row.title}>{row.title}</b>
                <span>Playlist · {row.entries.length} {row.entries.length === 1 ? 'episode' : 'episodes'} · {row.format} · {row.sizeLabel}</span>
              </div>
              <div class="dact">
                <button type="button" class="btn sm ghost" on:click|stopPropagation={() => revealGroup(row)}>Reveal</button>
                <button type="button" class="btn sm ghost" on:click|stopPropagation={() => openGroup(row)}>Open</button>
              </div>
            </div>
            {#if expanded}
              {#each row.entries as entry, index (entry.id)}
                {@const ep = episodeIndex(entry, index + 1)}
                <div class="drow sub" class:missing={entry.fileMissing} class:selected={selectedId === entry.id}>
                  <span class="epx">{episodeLabel(ep)}</span>
                  {#if thumbnailFor(entry)}
                    <img src={thumbnailFor(entry)} alt="" referrerpolicy="no-referrer" />
                  {:else}
                    <span class="thumb-fallback" aria-hidden="true"></span>
                  {/if}
                  <button class="copy" type="button" on:click={() => toggleDetails(entry)}>
                    <b title={entry.title}>{entry.title}</b>
                    <span>{entry.durationLabel ? `${entry.durationLabel} · ` : ''}{historySubtitle(entry)}</span>
                  </button>
                  <div class="dact">
                    <button type="button" class="btn sm ghost" aria-label="Show in Finder" disabled={entry.fileMissing} on:click={() => reveal(entry)}>Reveal</button>
                    <button type="button" class="btn sm ghost" aria-label="Open downloaded file" disabled={entry.fileMissing} on:click={() => open(entry)}>Open</button>
                  </div>
                  {#if selectedId === entry.id}
                    <div class="detail">
                      {#if entry.fileMissing}
                        <p class="missing-note">This file is no longer on disk. You can still remove the history entry.</p>
                      {/if}
                      <p class="path" title={entry.absolutePath}>{entry.absolutePath}</p>
                      <div class="dact">
                        <button type="button" class="btn sm ghost" aria-label="Remove from history" on:click={() => confirmRemoveHistory(entry)}>Remove</button>
                        <button type="button" class="btn sm ghost danger" aria-label="Delete downloaded file" disabled={entry.fileMissing} on:click={() => confirmDeleteFile(entry)}>Delete file</button>
                      </div>
                    </div>
                  {/if}
                </div>
              {/each}
            {/if}
          {:else}
            {@const entry = row.entry}
            <div class="drow" class:missing={entry.fileMissing} class:selected={selectedId === entry.id}>
              {#if thumbnailFor(entry)}
                <img src={thumbnailFor(entry)} alt="" referrerpolicy="no-referrer" />
              {:else}
                <span class="thumb-fallback" aria-hidden="true"></span>
              {/if}
              <button class="copy" type="button" aria-label={`${selectedId === entry.id ? 'Hide' : 'Show'} details for ${entry.title}`} aria-expanded={selectedId === entry.id} on:click={() => toggleDetails(entry)}>
                <b title={entry.title}>{entry.title}</b>
                <span>{historySubtitle(entry)}</span>
              </button>
              <div class="dact">
                <button type="button" class="btn sm ghost" aria-label="Show in Finder" disabled={entry.fileMissing} on:click={() => reveal(entry)}>Reveal</button>
                <button type="button" class="btn sm ghost" aria-label="Open downloaded file" disabled={entry.fileMissing} on:click={() => open(entry)}>Open</button>
              </div>
              {#if selectedId === entry.id}
                <div class="detail">
                  {#if entry.fileMissing}
                    <p class="missing-note">This file is no longer on disk. You can still remove the history entry.</p>
                  {/if}
                  <p class="path" title={entry.absolutePath}>{entry.absolutePath}</p>
                  <div class="dact">
                    <button type="button" class="btn sm ghost" aria-label="Remove from history" on:click={() => confirmRemoveHistory(entry)}>Remove</button>
                    <button type="button" class="btn sm ghost danger" aria-label="Delete downloaded file" disabled={entry.fileMissing} on:click={() => confirmDeleteFile(entry)}>Delete file</button>
                  </div>
                </div>
              {/if}
            </div>
          {/if}
        {/each}
      {/each}
    {:else}
      <div class="dgroup">{needle ? 'No matching downloads' : 'No downloads yet'}</div>
    {/if}
  </div>
</section>

<style>
  .downloads-page {
    position: absolute;
    inset: 0;
    padding: 0;
    gap: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .dhead {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    padding: 18px 18px 10px;
  }
  .dhead h1 {
    margin: 0;
    font-size: 17px;
    font-weight: 650;
    letter-spacing: -0.02em;
  }
  .downloads-page input[type="search"].dsearch {
    width: 240px;
    height: 28px;
    padding: 0 10px;
    border: 1px solid var(--border-default);
    border-radius: 7px;
    background: var(--surface-base);
    color: var(--text-primary);
    font-family: inherit;
    font-size: 12px;
    box-shadow: none;
  }
  .downloads-page input[type="search"].dsearch:focus {
    border-color: var(--accent-500);
    outline: none;
    box-shadow: none;
  }

  .dlist {
    flex: 1;
    overflow-y: auto;
    padding: 0 10px 14px;
  }
  .dgroup {
    padding: 14px 8px 6px;
    color: var(--text-muted);
    font-size: 11px;
    font-weight: 650;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .drow {
    display: grid;
    grid-template-columns: 48px minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
    min-height: 46px;
    padding: 0 8px;
    border-radius: 7px;
  }
  .drow:hover { background: #131316; }
  .drow.selected { background: var(--surface-raised); }
  .drow.missing { opacity: 0.92; }
  .drow img, .thumb-fallback {
    width: 48px;
    height: 28px;
    border-radius: 4px;
    object-fit: cover;
    display: block;
    background: var(--surface-base);
  }
  .drow.grp { grid-template-columns: 16px 48px minmax(0, 1fr) auto; cursor: pointer; }
  .drow.sub { grid-template-columns: 34px 48px minmax(0, 1fr) auto; padding-left: 14px; }
  .drow.sub b { font-weight: 450; font-size: 12px; }
  .chev {
    color: var(--text-muted);
    font-size: 10px;
    width: 16px;
    text-align: center;
    margin: 0;
  }
  .epx {
    color: #6b6b74;
    font-family: var(--font-mono);
    font-size: 10px;
    margin: 0;
  }
  .copy {
    min-width: 0;
    text-align: left;
  }
  .copy b {
    display: block;
    font-size: 12.5px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .copy span {
    display: block;
    color: var(--text-muted);
    font-size: 11px;
    margin-top: 1px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .dact { display: flex; gap: 6px; }

  .btn {
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
    transition: background-color 120ms ease, border-color 120ms ease;
  }
  .btn:hover:not(:disabled) { background: var(--surface-hover); }
  .btn:disabled { opacity: 0.4; cursor: default; }
  .btn.ghost { background: transparent; }
  .btn.sm { height: 24px; padding: 0 9px; font-size: 11px; border-radius: 6px; }
  .btn.danger { color: #FCA5A5; }

  .detail {
    grid-column: 1 / -1;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 0 0 10px;
  }
  .missing-note { margin: 0; color: var(--status-warning); font-size: 11px; }
  .path {
    margin: 0;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 11px;
  }
</style>
