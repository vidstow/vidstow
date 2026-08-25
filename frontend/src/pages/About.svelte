<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { showBanner, showError } from '../lib/stores.js';
  import { formatEngineVersion } from '../lib/format.js';
  import type { BuildInfo } from '../lib/types.js';

  const APP = {
    name: 'VidStow',
    tagline: 'A tidy YouTube video, Short, and playlist downloader for the desktop.',
    description:
      'VidStow is an open-source desktop app that downloads public YouTube videos, Shorts, playlists, and audio. ' +
      'Analyze a URL, choose the items and output, and let the queue run — no account or tracking required.',
    license: 'Apache-2.0',
    source: 'https://github.com/vidstow/vidstow',
    docs: 'https://github.com/vidstow/vidstow#readme',
  };

  const builtWith: Array<{ name: string; url: string }> = [
    { name: 'Go', url: 'https://go.dev' },
    { name: 'Wails', url: 'https://wails.io' },
    { name: 'Svelte', url: 'https://svelte.dev' },
    { name: 'ytdlp-go', url: 'https://github.com/tejasa97/ytdlp-go' },
    { name: 'FFmpeg', url: 'https://ffmpeg.org' },
  ];

  let build: BuildInfo = {
    version: 'Loading…', engineVersion: 'Loading…', os: '', architecture: '', goVersion: '',
  };

  onMount(async () => {
    try { build = await api.app.buildInfo(); }
    catch (err) { showError(err, 'Could not read build information'); }
  });

  function open(url: string) {
    if (url) window.runtime?.BrowserOpenURL?.(url);
  }

  async function copyDiagnostics() {
    try { await api.diagnostics.copy(); showBanner('info', 'Diagnostics copied');
    }
    catch (err) { showError(err, 'Could not copy diagnostics'); }
  }
</script>

<section class="page" aria-labelledby="about-title">
  <header class="page-header">
    <h1 id="about-title">About</h1>
    <p>{APP.tagline}</p>
  </header>

  <section class="group" aria-labelledby="app-title">
    <h2 id="app-title">{APP.name}</h2>
    <p class="lede">{APP.description}</p>

    <dl class="facts">
      <div>
        <dt>Version</dt>
        <dd>{build.version} · {APP.license}</dd>
      </div>
      <div>
        <dt>Engine</dt>
        <dd>ytdlp-go {formatEngineVersion(build.engineVersion)}</dd>
      </div>
      <div>
        <dt>Platform</dt>
        <dd>{build.os && build.architecture ? `${build.os}/${build.architecture}` : 'Loading…'}</dd>
      </div>
    </dl>

    <div class="setting">
      <div class="copy">
        <strong>Source</strong>
        <span>Open source on GitHub, with setup notes in the readme.</span>
      </div>
      <div class="actions">
        <button type="button" class="app-btn" on:click={() => open(APP.source)}>View source</button>
        <button type="button" class="app-btn primary" on:click={() => open(APP.docs)}>Read the docs</button>
      </div>
    </div>
    <div class="setting">
      <div class="copy">
        <strong>Support report</strong>
        <span>Includes app version and FFmpeg status. Paths stay private.</span>
      </div>
      <div class="actions">
        <button type="button" class="app-btn primary" on:click={copyDiagnostics}>Copy Diagnostics</button>
      </div>
    </div>
  </section>

  <section class="group" aria-labelledby="stack-title">
    <h2 id="stack-title">Built with open source</h2>
    <div class="setting">
      <div class="copy">
        <strong>Dependencies</strong>
        <span>VidStow depends on these projects.</span>
      </div>
      <ul class="stack">
        {#each builtWith as tool}
          <li>
            <button type="button" class="app-btn" on:click={() => open(tool.url)}>{tool.name}</button>
          </li>
        {/each}
      </ul>
    </div>
  </section>

  <section class="group" aria-labelledby="legal-title">
    <h2 id="legal-title">Legal</h2>
    <p class="legal">
      VidStow is distributed under the Apache License 2.0. It is not affiliated with, or endorsed by,
      YouTube or Google. Video and audio content is downloaded only where you have the right to do so —
      please respect each creator's terms and local laws. FFmpeg, ytdlp-go, Go, Wails, and Svelte are
      independent open-source projects with their own licenses.
    </p>
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
  .lede, .legal {
    margin: 0;
    padding: var(--sp-2) 0;
    border-top: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    font-size: var(--fs-xs);
    line-height: 1.45;
  }
  .facts { margin: 0; border-top: 1px solid var(--border-subtle); }
  .facts div {
    display: grid;
    grid-template-columns: 100px minmax(0, 1fr);
    min-height: 34px;
    align-items: center;
    border-bottom: 1px solid var(--border-subtle);
  }
  .facts dt { color: var(--text-muted); font-size: var(--fs-xs); font-weight: 600; }
  .facts dd {
    margin: 0;
    overflow-wrap: anywhere;
    color: var(--text-primary);
    font-family: var(--font-mono);
    font-size: var(--fs-xs);
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
  .copy { min-width: 0; flex: 1; }
  .copy strong, .copy span { display: block; }
  .copy strong { color: var(--text-primary); font-size: var(--fs-sm); font-weight: 600; }
  .copy span { margin-top: 2px; color: var(--text-secondary); font-size: var(--fs-xs); line-height: 1.4; }
  .actions { display: flex; align-items: center; justify-content: flex-end; gap: var(--sp-1); flex-shrink: 0; }
  .stack {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: var(--sp-1);
    list-style: none;
    margin: 0;
    padding: 0;
  }
  @media (max-width: 720px) {
    .setting { align-items: flex-start; flex-direction: column; }
    .actions, .stack { width: 100%; justify-content: flex-start; }
  }
</style>
