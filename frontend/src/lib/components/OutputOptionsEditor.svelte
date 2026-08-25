<script lang="ts">
  import type { OutputOptions, SubtitleLanguage } from '../types.js';

  export let value: OutputOptions = {};
  // Playlists apply one shared policy while each child is analyzed server-side.
  export let languages: SubtitleLanguage[] = [];
  export let allowSubtitles = true;
  export let collectionMode = false;
  export let videoLanguage = '';

  const commonLanguages: Array<[string, string]> = [
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
  $: selectedTrack = selectableLanguages.find((language) => language.code === selectedLanguage);
  $: subtitleSummary = noLanguagesReported
    ? 'None available'
    : subtitlesOn
      ? collectionMode
        ? selectedLanguage
          ? commonLanguageLabel(selectedLanguage)
          : 'Each video’s own language'
        : selectedTrack
          ? languageLabel(selectedTrack)
          : 'One soft track'
      : 'No subtitle track';

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

  function commonLanguageLabel(code: string): string {
    const match = commonLanguages.find(([value]) => value === code);
    return match ? `${match[1]} (${match[0]})` : code;
  }

  function languageLabel(language: SubtitleLanguage): string {
    const label = (language.name || language.code).replace(/\s*\(auto-generated\)$/i, '');
    return `${label}${language.auto ? ' (auto/transcribed)' : ''}`;
  }
</script>

{#if allowSubtitles}
  <section class="subtitle-control" aria-label="Subtitle policy">
    <div class="control-copy">
      <strong>Subtitles</strong>
      <span class:unavailable={noLanguagesReported}>{subtitleSummary}</span>
    </div>

    <div class="mode" role="group" aria-label="Subtitle inclusion">
      <button type="button" class:active={!subtitlesOn} aria-pressed={!subtitlesOn} aria-label="Do not include subtitles" on:click={() => setSubtitles(false)}>Off</button>
      <button type="button" class:active={subtitlesOn} aria-pressed={subtitlesOn} aria-label="Include subtitles" disabled={noLanguagesReported} on:click={() => setSubtitles(true)}>On</button>
    </div>

    {#if subtitlesOn}
      {#if collectionMode}
        <select aria-label="Subtitle language policy" value={selectedLanguage} on:change={(event) => selectLanguage(event.currentTarget.value)}>
          <option value="">Each video’s own language</option>
          {#each commonLanguages as language}
            <option value={language[0]}>{language[1]} ({language[0]})</option>
          {/each}
        </select>
      {:else}
        <select aria-label="Subtitle language" value={selectedLanguage} on:change={(event) => selectLanguage(event.currentTarget.value)}>
          {#each selectableLanguages as language (`${language.code}:${language.auto ? 'auto' : 'creator'}`)}
            <option value={language.code}>{languageLabel(language)}</option>
          {/each}
        </select>
      {/if}
    {/if}

    {#if collectionMode && subtitlesOn && selectedLanguage}
      <small>If unavailable, that item still downloads without subtitles.</small>
    {:else if subtitlesOn && value.subtitleSidecar}
      <small>Embedded track + additional .srt</small>
    {/if}
  </section>
{/if}

<style>
  .subtitle-control {
    grid-column: 1 / -1;
    display: grid;
    min-width: 250px;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 5px 8px;
    padding-top: 7px;
    border-top: 1px solid var(--border-subtle);
  }
  .control-copy { min-width: 0; display: flex; align-items: baseline; gap: 6px; }
  .control-copy strong { color: var(--text-secondary); font-size: 11px; font-weight: 600; }
  .control-copy span { overflow: hidden; color: var(--text-muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
  .control-copy span.unavailable { color: var(--text-muted); }

  .mode {
    display: inline-flex;
    padding: 1px;
    border: 1px solid var(--border-default);
    border-radius: var(--r-sm);
    background: var(--surface-bg);
  }
  .mode button {
    min-width: 36px;
    min-height: 21px;
    padding: 0 7px;
    border-radius: 3px;
    color: var(--text-muted);
    font-size: 10px;
    font-weight: 600;
  }
  .mode button:hover:not(:disabled) { color: var(--text-primary); }
  .mode button.active { background: var(--surface-active); color: var(--text-primary); }
  .mode button:disabled { cursor: not-allowed; opacity: 0.35; }

  select {
    grid-column: 1 / -1;
    width: 100%;
    height: 27px;
    min-width: 0;
    padding: 0 26px 0 8px;
    border-radius: var(--r-sm);
    background-color: var(--surface-bg);
    font-size: 10px;
  }
  small { grid-column: 1 / -1; color: var(--text-muted); font-size: 9px; line-height: 1.3; }

  @media (max-width: 860px) {
    .subtitle-control { min-width: 0; }
  }
</style>
