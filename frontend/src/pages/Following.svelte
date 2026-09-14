<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { api } from '../lib/api.js';
  import { ffmpeg, follows, modal, showBanner, showError } from '../lib/stores.js';
  import { applyFollowsView, followOutputSummary, formatLastChecked, pendingCount, REVIEW_PAGE_SIZE, word } from '../lib/follow.js';
  import OutputOptionsEditor from '../lib/components/OutputOptionsEditor.svelte';
  import type { FollowRecord, OutputOptions, Quality } from '../lib/types.js';

  const dispatch = createEventDispatcher<{ goto: 'home' | 'queue' | 'following' | 'downloads' | 'settings' | 'about' }>();
  const PLAYLIST_VIDEO_QUALITIES: Array<{ value: Quality; label: string }> = [
    { value: '1080p', label: '1080p' },
    { value: '720p', label: '720p' },
  ];
  const AUDIO_CHOICES = [
    { value: 'original', label: 'Original audio' },
    { value: '128', label: 'MP3 128' },
    { value: '192', label: 'MP3 192' },
    { value: '256', label: 'MP3 256' },
  ];

  let selectedId: string | null = null;
  let menuId: string | null = null;
  let selectedPending = new Set<string>();
  let editing = false;
  let editTab: 'video' | 'audio' = 'video';
  let editQuality: Quality = '1080p';
  let editAudio = 'original';
  let editOptions: OutputOptions = {};
  let editFolder = '';
  let confirmPage = false;
  let lastSkip: { followId: string; videoIds: string[] } | null = null;
  let busy = false;

  $: view = $follows;
  $: list = view.follows ?? [];
  $: reviewTotal = pendingCount(view);
  $: selected = list.find((item) => item.id === selectedId) ?? null;
  $: pending = selected?.pending ?? [];
  $: skipped = selected?.skipped ?? [];
  $: page = pending.slice(0, REVIEW_PAGE_SIZE);
  $: paged = pending.length > REVIEW_PAGE_SIZE;
  $: pageSelected = page.filter((item) => selectedPending.has(item.videoId) && item.available);
  $: checking = view.checking;

  $: if (selected) {
    const allowed = new Set(page.map((item) => item.videoId));
    const next = new Set([...selectedPending].filter((id) => allowed.has(id)));
    if (next.size !== selectedPending.size) selectedPending = next;
  }

  function sync(next = view) {
    follows.set(applyFollowsView(view, next));
  }

  function openReview(follow: FollowRecord) {
    selectedId = follow.id;
    menuId = null;
    editing = false;
    confirmPage = false;
    selectedPending = new Set((follow.pending ?? []).filter((item) => item.available).slice(0, REVIEW_PAGE_SIZE).map((item) => item.videoId));
  }

  function back() {
    selectedId = null;
    menuId = null;
    editing = false;
    confirmPage = false;
  }

  async function run(label: string, operation: () => Promise<unknown>) {
    if (busy) return;
    busy = true;
    try {
      const next = await operation();
      if (next && typeof next === 'object' && 'follows' in (next as object)) sync(next as typeof view);
    } catch (err) {
      showError(err, label);
      try { sync(await api.follows.list()); } catch { /* keep last view */ }
    } finally {
      busy = false;
    }
  }

  function checkOne(id: string) {
    void run("Couldn't check for updates.", () => api.follows.check(id));
  }

  function checkAll() {
    void run("Couldn't check for updates.", () => api.follows.checkAll());
  }

  function stopChecks() {
    // Stop must not wait on the in-flight Check now / Check all Wails call.
    void api.follows.stop().then((next) => sync(next)).catch((err) => showError(err, 'Could not stop the check.'));
  }

  function togglePending(id: string) {
    const next = new Set(selectedPending);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selectedPending = next;
  }

  function selectPage(all: boolean) {
    selectedPending = all
      ? new Set(page.filter((item) => item.available).map((item) => item.videoId))
      : new Set();
  }

  async function skipSelected() {
    if (!selected || !pageSelected.length) return;
    const videoIds = pageSelected.map((item) => item.videoId);
    lastSkip = { followId: selected.id, videoIds };
    await run('Could not skip those videos.', async () => {
      const next = await api.follows.skip({ followId: selected.id, videoIds });
      showBanner('info', `Skipped ${word(videoIds.length, 'video', 'videos')}.`);
      return next;
    });
  }

  async function undoSkip() {
    if (!lastSkip) return;
    const restore = lastSkip;
    lastSkip = null;
    await run('Could not restore skipped videos.', () => api.follows.restore(restore));
  }

  async function restoreOne(videoId: string) {
    if (!selected) return;
    await run('Could not restore that video.', () => api.follows.restore({ followId: selected.id, videoIds: [videoId] }));
  }

  async function downloadSelected() {
    if (!selected || !pageSelected.length) return;
    if (paged && !confirmPage) {
      confirmPage = true;
      return;
    }
    const videoIds = pageSelected.map((item) => item.videoId);
    await run('Could not add those videos to the queue.', async () => {
      const next = await api.follows.admit({ followId: selected.id, videoIds });
      confirmPage = false;
      showBanner('success', `Added ${word(videoIds.length, 'video', 'videos')} to queue`);
      dispatch('goto', 'queue');
      return next;
    });
  }

  function beginEdit(follow: FollowRecord) {
    selectedId = follow.id;
    menuId = null;
    editing = true;
    const audio = follow.output.quality === 'audio';
    editTab = audio ? 'audio' : 'video';
    editQuality = audio ? '1080p' : follow.output.quality;
    editAudio = audio && follow.output.audioBitrate ? String(follow.output.audioBitrate) : 'original';
    editOptions = { ...(follow.output.options ?? {}) };
    editFolder = follow.output.folder;
  }

  async function saveEdit() {
    if (!selected) return;
    const quality: Quality = editTab === 'audio' ? 'audio' : editQuality;
    const audioBitrate = editTab === 'audio' && editAudio !== 'original' ? Number(editAudio) : 0;
    await run('Could not save follow settings.', async () => {
      const next = await api.follows.updateOutput({
        followId: selected.id, quality, audioBitrate, options: editOptions, folder: editFolder,
      });
      editing = false;
      showBanner('info', 'Follow settings updated');
      return next;
    });
  }

  async function pickFollowFolder() {
    try {
      const path = await api.folder.pick();
      if (path) editFolder = path;
    } catch (err) {
      showError(err, 'Could not choose folder');
    }
  }

  function confirmUnfollow(follow: FollowRecord) {
    menuId = null;
    const pendingN = follow.pending?.length ?? 0;
    modal.set({
      kind: 'confirm',
      title: `Unfollow ${follow.title}?`,
      message: `This removes the playlist and its saved settings.${pendingN ? ` ${word(pendingN, 'video', 'videos')} waiting in review will leave Following along with it.` : ''} Your downloaded files and anything already queued stay right where they are.`,
      actions: [{
        label: 'Unfollow',
        primary: true,
        action: async () => {
          await run('Could not unfollow this playlist.', async () => {
            const next = await api.follows.unfollow(follow.id);
            if (selectedId === follow.id) back();
            showBanner('info', 'Unfollowed playlist');
            return next;
          });
        },
      }],
    });
  }

  function checkLine(follow: FollowRecord): string {
    if (follow.checkState === 'checking') return 'Checking…';
    if (follow.checkState === 'waiting') return 'Waiting to check';
    return `Last checked ${formatLastChecked(follow.lastCheckedAt)}`;
  }

  function hideBrokenImage(event: Event) {
    (event.currentTarget as HTMLImageElement).style.display = 'none';
  }
