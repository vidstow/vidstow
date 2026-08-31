<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { settings, ffmpeg, showBanner, showError } from '../lib/stores.js';
  import { formatEngineVersion } from '../lib/format.js';
  import { MAX_CONCURRENCY, MIN_CONCURRENCY } from '../lib/lifecycle-ui/types.js';
  import type { BuildInfo, Settings } from '../lib/types.js';

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
  $: engineReady = !!build.engineVersion && build.engineVersion !== 'Loading…';

  async function update(next: Settings, message = 'Settings updated') {
    saving = true;
    try {
      const saved = await api.settings.update(next);
      settings.set(saved);
      showBanner('success', message);
    } catch (err) { showError(err, 'Could not save settings'); }
    finally { saving = false; }
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

    <section class="sgroup" aria-labelledby="downloads-settings-title">
      <h2 id="downloads-settings-title">Downloads</h2>

      <div class="srow">
        <div class="scopy">
          <strong>Default download folder</strong>
          <span class="mono" title={folder}>{folder || 'Not set'}</span>
        </div>
        <div class="sact">
          <button type="button" class="btn sm ghost" disabled={!folder} on:click={showFolder}>Show in Finder</button>
          <button type="button" class="btn sm ghost" on:click={pickFolder}>Change…</button>
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
            class="toggle"
            class:on={$settings.perVideoSubfolder}
            aria-pressed={$settings.perVideoSubfolder}
            on:click={() => update({ ...$settings, perVideoSubfolder: !$settings.perVideoSubfolder })}
          >{$settings.perVideoSubfolder ? 'On' : 'Off'}</button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Confirm before starting downloads</strong>
          <span>Shows the selected output before adding it to the queue.</span>
        </div>
        <div class="sact">
          <button
            type="button"
            class="toggle"
            class:on={$settings.confirmBeforeDownload}
            aria-pressed={$settings.confirmBeforeDownload}
            on:click={() => update({ ...$settings, confirmBeforeDownload: !$settings.confirmBeforeDownload })}
          >{$settings.confirmBeforeDownload ? 'On' : 'Off'}</button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Concurrent downloads</strong>
          <span>How many jobs transfer at once.</span>
        </div>
        <div class="sact">
          <button type="button" class="btn sm ghost" aria-label="Decrease concurrent downloads" disabled={saving || concurrency <= MIN_CONCURRENCY} on:click={() => changeConcurrency(concurrency - 1)}>−</button>
          <b>{concurrency}</b>
          <button type="button" class="btn sm ghost" aria-label="Increase concurrent downloads" disabled={saving || concurrency >= MAX_CONCURRENCY} on:click={() => changeConcurrency(concurrency + 1)}>+</button>
        </div>
      </div>
      {#if concurrency > 4}
        <p class="swarn">More than 4 concurrent downloads may trigger YouTube rate limits (HTTP 429).</p>
      {/if}
    </section>

    <section class="sgroup" aria-labelledby="engine-title">
      <h2 id="engine-title">Engine</h2>

      <div class="srow">
        <div class="scopy">
          <strong>FFmpeg</strong>
          <span>
            {#if $ffmpeg.available && ffmpegVersion}
              Version {ffmpegVersion} · ready for merging and MP3 conversion.
            {:else}
              Needed to merge video and audio, and to convert to MP3.
            {/if}
          </span>
        </div>
        <div class="sact">
          <em class="sbadge" class:ok={$ffmpeg.available}>{$ffmpeg.available ? 'Ready' : 'Missing'}</em>
          <button type="button" class="btn sm ghost" on:click={recheck}>Recheck</button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>FFmpeg path</strong>
          <span class="mono" class:empty={!displayedFFmpegPath} title={displayedFFmpegPath}>{displayedFFmpegPath || 'Not configured'}</span>
        </div>
        <div class="sact">
          <button type="button" class="btn sm ghost" on:click={locateFFmpeg}>Change…</button>
        </div>
      </div>

      <div class="srow">
        <div class="scopy">
          <strong>Engine</strong>
          <span>ytdlp-go {formatEngineVersion(build.engineVersion)}</span>
        </div>
        <div class="sact">
          {#if engineReady}
            <em class="sbadge ok">Ready</em>
          {/if}
        </div>
      </div>
    </section>

    <section class="sgroup" aria-labelledby="diagnostics-title">
      <h2 id="diagnostics-title">Diagnostics</h2>
      <div class="srow">
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
    padding: 18px 18px 28px;
    overflow-y: auto;
    height: 100%;
    box-sizing: border-box;
  }
  .settings-page h1 {
    margin: 0 0 16px;
    font-size: 17px;
    font-weight: 650;
    letter-spacing: -0.02em;
  }
  .scol {
    max-width: 680px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .sgroup {
    border: 1px solid var(--border-default);
    border-radius: 10px;
    background: var(--surface-raised);
    padding: 4px 18px 6px;
  }
  .sgroup h2 {
    margin: 0;
    padding: 12px 0 4px;
    font-size: 11px;
    font-weight: 650;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-muted);
  }
  .srow {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
    padding: 11px 0;
    border-top: 1px solid #1F1F23;
    min-height: 48px;
  }
  .sgroup h2 + .srow { border-top: 0; }
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
  .sact b {
    font-family: var(--font-mono);
    font-size: 13px;
    min-width: 16px;
    text-align: center;
  }
  .sbadge {
    padding: 3px 10px;
    border-radius: 99px;
    background: var(--status-danger-soft);
    color: var(--status-danger);
    font-style: normal;
    font-size: 11px;
    font-weight: 650;
  }
  .sbadge.ok {
    background: rgba(34, 197, 94, 0.12);
    color: #4ADE80;
  }
  .swarn {
    margin: 0;
    padding: 2px 0 10px;
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
  .toggle {
    height: 24px;
    padding: 0 12px;
    border-radius: 99px;
    border: 1px solid var(--border-default);
    font-size: 11px;
    color: var(--text-secondary);
    background: var(--surface-base);
  }
  .toggle.on {
    background: var(--accent-soft);
    border-color: rgba(59, 130, 246, 0.55);
    color: #93C5FD;
  }
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
    padding: 16px 18px;
    border: 1px solid var(--border-default);
    border-radius: 10px;
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
    padding-top: 10px;
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
