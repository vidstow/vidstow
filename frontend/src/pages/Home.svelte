<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { errorMessage, ffmpeg, modal, pendingUrl, settings, showBanner } from '../lib/stores.js';
  import { formatBytes, formatViewCount, shortTitle, youtubeUrlFromText } from '../lib/format.js';
  import type { BatchAnalysisView, InfoSummary, OutputOptions, OutputPlan, PlaylistSummary, Quality, SubtitleLanguage, UrlCheckResult } from '../lib/types.js';
  import OutputOptionsEditor from '../lib/components/OutputOptionsEditor.svelte';

  const dispatch = createEventDispatcher<{ goto: 'home' | 'queue' | 'downloads' | 'settings' | 'about' }>();

  let inputMode: 'single' | 'batch' = 'single';
  let url = '';
  let analysisGeneration = 0;
  let busy = false;
  let batchText = '';
  let batchGeneration = 0;
  let batchBusy = false;
  let batchReview: BatchAnalysisView | null = null;
  let batchNow = Date.now();
  let batchTab: 'video' | 'audio' = 'video';
  let batchQuality: Quality = '1080p';
  let batchAudioChoice = 'original';
  let folder = '';
  let selectedPlanId = '';
  let search = '';
  let preview: InfoSummary | null = null;
  let playlist: PlaylistSummary | null = null;
  let selectedItems = new Set<number>();
  let tab: 'video' | 'audio' | 'info' = 'video';
  let playlistTab: 'video' | 'audio' = 'video';
  let playlistQuality: Quality = '1080p';
  let audioChoice = 'original';
  let rangeStart = '';
  let rangeEnd = '';
  let selectAllBox: HTMLInputElement | undefined;
  let linkedPlaylist: UrlCheckResult | null = null;
  let queuedVideoKey = '';
  let queuedPlaylistKey = '';
  let videoQueueBusy = false;
  let playlistQueueBusy = false;
  let videoOptions: OutputOptions = {};
  let playlistOptions: OutputOptions = {};
  const PLAYLIST_ADMIT_CAP = 500;
  const batchExpiryTimer = setInterval(() => batchNow = Date.now(), 1000);

  onMount(async () => {
    try {
      const clipboardURL = youtubeUrlFromText(await api.clipboard.getText());
      // Clipboard reads may resolve after a drop or a keystroke. Never replace work
      // that arrived while the runtime request was in flight.
      if (clipboardURL && !url.trim() && !$pendingUrl && !batchText.trim() && !preview && !playlist) {
        url = clipboardURL;
      }
    } catch {
      // Clipboard access is opportunistic (and can be denied by the OS/webview).
    }
  });
  onDestroy(() => clearInterval(batchExpiryTimer));

  $: folder = $settings.downloadFolder || folder;
  $: plans = preview?.plans ?? [];
  $: visiblePlans = tab === 'info' ? [] : plans.filter((plan) => plan.kind === tab);
  $: selectedPlan = plans.find((plan) => plan.id === selectedPlanId) ?? null;
  $: query = search.trim().toLowerCase();
  $: filteredEntries = playlist?.entries.filter((entry) => !query || entry.title.toLowerCase().includes(query)) ?? [];
  $: availableCount = playlist?.available ?? 0;
  $: playlistFirstIndex = playlist?.entries[0]?.index ?? 1;
  $: playlistLastIndex = playlist?.entries.at(-1)?.index ?? playlist?.entryCount ?? 1;
  $: allAvailableSelected = availableCount > 0 && selectedItems.size === availableCount;
  $: playlistAtCap = (playlist?.entries.length ?? 0) >= PLAYLIST_ADMIT_CAP;
  $: batchInputLineCount = batchText.split(/\r?\n/).filter((line) => line.trim()).length;
  $: batchReadyCount = batchReview?.counts.ready ?? 0;
  $: batchExpiry = batchReview?.expiresAt ? Date.parse(batchReview.expiresAt) : Number.NaN;
  $: batchTokenValid = !!batchReview?.token && Number.isFinite(batchExpiry) && batchExpiry > batchNow;
  $: batchCanStart = batchTokenValid && batchReadyCount >= 2 && !!folder && !batchBusy && (batchTab !== 'video' || $ffmpeg.available);
  $: videoRequestKey = preview && selectedPlan
    ? [preview.videoId, tab, selectedPlan.id, folder, outputOptionsIdentity(effectiveOptions(videoOptions, tab === 'video'))].join('|')
    : '';
  $: playlistSelectionKey = playlist
    ? [
        playlist.id,
        playlistTab,
        playlistTab === 'audio' ? audioChoice : playlistQuality,
        [...selectedItems].sort((a, b) => a - b).join(','),
        folder,
        outputOptionsIdentity(effectiveOptions(playlistOptions, playlistTab === 'video')),
      ].join('|')
    : '';
  $: videoJustQueued = !!videoRequestKey && queuedVideoKey === videoRequestKey;
  $: playlistJustQueued = !!playlistSelectionKey && queuedPlaylistKey === playlistSelectionKey;
  $: if (selectAllBox && playlist) {
    selectAllBox.indeterminate = selectedItems.size > 0 && selectedItems.size < availableCount;
  }

  $: if ($pendingUrl) {
    const droppedURL = $pendingUrl;
    pendingUrl.set('');
    inputMode = 'single';
    url = droppedURL;
    analyze();
  }

  const fallbackThumbnail = (videoId: string) => (videoId ? `https://i.ytimg.com/vi/${videoId}/hqdefault.jpg` : '');
  const batchReviewSummary = (review: BatchAnalysisView) => {
    const { pasted, ready, duplicate, invalid, analysisFailed } = review.counts;
    if (ready === pasted && duplicate === 0 && invalid === 0 && analysisFailed === 0) {
      return `${ready} ${ready === 1 ? 'video' : 'videos'} ready to download`;
    }
    return [
      ready ? `${ready} ready` : '',
      duplicate ? `${duplicate} duplicate` : '',
      invalid ? `${invalid} invalid` : '',
      analysisFailed ? `${analysisFailed} could not be analyzed` : '',
    ].filter(Boolean).join(' · ');
  };
  const hideBrokenImage = (event: Event) => {
    (event.currentTarget as HTMLImageElement).style.display = 'none';
  };

  function clearAnalysis() {
    preview = null;
    playlist = null;
    queuedVideoKey = '';
    queuedPlaylistKey = '';
    selectedItems = new Set();
    selectedPlanId = '';
    search = '';
    videoOptions = {};
    playlistOptions = {};
  }

  function updateURL(event: Event) {
    const nextURL = (event.currentTarget as HTMLInputElement).value;
    if (nextURL === url) return;
    url = nextURL;
    analysisGeneration += 1;
    busy = false;
    linkedPlaylist = null;
    if (preview || playlist) clearAnalysis();
  }

  function toggleBatchComposer() {
    inputMode = inputMode === 'batch' ? 'single' : 'batch';
  }

  function updateBatchText(event: Event) {
    const nextText = (event.currentTarget as HTMLTextAreaElement).value;
    if (nextText === batchText) return;
    batchText = nextText;
    batchGeneration += 1;
    batchBusy = false;
    batchReview = null;
  }

  async function analyzeBatch() {
    if (!batchText.trim() || batchBusy) return;
    const requestGeneration = ++batchGeneration;
    batchBusy = true;
    try {
      const review = await api.analyse.batch(batchText);
      if (requestGeneration !== batchGeneration) return;
      batchReview = review;
    } catch (err) {
      if (requestGeneration !== batchGeneration) return;
      modal.set({
        kind: 'error',
        title: 'Batch could not be reviewed',
        message: errorMessage(err, 'Paste between 2 and 20 individual public YouTube video or Short URLs.'),
      });
    } finally {
      if (requestGeneration === batchGeneration) batchBusy = false;
    }
  }

  function editBatchURLs() {
    batchGeneration += 1;
    batchBusy = false;
    batchReview = null;
  }

  async function enqueueBatch() {
    if (!batchReview?.token || !batchCanStart) return;
    const quality: Quality = batchTab === 'audio' ? 'audio' : batchQuality;
    const audioBitrate = batchTab === 'audio' && batchAudioChoice !== 'original' ? Number(batchAudioChoice) : 0;
    if (batchTab === 'video' && !$ffmpeg.available) {
      requireFFmpeg('FFmpeg is required to create complete video files with artwork and chapters (and subtitles when selected).');
      return;
    }
    if (audioBitrate && !$ffmpeg.available) {
      requireFFmpeg('MP3 conversion needs FFmpeg. Choose original audio or configure FFmpeg.');
      return;
    }
    batchBusy = true;
    try {
      const result = await api.jobs.startBatch({ token: batchReview.token, quality, audioBitrate });
      showBanner('success', `Added ${result.admitted} downloads to the queue`);
      batchReview = null;
      batchText = '';
      dispatch('goto', 'queue');
    } catch (err) {
      modal.set({ kind: 'error', title: 'Batch could not start', message: errorMessage(err, 'Could not add this batch to the queue.') });
    } finally {
      batchBusy = false;
    }
  }

  async function analyzeTarget(target: UrlCheckResult, requestGeneration: number) {
    if (requestGeneration !== analysisGeneration) return;
    clearAnalysis();
    if (target.kind === 'playlist') {
      const canonicalURL = target.playlistUrl!;
      const summary = await api.analyse.playlist(canonicalURL);
      if (requestGeneration !== analysisGeneration) return;
      url = canonicalURL;
      playlist = summary;
      playlistOptions = seedOutputOptions([], '', true);
      selectedItems = new Set(summary.entries.filter((entry) => entry.available).map((entry) => entry.index));
      rangeStart = summary.entries[0]?.index ? String(summary.entries[0].index) : '1';
      rangeEnd = summary.entries.at(-1)?.index ? String(summary.entries.at(-1)!.index) : String(summary.entryCount);
    } else {
      const canonicalURL = target.videoUrl!;
      const summary = await api.analyse.url(canonicalURL);
      if (requestGeneration !== analysisGeneration) return;
      url = canonicalURL;
      preview = summary;
      videoOptions = seedOutputOptions(summary.subtitles ?? [], summary.language);
      const recommended = summary.plans.find((plan) => plan.recommended) ?? summary.plans[0];
      selectedPlanId = recommended?.id ?? '';
      tab = recommended?.kind ?? 'video';
    }
  }

  // Subtitle embedding is an explicit opt-in. Artwork and chapters remain the
  // fixed complete-video policy and cannot be disabled in the frontend.
  function seedOutputOptions(languages: SubtitleLanguage[], videoLanguage = '', collectionMode = false): OutputOptions {
    const saved = $settings.outputOptions ?? {};
    const creators = languages.filter((language) => !language.auto);
    const creatorCodes = new Set(creators.map((language) => language.code.toLowerCase()));
    const automatic = languages.filter((language) => language.auto && !creatorCodes.has(language.code.toLowerCase()));
    const matchLanguage = (choices: SubtitleLanguage[], wanted: string) => {
      const normalized = wanted.trim().toLowerCase();
      if (!normalized) return undefined;
      const root = normalized.split(/[-_]/)[0];
      return choices.find((item) => item.code.toLowerCase() === normalized)
        ?? choices.find((item) => item.code.toLowerCase().split(/[-_]/)[0] === root);
    };
    const preferred = saved.subtitleLanguages?.[0] ?? '';
    const preferredTrack = matchLanguage(creators, preferred) ?? matchLanguage(automatic, preferred);
    const defaultTrack = creators.length
      ? matchLanguage(creators, videoLanguage) ?? matchLanguage(creators, 'en') ?? creators[0]
      : matchLanguage(automatic, videoLanguage) ?? matchLanguage(automatic, 'en') ?? automatic[0];
    const selectedTrack = preferredTrack ?? defaultTrack;
    const savedSubtitlesOn = saved.subtitleMode === 'embed' || saved.subtitleMode === 'sidecar';
    const subtitlesOn = savedSubtitlesOn && (collectionMode || !!selectedTrack);
    const sidecar = !!saved.subtitleSidecar || saved.subtitleMode === 'sidecar';
    return {
      ...saved,
      subtitleMode: subtitlesOn ? 'embed' : '',
      subtitleSidecar: sidecar,
      subtitleLanguages: subtitlesOn ? (collectionMode ? (preferred ? [preferred] : undefined) : [selectedTrack!.code]) : undefined,
      subtitleAutoCaptions: subtitlesOn,
      subtitleFormat: sidecar ? 'srt' : '',
      embedThumbnail: true,
      embedChapters: true,
    };
  }

  function outputOptionsIdentity(options: OutputOptions): string {
    return JSON.stringify({
      subtitleMode: options.subtitleMode ?? '',
      subtitleSidecar: !!options.subtitleSidecar,
      subtitleLanguages: options.subtitleLanguages?.slice(0, 1) ?? [],
      subtitleAutoCaptions: !!options.subtitleAutoCaptions,
      subtitleFormat: options.subtitleFormat ?? '',
      embedMetadata: !!options.embedMetadata,
      embedThumbnail: !!options.embedThumbnail,
      embedChapters: !!options.embedChapters,
    });
  }

  function effectiveOptions(options: OutputOptions, completeVideo: boolean): OutputOptions {
    if (completeVideo) {
      const subtitlesOn = options.subtitleMode === 'embed';
      return {
        ...options,
        subtitleMode: subtitlesOn ? 'embed' : '',
        subtitleLanguages: subtitlesOn ? options.subtitleLanguages?.slice(0, 1) : undefined,
        subtitleAutoCaptions: subtitlesOn,
        subtitleFormat: options.subtitleSidecar ? 'srt' : '',
        embedThumbnail: true,
        embedChapters: true,
      };
    }
    return {
      ...options,
      subtitleMode: '',
      subtitleSidecar: false,
      subtitleLanguages: undefined,
      subtitleAutoCaptions: false,
      subtitleFormat: '',
      embedMetadata: false,
      embedThumbnail: false,
      embedChapters: false,
    };
  }

  function subtitleOutcome(summary: InfoSummary, options: OutputOptions): string {
    const tracks = summary.subtitles ?? [];
    if (!tracks.length) return 'None available';
    if (options.subtitleMode !== 'embed') return 'Off';
    const code = options.subtitleLanguages?.[0];
    const selected = tracks.find((track) => track.code === code && !track.auto)
      ?? tracks.find((track) => track.code === code);
    if (!selected) return 'None';
    const label = (selected.name || selected.code).replace(/\s*\(auto-generated\)$/i, '');
    return `${label}${selected.auto ? ' (auto/transcribed)' : ''}`;
  }

  function chapterOutcome(summary: InfoSummary): string {
    if (typeof summary.chapterCount !== 'number') return 'chapter markers when available';
    if (summary.chapterCount > 0) return `${summary.chapterCount} chapter marker${summary.chapterCount === 1 ? '' : 's'}`;
    return 'no chapter markers reported';
  }

  function preferredLanguageLabel(code: string): string {
    const labels: Record<string, string> = {
      en: 'English', es: 'Spanish', fr: 'French', de: 'German', it: 'Italian', pt: 'Portuguese',
      ja: 'Japanese', ko: 'Korean', zh: 'Chinese', ar: 'Arabic', hi: 'Hindi', ru: 'Russian',
    };
    return labels[code] ? `${labels[code]} (${code})` : code;
  }

  function playlistSubtitleOutcome(options: OutputOptions): string {
    if (options.subtitleMode !== 'embed') return 'Off';
    const language = options.subtitleLanguages?.[0];
    return language ? `On · ${preferredLanguageLabel(language)}` : 'On · each video’s own language';
  }

  function settingsSubtitleOutcome(options: OutputOptions): string {
    if (options.subtitleMode !== 'embed' && options.subtitleMode !== 'sidecar') return 'Off';
    const language = options.subtitleLanguages?.[0];
    return language ? `On · ${preferredLanguageLabel(language)}` : 'On · each video’s own language';
  }

  async function analyze() {
    const submittedURL = url.trim();
    if (!submittedURL) return;
    inputMode = 'single';
    const requestGeneration = ++analysisGeneration;
    busy = true;
    linkedPlaylist = null;
    try {
      const accepted = await api.validation.url(submittedURL);
      if (requestGeneration !== analysisGeneration) return;
      if (accepted.kind === 'video_playlist') {
        linkedPlaylist = accepted;
        modal.set({
          kind: 'confirm',
          title: 'This link includes a playlist',
          message: 'Choose what you want to review.',
          actions: [
            { label: 'This video only', action: () => withBusy(() => analyzeTarget({ ...accepted, kind: 'single_video', url: accepted.videoUrl! }, requestGeneration), requestGeneration) },
            { label: 'Full playlist', primary: true, action: () => withBusy(() => analyzeTarget({ ...accepted, kind: 'playlist', url: accepted.playlistUrl! }, requestGeneration), requestGeneration) },
          ],
        });
      } else {
        await analyzeTarget(accepted, requestGeneration);
      }
    } catch (err) {
      if (requestGeneration !== analysisGeneration) return;
      modal.set({
        kind: 'error',
        title: 'Unsupported URL',
        message: errorMessage(err, 'VidStow could not extract information from this URL. Make sure it is a valid, publicly accessible YouTube video, Short, or playlist.'),
      });
    } finally {
      if (requestGeneration === analysisGeneration) busy = false;
    }
  }

  async function withBusy(action: () => Promise<void>, requestGeneration: number) {
    if (requestGeneration !== analysisGeneration) return;
    busy = true;
    try {
      await action();
    } catch (err) {
      if (requestGeneration !== analysisGeneration) return;
      modal.set({
        kind: 'error',
        title: 'Could not analyze link',
        message: errorMessage(err, 'Could not analyze this link.'),
      });
    } finally {
      if (requestGeneration === analysisGeneration) busy = false;
    }
  }

  async function pickFolder() {
    try {
      const path = await api.folder.pick();
      if (!path) return;
      const updated = await api.settings.update({ ...$settings, downloadFolder: path });
      settings.set(updated);
      folder = path;
    } catch (err) {
      modal.set({ kind: 'error', title: 'Folder could not be changed', message: errorMessage(err, 'Could not update the download folder.') });
    }
  }

  function choose(plan: OutputPlan) {
    selectedPlanId = plan.id;
  }

  function setTab(kind: 'video' | 'audio') {
    tab = kind;
    const compatible = plans.find((plan) => plan.kind === kind && plan.recommended) ?? plans.find((plan) => plan.kind === kind);
    selectedPlanId = compatible?.id ?? '';
  }

  function planDetail(plan: OutputPlan) {
    return [plan.container, plan.kind === 'video' ? plan.videoCodec : '', plan.audioCodec].filter(Boolean).join(' · ');
  }

  function toggle(index: number) {
    const next = new Set(selectedItems);
    if (next.has(index)) next.delete(index);
    else next.add(index);
    selectedItems = next;
  }

  function selectAll() {
    selectedItems = new Set(playlist?.entries.filter((entry) => entry.available).map((entry) => entry.index) ?? []);
  }

  function clearSelection() {
    selectedItems = new Set();
  }

  function toggleSelectAll() {
    if (allAvailableSelected) clearSelection();
    else selectAll();
  }

  function applyRange() {
    if (!playlist) return;
    const start = Number(rangeStart);
    const end = Number(rangeEnd);
    if (!Number.isInteger(start) || !Number.isInteger(end) || start < playlistFirstIndex || end < playlistFirstIndex || start > playlistLastIndex || end > playlistLastIndex) {
      showBanner('warning', `Enter whole-number positions from ${playlistFirstIndex} to ${playlistLastIndex}.`);
      return;
    }
    const low = Math.min(start, end);
    const high = Math.max(start, end);
    selectedItems = new Set(
      playlist.entries
        .filter((entry) => entry.available && entry.index >= low && entry.index <= high)
        .map((entry) => entry.index),
    );
  }

  function reviewLinkedPlaylist() {
    if (!linkedPlaylist?.playlistUrl) return;
    const requestGeneration = ++analysisGeneration;
    withBusy(
      () => analyzeTarget({ ...linkedPlaylist!, kind: 'playlist', url: linkedPlaylist!.playlistUrl! }, requestGeneration),
      requestGeneration,
    );
  }

  function requireFFmpeg(message: string) {
    modal.set({
      kind: 'ffmpeg-missing',
      title: 'FFmpeg Required',
      message,
      actions: [{ label: 'Open Settings', primary: true, action: () => dispatch('goto', 'settings') }],
    });
  }

  async function enqueueVideo() {
    if (!preview || !selectedPlan || !folder || videoQueueBusy || videoJustQueued) return;
    const submittedPreview = preview;
    const submittedPlan = selectedPlan;
    const submittedFolder = folder;
    if (tab === 'video' && !$ffmpeg.available) {
      requireFFmpeg('FFmpeg is required to create a complete video file with artwork and chapters (and subtitles when selected). Install FFmpeg or set its path in Settings.');
      return;
    }
    if (submittedPlan.requiresFfmpeg && !$ffmpeg.available) {
      requireFFmpeg('This output needs FFmpeg for merging or conversion. Install FFmpeg, set its path in Settings, or choose an original audio option.');
      return;
    }
    const options = effectiveOptions(videoOptions, tab === 'video');
    const submittedKey = videoRequestKey;
    videoQueueBusy = true;
    try {
      await api.jobs.start({
        url: submittedPreview.url,
        videoId: submittedPreview.videoId,
        title: submittedPreview.title,
        channel: submittedPreview.channel,
        planId: submittedPlan.id,
        outputDir: submittedFolder,
        duration: submittedPreview.duration,
        thumbnail: submittedPreview.thumbnail,
        options,
      });
      queuedVideoKey = submittedKey;
      showBanner('success', 'Added to queue');
    } catch (err) {
      modal.set({ kind: 'error', title: 'Download could not start', message: errorMessage(err, 'Could not start this download.') });
    } finally {
      videoQueueBusy = false;
    }
  }

  async function enqueuePlaylist() {
    if (!playlist || !selectedItems.size || playlistQueueBusy || playlistJustQueued) return;
    if (!folder) {
      showBanner('warning', 'Choose a download folder before adding this playlist.');
      return;
    }
    const submittedPlaylist = playlist;
    const submittedItems = [...selectedItems].sort((a, b) => a - b);
    const submittedKey = playlistSelectionKey;
    const quality: Quality = playlistTab === 'audio' ? 'audio' : playlistQuality;
    const audioBitrate = playlistTab === 'audio' && audioChoice !== 'original' ? Number(audioChoice) : 0;
    if (audioBitrate && !$ffmpeg.available) {
      requireFFmpeg('MP3 conversion needs FFmpeg. Choose original audio or configure FFmpeg.');
      return;
    }
    const options = effectiveOptions(playlistOptions, playlistTab === 'video');
    if (playlistTab === 'video' && !$ffmpeg.available) {
      requireFFmpeg('FFmpeg is required to create complete video files with artwork and chapters (and subtitles when selected). Install FFmpeg or set its path in Settings.');
      return;
    }
    playlistQueueBusy = true;
    try {
      await api.jobs.startPlaylist({
        url: submittedPlaylist.url,
        playlistId: submittedPlaylist.id,
        quality,
        audioBitrate,
        selectedItems: submittedItems,
        options,
      });
      queuedPlaylistKey = submittedKey;
      showBanner('success', `Added ${submittedItems.length} videos to queue`);
    } catch (err) {
      modal.set({ kind: 'error', title: 'Playlist could not start', message: errorMessage(err, 'Could not add this playlist to the queue.') });
    } finally {
      playlistQueueBusy = false;
    }
  }

