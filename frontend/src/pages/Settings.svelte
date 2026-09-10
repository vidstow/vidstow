<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { settings, ffmpeg, showBanner, showError } from '../lib/stores.js';
  import { formatEngineVersion } from '../lib/format.js';
  import { MAX_CONCURRENCY, MIN_CONCURRENCY } from '../lib/lifecycle-ui/types.js';
  import { COLLECTION_LANGUAGES } from '../lib/subtitle-languages.js';
  import type { BuildInfo, OutputOptions, Settings } from '../lib/types.js';

  const APP = {
    name: 'VidStow',
    license: 'Apache-2.0',
    source: 'https://github.com/vidstow/vidstow',
    docs: 'https://github.com/vidstow/vidstow#readme',
    tagline: 'A tidy YouTube video, Short, and playlist downloader for the desktop. Built with Go · Wails · Svelte · ytdlp-go · FFmpeg',
  };

  let folder = '';
  let ffmpegPath = '';
  let saving = false;
  let build: BuildInfo = {
    version: 'Loading…', engineVersion: 'Loading…', os: '', architecture: '', goVersion: '',
  };

  onMount(() => {
    folder = $settings.downloadFolder || '';
    ffmpegPath = $settings.ffmpegPath || $ffmpeg.path || '';
    api.app.buildInfo()
      .then((info) => { build = info; })
      .catch((err) => { showError(err, 'Could not read build information'); });
  });
  $: displayedFFmpegPath = ffmpegPath || $ffmpeg.path || '';
  $: concurrency = $settings.downloadConcurrency;
  $: ffmpegVersion = ($ffmpeg.version || '').replace(/^ffmpeg version /i, '').split(/\s+/)[0] || '';
  $: platform = build.os && build.architecture ? `${build.os}/${build.architecture}` : '';

  async function update(next: Settings, message = 'Settings updated') {
    saving = true;
    try {
      const saved = await api.settings.update(next);
      settings.set(saved);
      showBanner('success', message);
    } catch (err) { showError(err, 'Could not save settings'); }
    finally { saving = false; }
  }

  async function updateOutputOptions(patch: Partial<OutputOptions>) {
    await update({ ...$settings, outputOptions: { ...($settings.outputOptions ?? {}), ...patch } }, 'Output defaults updated');
  }

  async function pickFolder() {
    try {
      const path = await api.folder.pick();
      if (!path) return;
      folder = path;
      await update({ ...$settings, downloadFolder: path }, 'Download folder updated');
    } catch (err) { showError(err, 'Could not choose folder'); }
  }

  async function showFolder() {
    if (!folder) return;
    try { await api.fs.reveal(folder); }
    catch (err) { showError(err, 'Could not show the folder in Finder'); }
  }

  async function locateFFmpeg() {
    try {
      const path = await api.ffmpeg.pickPath();
      if (!path) return;
      const status = await api.ffmpeg.configure(path);
      ffmpeg.set(status);
      ffmpegPath = status.path;
      showBanner('success', 'FFmpeg configured');
    } catch (err) { showError(err, 'Could not configure FFmpeg'); }
  }

  async function recheck() {
    try { ffmpeg.set(await api.ffmpeg.probe()); showBanner('info', $ffmpeg.available ? 'FFmpeg is ready' : 'FFmpeg was not found'); }
    catch (err) { showError(err, 'Could not check FFmpeg'); }
  }

  async function copyDiagnostics() {
    try { await api.diagnostics.copy(); showBanner('info', 'Diagnostics copied'); }
    catch (err) { showError(err, 'Could not copy diagnostics'); }
  }

  async function clearDiagnostics() {
    try { await api.diagnostics.clear(); showBanner('info', 'Diagnostic history cleared'); }
    catch (err) { showError(err, 'Could not clear diagnostic history'); }
  }

  async function changeConcurrency(value: number) {
    const next = Math.min(MAX_CONCURRENCY, Math.max(MIN_CONCURRENCY, value));
    if (next === concurrency) return;
    await update({ ...$settings, downloadConcurrency: next });
  }

  async function setAutomaticDiagnostics(value: 'enabled' | 'disabled') {
    try {
      settings.set(await api.settings.setAutomaticDiagnostics(value));
      showBanner('success', value === 'enabled' ? 'Automatic diagnostics enabled' : 'Automatic diagnostics disabled');
    } catch (err) {
      try { settings.set(await api.settings.get()); } catch { /* retain the last known value */ }
      showError(err, 'Could not save the diagnostics preference');
    }
  }

  function open(url: string) {
    if (url) window.runtime?.BrowserOpenURL?.(url);
  }
