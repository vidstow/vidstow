<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { OutputOptions, SubtitleLanguage } from '../types.js';

  export let value: OutputOptions = {};
  // Empty for playlists, where the engine applies the preference per video.
  export let languages: SubtitleLanguage[] = [];
  export let allowSubtitles = true;
  export let ffmpegAvailable = true;
  export let collectionMode = false;

  const dispatch = createEventDispatcher<{ 'goto-settings': void }>();
  let open = false;

  $: selectedLanguages = new Set(value.subtitleLanguages ?? []);
  $: manualLanguages = languages.filter((language) => !language.auto);
  $: autoLanguages = languages.filter((language) => language.auto);
  $: noLanguagesReported = !collectionMode && languages.length === 0;
  $: summary = allowSubtitles
    ? `embedded subtitles${value.subtitleSidecar ? ' + SRT file' : ''} · artwork · chapters`
    : '';

  function toggleLanguage(code: string) {
    const next = new Set(value.subtitleLanguages ?? []);
    if (next.has(code)) next.delete(code);
    else next.add(code);
    value = { ...value, subtitleLanguages: [...next] };
  }

  function setFlag(flag: 'subtitleSidecar' | 'subtitleAutoCaptions' | 'embedMetadata', checked: boolean) {
    value = { ...value, [flag]: checked };
  }

  function languageLabel(language: SubtitleLanguage): string {
    return language.name || language.code;
  }
</script>

<section class="output-options" aria-label="Subtitles and details">
  <button type="button" class="disclosure" aria-expanded={open} on:click={() => (open = !open)}>
    <span class="chevron" aria-hidden="true">{open ? '▾' : '▸'}</span>
    <span class="label">Subtitles &amp; details</span>
    {#if summary}<span class="summary">{summary}</span>{/if}
  </button>

  {#if open}
    <div class="body">
      {#if allowSubtitles}
        <div class="group">
          <h3>Subtitles</h3>
          {#if noLanguagesReported}
            <p class="note">No subtitle tracks were reported. The video will still include artwork and chapters when available.</p>
            <label class="check">
              <input type="checkbox" checked={!!value.subtitleSidecar} on:change={(event) => setFlag('subtitleSidecar', event.currentTarget.checked)} />
              Also save an .srt file
            </label>
            <p class="hint">An .srt is saved only if a subtitle track becomes available; embedding remains enabled.</p>
          {:else}
            <p class="note">A selected creator subtitle is embedded in the video. If none is available, VidStow can use an auto-generated transcript.</p>
            {#if collectionMode}
              <p class="hint">Each video uses its own language, then English, then its first creator subtitle. Auto captions are the fallback.</p>
            {:else}
              <div class="languages" role="group" aria-label="Subtitle languages">
                {#each manualLanguages as language (language.code)}
                  <label class="lang">
                    <input type="checkbox" checked={selectedLanguages.has(language.code)} on:change={() => toggleLanguage(language.code)} />
                    {languageLabel(language)}
                  </label>
                {/each}
                {#if autoLanguages.length}
                  <span class="lang-group">Auto-generated</span>
                  {#each autoLanguages as language (`${language.code}:auto`)}
                    <label class="lang">
                      <input type="checkbox" checked={selectedLanguages.has(language.code)} on:change={() => toggleLanguage(language.code)} />
                      {languageLabel(language)}
                    </label>
                  {/each}
                {/if}
              </div>
            {/if}
            {#if autoLanguages.length || collectionMode}
              <label class="check">
                <input type="checkbox" checked={value.subtitleAutoCaptions !== false} on:change={(event) => setFlag('subtitleAutoCaptions', event.currentTarget.checked)} />
                Use auto-generated captions when needed
              </label>
            {/if}
            <label class="check">
              <input type="checkbox" checked={!!value.subtitleSidecar} on:change={(event) => setFlag('subtitleSidecar', event.currentTarget.checked)} />
              Also save an .srt file
            </label>
            <p class="hint">The .srt file is additional; subtitles remain embedded in the video.</p>
          {/if}
        </div>

        <div class="group">
          <h3>Included automatically</h3>
          <p class="note">Thumbnail artwork and chapter markers are embedded when YouTube provides them.</p>
          <label class="check">
            <input type="checkbox" checked={!!value.embedMetadata} on:change={(event) => setFlag('embedMetadata', event.currentTarget.checked)} />
            Include title &amp; channel metadata
          </label>
          {#if !ffmpegAvailable}
            <p class="ffmpeg-note">
              FFmpeg is required to create this complete video file.
              <button type="button" class="link" on:click={() => dispatch('goto-settings')}>Open Settings</button>
            </p>
          {/if}
        </div>
      {:else}
        <div class="group"><p class="note">Complete-file embedding applies to video downloads. Audio keeps its selected audio format.</p></div>
      {/if}
    </div>
  {/if}
</section>

<style>
  .output-options { border-top: 1px solid var(--border-subtle); background: var(--surface-subtle); flex-shrink: 0; }
  .disclosure { display: flex; align-items: center; gap: 8px; width: 100%; min-height: 38px; padding: 4px 16px; color: var(--text-secondary); font-size: var(--fs-xs); font-weight: 650; text-align: left; }
  .disclosure:hover { background: var(--surface-hover); }
  .chevron { width: 12px; color: var(--text-muted); }
  .summary { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-muted); font-weight: 550; }
  .body { display: flex; flex-wrap: wrap; gap: 12px 28px; padding: 2px 16px 14px 36px; }
  .group { min-width: 240px; max-width: 520px; display: flex; flex: 1; flex-direction: column; gap: 7px; }
  .group h3 { margin: 0; color: var(--text-muted); font-size: 10px; font-weight: 650; letter-spacing: 0.04em; text-transform: uppercase; }
  .note, .hint, .ffmpeg-note { margin: 0; color: var(--text-muted); font-size: 11px; line-height: 1.45; }
  .ffmpeg-note { color: var(--status-warning); font-weight: 600; }
  .languages { display: grid; grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 2px 10px; max-height: 148px; overflow: auto; padding: 8px 10px; border: 1px solid var(--border-subtle); border-radius: var(--r-md); background: var(--surface-base); }
  .lang { display: flex; align-items: center; gap: 7px; font-size: 11px; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .lang-group { grid-column: 1 / -1; margin-top: 4px; color: var(--text-muted); font-size: 10px; font-weight: 650; text-transform: uppercase; }
  .check { display: flex; align-items: center; gap: 8px; font-size: var(--fs-xs); font-weight: 550; color: var(--text-primary); cursor: pointer; }
  .check input { flex-shrink: 0; }
  .link { padding: 0; border: 0; background: none; color: var(--accent-600); font-size: 11px; font-weight: 650; text-decoration: underline; }
</style>
