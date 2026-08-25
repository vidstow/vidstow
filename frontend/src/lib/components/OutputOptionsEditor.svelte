<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { OutputOptions, SubtitleLanguage } from '../types.js';

  export let value: OutputOptions = {};
  // Playlists use one policy for every selected video rather than per-item tracks.
  export let languages: SubtitleLanguage[] = [];
  export let allowSubtitles = true;
  export let ffmpegAvailable = true;
  export let collectionMode = false;
  export let videoLanguage = '';

  const dispatch = createEventDispatcher<{ 'goto-settings': void }>();
  const commonLanguages = [
    ['en', 'English'], ['es', 'Spanish'], ['fr', 'French'], ['de', 'German'],
    ['it', 'Italian'], ['pt', 'Portuguese'], ['ja', 'Japanese'], ['ko', 'Korean'],
    ['zh', 'Chinese'], ['ar', 'Arabic'], ['hi', 'Hindi'], ['ru', 'Russian'],
  ];

  $: creatorLanguages = uniqueLanguages(languages.filter((language) => !language.auto));
  $: creatorCodes = new Set(creatorLanguages.map((language) => language.code.toLowerCase()));
  $: autoOnlyLanguages = uniqueLanguages(languages.filter((language) => language.auto && !creatorCodes.has(language.code.toLowerCase())));
  $: selectableLanguages = [...creatorLanguages, ...autoOnlyLanguages];
  $: noLanguagesReported = !collectionMode && selectableLanguages.length === 0;
  $: subtitlesOn = value.subtitleMode === 'embed' && (collectionMode || !noLanguagesReported);
  $: selectedLanguage = value.subtitleLanguages?.[0] ?? '';

  function uniqueLanguages(items: SubtitleLanguage[]): SubtitleLanguage[] {
    const seen = new Set<string>();
    return items.filter((item) => {
      const code = item.code.toLowerCase();
      if (seen.has(code)) return false;
      seen.add(code);
      return true;
    });
  }

  function matchingTrack(items: SubtitleLanguage[], wanted: string): SubtitleLanguage | undefined {
    const normalized = wanted.trim().toLowerCase();
    if (!normalized) return undefined;
    const root = normalized.split(/[-_]/)[0];
    return items.find((item) => item.code.toLowerCase() === normalized)
      ?? items.find((item) => item.code.toLowerCase().split(/[-_]/)[0] === root);
  }

  function preferredSingleLanguage(): string {
    const selected = matchingTrack(selectableLanguages, selectedLanguage);
    if (selected) return selected.code;
    if (creatorLanguages.length) {
      return (matchingTrack(creatorLanguages, videoLanguage)
        ?? matchingTrack(creatorLanguages, 'en')
        ?? creatorLanguages[0]).code;
    }
    return (matchingTrack(autoOnlyLanguages, videoLanguage)
      ?? matchingTrack(autoOnlyLanguages, 'en')
      ?? autoOnlyLanguages[0])?.code ?? '';
  }

  function setSubtitles(on: boolean) {
    if (!on || noLanguagesReported) {
      value = { ...value, subtitleMode: '', subtitleLanguages: undefined, subtitleAutoCaptions: false };
      return;
    }
    const language = collectionMode ? selectedLanguage : preferredSingleLanguage();
    value = {
      ...value,
      subtitleMode: 'embed',
      subtitleLanguages: language ? [language] : undefined,
      subtitleAutoCaptions: true,
    };
  }

  function selectLanguage(code: string) {
    value = { ...value, subtitleLanguages: code ? [code] : undefined, subtitleAutoCaptions: true };
  }

  function setMetadata(checked: boolean) {
    value = { ...value, embedMetadata: checked };
  }

  function languageLabel(language: SubtitleLanguage): string {
    const label = (language.name || language.code).replace(/\s*\(auto-generated\)$/i, '');
    return `${label}${language.auto ? ' (auto/transcribed)' : ''}`;
  }
</script>