</script>

<section class="page home" aria-label="Home workstation">
  <form class="analyze-bar" on:submit|preventDefault={analyze}>
    <label class="visually-hidden" for="video-url">YouTube video, Short, or playlist URL</label>
    <input id="video-url" type="url" value={url} on:input={updateURL} placeholder="https://www.youtube.com/watch?v=…" autocomplete="off" />
    <button
      class="batch-toggle"
      class:active={inputMode === 'batch'}
      type="button"
      aria-label={inputMode === 'batch' ? 'Close batch URL composer' : 'Batch URLs'}
      aria-pressed={inputMode === 'batch'}
      title={inputMode === 'batch' ? 'Close batch URL composer' : 'Batch URLs (2–20)'}
      on:click={toggleBatchComposer}
    >
      <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M3 4.25h1.5M3 8h1.5M3 11.75h1.5M7 4.25h6M7 8h6M7 11.75h6" /></svg>
    </button>
    <button class="app-btn primary analyze-button" type="submit" disabled={busy || !url.trim()}>
      {#if busy}
        Analyzing…
      {:else}
        <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="7" cy="7" r="3.75" /><path d="m10 10 3 3" /></svg>
        Analyze
      {/if}
    </button>
  </form>

  {#if inputMode === 'batch'}
    {#if batchReview}
      <section class="batch-review" aria-labelledby="batch-review-title">
        <header class="batch-review-header">
          <div>
            <h2 id="batch-review-title">Review URLs</h2>
            <p aria-live="polite">{batchReviewSummary(batchReview)}</p>
            {#if !batchTokenValid}<p class="batch-expired" role="alert">This review expired. Edit the lines and review them again.</p>{/if}
          </div>
          <button type="button" class="app-btn" on:click={editBatchURLs} disabled={batchBusy}>Edit URLs</button>
        </header>

        <div class="batch-lines" role="list" aria-label="Reviewed batch URLs">
          {#each batchReview.items as item (item.lineNumber)}
            <article class="batch-line" data-status={item.status} role="listitem">
              <span class="batch-line-number" aria-label={`Line ${item.lineNumber}`}>{item.lineNumber}</span>
              <div class="batch-thumbnail" aria-hidden="true">
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <rect x="3" y="5" width="18" height="14" rx="2"></rect>
                  <path d="m10 9 5 3-5 3Z"></path>
                </svg>
                {#if item.status === 'ready' && item.thumbnail}
                  <img src={item.thumbnail} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />
                {/if}
              </div>
              <div class="batch-line-copy">
                <strong title={item.title || item.input}>{item.title || item.input}</strong>
                <span title={item.input}>{item.input}</span>
                {#if item.title && (item.channel || item.duration)}<small>{[item.channel, item.duration].filter(Boolean).join(' · ')}</small>{/if}
              </div>
              <div class="batch-line-state">
                <span class="batch-state">{item.status === 'analysis_failed' ? 'Analysis failed' : item.status === 'duplicate' ? 'Duplicate' : item.status === 'invalid' ? 'Invalid URL' : 'Ready'}</span>
                {#if item.status !== 'ready'}<small>{item.message}</small>{/if}
              </div>
            </article>
          {/each}
        </div>

        <div class="batch-policy">
          <div>
            <strong>Format</strong>
            <small>Every ready video uses this format.</small>
          </div>
          <div class="segment" aria-label="Batch output type">
            <button type="button" aria-pressed={batchTab === 'video'} class:active={batchTab === 'video'} on:click={() => batchTab = 'video'}>Video</button>
            <button type="button" aria-pressed={batchTab === 'audio'} class:active={batchTab === 'audio'} on:click={() => batchTab = 'audio'}>Audio</button>
          </div>
          {#if batchTab === 'video'}
            <label class="visually-hidden" for="batch-quality">Batch video quality</label>
            <select id="batch-quality" bind:value={batchQuality}>
              <option value="best">Best available</option>
              <option value="4k">Up to 4K</option>
              <option value="1440p">Up to 1440p</option>
              <option value="1080p">Up to 1080p</option>
              <option value="720p">Up to 720p</option>
            </select>
          {:else}
            <label class="visually-hidden" for="batch-audio">Batch audio format</label>
            <select id="batch-audio" bind:value={batchAudioChoice}>
              <option value="original">Original audio</option>
              <option value="128">MP3 · 128 kbps</option>
              <option value="192">MP3 · 192 kbps</option>
              <option value="256">MP3 · 256 kbps</option>
            </select>
          {/if}
          {#if batchTab === 'video'}
            <p class="batch-complete-note">
              Subtitles: {settingsSubtitleOutcome($settings.outputOptions)} (Settings default)
              · SRT: {(($settings.outputOptions.subtitleMode === 'embed' || $settings.outputOptions.subtitleMode === 'sidecar') && ($settings.outputOptions.subtitleSidecar || $settings.outputOptions.subtitleMode === 'sidecar')) ? 'On' : 'Off'}
              · Title &amp; channel details: {$settings.outputOptions.embedMetadata ? 'On' : 'Off'}
              · Artwork and chapters: automatic
            </p>
          {/if}
        </div>

        <footer class="batch-save-bar">
          <div class="destination">
            <span>Save to</span>
            <strong title={folder}>{folder || 'Choose a download folder'}</strong>
          </div>
          <button type="button" class="app-btn" on:click={pickFolder} disabled={batchBusy}>Change…</button>
          <button type="button" class="app-btn primary" on:click={enqueueBatch} disabled={!batchCanStart}>
            {batchBusy ? 'Starting…' : `Start ${batchReadyCount} downloads`}
          </button>
        </footer>
      </section>
    {:else}
      <form class="batch-composer" on:submit|preventDefault={analyzeBatch}>
        <label for="batch-urls">YouTube video or Short URLs</label>
        <textarea id="batch-urls" value={batchText} on:input={updateBatchText} rows="9" placeholder={'https://www.youtube.com/watch?v=…\nhttps://youtu.be/…\nhttps://www.youtube.com/shorts/…'} autocomplete="off"></textarea>
        <div class="batch-composer-footer">
          <p>One URL per line. Blank lines are ignored; duplicates are identified before anything downloads.</p>
          <button class="app-btn primary" type="submit" disabled={batchBusy || batchInputLineCount < 2}>{batchBusy ? 'Analyzing…' : 'Review URLs'}</button>
        </div>
      </form>
    {/if}
  {:else}
  {#if playlist}
    <section class="workspace" aria-label="Playlist">
      <header class="identity">
        <div class="thumb">
          {#if playlist.thumbnail}<img src={playlist.thumbnail} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />{/if}
        </div>
        <div class="identity-copy">
          <strong title={playlist.title}>{playlist.title}</strong>
          <span>
            <em>Playlist</em>
            {#if playlist.channel} · {playlist.channel}{/if}
            · {playlist.entryCount} videos
            {#if playlist.unavailable} · {playlist.unavailable} unavailable{/if}
          </span>
          <small aria-live="polite">{selectedItems.size} of {availableCount} selected</small>
          {#if playlistAtCap}
            <small class="cap-note">VidStow can review up to {PLAYLIST_ADMIT_CAP} videos from a playlist.</small>
          {/if}
        </div>
        <div class="policy">
          <h2>Format</h2>
          <p class="policy-note">Every selected video uses this format.</p>
          <div class="segment" aria-label="Output type">
            <button type="button" aria-pressed={playlistTab === 'video'} class:active={playlistTab === 'video'} on:click={() => playlistTab = 'video'}>Video</button>
            <button type="button" aria-pressed={playlistTab === 'audio'} class:active={playlistTab === 'audio'} on:click={() => playlistTab = 'audio'}>Audio</button>
          </div>
          {#if playlistTab === 'video'}
            <label class="visually-hidden" for="playlist-quality">Video quality</label>
            <select id="playlist-quality" bind:value={playlistQuality}>
              <option value="best">Best available</option>
              <option value="4k">Up to 4K</option>
              <option value="1440p">Up to 1440p</option>
              <option value="1080p">Up to 1080p</option>
              <option value="720p">Up to 720p</option>
            </select>
          {:else}
            <label class="visually-hidden" for="playlist-audio">Audio format</label>
            <select id="playlist-audio" bind:value={audioChoice}>
              <option value="original">Original audio</option>
              <option value="128">MP3 · 128 kbps</option>
              <option value="192">MP3 · 192 kbps</option>
              <option value="256">MP3 · 256 kbps</option>
            </select>
          {/if}
          <OutputOptionsEditor
            bind:value={playlistOptions}
            collectionMode={true}
            allowSubtitles={playlistTab === 'video'}
          />
        </div>
      </header>

      <div class="toolbar">
        <label class="check">
          <input type="checkbox" bind:this={selectAllBox} checked={allAvailableSelected} on:change={toggleSelectAll} />
          All available
        </label>
        <button type="button" class="ghost" on:click={clearSelection} disabled={!selectedItems.size}>Clear</button>
        <form class="range" on:submit|preventDefault={applyRange}>
          <span>Range</span>
          <input type="number" min={playlistFirstIndex} max={playlistLastIndex} step="1" inputmode="numeric" bind:value={rangeStart} aria-label="Range start" />
          <span class="dash">–</span>
          <input type="number" min={playlistFirstIndex} max={playlistLastIndex} step="1" inputmode="numeric" bind:value={rangeEnd} aria-label="Range end" />
          <button type="submit" class="ghost">Apply</button>
        </form>
        <input type="search" bind:value={search} placeholder="Search playlist…" aria-label="Search playlist" />
      </div>

      <div class="entry-list" role="list">
        {#each filteredEntries as entry (entry.index)}
          <label class="entry" class:unavailable={!entry.available} class:selected={selectedItems.has(entry.index)} role="listitem">
            <input type="checkbox" checked={selectedItems.has(entry.index)} disabled={!entry.available} on:change={() => toggle(entry.index)} />
            <span class="number">{entry.index}</span>
            <span class="mini">
              {#if entry.thumbnail || entry.videoId}
                <img src={entry.thumbnail || fallbackThumbnail(entry.videoId)} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />
              {/if}
            </span>
            <strong title={entry.title}>{entry.title}</strong>
            {#if !entry.available}
              <span class="meta">Unavailable</span>
            {:else if entry.duration}
              <span class="meta">{entry.duration}</span>
            {/if}
          </label>
        {:else}
          <div class="empty-list">{query ? 'No videos match that search.' : 'No videos in this playlist.'}</div>
        {/each}
      </div>

      {#if playlistTab === 'video'}
        <aside class="complete-summary" aria-label="Complete file contents">
          <strong>Complete video files</strong>
          <span>Subtitles: {playlistSubtitleOutcome(playlistOptions)}.</span>
          <span>Artwork and chapter markers are included when provided.</span>
          {#if playlistOptions.subtitleMode === 'embed' && playlistOptions.subtitleSidecar}<span>An additional .srt will be saved when a track is available.</span>{/if}
          <span>MP4 preferred · MKV fallback when needed.</span>
          <label class="summary-option"><input type="checkbox" checked={!!playlistOptions.embedMetadata} on:change={(event) => (playlistOptions = { ...playlistOptions, embedMetadata: event.currentTarget.checked })} /> Title &amp; channel details</label>
          {#if !$ffmpeg.available}<span class="ffmpeg-required">FFmpeg is required before these videos can be added. Configure it in Settings.</span>{/if}
        </aside>
      {/if}

      <footer class="save-bar">
        <div class="destination">
          <span>Save to</span>
          <strong title={folder}>{folder}</strong>
          <small title={`${playlist.title} [${playlist.id}]`}>Playlist folder · {shortTitle(playlist.title, 48)}</small>
        </div>
        <button type="button" class="app-btn" on:click={pickFolder}>Change…</button>
        <button type="button" class="app-btn primary queue" class:queued={playlistJustQueued} on:click={enqueuePlaylist} disabled={!selectedItems.size || !folder || playlistQueueBusy || playlistJustQueued || (playlistTab === 'video' && !$ffmpeg.available)} title={playlistTab === 'video' && !$ffmpeg.available ? 'FFmpeg is required for complete video files' : ''}>
          {#if playlistQueueBusy}
            Adding…
          {:else if playlistJustQueued}
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m3.5 8.25 2.75 2.75 6.25-6.25" /></svg>
            Added to Queue
          {:else}
            Add {selectedItems.size} {selectedItems.size === 1 ? 'Video' : 'Videos'} to Queue
          {/if}
        </button>
      </footer>
    </section>
  {:else if preview}
    <section class="workspace video" aria-label="Video">
      <header class="identity">
        <div class="thumb thumbnail">
          {#if preview.thumbnail}<img src={preview.thumbnail} alt="" referrerpolicy="no-referrer" />{/if}
          {#if preview.duration}<span>{preview.duration}</span>{/if}
        </div>
        <div class="identity-copy">
          <strong title={preview.title}>{preview.title}</strong>
          <span>{preview.channel || 'YouTube'}{#if preview.mediaType === 'short'} · <em>Short</em>{/if}</span>
          <small>{preview.duration || 'Duration unavailable'}{preview.viewCount ? ` · ${formatViewCount(preview.viewCount)} views` : ''}</small>
          {#if linkedPlaylist?.playlistUrl}
            <button type="button" class="ghost review-playlist" on:click={reviewLinkedPlaylist}>Review the playlist instead</button>
          {/if}
        </div>
        <div class="policy">
          <h2>Choose Download</h2>
          <div class="segment" aria-label="Output type">
            <button type="button" aria-pressed={tab === 'video'} class:active={tab === 'video'} on:click={() => setTab('video')}>Video</button>
            <button type="button" aria-pressed={tab === 'audio'} class:active={tab === 'audio'} on:click={() => setTab('audio')}>Audio</button>
          </div>
          <OutputOptionsEditor
            bind:value={videoOptions}
            languages={preview?.subtitles ?? []}
            videoLanguage={preview.language}
            allowSubtitles={tab === 'video'}
          />
        </div>
      </header>

      {#if visiblePlans.length}
        <div class="plan-list pane" role="radiogroup" aria-label={`${tab} output options`}>
          {#each visiblePlans as plan (plan.id)}
            <button type="button" class="plan-row" class:selected={selectedPlanId === plan.id} role="radio" aria-checked={selectedPlanId === plan.id} on:click={() => choose(plan)}>
              <span class="radio"></span>
              <span class="plan-copy">
                <strong>
                  {plan.label}
                  {#if plan.recommended}<em>Recommended</em>{/if}
                </strong>
                <small>{planDetail(plan) || plan.container}</small>
              </span>
              <span class="plan-size">{plan.approxBytes ? `${plan.sizeIsApproximate ? '~' : ''}${formatBytes(plan.approxBytes)}` : '—'}</span>
            </button>
          {/each}
        </div>
      {:else}
        <div class="empty pane">No {tab} outputs were reported for this video.</div>
      {/if}

      {#if tab === 'video'}
        <aside class="complete-summary" aria-label="Complete file contents">
          <strong>What this file will include</strong>
          <span>Subtitles: {subtitleOutcome(preview, videoOptions)}.</span>
          <span>Artwork: {preview.thumbnail ? 'thumbnail artwork will be embedded' : 'none reported'}.</span>
          <span>Chapters: {chapterOutcome(preview)}.</span>
          {#if videoOptions.subtitleMode === 'embed' && videoOptions.subtitleSidecar}<span>Additional SRT: saved when the selected track is available.</span>{/if}
          <span>Container: likely MP4, with MKV fallback when needed.</span>
          <label class="summary-option"><input type="checkbox" checked={!!videoOptions.embedMetadata} on:change={(event) => (videoOptions = { ...videoOptions, embedMetadata: event.currentTarget.checked })} /> Title &amp; channel details</label>
          {#if !$ffmpeg.available}<span class="ffmpeg-required">FFmpeg is required to create this complete file. Configure it in Settings before adding.</span>{/if}
        </aside>
      {/if}

      <footer class="save-bar">
        <div class="destination">
          <span>Save to</span>
          <strong title={folder}>{folder}</strong>
          {#if selectedPlan}
            <small>{selectedPlan.label} · {selectedPlan.container}{selectedPlan.approxBytes ? ` · ${selectedPlan.sizeIsApproximate ? '~' : ''}${formatBytes(selectedPlan.approxBytes)}` : ''}</small>
          {/if}
        </div>
        <button type="button" class="app-btn" on:click={pickFolder}>Change…</button>
        <button type="button" class="app-btn primary queue" class:queued={videoJustQueued} on:click={enqueueVideo} disabled={!selectedPlan || !folder || videoQueueBusy || videoJustQueued || (tab === 'video' && !$ffmpeg.available)} title={tab === 'video' && !$ffmpeg.available ? 'FFmpeg is required for a complete video file' : ''}>
          {#if videoQueueBusy}
            Adding…
          {:else if videoJustQueued}
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="m3.5 8.25 2.75 2.75 6.25-6.25" /></svg>
            Added to Queue
          {:else}
            Add to Queue
          {/if}
        </button>
      </footer>
    </section>
  {:else}
    <section class="welcome">
      <svg class="download-mark" viewBox="0 0 16 16" aria-hidden="true">
        <path d="M8 2.75v7.5m0 0L5.5 7.75M8 10.25l2.5-2.5M3.5 12.75h9" />
      </svg>
      <p>Paste a YouTube URL to analyze</p>
    </section>
    {/if}
  {/if}
</section>

<style>
  .home {
    height: 100%;
    min-height: 0;
    overflow: hidden;
    padding: 0;
    gap: 0;
    background: var(--surface-bg);
  }

  .batch-composer,
  .batch-review {
    display: flex;
    min-height: 0;
    margin: 10px 12px 12px;
    flex-direction: column;
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    background: var(--surface-base);
    box-shadow: none;
  }
  .batch-composer { padding: var(--sp-3); gap: var(--sp-3); overflow: auto; }
  .batch-composer > label { font-weight: 700; }
  .batch-composer textarea {
    width: 100%;
    min-height: 210px;
    resize: vertical;
    padding: var(--sp-4);
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    background: var(--surface-base);
    color: var(--text-primary);
    font: inherit;
    line-height: 1.6;
  }
  .batch-composer-footer,
  .batch-review-header,
  .batch-save-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-4);
  }
  .batch-composer-footer p,
  .batch-review-header p { margin: 0; color: var(--text-muted); font-size: var(--fs-sm); }
  .batch-review { flex: 1; overflow: hidden; }
  .batch-review-header { padding: var(--sp-4); }
  .batch-review-header h2 { margin: 0 0 3px; font-size: var(--fs-lg); }
  .batch-review-header .batch-expired { margin-top: var(--sp-2); color: var(--status-danger); }
  .batch-lines { display: flex; min-height: 0; flex: 1; overflow: auto; flex-direction: column; border-top: 1px solid var(--border-subtle); }
  .batch-line {
    display: grid;
    grid-template-columns: 28px 64px minmax(0, 1fr) minmax(130px, 210px);
    align-items: center;
    gap: var(--sp-3);
    padding: 7px 12px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-base);
  }
  .batch-line-number { color: var(--text-muted); font-variant-numeric: tabular-nums; text-align: center; }
  .batch-thumbnail {
    width: 64px;
    aspect-ratio: 16 / 9;
    position: relative;
    display: grid;
    place-items: center;
    overflow: hidden;
    border-radius: var(--r-sm);
    background: var(--surface-sunken);
    color: var(--text-muted);
  }
  .batch-thumbnail img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: cover; }
  .batch-thumbnail svg { width: 28px; height: 28px; fill: none; stroke: currentColor; stroke-width: 1.5; }
  .batch-thumbnail svg path { fill: currentColor; stroke: none; }
  .batch-line-copy,
  .batch-line-state { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
  .batch-line-copy strong,
  .batch-line-copy span,
  .batch-line-state small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .batch-line-copy span,
  .batch-line-copy small,
  .batch-line-state small { color: var(--text-muted); font-size: var(--fs-xs); }
  .batch-line-state { align-items: flex-end; text-align: right; }
  .batch-state { display: inline-flex; padding: 3px 8px; border-radius: var(--r-full); font-size: var(--fs-xs); font-weight: 700; }
  .batch-line[data-status='ready'] .batch-state { color: var(--status-success); background: var(--status-success-soft); }
  .batch-line[data-status='duplicate'] .batch-state { color: var(--status-warning); background: var(--status-warning-soft); }
  .batch-line[data-status='invalid'] .batch-state,
  .batch-line[data-status='analysis_failed'] .batch-state { color: var(--status-danger); background: var(--status-danger-soft); }
  .batch-policy {
    display: grid;
    grid-template-columns: minmax(180px, 1fr) auto minmax(170px, 220px);
    align-items: center;
    gap: 8px var(--sp-3);
    padding: 9px 12px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-sunken);
  }
  .batch-policy > div:first-child { display: flex; flex-direction: column; }
  .batch-policy small { color: var(--text-muted); }
  .batch-policy select { height: 32px; font-size: var(--fs-xs); }
  .batch-complete-note { grid-column: 1 / -1; margin: -1px 0 0; color: var(--text-muted); font-size: var(--fs-xs); }
  .batch-save-bar { padding: 9px 12px; }
  .batch-save-bar .destination { flex: 1; min-width: 0; }

  .analyze-bar {
    display: grid;
    height: var(--topbar-h);
    padding: 3px 12px;
    grid-template-columns: minmax(0, 1fr) 26px 76px;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-bg);
  }
  .analyze-bar input {
    height: 26px;
    padding: 0 10px;
    font-family: var(--font-mono);
    font-size: var(--fs-xs);
  }
  .analyze-bar .app-btn { min-height: 24px; height: 24px; padding: 0 8px; }
  .analyze-button { gap: 5px; }
  .analyze-button svg,
  .batch-toggle svg,
  .queue svg { width: 13px; height: 13px; fill: none; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
  .batch-toggle {
    width: 26px;
    height: 24px;
    display: grid;
    place-items: center;
    border: 1px solid transparent;
    border-radius: var(--r-md);
    color: var(--text-muted);
  }
  .batch-toggle:hover,
  .batch-toggle.active { border-color: var(--border-default); background: var(--surface-raised); color: var(--text-primary); }
  .batch-toggle.active { color: var(--accent-400); }
  .ghost {
    min-height: 32px;
    padding: 0 10px;
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    background: var(--surface-base);
    color: var(--text-primary);
    font-size: var(--fs-sm);
    font-weight: 600;
  }
  .ghost:hover:not(:disabled) { background: var(--surface-hover); }

  .workspace {
    flex: 1;
    min-height: 0;
    margin: 10px 12px 12px;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    background: var(--surface-base);
    box-shadow: none;
    overflow: hidden;
  }
  .identity {
    display: grid;
    grid-template-columns: 72px minmax(0, 1fr) auto;
    gap: 14px;
    align-items: center;
    padding: 10px 12px;
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }
  .thumb, .mini, .thumbnail {
    overflow: hidden;
    border-radius: var(--r-sm);
    background: var(--surface-sunken);
  }
  .thumb {
    width: 72px;
    aspect-ratio: 16 / 9;
    position: relative;
  }
  .thumb span {
    position: absolute;
    right: 4px;
    bottom: 4px;
    padding: 1px 5px;
    border-radius: 3px;
    background: #111d;
    color: #fff;
    font-size: 10px;
  }
  .thumb img, .mini img, .thumbnail img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .identity-copy {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .identity-copy strong, .entry strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .identity-copy strong { font-size: var(--fs-md); letter-spacing: -0.015em; }
  .identity-copy span, .identity-copy small { color: var(--text-secondary); font-size: var(--fs-xs); }
  .identity-copy em {
    font-style: normal;
    font-weight: 650;
    color: var(--text-secondary);
  }
  .identity-copy small { color: var(--text-secondary); font-weight: 550; }

  .policy {
    display: grid;
    grid-template-columns: auto minmax(168px, 200px);
    gap: 8px 10px;
    align-items: center;
  }
  .policy h2 {
    grid-column: 1 / -1;
    margin: 0;
    color: var(--text-secondary);
    font-size: 11px;
    font-weight: 600;
  }
  .policy-note {
    grid-column: 1 / -1;
    margin: -4px 0 0;
    color: var(--text-muted);
    font-size: 11px;
    font-weight: 500;
  }
  .cap-note, .review-playlist {
    color: var(--accent-600);
    font-weight: 600;
  }
  .review-playlist { justify-self: start; margin-top: 4px; min-height: 28px; padding: 0 8px; }
  .segment {
    display: inline-flex;
    padding: 3px;
    background: var(--surface-sunken);
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
  }
  .segment button {
    min-width: 68px;
    min-height: 28px;
    padding: 0 10px;
    border-radius: 6px;
    color: var(--text-secondary);
    font-size: var(--fs-xs);
    font-weight: 600;
  }
  .segment button.active {
    color: var(--text-primary);
    background: var(--surface-base);
    box-shadow: var(--shadow-card);
  }
  .policy select { height: 34px; padding: 0 10px; font-size: var(--fs-xs); }

  .toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 12px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-subtle);
    flex-shrink: 0;
  }
  .check {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: var(--fs-xs);
    font-weight: 600;
    color: var(--text-secondary);
    white-space: nowrap;
  }
  .range {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--text-secondary);
    font-size: var(--fs-xs);
    font-weight: 600;
  }
  .range input {
    width: 56px;
    height: 32px;
    padding: 0 8px;
    text-align: center;
  }
  .toolbar input[type='search'] {
    margin-left: auto;
    width: min(240px, 100%);
    height: 32px;
    padding: 0 10px;
    font-size: var(--fs-xs);
  }

  .entry-list {
    flex: 1;
    min-height: 150px;
    overflow: auto;
  }
  .entry {
    display: grid;
    grid-template-columns: 16px 36px 64px minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    padding: 7px 12px;
    border-bottom: 1px solid var(--border-subtle);
    cursor: pointer;
  }
  .entry:hover { background: var(--surface-hover); }
  .entry.unavailable {
    opacity: 0.55;
    cursor: default;
  }
  .entry .number, .entry .meta {
    font-size: var(--fs-xs);
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
  }
  .entry strong { font-size: var(--fs-sm); font-weight: 550; }
  .mini {
    width: 64px;
    aspect-ratio: 16 / 9;
  }
  .empty-list {
    min-height: 160px;
    display: grid;
    place-items: center;
    color: var(--text-muted);
    font-size: var(--fs-sm);
  }

  .complete-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 3px 10px;
    padding: 7px 12px;
    border-top: 1px solid var(--border-subtle);
    background: var(--surface-subtle);
    color: var(--text-secondary);
    font-size: 11px;
    line-height: 1.45;
  }
  .complete-summary strong { flex-basis: 100%; color: var(--text-primary); font-size: var(--fs-xs); }
  .complete-summary .ffmpeg-required { flex-basis: 100%; color: var(--status-warning); font-weight: 700; }
  .summary-option { display: inline-flex; align-items: center; gap: 6px; margin-left: auto; color: var(--text-secondary); font-size: 10px; white-space: nowrap; cursor: pointer; }
  .summary-option input { margin: 0; }

  .save-bar {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    gap: 10px;
    align-items: center;
    padding: 9px 12px;
    border-top: 1px solid var(--border-subtle);
    background: var(--surface-base);
    flex-shrink: 0;
  }
  .destination {
    display: flex;
    min-width: 0;
    flex-wrap: wrap;
    gap: 6px 8px;
    align-items: baseline;
  }
  .destination span { color: var(--text-secondary); font-size: var(--fs-sm); }
  .destination strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--fs-sm);
  }
  .destination small {
    flex-basis: 100%;
    color: var(--text-muted);
    font-size: 11px;
  }
  .queue { min-width: 168px; min-height: 30px; gap: 6px; }
  .queue.queued:disabled {
    border-color: rgba(34, 197, 94, 0.4);
    background: var(--status-success-soft);
    color: var(--status-success);
    opacity: 1;
  }

  .pane {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .plan-list {
    display: flex;
    flex-direction: column;
  }
  .plan-row {
    display: grid;
    grid-template-columns: 20px minmax(0, 1fr) auto;
    gap: 12px;
    align-items: center;
    width: 100%;
    min-height: 44px;
    padding: 8px 14px;
    border-top: 1px solid var(--border-subtle);
    text-align: left;
    color: var(--text-secondary);
  }
  .plan-row:first-child { border-top: 0; }
  .plan-row:hover { background: var(--surface-hover); }
  .plan-row.selected { background: var(--accent-soft); }
  .plan-copy { min-width: 0; }
  .plan-copy strong {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-primary);
    font-size: var(--fs-sm);
  }
  .plan-copy em {
    font-style: normal;
    font-size: 10px;
    font-weight: 650;
    letter-spacing: 0.02em;
    text-transform: uppercase;
    color: var(--accent-600);
  }
  .plan-copy small {
    display: block;
    margin-top: 3px;
    color: var(--text-secondary);
    font-size: 11px;
  }
  .plan-size {
    color: var(--text-primary);
    font-size: var(--fs-sm);
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .radio {
    width: 14px;
    height: 14px;
    border: 1.5px solid var(--border-strong);
    border-radius: 50%;
  }
  .selected .radio { border: 4px solid var(--accent-600); }
  .empty {
    display: grid;
    place-items: center;
    color: var(--text-secondary);
    font-size: var(--fs-sm);
  }

  .welcome {
    flex: 1;
    min-height: 0;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 12px;
    text-align: center;
    color: var(--text-secondary);
  }
  .welcome p { margin: 0; font-size: var(--fs-sm); }
  .download-mark {
    width: 14px;
    height: 14px;
    fill: none;
    stroke: var(--border-strong);
    stroke-width: 1.25;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  @media (max-width: 860px) {
    .batch-policy { grid-template-columns: 1fr auto; }
    .batch-policy select { grid-column: 1 / -1; width: 100%; }
    .identity { grid-template-columns: 72px 1fr; }
    .policy { grid-column: 1 / -1; }
    .toolbar input[type='search'] { margin-left: 0; width: 100%; flex-basis: 100%; }
  }
  @media (max-width: 720px) {
    .analyze-bar { padding-right: 8px; padding-left: 8px; grid-template-columns: minmax(0, 1fr) 26px 72px; gap: 6px; }
    .batch-composer,
    .batch-review,
    .workspace { margin-right: 14px; margin-left: 14px; }
    .batch-composer-footer,
    .batch-review-header,
    .batch-save-bar { align-items: stretch; flex-direction: column; }
    .batch-composer-footer .app-btn,
    .batch-review-header .app-btn,
    .batch-save-bar .app-btn { width: 100%; }
    .batch-line { grid-template-columns: 28px 72px minmax(0, 1fr); }
    .batch-thumbnail { width: 72px; }
    .batch-line-state { grid-column: 3; align-items: flex-start; text-align: left; }
    .save-bar { grid-template-columns: 1fr auto; }
    .destination { grid-column: 1 / -1; }
    .queue { grid-column: 1 / -1; }
    .entry { grid-template-columns: 16px 28px minmax(0, 1fr) auto; }
    .mini { display: none; }
  }
</style>
