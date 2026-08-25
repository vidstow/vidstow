<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { settings, ffmpeg, showBanner, showError } from '../lib/stores.js';
  import type { Settings } from '../lib/types.js';
  import QueueSettingsCard from '../lib/lifecycle-ui/QueueSettingsCard.svelte';

  let folder = '';
  let ffmpegPath = '';
  let saving = false;

  onMount(() => {
    folder = $settings.downloadFolder || '';
    ffmpegPath = $settings.ffmpegPath || $ffmpeg.path || '';
  });
  $: displayedFFmpegPath = ffmpegPath || $ffmpeg.path || '';
  $: concurrency = $settings.downloadConcurrency;
  $: ffmpegVersion = ($ffmpeg.version || '').replace(/^ffmpeg version /i, '').split(/\s+/)[0] || '';

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
    await update({ ...$settings, downloadConcurrency: value });
  }

  async function setAutomaticDiagnostics(value: 'enabled' | 'disabled') {
    try {
      settings.set(await api.settings.setAutomaticDiagnostics(value));
      showBanner('success', value === 'enabled' ? 'Automatic diagnostics enabled' : 'Automatic diagnostics disabled');
    } catch (err) {
      // Disabling remains persisted even if its best-effort local purge reports
      // an error, so refresh instead of leaving the control misleadingly on.
      try { settings.set(await api.settings.get()); } catch { /* retain the last known value */ }
      showError(err, 'Could not save the diagnostics preference');
    }
  }
</script>

