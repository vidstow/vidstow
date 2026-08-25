<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { errorMessage, ffmpeg, modal, pendingUrl, settings, showBanner } from '../lib/stores.js';
  import { formatBytes, formatViewCount, shortTitle, youtubeUrlFromText } from '../lib/format.js';
  import type { BatchAnalysisView, InfoSummary, OutputPlan, PlaylistSummary, Quality, UrlCheckResult } from '../lib/types.js';

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
  let queuedVideoPlanId = '';
  let queuedPlaylistKey = '';
  let videoQueueBusy = false;
  let playlistQueueBusy = false;
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
  $: batchCanStart = batchTokenValid && batchReadyCount >= 2 && !!folder && !batchBusy;
  $: playlistSelectionKey = playlist
    ? [playlist.id, playlistTab, playlistTab === 'audio' ? audioChoice : playlistQuality, [...selectedItems].sort((a, b) => a - b).join(',')].join('|')
    : '';
  $: videoJustQueued = !!selectedPlan && queuedVideoPlanId === selectedPlan.id;
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
    queuedVideoPlanId = '';
    queuedPlaylistKey = '';
    selectedItems = new Set();
    selectedPlanId = '';
    search = '';
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
      selectedItems = new Set(summary.entries.filter((entry) => entry.available).map((entry) => entry.index));
      rangeStart = summary.entries[0]?.index ? String(summary.entries[0].index) : '1';
      rangeEnd = summary.entries.at(-1)?.index ? String(summary.entries.at(-1)!.index) : String(summary.entryCount);
    } else {
      const canonicalURL = target.videoUrl!;
      const summary = await api.analyse.url(canonicalURL);
      if (requestGeneration !== analysisGeneration) return;
      url = canonicalURL;
      preview = summary;
      const recommended = summary.plans.find((plan) => plan.recommended) ?? summary.plans[0];
      selectedPlanId = recommended?.id ?? '';
      tab = recommended?.kind ?? 'video';
    }
  }

  async function analyze() {
    const submittedURL = url.trim();
    if (!submittedURL) return;
    // The URL strip stays available above the batch workflow. A submitted
    // single URL always returns to the matching video or playlist workspace.
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
    if (submittedPlan.requiresFfmpeg && !$ffmpeg.available) {
      requireFFmpeg('This output needs FFmpeg for merging or conversion. Install FFmpeg, set its path in Settings, or choose an original audio option.');
      return;
    }
    const start = async () => {
      if (videoQueueBusy || queuedVideoPlanId === submittedPlan.id) return;
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
        });
        queuedVideoPlanId = submittedPlan.id;
        showBanner('success', 'Added to queue');
      } catch (err) {
        modal.set({ kind: 'error', title: 'Download could not start', message: errorMessage(err, 'Could not start this download.') });
      } finally {
        videoQueueBusy = false;
      }
    };
    if ($settings.confirmBeforeDownload) {
      modal.set({
        kind: 'confirm',
        title: 'Add this download?',
        message: `${submittedPlan.label} · ${submittedPlan.container}${submittedPlan.approxBytes ? ` · about ${formatBytes(submittedPlan.approxBytes)}` : ''}`,
        actions: [{ label: 'Add to Queue', primary: true, action: start }],
      });
      return;
    }
    await start();
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
    const start = async () => {
      if (playlistQueueBusy || queuedPlaylistKey === submittedKey) return;
      playlistQueueBusy = true;
      try {
        await api.jobs.startPlaylist({
          url: submittedPlaylist.url,
          playlistId: submittedPlaylist.id,
          quality,
          audioBitrate,
          selectedItems: submittedItems,
        });
        queuedPlaylistKey = submittedKey;
        showBanner('success', `Added ${submittedItems.length} videos to queue`);
      } catch (err) {
        modal.set({ kind: 'error', title: 'Playlist could not start', message: errorMessage(err, 'Could not add this playlist to the queue.') });
      } finally {
        playlistQueueBusy = false;
      }
    };
    if (submittedItems.length > 100 || $settings.confirmBeforeDownload) {
      modal.set({
        kind: 'confirm',
        title: 'Add this playlist?',
        message: `${submittedItems.length} videos will be added to the queue.`,
        actions: [{ label: 'Add to Queue', primary: true, action: start }],
      });
      return;
    }
    await start();
  }
</script>
