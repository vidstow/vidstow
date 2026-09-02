<script lang="ts">
  import PageEmpty from '../lib/components/PageEmpty.svelte';
  import { history, modal, route, showBanner, showError } from '../lib/stores.js';
  import { api } from '../lib/api.js';
  import {
    buildHistorySections,
    episodeIndex,
    episodeLabel,
    episodeSubtitle,
    historySubtitle,
    thumbnailFor,
    type HistoryCollectionRow,
    type HistoryRecord,
  } from '../lib/download-history.js';

  let query = '';
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
    try {
      await api.fs.open(entry.absolutePath);
    } catch (err) {
      showError(err, 'That file is no longer at this path');
    }
  };

  const reveal = async (entry: HistoryRecord) => {
    try {
      await api.fs.reveal(entry.absolutePath);
    } catch (err) {
      showError(err, 'That file is no longer at this path');
    }
  };

  function firstPresent(entries: HistoryRecord[]): HistoryRecord | undefined {
    return entries[0];
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

  function goHome() {
    route.set('home');
  }

  function clearSearch() {
    query = '';
  }
</script>

<section class="page downloads-page" aria-labelledby="downloads-title">
  <header class="dhead">
    <h1 id="downloads-title">Downloads</h1>
    <input
      class="dsearch"
      type="search"
      bind:value={query}
      placeholder="Search"
      aria-label="Search downloads"
    />
  </header>

  {#if sections.length}
    <div class="dlist">
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
                <button type="button" class="btn sm ghost" on:click|stopPropagation={() => revealGroup(row)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3.5 7A1.5 1.5 0 0 1 5 5.5h4l2 2h8A1.5 1.5 0 0 1 20.5 9v8A1.5 1.5 0 0 1 19 18.5H5A1.5 1.5 0 0 1 3.5 17Z"/></svg>Reveal</button>
                <button type="button" class="btn sm pri" on:click|stopPropagation={() => openGroup(row)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 5.5v13l11-6.5Z"/></svg>Open</button>
              </div>
            </div>
            {#if expanded}
              {#each row.entries as entry, index (entry.id)}
                {@const ep = episodeIndex(entry, index + 1)}
                <div class="drow sub" class:selected={selectedId === entry.id}>
                  <span class="epx">{episodeLabel(ep)}</span>
                  {#if thumbnailFor(entry)}
                    <img src={thumbnailFor(entry)} alt="" referrerpolicy="no-referrer" />
                  {:else}
                    <span class="thumb-fallback" aria-hidden="true"></span>
                  {/if}
                  <button class="copy" type="button" on:click={() => toggleDetails(entry)}>
                    <b title={entry.title}>{entry.title}</b>
                    <span>{episodeSubtitle(entry)}</span>
                  </button>
                  <div class="dact">
                    <button type="button" class="btn sm ghost" aria-label="Show in Finder" on:click={() => reveal(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3.5 7A1.5 1.5 0 0 1 5 5.5h4l2 2h8A1.5 1.5 0 0 1 20.5 9v8A1.5 1.5 0 0 1 19 18.5H5A1.5 1.5 0 0 1 3.5 17Z"/></svg>Reveal</button>
                    <button type="button" class="btn sm pri" aria-label="Open downloaded file" on:click={() => open(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 5.5v13l11-6.5Z"/></svg>Open</button>
                  </div>
                  {#if selectedId === entry.id}
                    <div class="detail">
                      <p class="path" title={entry.absolutePath}>{entry.absolutePath}</p>
                      <div class="dact">
                        <button type="button" class="btn sm ghost" aria-label="Remove from history" on:click={() => confirmRemoveHistory(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6.5 6.5l11 11"/><path d="M17.5 6.5l-11 11"/></svg>Remove</button>
                        <button type="button" class="btn sm ghost danger" aria-label="Delete downloaded file" on:click={() => confirmDeleteFile(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 7h16"/><path d="M9 7V4h6v3"/><path d="M6.5 7l1 13h9l1-13"/><path d="M10 11v5.5"/><path d="M14 11v5.5"/></svg>Delete file</button>
                      </div>
                    </div>
                  {/if}
                </div>
              {/each}
            {/if}
          {:else}
            {@const entry = row.entry}
            {@const expanded = selectedId === entry.id}
            <div class="drow" class:selected={expanded} class:open={expanded}>
              <button
                class="expand"
                type="button"
                title={expanded ? 'Hide details' : 'Show details'}
                aria-label={`${expanded ? 'Hide' : 'Show'} details for ${entry.title}`}
                aria-expanded={expanded}
                on:click={() => toggleDetails(entry)}
              >
                <span class="chev" aria-hidden="true">{expanded ? '▾' : '▸'}</span>
                {#if thumbnailFor(entry)}
                  <img src={thumbnailFor(entry)} alt="" referrerpolicy="no-referrer" />
                {:else}
                  <span class="thumb-fallback" aria-hidden="true"></span>
                {/if}
                <div class="copy">
                  <b title={entry.title}>{entry.title}</b>
                  <span>{historySubtitle(entry)}</span>
                </div>
              </button>
              <div class="dact">
                <button type="button" class="btn sm ghost" aria-label="Show in Finder" on:click={() => reveal(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3.5 7A1.5 1.5 0 0 1 5 5.5h4l2 2h8A1.5 1.5 0 0 1 20.5 9v8A1.5 1.5 0 0 1 19 18.5H5A1.5 1.5 0 0 1 3.5 17Z"/></svg>Reveal</button>
                <button type="button" class="btn sm pri" aria-label="Open downloaded file" on:click={() => open(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 5.5v13l11-6.5Z"/></svg>Open</button>
              </div>
              {#if expanded}
                <div class="detail">
                  <p class="path" title={entry.absolutePath}>{entry.absolutePath}</p>
                  <div class="dact">
                    <button type="button" class="btn sm ghost" aria-label="Remove from history" on:click={() => confirmRemoveHistory(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6.5 6.5l11 11"/><path d="M17.5 6.5l-11 11"/></svg>Remove</button>
                    <button type="button" class="btn sm ghost danger" aria-label="Delete downloaded file" on:click={() => confirmDeleteFile(entry)}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 7h16"/><path d="M9 7V4h6v3"/><path d="M6.5 7l1 13h9l1-13"/><path d="M10 11v5.5"/><path d="M14 11v5.5"/></svg>Delete file</button>
                  </div>
                </div>
              {/if}
            </div>
          {/if}
        {/each}
      {/each}
    </div>
  {:else if $history.length && needle}
    <PageEmpty
      icon="search"
      title="No matching downloads"
      message="Try a different title, channel, or filename."
      action="Clear search"
      actionKind="ghost"
      onAction={clearSearch}
    />
  {:else}
    <PageEmpty
      icon="downloads"
      title="Nothing here yet"
      message={"Paste a link on Home.\nFinished files show up here."}
      action="Go to Home"
      onAction={goHome}
    />
  {/if}
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
    flex-shrink: 0;
    min-height: calc(var(--page-pad-y) + 28px + 16px);
    padding: var(--page-pad-y) var(--page-pad-x) 16px;
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
    padding: 0 calc(var(--page-pad-x) - 8px) 16px;
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
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
    min-height: 46px;
    padding: 0 8px;
    border-radius: 7px;
  }
  .drow:hover { background: #131316; }
  .drow.selected { background: var(--surface-raised); }
  .drow.open .chev { color: var(--text-secondary); }
  .drow img, .thumb-fallback {
    width: 48px;
    height: 28px;
    border-radius: 4px;
    object-fit: cover;
    display: block;
    background: var(--surface-base);
  }
  .expand {
    display: grid;
    grid-template-columns: 16px 48px minmax(0, 1fr);
    gap: 12px;
    min-width: 0;
    align-items: center;
    text-align: left;
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
  .dact .btn:has(> svg) { display: inline-flex; align-items: center; gap: 5px; }
  .dact .btn:has(> svg) svg { width: 12px; height: 12px; flex-shrink: 0; }

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
    cursor: pointer;
    transition: background-color 120ms ease, border-color 120ms ease;
  }
  .btn:hover:not(:disabled) { background: var(--surface-hover); }
  .btn:disabled { opacity: 0.4; cursor: default; }
  .btn.ghost { background: transparent; }
  .btn.pri { background: var(--accent-600); border-color: var(--accent-600); color: #fff; }
  .btn.pri:hover:not(:disabled) { background: #1D4ED8; }
  .btn.sm { height: 24px; padding: 0 9px; font-size: 11px; border-radius: 6px; }
  .btn.danger { color: #FCA5A5; }

  .detail {
    grid-column: 1 / -1;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 0 0 10px;
  }
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