{#if allowSubtitles}
<section class="output-options" aria-label="Subtitles and details">
    <div class="group subtitles">
      <div class="heading">
        <div>
          <h3>Subtitles {subtitlesOn ? 'On' : 'Off'}</h3>
          {#if noLanguagesReported}<p>None available</p>{/if}
        </div>
        <label class="switch">
          <input type="checkbox" aria-label="Include subtitles" checked={subtitlesOn} disabled={noLanguagesReported} on:change={(event) => setSubtitles(event.currentTarget.checked)} />
          <span>{subtitlesOn ? 'On' : 'Off'}</span>
        </label>
      </div>

      {#if subtitlesOn}
        {#if collectionMode}
          <label class="language-select">
            Subtitle language policy
            <select value={selectedLanguage} on:change={(event) => selectLanguage(event.currentTarget.value)}>
              <option value="">Each video’s own language</option>
              {#each commonLanguages as language}
                <option value={language[0]}>{language[1]} ({language[0]})</option>
              {/each}
            </select>
          </label>
          {#if selectedLanguage}
            <p class="hint">If that language is unavailable, the item still downloads without subtitles.</p>
          {:else}
            <p class="hint">Each item prefers a creator track in its own language, then English, then its first creator track. Auto captions are used only when no creator track exists.</p>
          {/if}
        {:else}
          <div class="languages" role="radiogroup" aria-label="Subtitle language">
            {#each selectableLanguages as language (`${language.code}:${language.auto ? 'auto' : 'creator'}`)}
              <label class="lang">
                <input type="radio" name="subtitle-language" checked={selectedLanguage === language.code} on:change={() => selectLanguage(language.code)} />
                {languageLabel(language)}
              </label>
            {/each}
          </div>
        {/if}
      {/if}
    </div>

    <div class="group details">
      <h3>Included automatically</h3>
      <p class="note">Thumbnail artwork and chapter markers are included when YouTube provides them.</p>
      <label class="check">
        <input type="checkbox" checked={!!value.embedMetadata} on:change={(event) => setMetadata(event.currentTarget.checked)} />
        Include title &amp; channel metadata
      </label>
      {#if !ffmpegAvailable}
        <p class="ffmpeg-note">FFmpeg is required to create this video file. <button type="button" class="link" on:click={() => dispatch('goto-settings')}>Open Settings</button></p>
      {/if}
    </div>
</section>
{/if}

<style>
  .output-options { display: flex; flex-wrap: wrap; gap: 12px 28px; padding: 12px 16px; border-top: 1px solid var(--border-subtle); background: var(--surface-subtle); flex-shrink: 0; }
  .group { min-width: 240px; max-width: 520px; display: flex; flex: 1; flex-direction: column; gap: 8px; }
  .heading { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
  .heading div { display: flex; align-items: baseline; gap: 8px; }
  .group h3 { margin: 0; color: var(--text-primary); font-size: var(--fs-xs); font-weight: 700; }
  .heading p { margin: 0; color: var(--text-muted); font-size: 11px; }
  .switch, .check, .lang { display: flex; align-items: center; gap: 7px; color: var(--text-primary); font-size: var(--fs-xs); font-weight: 550; }
  .switch { cursor: pointer; }
  .language-select { display: flex; flex-direction: column; gap: 5px; color: var(--text-secondary); font-size: 11px; font-weight: 650; }
  .language-select select { max-width: 280px; }
  .languages { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 5px 12px; max-height: 148px; overflow: auto; padding: 8px 10px; border: 1px solid var(--border-subtle); border-radius: var(--r-md); background: var(--surface-base); }
  .note, .hint, .ffmpeg-note { margin: 0; color: var(--text-muted); font-size: 11px; line-height: 1.45; }
  .ffmpeg-note { color: var(--status-warning); font-weight: 600; }
  .link { padding: 0; border: 0; background: none; color: var(--accent-600); font-size: 11px; font-weight: 650; text-decoration: underline; }
</style>