</script>

<svelte:window on:click={() => (menuId = null)} />

<section class="page following-page" aria-labelledby="following-title">
  {#if selected}
    <button type="button" class="back" on:click={back}>
      <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 6l-6 6 6 6"/></svg>
      Back to Following
    </button>
    <header class="detail-identity">
      <div class="art large">
        {#if selected.thumbnail}<img src={selected.thumbnail} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />{/if}
      </div>
      <div>
        <h1 id="following-title">{selected.title}</h1>
        <p>{word(pending.length, 'video', 'videos')} to review · {selected.channel || 'YouTube'} · {selected.videoCount} videos</p>
      </div>
      <div class="menu-wrap identity-menu">
        <button type="button" class="app-btn quiet more" aria-label={`Manage ${selected.title}`} on:click|stopPropagation={() => (menuId = menuId === selected.id ? null : selected.id)}>···</button>
        {#if menuId === selected.id}
          <div class="menu">
            <button type="button" on:click={() => beginEdit(selected)}>Edit download settings</button>
            <button type="button" class="danger" on:click={() => confirmUnfollow(selected)}>Unfollow playlist…</button>
          </div>
        {/if}
      </div>
    </header>

    {#if editing}
      <section class="saved-settings">
        <OutputOptionsEditor bind:value={editOptions} collectionMode={true} allowSubtitles={editTab === 'video'} ffmpegAvailable={$ffmpeg.available} on:goto-settings={() => dispatch('goto', 'settings')}>
          <svelte:fragment slot="output">
            <div class="output-controls">
              <div class="type-pills" role="radiogroup" aria-label="Output type">
                <button type="button" class:active={editTab === 'video'} on:click={() => (editTab = 'video')}>Video</button>
                <button type="button" class:active={editTab === 'audio'} on:click={() => (editTab = 'audio')}>Audio</button>
              </div>
              {#if editTab === 'video'}
                {#each PLAYLIST_VIDEO_QUALITIES as option}
                  <button type="button" class="chip" class:on={editQuality === option.value} on:click={() => (editQuality = option.value)}>{option.label}</button>
                {/each}
              {:else}
                {#each AUDIO_CHOICES as option}
                  <button type="button" class="chip" class:on={editAudio === option.value} on:click={() => (editAudio = option.value)}>{option.label}</button>
                {/each}
              {/if}
            </div>
          </svelte:fragment>
        </OutputOptionsEditor>
        <label class="folder-field">Download folder
          <span class="folder-row">
            <input type="text" value={editFolder} readonly />
            <button type="button" class="app-btn" on:click={pickFollowFolder}>Change</button>
          </span>
        </label>
        <p>This applies to new downloads going forward, including the videos still waiting in review. Anything already in the queue keeps the settings it started with.</p>
        <div class="edit-actions">
          <button type="button" class="app-btn" on:click={() => (editing = false)}>Cancel</button>
          <button type="button" class="app-btn primary" on:click={saveEdit} disabled={busy}>Save settings</button>
        </div>
      </section>
    {:else}
      <section class="saved-settings summary">
        <div>
          <strong>{followOutputSummary(selected)}</strong>
          <span class="path">{selected.output.folder}</span>
        </div>
        <button type="button" class="app-btn" on:click={() => beginEdit(selected)}>Edit</button>
      </section>
    {/if}

    {#if selected.lastCheckError}
      <div class="notice danger-notice">
        <span>The last check failed. These videos were found earlier and remain available to review.</span>
        <button type="button" class="app-btn" on:click={() => checkOne(selected.id)} disabled={checking || busy}>Retry check</button>
      </div>
    {/if}

    {#if pending.length}
      {#if paged}
        <div class="page-note"><span class="dchip good">{word(pending.length, 'video', 'videos')} to review · showing first {page.length}</span></div>
      {/if}
      {#if confirmPage}
        <div class="notice">
          <span>Queue these {pageSelected.length} shown videos? The other {pending.length - pageSelected.length} will stay put for later.</span>
          <span class="notice-actions">
            <button type="button" class="app-btn primary" on:click={downloadSelected} disabled={busy}>Confirm</button>
            <button type="button" class="app-btn" on:click={() => (confirmPage = false)}>Cancel</button>
          </span>
        </div>
      {/if}
      <div class="eplist">
        <div class="ephead">
          <span>{pageSelected.length} of {paged ? word(page.length, 'shown video', 'shown videos') : word(pending.length, 'video', 'videos')} selected</span>
          <span class="rng">
            <button type="button" on:click={() => selectPage(true)}>All available</button>
            <button type="button" on:click={() => selectPage(false)}>None</button>
          </span>
        </div>
        <div class="epscroll" role="list" aria-label="New videos">
          {#each page as item (item.videoId)}
            <button type="button" class="eprow" class:sel={selectedPending.has(item.videoId)} class:unsel={!selectedPending.has(item.videoId)} class:unavailable={!item.available} disabled={!item.available} on:click={() => item.available && togglePending(item.videoId)}>
              <span class="chk" aria-hidden="true"></span>
              <span class="eptitle"><b>{item.title}</b>{#if !item.available}<span class="epnote">Unavailable</span>{/if}</span>
              {#if item.duration}<span class="epd">{item.duration}</span>{/if}
            </button>
          {/each}
        </div>
        <div class="epcommit">
          <span>{pageSelected.length} selected · {followOutputSummary(selected)}</span>
          <div class="epacts">
            <button type="button" class="dbtn" on:click={skipSelected} disabled={!pageSelected.length || busy}>Skip {pageSelected.length} selected</button>
            <button type="button" class="dbtn pri" on:click={downloadSelected} disabled={!pageSelected.length || busy}>
              Download {paged ? word(pageSelected.length, 'shown video', 'shown videos') : word(pageSelected.length, 'video', 'videos')}
            </button>
          </div>
        </div>
      </div>
      {#if lastSkip && lastSkip.followId === selected.id}
        <p class="undo-line">Skipped {word(lastSkip.videoIds.length, 'video', 'videos')}. <button type="button" on:click={undoSkip}>Undo</button></p>
      {/if}
    {:else}
      <div class="review-empty">
        <h2>You’re caught up.</h2>
        <p>Hit Check now whenever you want to hunt for new videos in this playlist.</p>
        <button type="button" class="app-btn" on:click={() => checkOne(selected.id)} disabled={checking || busy}>Check now</button>
      </div>
    {/if}

    {#if skipped.length}
      <section class="skipped-list">
        <span class="skipped-heading">Skipped videos ({skipped.length}) — bring them back anytime</span>
        {#each skipped as item (item.videoId)}
          <div class="skipped-row">
            <span>{item.title}</span>
            <button type="button" class="app-btn quiet" on:click={() => restoreOne(item.videoId)}>Return to review</button>
          </div>
        {/each}
      </section>
    {/if}
  {:else}
    <header class="page-header follow-header">
      <div>
        <h1 id="following-title">Following</h1>
        {#if checking && view.checkTotal}
          <p>Checking {Math.min(view.checkDone + 1, view.checkTotal)} of {view.checkTotal} · Going through your playlists one at a time — anything found along the way is kept.</p>
        {:else}
          <p>{word(list.length, 'playlist', 'playlists')} · {word(reviewTotal, 'video', 'videos')} to review · Checks are manual</p>
        {/if}
      </div>
      {#if checking}
        <button type="button" class="app-btn" on:click={stopChecks}>Stop checking</button>
      {:else}
        <button type="button" class="app-btn" on:click={checkAll} disabled={!list.length || busy}>Check all</button>
      {/if}
    </header>

    {#if !list.length}
      <div class="empty-follow">
        <h2>You are not following any playlists.</h2>
        <p>Head to Home, analyze a public playlist, and choose Follow — checks are manual, and you decide which new videos actually download.</p>
        <button type="button" class="app-btn primary" on:click={() => dispatch('goto', 'home')}>Go to Home</button>
      </div>
    {:else}
      <div class="follow-list">
        {#each list as follow (follow.id)}
          {@const pendingN = follow.pending?.length ?? 0}
          <article class="follow-row">
            <div class="art">
              {#if follow.thumbnail}<img src={follow.thumbnail} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />{/if}
            </div>
            <div class="follow-copy">
              <button type="button" class="title-button" on:click={() => openReview(follow)}>
                {follow.title} <span class="tmeta">· {follow.channel || 'YouTube'} · {follow.videoCount}</span>
              </button>
              <span>
                {#if pendingN}<b>{word(pendingN, 'video', 'videos')} to review</b> · {/if}{checkLine(follow)}
                {#if follow.lastCheckError && follow.checkState !== 'checking'} <span class="dchip bad">Check failed</span>{/if}
              </span>
              <span>{followOutputSummary(follow)}</span>
            </div>
            <div class="row-actions">
              {#if pendingN}
                <button type="button" class="app-btn primary" on:click={() => openReview(follow)}>Review {pendingN} new</button>
              {/if}
              <button type="button" class="app-btn quiet" on:click={() => checkOne(follow.id)} disabled={checking || busy}>
                {follow.lastCheckError ? 'Retry check' : 'Check now'}
              </button>
              <div class="menu-wrap">
                <button type="button" class="app-btn quiet more" aria-label={`Manage ${follow.title}`} on:click|stopPropagation={() => (menuId = menuId === follow.id ? null : follow.id)}>···</button>
                {#if menuId === follow.id}
                  <div class="menu">
                    <button type="button" on:click={() => beginEdit(follow)}>Edit download settings</button>
                    <button type="button" class="danger" on:click={() => confirmUnfollow(follow)}>Unfollow playlist…</button>
                  </div>
                {/if}
              </div>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  {/if}
</section>

<style>
  .following-page { min-height: 100%; }
  .follow-header { display: flex; align-items: flex-start; gap: 16px; }
  .follow-header > div { flex: 1; }
  .back {
    display: inline-flex; align-items: center; gap: 6px; width: fit-content;
    color: var(--text-secondary); font-size: 13px; font-weight: 600;
  }
  .back:hover { color: var(--text-primary); }
  .detail-identity { display: flex; align-items: center; gap: 16px; }
  .detail-identity h1 { margin: 0; font-size: var(--fs-2xl); font-weight: 700; letter-spacing: -0.03em; }
  .detail-identity p { margin: 6px 0 0; color: var(--text-secondary); font-size: var(--fs-sm); }
  .art {
    width: 80px; aspect-ratio: 16 / 10; overflow: hidden; border: 1px solid var(--border-default);
    border-radius: var(--r-md); background: var(--surface-raised);
  }
  .art.large { width: 104px; flex: 0 0 auto; }
  .art img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .follow-list { border-top: 1px solid var(--border-default); }
  .follow-row {
    display: grid; grid-template-columns: 80px minmax(0, 1fr) auto; gap: 16px; align-items: center;
    padding: 18px 0; border-bottom: 1px solid var(--border-default);
  }
  .follow-copy { min-width: 0; display: flex; flex-direction: column; gap: 3px; color: var(--text-secondary); font-size: var(--fs-sm); }
  .follow-copy b { color: var(--accent-400); font-weight: 600; }
  .title-button { width: fit-content; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-primary); font-size: var(--fs-lg); font-weight: 650; text-align: left; }
  .title-button:hover { text-decoration: underline; }
  .tmeta { color: var(--text-muted); font-size: 12px; font-weight: 500; }
  .row-actions { display: flex; align-items: center; justify-content: flex-end; gap: 6px; }
  .more { min-width: 32px; padding: 0 8px; }
  .menu-wrap { position: relative; }
  .menu {
    position: absolute; z-index: 20; right: 0; top: calc(100% + 4px); width: 210px; padding: 5px;
    border: 1px solid var(--border-strong); border-radius: var(--r-md); background: var(--surface-raised);
    box-shadow: var(--shadow-modal);
  }
  .menu button { width: 100%; padding: 8px 9px; border-radius: var(--r-sm); text-align: left; color: var(--text-primary); }
  .menu button:hover { background: var(--surface-active); }
  .menu button.danger { color: #FCA5A5; }
  .empty-follow, .review-empty {
    flex: 1; display: grid; place-content: center; justify-items: center; text-align: center; padding: 60px 20px;
  }
  .empty-follow h2, .review-empty h2 { margin: 0; font-size: var(--fs-xl); }
  .empty-follow p, .review-empty p { max-width: 440px; margin: 9px 0 16px; color: var(--text-secondary); }
  .dchip {
    display: inline-block; padding: 1px 8px; border-radius: 99px; font-size: 11px;
    border: 1px solid var(--border-default); color: var(--text-muted); white-space: nowrap;
  }
  .dchip.bad { border-color: rgba(239, 68, 68, 0.4); color: #FCA5A5; }
  .dchip.good { border-color: rgba(59, 130, 246, 0.45); color: #93C5FD; background: var(--accent-soft); }
  .saved-settings {
    display: flex; flex-direction: column; gap: 10px; padding: 12px 14px;
    border: 1px solid var(--border-default); border-radius: 10px; background: var(--surface-raised);
  }
  .saved-settings.summary { flex-direction: row; align-items: center; justify-content: space-between; gap: 12px; }
  .saved-settings .path, .folder-field input {
    display: block; margin-top: 4px; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;
  }
  .saved-settings p { margin: 0; color: var(--text-secondary); font-size: 12px; }
  .folder-field { color: var(--text-secondary); font-size: 12px; }
  .folder-row { display: flex; align-items: center; gap: 8px; margin-top: 6px; }
  .folder-row input { flex: 1; min-width: 0; height: 32px; padding: 0 10px; border: 1px solid var(--border-default); border-radius: 7px; background: var(--surface-base); color: var(--text-primary); }
  .edit-actions { display: flex; justify-content: flex-end; gap: 8px; }
  .output-controls { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; }
  .type-pills { display: inline-flex; padding: 2px; background: var(--surface-sunken); border: 1px solid var(--border-default); border-radius: 6px; gap: 2px; }
  .type-pills button { min-height: 22px; padding: 0 8px; border-radius: 4px; font-size: 11px; font-weight: 600; color: var(--text-secondary); }
  .type-pills button.active { background: var(--surface-base); color: var(--text-primary); }
  .chip { height: 24px; padding: 0 9px; border: 1px solid var(--border-default); border-radius: 6px; background: var(--surface-base); color: var(--text-secondary); font-size: 11px; }
  .chip.on { border-color: rgba(59, 130, 246, 0.55); background: var(--accent-soft); color: #93C5FD; }
  .notice, .danger-notice {
    display: flex; align-items: center; justify-content: space-between; gap: 12px;
    padding: 10px 12px; border: 1px solid var(--border-default); border-radius: 8px; color: var(--text-secondary); font-size: 13px;
  }
  .danger-notice { border-color: rgba(239, 68, 68, 0.35); color: #FCA5A5; }
  .notice-actions { display: flex; gap: 6px; }
  .eplist { display: flex; flex-direction: column; overflow: hidden; border: 1px solid var(--border-default); border-radius: 10px; background: var(--surface-base); }
  .ephead { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 8px 12px; border-bottom: 1px solid var(--border-default); background: var(--surface-raised); color: var(--text-secondary); font-size: 11.5px; }
  .rng { display: flex; gap: 10px; }
  .rng button { color: #93C5FD; font-size: 11.5px; }
  .rng button:hover { text-decoration: underline; }
  .epscroll { max-height: 360px; overflow-y: auto; }
  .eprow { display: grid; grid-template-columns: 14px minmax(0, 1fr) auto; gap: 10px; align-items: center; width: 100%; padding: 6px 12px; color: var(--text-secondary); font-size: 12px; text-align: left; }
  .eprow + .eprow { border-top: 1px solid var(--border-default); }
  .eprow:hover:not(:disabled) { background: var(--surface-raised); }
  .chk { width: 12px; height: 12px; border: 1px solid var(--border-strong); border-radius: 3px; }
  .eprow.sel .chk { background: var(--accent-600); border-color: var(--accent-600); }
  .eprow b { overflow: hidden; color: var(--text-primary); font-size: 12px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
  .eprow .epd { color: var(--text-muted); font-family: var(--font-mono); font-size: 10.5px; }
  .eprow.unsel b, .eprow.unsel .epd { opacity: 0.5; }
  .epnote { margin-left: 6px; color: var(--text-muted); font-size: 10.5px; }
  .epcommit { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 5px 6px 5px 10px; border-top: 1px solid var(--border-strong); background: var(--surface-raised); color: var(--text-secondary); font-size: 11.5px; }
  .epacts { display: flex; align-items: center; gap: 6px; }
  .dbtn { display: inline-flex; align-items: center; height: 24px; padding: 0 9px; border: 1px solid var(--border-default); border-radius: 6px; background: var(--surface-raised); color: var(--text-primary); font-size: 11px; font-weight: 500; }
  .dbtn.pri { background: var(--accent-600); border-color: var(--accent-600); color: #fff; }
  .dbtn:disabled { opacity: 0.4; }
  .undo-line { margin: 0; color: var(--text-secondary); font-size: 12px; }
  .undo-line button { color: #93C5FD; font-weight: 600; }
  .skipped-list { display: flex; flex-direction: column; gap: 6px; }
  .skipped-heading { color: var(--text-muted); font-size: 12px; }
  .skipped-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; color: var(--text-secondary); font-size: 13px; }
  .app-btn.quiet { background: transparent; border-color: transparent; color: var(--text-secondary); }
  .app-btn.quiet:hover:not(:disabled) { color: var(--text-primary); background: var(--surface-hover); }
</style>
