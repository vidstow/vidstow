<script lang="ts">
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { api } from '../lib/api.js';
  import { errorMessage, ffmpeg, modal, pendingUrl, settings, showBanner } from '../lib/stores.js';
  import { formatBytes, formatPlanSize, formatViewCount, shortTitle } from '../lib/format.js';
  import FormatPicker from '../lib/components/FormatPicker.svelte';
  import type { BatchAnalysisView, InfoSummary, OutputPlan, PlaylistSummary, Quality, UrlCheckResult } from '../lib/types.js';

  const dispatch = createEventDispatcher<{ goto: 'home' | 'queue' | 'downloads' | 'settings' | 'about' }>();

  const PLAYLIST_ADMIT_CAP = 500;
  const VIDEO_QUALITIES: Array<{ value: Quality; label: string }> = [
    { value: 'best', label: 'Best available' },
    { value: '4k', label: '4K' },
    { value: '1440p', label: '1440p' },
    { value: '1080p', label: '1080p' },
    { value: '720p', label: '720p' },
  ];
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
  const KIND_OPTIONS = [
    { id: 'video', label: 'Video' },
    { id: 'audio', label: 'Audio' },
  ];
  const TRY_CHIPS: Array<{ kind: 'video' | 'playlist' | 'batch'; label: string }> = [
    { kind: 'video', label: 'a video' },
    { kind: 'playlist', label: 'a playlist' },
    { kind: 'batch', label: 'several links' },
  ];

  function downloadVideosLabel(count: number): string {
    return `Download ${count} ${count === 1 ? 'video' : 'videos'}`;
  }

  let url = '';
  let urlField: HTMLTextAreaElement | undefined;
  let analysisGeneration = 0;
  let busy = false;
  let batchText = '';
  let batchGeneration = 0;
  let batchBusy = false;
  let playlistBusy = false;
  let batchReview: BatchAnalysisView | null = null;
  let batchNow = Date.now();
  let batchTab: 'video' | 'audio' = 'video';
  let batchQuality: Quality = '1080p';
  let batchAudioChoice = 'original';
  let folder = '';
  let selectedPlanId = '';
  let preview: InfoSummary | null = null;
  let playlist: PlaylistSummary | null = null;
  let selectedItems = new Set<number>();
  let tab: 'video' | 'audio' = 'video';
  let playlistTab: 'video' | 'audio' = 'video';
  let playlistQuality: Quality = '1080p';
  let audioChoice = 'original';
  let rangeStart = '';
  let rangeEnd = '';
  let rangeWarn = false;
  let linkedPlaylist: UrlCheckResult | null = null;
  let scopeChoice: UrlCheckResult | null = null;
  let scopeVideo: InfoSummary | null = null;
  let scopePlaylist: PlaylistSummary | null = null;
  let scopePlaylistTask: Promise<PlaylistSummary> | null = null;
  let scopeFocus: 'video' | 'playlist' = 'playlist';
  let analyzeError: { title: string; message: string } | null = null;
  let detailsOpen = false;

  const batchExpiryTimer = setInterval(() => {
    if (batchReview) batchNow = Date.now();
  }, 1000);
  onDestroy(() => clearInterval(batchExpiryTimer));

  $: folder = $settings.downloadFolder || folder;
  $: plans = preview?.plans ?? [];
  $: selectedPlan = plans.find((plan) => plan.id === selectedPlanId) ?? null;
  $: visiblePlans = plans.filter((plan) => plan.available && plan.kind === tab);
  $: kindOptions = (
    [
      plans.some((plan) => plan.available && plan.kind === 'video') ? { id: 'video', label: 'Video' } : null,
      plans.some((plan) => plan.available && plan.kind === 'audio') ? { id: 'audio', label: 'Audio' } : null,
    ] as Array<{ id: string; label: string } | null>
  ).filter((option): option is { id: string; label: string } => option !== null);
  $: videoPlanOptions = visiblePlans.map((plan) => ({
    id: plan.id,
    label: plan.label,
    size: plan.approxBytes ? `${plan.sizeIsApproximate ? '~' : ''}${formatBytes(plan.approxBytes)}` : undefined,
  }));
  $: playlistPlanOptions = playlistTab === 'audio'
    ? AUDIO_CHOICES.map((option) => ({ id: option.value, label: option.label }))
    : PLAYLIST_VIDEO_QUALITIES.map((option) => ({ id: option.value, label: option.label }));
  $: batchPlanOptions = batchTab === 'audio'
    ? AUDIO_CHOICES.map((option) => ({ id: option.value, label: option.label }))
    : VIDEO_QUALITIES.map((option) => ({ id: option.value, label: option.label }));
  $: playlistFormatValue = playlistTab === 'audio' ? audioChoice : playlistQuality;
  $: batchFormatValue = batchTab === 'audio' ? batchAudioChoice : batchQuality;
  $: availableCount = playlist?.available ?? 0;
  $: playlistFirstIndex = playlist?.entries[0]?.index ?? 1;
  $: playlistLastIndex = playlist?.entries.at(-1)?.index ?? playlist?.entryCount ?? 1;
  $: playlistAtCap = (playlist?.entries.length ?? 0) >= PLAYLIST_ADMIT_CAP;
  $: batchReadyCount = batchReview?.counts.ready ?? 0;
  $: batchExpiry = batchReview?.expiresAt ? Date.parse(batchReview.expiresAt) : Number.NaN;
  $: batchTokenValid = !!batchReview?.token && Number.isFinite(batchExpiry) && batchExpiry > batchNow;
  $: batchCanStart = batchTokenValid && batchReadyCount >= 2 && !!folder && !batchBusy;
  $: hasDock = !!(preview || playlist || batchReview || scopeChoice);
  $: linkedSwap = !!(linkedPlaylist?.playlistUrl && linkedPlaylist.videoUrl && (preview || playlist));
  $: scopeVideoPlan = scopeVideo?.plans.find((plan) => plan.recommended) ?? scopeVideo?.plans[0] ?? null;
  $: scopeVideoMeta = [scopeVideo?.channel, scopeVideo?.duration, scopeVideoPlan?.approxBytes ? `${scopeVideoPlan.sizeIsApproximate ? '~' : ''}${formatBytes(scopeVideoPlan.approxBytes)}` : ''].filter(Boolean).join(' · ');
  $: scopePlaylistMeta = scopePlaylist
    ? [`${scopePlaylist.entryCount} videos`, scopePlaylist.duration, scopePlaylist.channel].filter(Boolean).join(' · ')
    : scopePlaylistTask ? 'Reading playlist…' : 'Review every video in this list';
  $: playlistMeta = playlist
    ? [
        'Playlist',
        `${playlist.entryCount} videos`,
        playlist.duration,
        playlist.channel,
        playlist.unavailable ? `${playlist.unavailable} unavailable` : '',
      ].filter(Boolean).join(' · ')
    : '';
  $: playlistSavePath = playlist && folder ? joinSavePath(folder, playlist.title) : '';

  function syncFieldHeight() {
    const node = urlField;
    if (!node) return;
    node.style.height = 'auto';
    node.style.height = `${Math.min(Math.max(node.scrollHeight, 28), 96)}px`;
  }

  $: if (url !== undefined) {
    queueMicrotask(syncFieldHeight);
  }

  $: if ($pendingUrl) {
    const droppedURL = $pendingUrl;
    pendingUrl.set('');
    applyUrl(droppedURL);
    if (urlField) urlField.value = droppedURL;
    void submitPaste(droppedURL);
  }

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

  function qualityChipLabel(value: Quality): string {
    return PLAYLIST_VIDEO_QUALITIES.find((option) => option.value === value)?.label
      ?? VIDEO_QUALITIES.find((option) => option.value === value)?.label
      ?? value;
  }

  function playlistPolicyCopy(): { link: string; detail: string } {
    const count = playlist?.entryCount ?? 0;
    const link = `${count} episode${count === 1 ? '' : 's'}`;
    if (playlistTab === 'audio') {
      if (audioChoice === 'original') return { link, detail: 'Original audio' };
      return { link, detail: `MP3 ${audioChoice} · converted with FFmpeg` };
    }
    const label = qualityChipLabel(playlistQuality);
    return { link, detail: label === 'Best available' ? 'best available' : `up to ${label}` };
  }

  function planSizeCopy(plan: OutputPlan): string {
    if (!plan.approxBytes) return '';
    return formatPlanSize(plan.approxBytes, !!plan.sizeIsApproximate);
  }

  function pasteItems(raw: string): string[] {
    const items: string[] = [];
    for (const line of raw.split(/\r?\n/)) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      const urls = trimmed.match(/https?:\/\/[^\s,]+/gi);
      if (urls && urls.length > 1) items.push(...urls);
      else items.push(trimmed);
    }
    return items;
  }

  function resetDock() {
    preview = null;
    playlist = null;
    selectedItems = new Set();
    selectedPlanId = '';
    detailsOpen = false;
    rangeWarn = false;
  }

  function clearLinkContext() {
    linkedPlaylist = null;
    scopeChoice = null;
    scopeVideo = null;
    scopePlaylist = null;
    scopePlaylistTask = null;
    scopeFocus = 'playlist';
  }

  function clearAnalysis() {
    resetDock();
    clearLinkContext();
  }

  function applyUrl(next: string) {
    if (next === url) return;
    url = next;
    analysisGeneration += 1;
    busy = false;
    linkedPlaylist = null;
    analyzeError = null;
    if (preview || playlist || scopeChoice) clearAnalysis();
    if (batchReview) {
      batchGeneration += 1;
      batchBusy = false;
      batchReview = null;
    }
    playlistBusy = false;
  }

  function updatePaste(event: Event) {
    applyUrl((event.currentTarget as HTMLTextAreaElement).value);
  }

  function onUrlKey(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return;
    event.preventDefault();
    void submitPaste();
  }

  async function submitPaste(source?: string) {
    applyUrl(typeof source === 'string' ? source : (urlField?.value ?? url));
    const items = pasteItems(url);
    if (!items.length || busy || batchBusy || scopeChoice) return;
    analyzeError = null;
    if (items.length > 20) {
      analyzeError = {
        title: 'Too many URLs',
        message: 'Paste between 2 and 20 individual public YouTube video or Short URLs.',
      };
      return;
    }
    if (items.length === 1) {
      await analyze();
      return;
    }
    batchText = items.join('\n');
    await analyzeBatch();
  }

  async function analyzeBatch() {
    if (!batchText.trim() || batchBusy) return;
    const requestGeneration = ++batchGeneration;
    batchBusy = true;
    try {
      const review = await api.analyse.batch(batchText);
      if (requestGeneration !== batchGeneration) return;
      batchReview = review;
      detailsOpen = true;
    } catch (err) {
      if (requestGeneration !== batchGeneration) return;
      analyzeError = {
        title: 'Batch could not be reviewed',
        message: errorMessage(err, 'Paste between 2 and 20 individual public YouTube video or Short URLs.'),
      };
    } finally {
      if (requestGeneration === batchGeneration) batchBusy = false;
    }
  }

  function editBatchURLs() {
    batchGeneration += 1;
    batchBusy = false;
    batchReview = null;
    detailsOpen = false;
    queueMicrotask(() => {
      urlField?.focus();
      urlField?.select();
    });
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
      url = '';
      batchText = '';
      dispatch('goto', 'queue');
    } catch (err) {
      modal.set({ kind: 'error', title: 'Batch could not start', message: errorMessage(err, 'Could not add this batch to the queue.') });
    } finally {
      batchBusy = false;
    }
  }

  function applyVideoDock(summary: InfoSummary) {
    resetDock();
    preview = summary;
    const recommended = summary.plans.find((plan) => plan.recommended) ?? summary.plans[0];
    selectedPlanId = recommended?.id ?? '';
    tab = recommended?.kind ?? 'video';
    detailsOpen = false;
  }

  function applyPlaylistDock(summary: PlaylistSummary) {
    resetDock();
    playlist = summary;
    selectedItems = new Set(summary.entries.filter((entry) => entry.available).map((entry) => entry.index));
    rangeStart = summary.entries[0]?.index ? String(summary.entries[0].index) : '1';
    rangeEnd = summary.entries.at(-1)?.index ? String(summary.entries.at(-1)!.index) : String(summary.entryCount);
    detailsOpen = false;
    rangeWarn = false;
  }

  async function analyzeTarget(target: UrlCheckResult, requestGeneration: number) {
    if (requestGeneration !== analysisGeneration) return;
    resetDock();
    if (target.kind === 'playlist') {
      const canonicalURL = target.playlistUrl!;
      const summary = await api.analyse.playlist(canonicalURL);
      if (requestGeneration !== analysisGeneration) return;
      if (!linkedPlaylist) url = canonicalURL;
      applyPlaylistDock(summary);
    } else {
      const canonicalURL = target.videoUrl!;
      const summary = await api.analyse.url(canonicalURL);
      if (requestGeneration !== analysisGeneration) return;
      if (!linkedPlaylist) url = canonicalURL;
      applyVideoDock(summary);
    }
  }

  function prefetchLinkedPlaylist(accepted: UrlCheckResult, requestGeneration: number) {
    if (!accepted.playlistUrl) return;
    scopePlaylistTask = api.analyse.playlist(accepted.playlistUrl).then((summary) => {
      if (requestGeneration === analysisGeneration) scopePlaylist = summary;
      return summary;
    });
  }

  async function ensureLinkedPlaylist(requestGeneration: number): Promise<PlaylistSummary> {
    if (scopePlaylist) return scopePlaylist;
    if (scopePlaylistTask) {
      try {
        const summary = await scopePlaylistTask;
        if (requestGeneration !== analysisGeneration) throw new Error('stale');
        scopePlaylist = summary;
        return summary;
      } catch (err) {
        scopePlaylistTask = null;
        if (requestGeneration !== analysisGeneration) throw err;
      }
    }
    if (!linkedPlaylist?.playlistUrl) throw new Error('Playlist URL missing');
    const summary = await api.analyse.playlist(linkedPlaylist.playlistUrl);
    if (requestGeneration !== analysisGeneration) throw new Error('stale');
    scopePlaylist = summary;
    return summary;
  }

  function chooseScopeVideo() {
    if (!scopeVideo) return;
    scopeChoice = null;
    applyVideoDock(scopeVideo);
  }

  async function chooseScopePlaylist() {
    if (!linkedPlaylist?.playlistUrl) return;
    if (scopePlaylist) {
      scopeChoice = null;
      applyPlaylistDock(scopePlaylist);
      return;
    }
    const requestGeneration = analysisGeneration;
    scopeChoice = null;
    await withBusy(async () => {
      const summary = await ensureLinkedPlaylist(requestGeneration);
      applyPlaylistDock(summary);
    }, requestGeneration);
  }

  function cancelScope() {
    scopeChoice = null;
    scopeVideo = null;
    scopePlaylist = null;
    scopePlaylistTask = null;
    linkedPlaylist = null;
    scopeFocus = 'playlist';
  }

  function swapLinkedScope() {
    if (!linkedPlaylist) return;
    if (preview) void chooseScopePlaylist();
    else if (playlist) chooseScopeVideo();
  }

  function onHomeKey(event: KeyboardEvent) {
    if (!scopeChoice) return;
    if (event.key === 'Escape') {
      event.preventDefault();
      cancelScope();
      return;
    }
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      scopeFocus = event.key === 'ArrowDown' ? 'playlist' : 'video';
      return;
    }
    if (event.key === 'Enter') {
      if (event.target instanceof HTMLElement && event.target.closest('.scard')) return;
      event.preventDefault();
      if (scopeFocus === 'playlist') void chooseScopePlaylist();
      else chooseScopeVideo();
    }
  }

  async function analyze() {
    const submittedURL = url.trim();
    if (!submittedURL) return;
    const requestGeneration = ++analysisGeneration;
    busy = true;
    clearLinkContext();
    try {
      const accepted = await api.validation.url(submittedURL);
      if (requestGeneration !== analysisGeneration) return;
      if (accepted.kind === 'video_playlist') {
        linkedPlaylist = accepted;
        const summary = await api.analyse.url(accepted.videoUrl!);
        if (requestGeneration !== analysisGeneration) return;
        scopeVideo = summary;
        scopeChoice = accepted;
        scopeFocus = 'playlist';
        prefetchLinkedPlaylist(accepted, requestGeneration);
      } else {
        await analyzeTarget(accepted, requestGeneration);
      }
    } catch (err) {
      if (requestGeneration !== analysisGeneration) return;
      analyzeError = {
        title: 'Unsupported URL',
        message: errorMessage(err, 'VidStow could not extract information from this URL. Make sure it is a valid, publicly accessible YouTube video, Short, or playlist.'),
      };
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
      analyzeError = {
        title: 'Could not analyze link',
        message: errorMessage(err, 'Could not analyze this link.'),
      };
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

  function setTab(kind: 'video' | 'audio') {
    tab = kind;
    const compatible = plans.find((plan) => plan.kind === kind && plan.available && plan.recommended)
      ?? plans.find((plan) => plan.kind === kind && plan.available);
    selectedPlanId = compatible?.id ?? '';
  }

  function setPlaylistTab(kind: string) {
    playlistTab = kind === 'audio' ? 'audio' : 'video';
  }

  function setBatchTab(kind: string) {
    batchTab = kind === 'audio' ? 'audio' : 'video';
  }

  function setPlaylistFormat(id: string) {
    if (playlistTab === 'audio') audioChoice = id;
    else playlistQuality = id as Quality;
  }

  function setBatchFormat(id: string) {
    if (batchTab === 'audio') batchAudioChoice = id;
    else batchQuality = id as Quality;
  }

  function planDetail(plan: OutputPlan) {
    return [plan.container, plan.kind === 'video' ? plan.videoCodec : '', plan.audioCodec].filter(Boolean).join(' · ');
  }

  function toggle(index: number) {
    const entry = playlist?.entries.find((item) => item.index === index);
    if (!entry?.available) return;
    const next = new Set(selectedItems);
    if (next.has(index)) next.delete(index);
    else next.add(index);
    selectedItems = next;
  }

  function selectAll() {
    selectedItems = new Set(playlist?.entries.filter((entry) => entry.available).map((entry) => entry.index) ?? []);
    rangeStart = String(playlistFirstIndex);
    rangeEnd = String(playlistLastIndex);
    rangeWarn = false;
  }

  function clearSelection() {
    selectedItems = new Set();
    rangeStart = '';
    rangeEnd = '';
    rangeWarn = false;
  }

  function applyRange() {
    if (!playlist) return;
    const start = Number(rangeStart);
    const end = Number(rangeEnd);
    if (!Number.isInteger(start) || !Number.isInteger(end) || start < playlistFirstIndex || end < playlistFirstIndex || start > playlistLastIndex || end > playlistLastIndex) {
      rangeWarn = true;
      return;
    }
    rangeWarn = false;
    const low = Math.min(start, end);
    const high = Math.max(start, end);
    selectedItems = new Set(
      playlist.entries
        .filter((entry) => entry.available && entry.index >= low && entry.index <= high)
        .map((entry) => entry.index),
    );
  }

  function joinSavePath(root: string, name: string) {
    const sep = /\\/.test(root) && !root.includes('/') ? '\\' : '/';
    return `${root.replace(/[\\/]+$/, '')}${sep}${name}`;
  }

  function requireFFmpeg(message: string) {
    modal.set({
      kind: 'ffmpeg-missing',
      title: 'FFmpeg Required',
      message,
      actions: [{ label: 'Open Settings', primary: true, action: () => dispatch('goto', 'settings') }],
    });
  }

  function pasteAnotherLink() {
    analyzeError = null;
    queueMicrotask(() => {
      urlField?.focus();
      urlField?.select();
    });
  }

  function fillExample(kind: 'video' | 'playlist' | 'batch') {
    if (kind === 'video') applyUrl('https://www.youtube.com/watch?v=jNQXAC9IVRw');
    else if (kind === 'playlist') applyUrl('https://www.youtube.com/playlist?list=PLPTV0NXA_ZSgsLAr8YCgCwhPIJNNtexWu');
    else applyUrl('https://www.youtube.com/watch?v=jNQXAC9IVRw\nhttps://www.youtube.com/watch?v=aqz-KE-bpKQ\nhttps://www.youtube.com/watch?v=1PZNsDFItl4');
    queueMicrotask(() => {
      urlField?.focus();
      const node = urlField;
      if (!node) return;
      node.setSelectionRange(node.value.length, node.value.length);
      syncFieldHeight();
    });
  }

  async function enqueueVideo() {
    if (!preview || !selectedPlan || !folder) return;
    if (selectedPlan.requiresFfmpeg && !$ffmpeg.available) {
      requireFFmpeg('This output needs FFmpeg for merging or conversion. Install FFmpeg, set its path in Settings, or choose an original audio option.');
      return;
    }
    const start = async () => {
      try {
        await api.jobs.start({
          url: preview!.url,
          videoId: preview!.videoId,
          title: preview!.title,
          channel: preview!.channel,
          planId: selectedPlan!.id,
          outputDir: folder,
          duration: preview!.duration,
          thumbnail: preview!.thumbnail,
        });
        showBanner('success', 'Queued for download');
        clearAnalysis();
        dispatch('goto', 'queue');
      } catch (err) {
        modal.set({ kind: 'error', title: 'Download could not start', message: errorMessage(err, 'Could not start this download.') });
      }
    };
    if ($settings.confirmBeforeDownload) {
      modal.set({
        kind: 'confirm',
        title: 'Add this download?',
        message: `${selectedPlan.label} · ${selectedPlan.container}${selectedPlan.approxBytes ? ` · about ${formatBytes(selectedPlan.approxBytes)}` : ''}`,
        actions: [{ label: 'Download', primary: true, action: start }],
      });
      return;
    }
    await start();
  }

  async function enqueuePlaylist() {
    if (!playlist || !selectedItems.size || playlistBusy) return;
    if (!folder) {
      showBanner('warning', 'Choose a download folder before adding this playlist.');
      return;
    }
    const quality: Quality = playlistTab === 'audio' ? 'audio' : playlistQuality;
    const audioBitrate = playlistTab === 'audio' && audioChoice !== 'original' ? Number(audioChoice) : 0;
    if (audioBitrate && !$ffmpeg.available) {
      requireFFmpeg('MP3 conversion needs FFmpeg. Choose original audio or configure FFmpeg.');
      return;
    }
    const start = async () => {
      if (playlistBusy) return;
      playlistBusy = true;
      try {
        const result = await api.jobs.startPlaylist({
          url: playlist!.url,
          playlistId: playlist!.id,
          quality,
          audioBitrate,
          selectedItems: [...selectedItems].sort((a, b) => a - b),
        });
        const admittedLabel = `${result.admitted} ${result.admitted === 1 ? 'video' : 'videos'}`;
        if (result.skipped) {
          const skippedLabel = `${result.skipped} selected ${result.skipped === 1 ? 'video' : 'videos'}`;
          showBanner('warning', `Added ${admittedLabel} to queue. ${skippedLabel} could not be downloaded.`);
        } else {
          showBanner('success', `Added ${admittedLabel} to queue`);
        }
        dispatch('goto', 'queue');
      } catch (err) {
        modal.set({ kind: 'error', title: 'Playlist could not start', message: errorMessage(err, 'Could not add this playlist to the queue.') });
      } finally {
        playlistBusy = false;
      }
    };
    if (selectedItems.size > 100 || $settings.confirmBeforeDownload) {
      modal.set({
        kind: 'confirm',
        title: 'Add this playlist?',
        message: `${selectedItems.size} videos will be added to the queue.`,
        actions: [{ label: downloadVideosLabel(selectedItems.size), primary: true, action: start }],
      });
      return;
    }
    await start();
  }
</script>

<svelte:window on:keydown={onHomeKey} />

{#snippet searchMark()}
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m20.5 20.5-3.8-3.8"/></svg>
{/snippet}

{#snippet downloadMark()}
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/></svg>
{/snippet}

<section class="page home" class:fill={hasDock} aria-label="Home">
  <form class="composer" on:submit|preventDefault={submitPaste}>
    <div class="fieldwrap">
      <label class="visually-hidden" for="video-url">YouTube video, Short, or playlist URL</label>
      <textarea
        id="video-url"
        bind:this={urlField}
        value={url}
        on:input={updatePaste}
        on:change={updatePaste}
        on:keydown={onUrlKey}
        placeholder="Paste a YouTube URL"
        autocomplete="off"
        spellcheck="false"
        rows="1"
      ></textarea>
      <button class="dbtn query" type="submit" disabled={busy || batchBusy || playlistBusy || !url.trim() || !!scopeChoice} aria-busy={busy || batchBusy || playlistBusy}>
        {#if busy || batchBusy || playlistBusy}
          <span class="query-spin" aria-hidden="true"></span>
        {:else}
          {@render searchMark()}
        {/if}
        Analyze
      </button>
    </div>
  </form>

  {#if !hasDock && !analyzeError && !(busy || batchBusy)}
    <p class="hint">
      One link, a playlist, or a handful — the link decides.
      <span class="tries">
        Try
        {#each TRY_CHIPS as chip}
          <button type="button" class="try" on:click={() => fillExample(chip.kind)}>{chip.label}</button>
        {/each}
      </span>
    </p>
  {:else if (busy || batchBusy) && !hasDock && !analyzeError}
    <div class="skel" aria-busy="true">Reading…</div>
  {:else if analyzeError}
    <div class="errslot" role="alert">
      <b>{analyzeError.title}</b>
      <span>{analyzeError.message}</span>
      <button type="button" class="dbtn query" on:click={pasteAnotherLink}>Paste another link</button>
    </div>
  {/if}

  {#if scopeChoice && scopeVideo}
    <div class="sdialog" role="group" aria-labelledby="scope-title">
      <b id="scope-title">This link includes a playlist</b>
      <span class="dsub">Choose what to download.</span>
      <button
        type="button"
        class="scard"
        class:focus={scopeFocus === 'video'}
        aria-pressed={scopeFocus === 'video'}
        on:click={chooseScopeVideo}
        on:focus={() => (scopeFocus = 'video')}
      >
        {#if scopeVideo.thumbnail}
          <img src={scopeVideo.thumbnail} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />
        {:else}
          <span class="sthumb"></span>
        {/if}
        <span>
          <span class="sc-k">Video</span>
          <span class="sc-t">{scopeVideo.title}</span>
          {#if scopeVideoMeta}<span class="sc-m">{scopeVideoMeta}</span>{/if}
        </span>
      </button>
      <button
        type="button"
        class="scard pri"
        class:focus={scopeFocus === 'playlist'}
        aria-pressed={scopeFocus === 'playlist'}
        on:click={() => void chooseScopePlaylist()}
        on:focus={() => (scopeFocus = 'playlist')}
      >
        <span class="sthumb">
          {#if scopePlaylist?.thumbnail}<img src={scopePlaylist.thumbnail} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />{/if}
          {#if scopePlaylist}<span class="scnt">{scopePlaylist.entryCount}</span>{/if}
        </span>
        <span>
          <span class="sc-k">Playlist</span>
          <span class="sc-t">{scopePlaylist?.title || 'Full playlist'}</span>
          <span class="sc-m">{scopePlaylistMeta}</span>
        </span>
      </button>
      <div class="sdlg-foot">Esc to cancel</div>
    </div>
  {:else if playlist}
    {@const policy = playlistPolicyCopy()}
    <div class="playlist-pane">
    <section class="dock" aria-label="Playlist">
      <div class="drow drow2">
        <div class="thumb">
          {#if playlist.thumbnail}<img src={playlist.thumbnail} alt="" referrerpolicy="no-referrer" on:error={hideBrokenImage} />{/if}
        </div>
        <div class="dmain dmain2">
          <div class="dtop">
            <b title={playlist.title}>{playlist.title}</b>
          </div>
          <span class="dmeta">{playlistMeta}</span>
          {#if linkedSwap && scopeVideo}
            <button type="button" class="swapline" on:click={swapLinkedScope}>
              Pasted video: <b>{scopeVideo.title}{#if scopeVideo.duration} · {scopeVideo.duration}{/if}</b>
              <span class="arr" aria-hidden="true">→</span>
            </button>
          {/if}
        </div>
        <div class="dformat">
          <FormatPicker label="Type" options={KIND_OPTIONS} value={playlistTab} onChange={setPlaylistTab} />
          <div class="chips" role="radiogroup" aria-label="Format">
            {#each playlistPlanOptions as option (option.id)}
              <button
                type="button"
                class="seg"
                class:on={playlistFormatValue === option.id}
                role="radio"
                aria-checked={playlistFormatValue === option.id}
                on:click={() => setPlaylistFormat(option.id)}
              >{option.label}</button>
            {/each}
          </div>
        </div>
      </div>
      <button type="button" class="ddisc has" aria-expanded={detailsOpen} on:click={() => detailsOpen = !detailsOpen}>
        <span class="chev">▸</span>
        <span class="dlnk">{policy.link}</span>
        <span class="dp">{policy.detail}</span>
        {#if playlistAtCap}<span class="cov part">VidStow can review up to {PLAYLIST_ADMIT_CAP} videos from a playlist.</span>{/if}
      </button>
      <footer class="dfoot">
        <span class="dleft">
          <button type="button" class="dbtn" on:click={pickFolder}>Change</button>
          <span class="dpath" title={playlistSavePath || folder}>{playlistSavePath || (folder ? folder : 'Choose a download folder')}</span>
        </span>
        {#if !detailsOpen}
          <button type="button" class="dbtn pri" on:click={enqueuePlaylist} disabled={!selectedItems.size || !folder || playlistBusy} aria-busy={playlistBusy}>
            {#if playlistBusy}<span class="query-spin" aria-hidden="true"></span>Adding…{:else if selectedItems.size}{@render downloadMark()}{downloadVideosLabel(selectedItems.size)}{:else}Nothing selected{/if}
          </button>
        {/if}
      </footer>
    </section>

    {#if detailsOpen}
      <div class="eplist">
        <div class="ephead">
          <span>{selectedItems.size} of {availableCount} selected</span>
          <span class="rng">
            <button type="button" on:click={selectAll}>All</button>
            <button type="button" on:click={clearSelection}>None</button>
            <form class="eprange" on:submit|preventDefault={applyRange}>
              Range
              <input type="number" min={playlistFirstIndex} max={playlistLastIndex} step="1" inputmode="numeric" bind:value={rangeStart} aria-label="Range start" placeholder={String(playlistFirstIndex)} />
              <span>–</span>
              <input type="number" min={playlistFirstIndex} max={playlistLastIndex} step="1" inputmode="numeric" bind:value={rangeEnd} aria-label="Range end" placeholder={String(playlistLastIndex)} />
              <button type="submit">Apply</button>
              {#if rangeWarn}<span class="rwarn">Enter positions from {playlistFirstIndex} to {playlistLastIndex}</span>{/if}
            </form>
          </span>
        </div>
        <div class="epscroll" role="list">
          {#each playlist.entries as entry (entry.index)}
            <button
              type="button"
              class="eprow"
              class:sel={selectedItems.has(entry.index)}
              class:unsel={!selectedItems.has(entry.index)}
              class:unavailable={!entry.available}
              aria-pressed={selectedItems.has(entry.index)}
              disabled={!entry.available}
              title={entry.title}
              on:click={() => toggle(entry.index)}
            >
              <span class="chk" aria-hidden="true"></span>
              <span class="eptitle"><b>{entry.title}</b>{#if !entry.available}<span class="epnote">Unavailable</span>{/if}</span>
              {#if entry.duration}<span class="epd">{entry.duration}</span>{/if}
            </button>
          {:else}
            <div class="empty-list">No videos in this playlist.</div>
          {/each}
        </div>
        <div class="epcommit">
          <span>{selectedItems.size} selected · {policy.detail}</span>
          <button type="button" class="dbtn pri" on:click={enqueuePlaylist} disabled={!selectedItems.size || !folder || playlistBusy} aria-busy={playlistBusy}>
            {#if playlistBusy}<span class="query-spin" aria-hidden="true"></span>Adding…{:else if selectedItems.size}{@render downloadMark()}{downloadVideosLabel(selectedItems.size)}{:else}Nothing selected{/if}
          </button>
        </div>
      </div>
    {/if}
    </div>
  {:else if preview}
    <section class="dock" aria-label="Video">
      <div class="drow drow2 v2">
        <div class="thumb thumbnail">
          {#if preview.thumbnail}<img src={preview.thumbnail} alt="" referrerpolicy="no-referrer" />{/if}
          {#if preview.duration}<span>{preview.duration}</span>{/if}
        </div>
        <div class="dmain dmain2">
          <div class="dtop">
            <b title={preview.title}>{preview.title}</b>
          </div>
          <span class="dmeta">
            {preview.channel || 'YouTube'}{#if preview.mediaType === 'short'} · <em>Short</em>{/if}
            · {preview.duration || 'Duration unavailable'}{preview.viewCount ? ` · ${formatViewCount(preview.viewCount)} views` : ''}
          </span>
          {#if selectedPlan && (planDetail(selectedPlan) || planSizeCopy(selectedPlan))}
            <div class="dplan">
              {#if planDetail(selectedPlan)}<span class="dcodec">{planDetail(selectedPlan)}</span>{/if}
              {#if planSizeCopy(selectedPlan)}<span class="dcost">{planSizeCopy(selectedPlan)}</span>{/if}
            </div>
          {/if}
          {#if linkedSwap}
            <button type="button" class="swapline" on:click={swapLinkedScope}>
              Part of playlist:{' '}
              <b>{scopePlaylist ? `${scopePlaylist.title} · ${scopePlaylist.entryCount} videos` : 'this list'}</b>
              <span class="arr" aria-hidden="true">→</span>
            </button>
          {/if}
        </div>
        {#if kindOptions.length}
          <div class="dformat">
            <FormatPicker label="Type" options={kindOptions} value={tab} onChange={(kind) => setTab(kind === 'audio' ? 'audio' : 'video')} />
            {#if videoPlanOptions.length}
              <div class="chips" role="radiogroup" aria-label="Format">
                {#each videoPlanOptions as plan (plan.id)}
                  <button
                    type="button"
                    class="seg"
                    class:on={selectedPlanId === plan.id}
                    role="radio"
                    aria-checked={selectedPlanId === plan.id}
                    on:click={() => selectedPlanId = plan.id}
                  >{plan.label}</button>
                {/each}
              </div>
            {:else}
              <span class="dmeta">No {tab} outputs were reported for this video.</span>
            {/if}
          </div>
        {:else}
          <span class="dformat dmeta">No outputs were reported for this video.</span>
        {/if}
      </div>
      <footer class="dfoot">
        <span class="dleft">
          <button type="button" class="dbtn" on:click={pickFolder}>Change</button>
          <span class="dpath" title={folder}>{folder ? folder : 'Choose a download folder'}</span>
        </span>
        <button type="button" class="dbtn pri" on:click={enqueueVideo} disabled={!selectedPlan || !folder}>{@render downloadMark()}Download</button>
      </footer>
    </section>
  {:else if batchReview}
    <section class="dock" aria-label="Batch">
      <div class="drow drow2 v2">
        <div class="thumb count">{batchReview.counts.pasted}</div>
        <div class="dmain dmain2">
          <div class="dtop">
            <b>Batch of public videos</b>
          </div>
          <span class="dmeta" aria-live="polite">{batchReviewSummary(batchReview)}</span>
          {#if !batchTokenValid}<span class="dmeta expired" role="alert">This review expired. Edit the lines and review them again.</span>{/if}
          <div class="dpolhint">
            {#if batchTab === 'audio' && batchAudioChoice !== 'original'}MP3 conversion requires FFmpeg · {/if}
            <button type="button" class="tlink" on:click={() => detailsOpen = !detailsOpen}>{batchReview.items.length} titles</button>
          </div>
        </div>
        <div class="dformat">
          <FormatPicker label="Type" options={KIND_OPTIONS} value={batchTab} onChange={setBatchTab} />
          <div class="chips" role="radiogroup" aria-label="Format">
            {#each batchPlanOptions as option (option.id)}
              <button
                type="button"
                class="seg"
                class:on={batchFormatValue === option.id}
                role="radio"
                aria-checked={batchFormatValue === option.id}
                on:click={() => setBatchFormat(option.id)}
              >{option.label}</button>
            {/each}
          </div>
        </div>
      </div>
      {#if detailsOpen}
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
      {/if}
      <footer class="dfoot">
        <span class="dleft">
          <button type="button" class="dbtn" on:click={pickFolder} disabled={batchBusy}>Change</button>
          <span class="dpath" title={folder}>{folder ? folder : 'Choose a download folder'}</span>
        </span>
        <div class="dacts">
          <button type="button" class="dbtn" on:click={editBatchURLs} disabled={batchBusy}>Edit URLs</button>
          <button type="button" class="dbtn pri" on:click={enqueueBatch} disabled={!batchCanStart}>
            {@render downloadMark()}{downloadVideosLabel(batchReadyCount)}
          </button>
        </div>
      </footer>
    </section>
  {/if}
</section>

<style>
  .page.home {
    gap: 12px;
  }
  .page.fill {
    height: 100%;
    min-height: 0;
    overflow: hidden;
    padding-bottom: 20px;
  }
  .playlist-pane {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    gap: 8px;
  }
  .composer {
    width: 100%;
    flex-shrink: 0;
  }
  .fieldwrap {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 42px;
    padding: 0 6px 0 14px;
    border: 1px solid var(--border-default);
    border-radius: 10px;
    background: var(--surface-base);
    transition: border-color 120ms ease;
  }
  .fieldwrap:focus-within { border-color: var(--accent-500); }
  .fieldwrap textarea {
    flex: 1;
    width: auto;
    min-width: 0;
    height: auto;
    min-height: 28px;
    max-height: 96px;
    padding: 6px 0;
    border: 0;
    border-radius: 0;
    background: none;
    box-shadow: none;
    outline: none;
    resize: none;
    font-family: var(--font-mono);
    font-size: 13px;
    line-height: 1.4;
    color: var(--text-secondary);
    overflow-wrap: anywhere;
    word-break: break-all;
  }
  .fieldwrap:focus-within textarea {
    color: var(--text-primary);
  }
  .fieldwrap textarea::placeholder {
    color: var(--text-muted);
  }
  .fieldwrap textarea:focus,
  .fieldwrap textarea:focus-visible {
    border: 0;
    box-shadow: none;
    outline: none;
    outline-offset: 0;
    background: none;
  }
  .fieldwrap .dbtn { flex-shrink: 0; align-self: center; }
  .fieldwrap .dbtn.query { gap: 5px; }
  .fieldwrap .dbtn.query svg,
  .fieldwrap .dbtn.query .query-spin { width: 12px; height: 12px; flex-shrink: 0; }
  .query-spin {
    box-sizing: border-box;
    border: 1.5px solid currentColor;
    border-right-color: transparent;
    border-radius: 99px;
    animation: query-spin 0.7s linear infinite;
  }
  @keyframes query-spin { to { transform: rotate(360deg); } }
  @media (prefers-reduced-motion: reduce) {
    .query-spin { animation: none; border-right-color: currentColor; opacity: 0.45; }
  }

  .hint {
    width: min(780px, 100%);
    margin: 0 auto;
    padding: 14px 4px 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    color: var(--text-muted);
    font-size: 13px;
    text-align: center;
    line-height: 1.7;
  }
  .tries {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    gap: 6px;
    font-size: 12px;
  }
  .try {
    display: inline-flex;
    align-items: center;
    height: 24px;
    padding: 0 9px;
    border: 1px solid var(--border-default);
    border-radius: 7px;
    background: var(--surface-raised);
    color: var(--text-secondary);
    font: inherit;
    font-size: 12px;
    line-height: 1;
    cursor: pointer;
    transition: border-color 120ms ease, color 120ms ease, background 120ms ease;
  }
  .try:hover {
    border-color: var(--border-strong);
    color: var(--text-primary);
  }

  .skel {
    width: min(780px, 100%);
    margin: 0 auto;
    padding: 14px;
    border: 1px dashed var(--border-default);
    border-radius: 10px;
    color: var(--text-muted);
    font-size: 12px;
  }
  .errslot {
    width: min(780px, 100%);
    margin: 0 auto;
    padding: 14px 16px;
    border: 1px solid rgba(239, 68, 68, 0.45);
    border-radius: 10px;
    background: rgba(239, 68, 68, 0.06);
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  .errslot b { font-size: 13px; font-weight: 600; color: #FCA5A5; }
  .errslot span { color: var(--text-secondary); font-size: 12px; }
  .composer + .errslot,
  .composer + .skel,
  .composer + .sdialog { width: 100%; }

  .sdialog {
    padding: 16px 18px;
    border: 1px solid var(--border-default);
    border-radius: 10px;
    background: var(--surface-raised);
  }
  .sdialog b { display: block; font-size: 13px; font-weight: 600; }
  .sdialog .dsub { display: block; margin-top: 3px; color: var(--text-secondary); font-size: 12px; }
  .scard {
    display: grid;
    grid-template-columns: 96px minmax(0, 1fr);
    gap: 14px;
    align-items: center;
    width: 100%;
    margin-top: 8px;
    padding: 10px 12px;
    border: 1px solid var(--border-default);
    border-radius: 10px;
    background: var(--surface-base);
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .scard:hover { background: var(--surface-raised); border-color: var(--border-strong); }
  .scard img { width: 96px; height: 54px; border-radius: 6px; object-fit: cover; display: block; }
  .scard .sc-k { display: block; margin-bottom: 4px; color: var(--text-muted); font-size: 9.5px; font-weight: 650; letter-spacing: 0.08em; text-transform: uppercase; }
  .scard .sc-t { display: block; overflow: hidden; color: var(--text-primary); font-size: 13px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
  .scard .sc-m { display: block; margin-top: 3px; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; }
  .scard .sthumb { position: relative; display: block; width: 96px; height: 54px; border-radius: 6px; background: var(--surface-sunken); }
  .scard .sthumb img { position: relative; z-index: 1; width: 100%; height: 100%; }
  .scard .sthumb::before,
  .scard .sthumb::after { content: ''; position: absolute; inset: 0; border: 1px solid var(--border-default); border-radius: 6px; background: var(--surface-base); }
  .scard .sthumb::before { transform: translate(5px, 5px); }
  .scard .sthumb::after { transform: translate(2.5px, 2.5px); }
  .scard .scnt {
    position: absolute;
    z-index: 2;
    right: 3px;
    bottom: 3px;
    padding: 1px 5px;
    border-radius: 4px;
    background: rgba(0, 0, 0, 0.78);
    color: #fff;
    font-family: var(--font-mono);
    font-size: 10px;
    font-weight: 600;
  }
  .scard.pri { border-color: var(--accent-500); }
  .scard.pri:hover { border-color: #60A5FA; }
  .scard.focus { box-shadow: 0 0 0 2px var(--accent-ring); }
  .sdlg-foot { display: flex; justify-content: flex-end; margin-top: 10px; color: var(--text-muted); font-size: 10.5px; }
  .swapline {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    margin: 7px -8px 0;
    padding: 4px 8px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--text-muted);
    font: inherit;
    font-size: 11px;
    text-align: left;
    cursor: pointer;
  }
  .swapline:hover { background: var(--surface-base); }
  .swapline b { overflow: hidden; color: #93C5FD; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
  .swapline:hover b { text-decoration: underline; }
  .swapline .arr { flex-shrink: 0; color: var(--text-muted); font-size: 10px; }

  .dock {
    display: flex;
    flex-direction: column;
    min-height: 0;
    border: 1px solid var(--border-default);
    border-radius: 10px;
    background: var(--surface-raised);
    overflow: visible;
  }
  .drow {
    display: grid;
    grid-template-columns: 96px minmax(0, 1fr);
    gap: 14px;
    align-items: center;
    padding: 12px 14px 0;
  }
  .drow2 {
    display: grid;
    grid-template-columns: 160px minmax(0, 1fr) auto;
    grid-template-areas: 'thumb identity format';
    gap: 10px 14px;
    align-items: start;
    padding: 16px 16px 0;
  }
  .drow2 .thumb {
    grid-area: thumb;
    width: 160px;
    height: 90px;
    max-height: 90px;
    aspect-ratio: 16 / 9;
    align-self: start;
  }
  .thumb, .thumbnail {
    overflow: hidden;
    border-radius: var(--r-sm);
    background: var(--surface-sunken);
  }
  .thumb {
    width: 96px;
    aspect-ratio: 16 / 9;
    position: relative;
  }
  .thumb.count {
    display: grid;
    place-items: center;
    color: var(--accent-400);
    font-weight: 700;
    font-size: 16px;
    border: 1px solid var(--border-default);
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
  .thumb img, .thumbnail img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    max-height: 100%;
    object-fit: cover;
    object-position: center;
    display: block;
  }
  .dmain { min-width: 0; padding-bottom: 8px; }
  .dmain2 {
    display: flex;
    flex-direction: column;
    grid-area: identity;
    min-width: 0;
    padding-bottom: 8px;
  }
  .dformat {
    display: flex;
    flex-direction: column;
    grid-area: format;
    align-items: stretch;
    gap: 6px;
    width: max-content;
    min-width: 148px;
    max-width: 240px;
    justify-self: end;
    padding-bottom: 8px;
  }
  .dformat :global(.fmt) {
    width: 100%;
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    min-width: 0;
  }
  .seg {
    height: 26px;
    padding: 0 10px;
    border: 1px solid var(--border-default);
    border-radius: 7px;
    background: var(--surface-base);
    color: var(--text-secondary);
    font-size: 12px;
    white-space: nowrap;
    transition: border-color 120ms ease, color 120ms ease, background 120ms ease;
  }
  .seg:hover { border-color: var(--border-strong); color: var(--text-primary); }
  .seg.on {
    border-color: rgba(59, 130, 246, 0.55);
    background: var(--accent-soft);
    color: #93C5FD;
  }
  .drow2.v2 {
    align-items: start;
  }
  .drow2.v2 .thumb {
    width: 160px;
    height: 90px;
    max-height: 90px;
    min-height: 0;
    align-self: start;
  }
  .drow2.v2 .thumb.count { font-size: 28px; }
  .dpolhint {
    margin-top: 4px;
    color: var(--text-muted);
    font-size: 10.5px;
  }
  .dpolhint .tlink {
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-decoration: underline;
    cursor: pointer;
  }
  .dtop {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .dtop b {
    min-width: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    font-size: 15px;
    font-weight: 600;
    line-height: 1.35;
  }
  .dmeta {
    display: block;
    margin-top: 3px;
    color: var(--text-secondary);
    font-size: 13px;
  }
  .dmeta em { font-style: normal; font-weight: 650; }
  .dmeta.expired { color: var(--status-danger); }
  .ddisc {
    display: flex;
    align-items: baseline;
    gap: 10px;
    width: auto;
    margin: 2px 16px 0;
    padding: 5px 8px;
    border-radius: 6px;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 11px;
    text-align: left;
  }
  .ddisc.has { cursor: pointer; }
  .ddisc.has:hover { background: var(--surface-base); }
  .ddisc.has:hover .dlnk { text-decoration: underline; }
  .ddisc .dlnk { color: #93C5FD; white-space: nowrap; }
  .ddisc .cov { color: var(--text-secondary); }
  .ddisc .cov.part { color: #FBBF24; }
  .ddisc .chev {
    font-size: 9px;
    align-self: center;
    transition: transform 120ms ease;
  }
  .ddisc.has[aria-expanded='true'] .chev { transform: rotate(90deg); }
  .dfoot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 8px;
    padding: 8px 10px 8px 16px;
    border-top: 1px solid var(--border-subtle);
    background: var(--surface-base);
  }
  .dpath {
    min-width: 0;
    overflow: hidden;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 11px;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .dleft {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }
  .dleft .dbtn { flex-shrink: 0; }
  .dfoot > .dbtn { flex-shrink: 0; }
  .dacts { display: flex; flex-shrink: 0; align-items: center; gap: 8px; }
  .dplan {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    margin-top: 5px;
    min-width: 0;
    background: none;
  }
  .dcodec,
  .dcost {
    display: block;
    max-width: 100%;
    overflow: hidden;
    padding: 0;
    border: 0;
    background: none;
    box-shadow: none;
    color: var(--text-muted);
    font: inherit;
    font-size: 13px;
    font-variant-numeric: tabular-nums;
    text-overflow: ellipsis;
    white-space: nowrap;
    user-select: none;
    -webkit-user-select: none;
  }
  .dbtn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 24px;
    padding: 0 9px;
    border: 1px solid var(--border-default);
    border-radius: 6px;
    background: var(--surface-raised);
    color: var(--text-primary);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
  }
  .dbtn:hover:not(:disabled) { background: var(--surface-hover); }
  .dbtn:disabled { opacity: 0.4; cursor: default; }
  .dbtn.pri { background: var(--accent-600); border-color: var(--accent-600); color: #fff; }
  .dbtn.pri:hover:not(:disabled) { background: #1D4ED8; }
  .dbtn.pri:has(> svg) { gap: 5px; }
  .dbtn.pri:has(> svg) svg { width: 12px; height: 12px; flex-shrink: 0; }
  .dbtn.query {
    background: #FAFAFA;
    border-color: #FAFAFA;
    color: #09090B;
    font-weight: 600;
  }
  .dbtn.query:hover:not(:disabled) { background: #E4E4E7; border-color: #E4E4E7; }

  .eplist {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
    border: 1px solid var(--border-default);
    border-radius: 10px;
    background: var(--surface-base);
  }
  .ephead {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 10px 12px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border-default);
    background: var(--surface-raised);
    color: var(--text-secondary);
    font-size: 11.5px;
  }
  .rng {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
    color: var(--text-muted);
  }
  .rng > button {
    padding: 0;
    border: 0;
    background: none;
    color: #93C5FD;
    font-size: 11.5px;
    cursor: pointer;
  }
  .rng > button:hover { text-decoration: underline; }
  .eprange {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .eprange input {
    width: 46px;
    height: 22px;
    padding: 0 6px;
    text-align: center;
    font-family: var(--font-mono);
    font-size: 11px;
  }
  .eprange button {
    padding: 0;
    border: 0;
    background: none;
    color: #93C5FD;
    font-size: 11.5px;
    cursor: pointer;
  }
  .eprange button:hover { text-decoration: underline; }
  .rwarn {
    color: #fbbf24;
    font-size: 10.5px;
  }
  .epscroll {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }
  .eprow {
    display: grid;
    grid-template-columns: 14px minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    width: 100%;
    padding: 6px 12px;
    border: 0;
    border-radius: 0;
    background: none;
    color: var(--text-secondary);
    font-size: 12px;
    text-align: left;
    cursor: pointer;
  }
  .eprow + .eprow { border-top: 1px solid var(--border-default); }
  .eprow:hover { background: var(--surface-raised); }
  .eprow.unavailable { cursor: default; }
  .eprow .chk {
    width: 12px;
    height: 12px;
    border: 1px solid var(--border-strong);
    border-radius: 3px;
    flex-shrink: 0;
  }
  .eprow.sel .chk {
    background: var(--accent-600);
    border-color: var(--accent-600);
  }
  .eptitle {
    min-width: 0;
    display: flex;
    align-items: baseline;
  }
  .eprow b {
    overflow: hidden;
    color: var(--text-primary);
    font-size: 12px;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .eprow .epnote { margin-left: 6px; color: var(--text-muted); font-size: 10.5px; }
  .eprow .epd {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 10.5px;
  }
  .eprow.unsel b,
  .eprow.unsel .epd,
  .eprow.unsel .epnote { opacity: 0.5; }
  .epcommit {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 5px 6px 5px 10px;
    border-top: 1px solid var(--border-strong);
    background: var(--surface-raised);
    color: var(--text-secondary);
    font-size: 11.5px;
  }
  .empty-list {
    min-height: 120px;
    display: grid;
    place-items: center;
    color: var(--text-muted);
    font-size: 12.5px;
  }

  .batch-lines {
    display: flex;
    min-height: 0;
    max-height: none;
    flex: 1;
    flex-direction: column;
    overflow: auto;
    border-top: 1px solid var(--border-subtle);
  }
  .batch-line {
    display: grid;
    grid-template-columns: 34px 112px minmax(0, 1fr) minmax(150px, 230px);
    align-items: center;
    gap: var(--sp-3);
    padding: var(--sp-3) var(--sp-4);
    border-bottom: 1px solid var(--border-subtle);
    background: var(--surface-base);
  }
  .batch-line-number { color: var(--text-muted); font-variant-numeric: tabular-nums; text-align: center; }
  .batch-thumbnail {
    width: 112px;
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

  @media (max-width: 860px) {
    .drow { grid-template-columns: 72px 1fr; }
    .drow2,
    .drow2.v2 {
      grid-template-columns: 96px minmax(0, 1fr);
      grid-template-areas:
        'thumb identity'
        'thumb format';
    }
    .drow2 .thumb { width: 96px; height: 54px; }
    .drow2.v2 .thumb { width: 96px; height: 54px; max-height: 54px; min-height: 0; }
    .dformat {
      justify-self: stretch;
      width: auto;
      max-width: none;
    }
  }
  @media (max-width: 720px) {
    .batch-line { grid-template-columns: 28px 72px minmax(0, 1fr); }
    .batch-thumbnail { width: 72px; }
    .batch-line-state { grid-column: 3; align-items: flex-start; text-align: left; }
    .dfoot { flex-direction: column; align-items: stretch; }
    .dacts { width: 100%; justify-content: flex-end; }
  }
</style>