<section class="page" aria-labelledby="settings-title">
  <header class="page-header">
    <h1 id="settings-title">Settings</h1>
    <p>Configure downloads, queue behavior, and external tools.</p>
  </header>

  <section class="group" aria-labelledby="general-title">
    <h2 id="general-title">Downloads</h2>

    <div class="setting">
      <div class="copy">
        <strong>Default download folder</strong>
        <span class="mono" title={folder}>{folder || 'Not set'}</span>
      </div>
      <div class="actions">
        <button type="button" class="app-btn" disabled={!folder} on:click={showFolder}>Show in Finder</button>
        <button type="button" class="app-btn primary" on:click={pickFolder}>Change…</button>
      </div>
    </div>

    <label class="setting">
      <span class="copy">
        <strong>Create a subfolder for each download</strong>
        <small>Places all files for one video together.</small>
      </span>
      <input type="checkbox" checked={$settings.perVideoSubfolder} on:change={(e) => update({ ...$settings, perVideoSubfolder: e.currentTarget.checked })} />
    </label>

    <label class="setting">
      <span class="copy">
        <strong>Confirm before starting downloads</strong>
        <small>Shows the selected output before adding it to the queue.</small>
      </span>
      <input type="checkbox" checked={$settings.confirmBeforeDownload} on:change={(e) => update({ ...$settings, confirmBeforeDownload: e.currentTarget.checked })} />
    </label>

    <QueueSettingsCard
      model={{ concurrency, minimum: 1, maximum: 10, defaultValue: 2, disabled: saving }}
      onConcurrencyChange={changeConcurrency}
    />
    {#if concurrency > 4}
      <p class="warning">More than 4 simultaneous downloads may reduce stability or trigger rate limits.</p>
    {/if}
  </section>

  <section class="group" aria-labelledby="ffmpeg-title">
    <h2 id="ffmpeg-title">FFmpeg</h2>

    <div class="setting">
      <div class="copy">
        <strong>FFmpeg status</strong>
        <span>
          {#if $ffmpeg.available && ffmpegVersion}
            Version {ffmpegVersion} · ready for merging and MP3 conversion
          {:else}
            Needed to merge video and audio, and to convert to MP3.
          {/if}
        </span>
      </div>
      <div class="actions">
        <em class="badge" class:ok={$ffmpeg.available}>{$ffmpeg.available ? 'Ready' : 'Not found'}</em>
        <button type="button" class="app-btn" on:click={recheck}>Recheck</button>
      </div>
    </div>

    <div class="setting">
      <div class="copy">
        <strong>FFmpeg path</strong>
        <span class="mono" class:empty={!displayedFFmpegPath} title={displayedFFmpegPath}>{displayedFFmpegPath || 'Not configured'}</span>
      </div>
      <div class="actions">
        <button type="button" class="app-btn primary" on:click={locateFFmpeg}>Change…</button>
      </div>
    </div>

    {#if !$ffmpeg.available}
      <div class="setting hint">
        <p>Install FFmpeg, then Recheck or choose the binary with Change…</p>
        <button type="button" class="app-btn" on:click={() => window.runtime.BrowserOpenURL('https://ffmpeg.org/download.html')}>Installation guide ↗</button>
      </div>
    {/if}
  </section>

  <section class="group" aria-labelledby="diagnostics-title">
    <h2 id="diagnostics-title">Diagnostics</h2>
    <div class="setting">
      <div class="copy">
        <strong>Send operational diagnostics</strong>
        <span>When VidStow cannot complete a requested download or encounters an app failure, send a small sanitized report in the background. Video IDs, links, paths, filenames, cookies, tokens, and error text stay private.</span>
        <small>Local diagnostic history remains available either way. Disabling this immediately deletes anything waiting to be sent.</small>
        <button class="privacy-link" type="button" on:click={() => window.runtime.BrowserOpenURL('https://diagnostics.vidstow.workers.dev/privacy')}>Diagnostics privacy notice ↗</button>
      </div>
      <div class="actions choices" role="radiogroup" aria-label="Automatic diagnostics">
        <label><input type="radio" name="automatic-diagnostics" checked={$settings.automaticDiagnostics === 'enabled'} on:change={() => setAutomaticDiagnostics('enabled')} /> Send diagnostics</label>
        <label><input type="radio" name="automatic-diagnostics" checked={$settings.automaticDiagnostics === 'disabled'} on:change={() => setAutomaticDiagnostics('disabled')} /> Don’t send</label>
      </div>
    </div>
    <div class="setting">
      <div class="copy">
        <strong>Support report</strong>
        <span>Includes app and FFmpeg status plus recent sanitized failures. URLs and paths stay private.</span>
      </div>
      <div class="actions">
        <button type="button" class="app-btn" on:click={clearDiagnostics}>Clear history</button>
        <button type="button" class="app-btn primary" on:click={copyDiagnostics}>Copy Diagnostics</button>
      </div>
    </div>
  </section>
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

  .group {
    overflow: hidden;
    padding: 0 var(--sp-3) var(--sp-1);
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    background: var(--surface-base);
  }
  .group h2 {
    margin: 0;
    padding: var(--sp-2) 0 6px;
    color: var(--text-muted);
    font-size: var(--fs-xs);
    font-weight: 650;
    letter-spacing: 0.06em;
    line-height: 1;
    text-transform: uppercase;
  }
  .setting {
    display: flex;
    min-height: 44px;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-4);
    padding: 7px 0;
    border-top: 1px solid var(--border-subtle);
  }
  .group > h2 + .setting { border-top: 0; }
  .copy { min-width: 0; flex: 1; }
  .copy strong, .copy span, .copy small { display: block; }
  .copy strong { color: var(--text-primary); font-size: var(--fs-sm); font-weight: 600; }
  .copy span, .copy small {
    margin-top: 2px;
    color: var(--text-secondary);
    font-size: var(--fs-xs);
    line-height: 1.4;
  }
  .copy .mono {
    overflow: hidden;
    font-family: var(--font-mono);
    font-size: var(--fs-xs);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .copy .mono.empty { font-family: var(--font-sans); font-style: italic; }
  label.setting { cursor: pointer; }
  label.setting input { margin-left: var(--sp-2); flex-shrink: 0; }
  .choices { min-width: 146px; align-items: flex-start; flex-direction: column; gap: 5px; }
  .choices label { display: flex; align-items: center; gap: 6px; font-size: var(--fs-xs); cursor: pointer; }
  .choices input { margin: 0; }
  .privacy-link {
    margin-top: 4px;
    padding: 0;
    color: var(--accent-primary);
    font-size: var(--fs-xs);
    text-decoration: underline;
    text-underline-offset: 2px;
  }
  .actions { display: flex; align-items: center; justify-content: flex-end; gap: var(--sp-1); flex-shrink: 0; }
  .badge {
    padding: 2px 6px;
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: var(--r-sm);
    background: var(--status-danger-soft);
    color: var(--status-danger);
    font-style: normal;
    font-size: var(--fs-xs);
    font-weight: 650;
  }
  .badge.ok { border-color: rgba(34, 197, 94, 0.3); background: var(--status-success-soft); color: var(--status-success); }
  .warning, .hint {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-3);
    margin: 0 0 var(--sp-1);
    padding: 7px var(--sp-2);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-sm);
    font-size: var(--fs-xs);
  }
  .warning { border-color: rgba(245, 158, 11, 0.25); background: var(--status-warning-soft); color: var(--status-warning); }
  .hint { border-color: rgba(59, 130, 246, 0.25); background: var(--status-info-soft); color: var(--text-primary); }
  .hint p { margin: 0; color: var(--text-secondary); font-size: var(--fs-xs); }

  @media (max-width: 720px) {
    .setting, .hint { align-items: flex-start; flex-direction: column; }
    .actions { width: 100%; justify-content: flex-start; flex-wrap: wrap; }
  }
</style>