</script>

<section class="page settings-page" aria-labelledby="settings-title">
  <div class="scol">
    <h1 id="settings-title">Settings</h1>

    <section class="sgroup" aria-labelledby="general-settings-title">
      <h2 id="general-settings-title">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 7.5A1.5 1.5 0 0 1 4.5 6h4.2l1.6 1.8H19.5A1.5 1.5 0 0 1 21 9.3v8.2a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 17.5Z" /></svg>
        <span>General</span>
      </h2>

      <div class="srow">
        <div class="scopy">
          <strong>Default download folder</strong>
          <span class="mono" title={folder}>{folder || 'Not set'}</span>
        </div>
        <div class="sact">
          <button type="button" class="btn sm ghost" disabled={!folder} on:click={showFolder}>Show in Finder</button>
          <button type="button" class="btn sm ghost" on:click={pickFolder}>Change Folder</button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Create a subfolder for each download</strong>
          <span>Places all files for one video together.</span>
        </div>
        <div class="sact">
          <button
            type="button"
            class="switch"
            class:on={$settings.perVideoSubfolder}
            role="switch"
            aria-checked={$settings.perVideoSubfolder}
            aria-label="Create a subfolder for each download"
            on:click={() => update({ ...$settings, perVideoSubfolder: !$settings.perVideoSubfolder })}
          ><span class="switch-thumb"></span></button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Interrupted jobs</strong>
          <span>The download that was in progress continues when VidStow opens. Waiting stays waiting. Paused stays paused.</span>
        </div>
        <div class="sact">
          <span class="sfixed">In progress continues</span>
        </div>
      </div>
    </section>

    <section class="sgroup" aria-labelledby="performance-settings-title">
      <h2 id="performance-settings-title">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 21a9 9 0 1 1 9-9" /><path d="M12 12l5-3" /><circle cx="12" cy="12" r="1.6" /></svg>
        <span>Performance</span>
      </h2>

      <div class="srow">
        <div class="scopy">
          <strong>Maximum concurrent downloads</strong>
          <span>Range: {MIN_CONCURRENCY}–{MAX_CONCURRENCY}; Recommended: 2–4.</span>
        </div>
        <div class="sact">
          <div class="stepper">
            <button type="button" class="btn sm ghost" aria-label="Decrease concurrent downloads" disabled={saving || concurrency <= MIN_CONCURRENCY} on:click={() => changeConcurrency(concurrency - 1)}>−</button>
            <b>{concurrency}</b>
            <button type="button" class="btn sm ghost" aria-label="Increase concurrent downloads" disabled={saving || concurrency >= MAX_CONCURRENCY} on:click={() => changeConcurrency(concurrency + 1)}>+</button>
          </div>
        </div>
      </div>
      {#if concurrency > 4}
        <p class="swarn">More than 4 concurrent downloads may trigger YouTube rate limits (HTTP 429).</p>
      {/if}
    </section>

    <section class="sgroup" aria-labelledby="video-files-settings-title">
      <h2 id="video-files-settings-title">
        <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="5" width="18" height="14" rx="2" /><path d="m10 9 5 3-5 3Z" /></svg>
        <span>Video files</span>
      </h2>

      <div class="srow">
        <div class="scopy">
          <strong>Subtitles on new adds</strong>
          <span>
            {#if $settings.outputOptions?.subtitleMode === 'embed'}
              Writes captions inside the video. The folder still shows one MP4.
            {:else if $settings.outputOptions?.subtitleMode === 'sidecar'}
              Saves a caption file next to the video.
            {:else}
              Pre-selects subtitles for new downloads when the video offers them.
            {/if}
          </span>
        </div>
        <div class="sact">
          <select
            class="sselect"
            aria-label="Default subtitle mode"
            value={$settings.outputOptions?.subtitleMode ?? ''}
            on:change={(e) => updateOutputOptions({ subtitleMode: (e.currentTarget as HTMLSelectElement).value as OutputOptions['subtitleMode'] })}
          >
            <option value="">Off</option>
            <option value="sidecar">Subtitle file</option>
            <option value="embed" disabled={!$ffmpeg.available}>Embed in video</option>
          </select>
        </div>
      </div>

      {#if $settings.outputOptions?.subtitleMode === 'sidecar' || $settings.outputOptions?.subtitleMode === 'embed'}
        <div class="srow">
          <div class="scopy">
            <strong>Default subtitle language</strong>
            <span>Prefills Home. If a playlist or batch video does not have the language on the card, VidStow uses this instead.</span>
          </div>
          <div class="sact">
            <select
              class="sselect"
              aria-label="Default subtitle language"
              value={$settings.outputOptions?.subtitleLanguages?.[0] ?? ''}
              on:change={(e) => {
                const code = (e.currentTarget as HTMLSelectElement).value;
                updateOutputOptions({ subtitleLanguages: code ? [code] : [] });
              }}
            >
              <option value="">English or first available</option>
              {#each COLLECTION_LANGUAGES as language (language.code)}
                <option value={language.code}>{language.name}</option>
              {/each}
            </select>
          </div>
        </div>
      {/if}

      {#if $settings.outputOptions?.subtitleMode === 'sidecar'}
        <div class="srow">
          <div class="scopy">
            <strong>Subtitle file format</strong>
            <span>{$ffmpeg.available ? 'Converted with FFmpeg when needed.' : 'FFmpeg is needed to convert; the original format is kept.'}</span>
          </div>
          <div class="sact">
            <select
              class="sselect"
              aria-label="Default subtitle file format"
              disabled={!$ffmpeg.available}
              value={$settings.outputOptions?.subtitleFormat ?? ''}
              on:change={(e) => updateOutputOptions({ subtitleFormat: (e.currentTarget as HTMLSelectElement).value as OutputOptions['subtitleFormat'] })}
            >
              <option value="">Original</option>
              <option value="srt">SRT</option>
              <option value="vtt">VTT</option>
            </select>
          </div>
        </div>
      {/if}

      <div class="srow">
        <div class="scopy">
          <strong>Include auto-generated captions</strong>
          <span>Uses auto-generated tracks when a language has no manual subtitles.</span>
        </div>
        <div class="sact">
          <button
            type="button"
            class="switch"
            class:on={!!$settings.outputOptions?.subtitleAutoCaptions}
            role="switch"
            aria-checked={!!$settings.outputOptions?.subtitleAutoCaptions}
            aria-label="Include auto-generated captions"
            on:click={() => updateOutputOptions({ subtitleAutoCaptions: !$settings.outputOptions?.subtitleAutoCaptions })}
          ><span class="switch-thumb"></span></button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Embed title &amp; channel details</strong>
          <span>Writes the video's details into the downloaded file.</span>
        </div>
        <div class="sact">
          <button
            type="button"
            class="switch"
            class:on={!!$settings.outputOptions?.embedMetadata}
            role="switch"
            aria-checked={!!$settings.outputOptions?.embedMetadata}
            aria-label="Embed title and channel details"
            disabled={!$ffmpeg.available}
            on:click={() => updateOutputOptions({ embedMetadata: !$settings.outputOptions?.embedMetadata })}
          ><span class="switch-thumb"></span></button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Embed thumbnail artwork</strong>
          <span>Temporarily unavailable. Artwork is not added to downloads.</span>
        </div>
        <div class="sact">
          <button
            type="button"
            class="switch"
            role="switch"
            aria-checked="false"
            aria-label="Embed thumbnail artwork"
            disabled
          ><span class="switch-thumb"></span></button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Embed chapter markers</strong>
          <span>Adds the video's chapters where the format supports them.</span>
        </div>
        <div class="sact">
          <button
            type="button"
            class="switch"
            class:on={!!$settings.outputOptions?.embedChapters}
            role="switch"
            aria-checked={!!$settings.outputOptions?.embedChapters}
            aria-label="Embed chapter markers"
            disabled={!$ffmpeg.available}
            on:click={() => updateOutputOptions({ embedChapters: !$settings.outputOptions?.embedChapters })}
          ><span class="switch-thumb"></span></button>
        </div>
      </div>

      {#if !$ffmpeg.available}
        <p class="swarn">Embedding needs FFmpeg. Install it or set its path under Advanced.</p>
      {/if}
    </section>

    <section class="sgroup" aria-labelledby="advanced-settings-title">
      <h2 id="advanced-settings-title">
        <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="7" y="7" width="10" height="10" rx="1.4" /><path d="M12 3v2.5M12 18.5V21M3 12h2.5M18.5 12H21M6 6l1.8 1.8M16.2 16.2 18 18M18 6l-1.8 1.8M7.8 16.2 6 18" /></svg>
        <span>Advanced</span>
      </h2>

      <div class="srow">
        <div class="scopy">
          <strong>Engine &amp; dependencies</strong>
          <span>
            {#if $ffmpeg.available && ffmpegVersion}
              FFmpeg version {ffmpegVersion} · ready for merging and MP3 conversion.
            {:else}
              FFmpeg is needed to merge video and audio, and to convert to MP3.
            {/if}
          </span>
          <span>ytdlp-go {formatEngineVersion(build.engineVersion)}</span>
        </div>
        <div class="sact">
          <em class="sbadge" class:ok={$ffmpeg.available}>
            {#if $ffmpeg.available}
              <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M3.5 8.5 6.5 11.5 12.5 4.5" /></svg>
              Ready
            {:else}
              Missing
            {/if}
          </em>
          <button type="button" class="btn sm ghost" on:click={recheck}>Recheck Dependencies</button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>FFmpeg path</strong>
          <span class="mono" class:empty={!displayedFFmpegPath} title={displayedFFmpegPath}>{displayedFFmpegPath || 'Not configured'}</span>
        </div>
        <div class="sact">
          <button type="button" class="btn sm ghost" on:click={locateFFmpeg}>Change Path</button>
        </div>
      </div>
    </section>

    <section class="sgroup" aria-labelledby="diagnostics-title">
      <h2 id="diagnostics-title">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M22 12h-4l-3 8-6-16-3 8H2" /></svg>
        <span>Diagnostics</span>
      </h2>
      <div class="srow stack">
        <div class="scopy">
          <strong>Send operational diagnostics</strong>
          <span>When VidStow cannot complete a requested download, send a small sanitized report. Links, paths, and error text stay private.</span>
          <small>
            <button class="slink" type="button" on:click={() => open('https://diagnostics.vidstow.workers.dev/privacy')}>Diagnostics privacy notice ↗</button>
          </small>
        </div>
        <div class="sact" role="radiogroup" aria-label="Automatic diagnostics">
          <button
            type="button"
            class="btn sm ghost"
            class:on={$settings.automaticDiagnostics === 'enabled'}
            aria-pressed={$settings.automaticDiagnostics === 'enabled'}
            on:click={() => setAutomaticDiagnostics('enabled')}
          >Send</button>
          <button
            type="button"
            class="btn sm ghost"
            class:on={$settings.automaticDiagnostics === 'disabled'}
            aria-pressed={$settings.automaticDiagnostics === 'disabled'}
            on:click={() => setAutomaticDiagnostics('disabled')}
          >Don’t send</button>
        </div>
      </div>
      <div class="srow">
        <div class="scopy">
          <strong>Support report</strong>
          <span>Includes app and FFmpeg status plus recent sanitized failures.</span>
        </div>
        <div class="sact">
          <button type="button" class="btn sm ghost" on:click={copyDiagnostics}>Copy diagnostics</button>
          <button type="button" class="btn sm ghost" on:click={clearDiagnostics}>Clear history</button>
        </div>
      </div>
    </section>

    <div class="colophon">
      <div class="cleft">
        <span class="cmark" aria-hidden="true">V</span>
        <div>
          <b>{APP.name}</b>
          <span>{build.version} · {APP.license}{platform ? ` · ${platform}` : ''}</span>
        </div>
      </div>
      <div class="cact">
        <button type="button" class="btn sm ghost" on:click={() => open(APP.source)}>View source</button>
        <button type="button" class="btn sm ghost" on:click={() => open(APP.docs)}>Read the docs</button>
      </div>
      <div class="cbuilt">{APP.tagline}</div>
    </div>
  </div>
</section>

<style>
  .settings-page {
    overflow-y: auto;
    height: 100%;
  }
  .settings-page h1 {
    margin: 0 0 4px;
    font-size: 18px;
    font-weight: 650;
    letter-spacing: -0.02em;
  }
  .scol {
    max-width: 680px;
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
  .sgroup {
    border: 1px solid var(--border-default);
    border-radius: 12px;
    background: var(--surface-raised);
    padding: 2px 20px 10px;
  }
  .sgroup h2 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    padding: 14px 0 12px;
    border-bottom: 1px solid #1F1F23;
    font-size: 11px;
    font-weight: 650;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }
  .sgroup h2 svg {
    width: 14px;
    height: 14px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
    flex-shrink: 0;
  }
  .srow {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
    padding: 14px 0;
    border-top: 1px solid #1F1F23;
    min-height: 56px;
  }
  .sgroup h2 + .srow { border-top: 0; }
  .srow.stack {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
  .scopy { min-width: 0; flex: 1; }
  .scopy strong, .scopy span, .scopy small { display: block; }
  .scopy strong { font-size: 13px; font-weight: 600; }
  .scopy span {
    margin-top: 3px;
    color: var(--text-secondary);
    font-size: 11.5px;
    line-height: 1.45;
  }
  .scopy small {
    margin-top: 3px;
    color: var(--text-muted);
    font-size: 11px;
    line-height: 1.45;
  }
  .scopy .mono {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--font-mono);
    font-size: 11px;
  }
  .scopy .mono.empty { font-family: var(--font-sans); font-style: italic; }
  .sact {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }
  .srow.stack .sact { width: 100%; }
  .sselect {
    height: 28px;
    min-width: 140px;
    padding: 0 26px 0 8px;
    border: 1px solid var(--border-default);
    border-radius: 7px;
    background: var(--surface-input);
    color: var(--text-primary);
    font-family: inherit;
    font-size: 12px;
  }
  .sselect:focus { border-color: var(--accent-500); outline: none; }
  .sselect:disabled { opacity: 0.4; }
  .sfixed {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 11.5px;
    font-weight: 600;
    white-space: nowrap;
  }
  .stepper {
    display: inline-flex;
    align-items: center;
    border: 1px solid var(--border-default);
    border-radius: 8px;
    overflow: hidden;
  }
  .stepper .btn {
    width: 28px;
    padding: 0;
    border: 0;
    border-radius: 0;
  }
  .stepper b {
    font-family: var(--font-mono);
    font-size: 13px;
    min-width: 28px;
    text-align: center;
  }
  .sbadge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 3px 10px;
    border-radius: 99px;
    background: var(--status-danger-soft);
    color: var(--status-danger);
    font-style: normal;
    font-size: 11px;
    font-weight: 650;
  }
  .sbadge svg {
    width: 12px;
    height: 12px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .sbadge.ok {
    background: rgba(34, 197, 94, 0.12);
    color: #4ADE80;
  }
  .swarn {
    margin: 0;
    padding: 0 0 12px;
    color: #FBBF24;
    font-size: 11.5px;
    line-height: 1.45;
  }
  .slink {
    color: #93C5FD;
    text-decoration: underline;
    text-underline-offset: 3px;
    cursor: pointer;
  }
  .switch {
    width: 40px;
    height: 22px;
    padding: 2px;
    border-radius: 99px;
    border: 1px solid var(--border-default);
    background: #27272A;
    flex-shrink: 0;
  }
  .switch-thumb {
    display: block;
    width: 16px;
    height: 16px;
    border-radius: 99px;
    background: #FAFAFA;
    transform: translateX(0);
    transition: transform 120ms ease;
  }
  .switch.on {
    background: var(--accent-600);
    border-color: var(--accent-600);
  }
  .switch.on .switch-thumb { transform: translateX(18px); }
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
  .btn.on {
    background: var(--accent-soft);
    color: #93C5FD;
  }

  .colophon {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 8px 16px;
    align-items: center;
    padding: 10px 14px;
    border: 1px solid var(--border-default);
    border-radius: 12px;
    background: var(--surface-raised);
  }
  .cleft { display: flex; align-items: center; gap: 13px; }
  .cmark {
    width: 32px;
    height: 32px;
    display: grid;
    place-items: center;
    border-radius: 9px;
    background: linear-gradient(180deg, #3B82F6, #1D4ED8);
    font-weight: 700;
    font-size: 15px;
    color: #fff;
    flex-shrink: 0;
  }
  .cleft b { display: block; font-size: 13.5px; font-weight: 650; }
  .cleft span {
    display: block;
    color: var(--text-muted);
    font-size: 11px;
    margin-top: 2px;
    font-family: var(--font-mono);
  }
  .cact { display: flex; gap: 6px; }
  .cbuilt {
    grid-column: 1 / -1;
    padding-top: 8px;
    border-top: 1px solid #1F1F23;
    color: var(--text-muted);
    font-size: 11px;
  }

  @media (max-width: 720px) {
    .srow { flex-direction: column; align-items: flex-start; }
    .sact { width: 100%; justify-content: flex-start; flex-wrap: wrap; }
    .colophon { grid-template-columns: 1fr; }
  }
</style>
