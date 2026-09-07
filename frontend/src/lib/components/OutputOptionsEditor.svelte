<script lang="ts">
  // Inspector rows under identity: Output (slot), Subtitles, In the file.
  // Audio greys Subtitles in place so the card does not jump. Download still
  // strips captions via effectiveOptions. FFmpeg-dependent choices stay
  // disabled and are clamped away while FFmpeg is missing.
  import { createEventDispatcher, onMount } from 'svelte';
  import type { OutputOptions, SubtitleLanguage } from '../types.js';

  export let value: OutputOptions = {};
  // Languages reported by analysis; empty for collections (playlists), where
  // the engine's English-first default picks per video.
  export let languages: SubtitleLanguage[] = [];
  export let allowSubtitles = true;
  export let ffmpegAvailable = true;
  export let collectionMode = false;

  const dispatch = createEventDispatcher<{ 'goto-settings': void }>();
  const TRANSLATED_PREVIEW = 8;
  const MAX_LANGUAGES = 16;

  let langOpen = false;
  let fmtOpen = false;
  let langRoot: HTMLDivElement | undefined;
  let fmtRoot: HTMLDivElement | undefined;

  $: mode = value.subtitleMode ?? '';
  $: selectedLanguages = new Set(value.subtitleLanguages ?? []);
  $: manualLanguages = uniqueLanguages(languages.filter((language) => !language.auto));
  $: autoLanguages = uniqueLanguages(languages.filter((language) => language.auto));
  $: split = splitAutoLanguages(manualLanguages, autoLanguages);
  $: noLanguagesReported = !collectionMode && languages.length === 0;
  $: subtitleChoiceBlocked = !allowSubtitles || noLanguagesReported;
  $: embedBlocked = !ffmpegAvailable;
  $: subtitlesOff = mode === '';
  $: langDisabled = subtitleChoiceBlocked || subtitlesOff;
  $: fmtDisabled = subtitleChoiceBlocked || subtitlesOff || mode === 'embed' || !ffmpegAvailable;
  $: rowLockReason = !allowSubtitles
    ? 'Subtitles need a video output.'
    : noLanguagesReported
      ? 'No subtitles were reported for this video.'
      : '';
  $: if (subtitleChoiceBlocked) {
    langOpen = false;
    fmtOpen = false;
  }
  $: formatValue = value.subtitleFormat ?? '';
  $: formatLabel = formatValue === 'srt' ? 'SRT' : formatValue === 'vtt' ? 'VTT' : 'Original';
  $: languageButtonLabel = languageSummary(manualLanguages, split.asr, split.translated, selectedLanguages);

  $: if (!ffmpegAvailable && requiresFFmpeg(value)) {
    value = withoutFFmpegChoices(value);
  }

  onMount(() => {
    const onPtr = (event: PointerEvent) => {
      const target = event.target as Node;
      if (langOpen && langRoot && !langRoot.contains(target)) langOpen = false;
      if (fmtOpen && fmtRoot && !fmtRoot.contains(target)) fmtOpen = false;
    };
    const onDocKey = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      langOpen = false;
      fmtOpen = false;
    };
    document.addEventListener('pointerdown', onPtr);
    document.addEventListener('keydown', onDocKey);
    return () => {
      document.removeEventListener('pointerdown', onPtr);
      document.removeEventListener('keydown', onDocKey);
    };
  });

  function uniqueLanguages(items: SubtitleLanguage[]): SubtitleLanguage[] {
    const seen = new Set<string>();
    return items.filter((item) => {
      const key = `${item.code.toLowerCase()}:${item.auto ? 'auto' : 'manual'}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  }

  function splitAutoLanguages(manual: SubtitleLanguage[], autos: SubtitleLanguage[]): { asr: SubtitleLanguage[]; translated: SubtitleLanguage[] } {
    if (!autos.length) return { asr: [], translated: [] };
    const creator = new Set(manual.map((language) => language.code.toLowerCase()));
    const asrCodes = new Set<string>();
    for (const language of autos) {
      if (creator.has(language.code.toLowerCase())) asrCodes.add(language.code.toLowerCase());
    }
    if (!asrCodes.size) {
      const english = autos.find((language) => /^en([-_]|$)/i.test(language.code));
      asrCodes.add((english ?? autos[0]).code.toLowerCase());
    }
    return {
      asr: autos.filter((language) => asrCodes.has(language.code.toLowerCase())),
      translated: autos.filter((language) => !asrCodes.has(language.code.toLowerCase())),
    };
  }

  function defaultLanguage(): SubtitleLanguage | undefined {
    const pool = manualLanguages.length ? manualLanguages : languages;
    return pool.find((language) => /^en([-_]|$)/i.test(language.code)) ?? pool[0];
  }

  function languageSummary(
    manual: SubtitleLanguage[],
    asr: SubtitleLanguage[],
    translated: SubtitleLanguage[],
    selected: Set<string>,
  ): string {
    const listed = [...manual, ...asr, ...translated];
    const codes = selected.size ? [...selected] : (defaultLanguage() ? [defaultLanguage()!.code] : []);
    const names = codes.map((code) => {
      const match = listed.find((language) => language.code === code);
      return match ? displayName(match) : code;
    });
    if (names.length <= 1) return names[0] ?? '';
    return `${names.length} languages`;
  }

  function displayName(language: SubtitleLanguage): string {
    return (language.name || language.code).replace(/\s*\(auto-generated\)$/i, '');
  }

  function requiresFFmpeg(options: OutputOptions): boolean {
    return (
      options.subtitleMode === 'embed' ||
      !!options.embedMetadata ||
      !!options.embedThumbnail ||
      !!options.embedChapters ||
      (options.subtitleMode === 'sidecar' && !!options.subtitleFormat)
    );
  }

  function withoutFFmpegChoices(options: OutputOptions): OutputOptions {
    return {
      ...options,
      subtitleMode: options.subtitleMode === 'embed' ? '' : options.subtitleMode,
      subtitleFormat: '',
      embedMetadata: false,
      embedThumbnail: false,
      embedChapters: false,
    };
  }

  function setMode(next: '' | 'sidecar' | 'embed') {
    if (subtitleChoiceBlocked) return;
    if (next === 'embed' && embedBlocked) return;
    const nextValue: OutputOptions = { ...value, subtitleMode: next, subtitleAutoCaptions: next !== '' };
    if (next === '') {
      langOpen = false;
      fmtOpen = false;
    } else if (next === 'sidecar' && !nextValue.subtitleFormat && ffmpegAvailable) {
      nextValue.subtitleFormat = 'srt';
    }
    if (next !== '' && !collectionMode && !nextValue.subtitleLanguages?.length) {
      const preferred = defaultLanguage()?.code;
      if (preferred) nextValue.subtitleLanguages = [preferred];
    }
    value = nextValue;
  }

  function toggleLanguage(code: string) {
    if (langDisabled) return;
    const next = new Set(value.subtitleLanguages ?? []);
    if (next.has(code)) next.delete(code);
    else if (next.size < MAX_LANGUAGES) next.add(code);
    const fallback = defaultLanguage()?.code;
    const languages = next.size ? [...next] : fallback ? [fallback] : undefined;
    value = { ...value, subtitleLanguages: languages, subtitleAutoCaptions: true };
  }

  function setFormat(next: string) {
    if (fmtDisabled) return;
    if (next !== '' && next !== 'srt' && next !== 'vtt') return;
    value = { ...value, subtitleFormat: next };
    fmtOpen = false;
  }

  function setFlag(flag: 'embedMetadata' | 'embedThumbnail' | 'embedChapters', checked: boolean) {
    value = { ...value, [flag]: checked };
  }

  function isDefaultLanguage(language: SubtitleLanguage): boolean {
    return defaultLanguage()?.code === language.code && !language.auto;
  }
</script>

<div class="opt-sections">
  <div class="erow">
    <span class="elab">Output</span>
    <div class="ectl">
      <slot name="output" />
    </div>
  </div>

  <div
    class="erow"
    class:disabled={subtitleChoiceBlocked}
    aria-disabled={subtitleChoiceBlocked}
    title={rowLockReason}
  >
    <span class="elab">Subtitles{#if !allowSubtitles}<span class="only">Video only</span>{/if}</span>
    <div class="ectl">
      <div class="triad" role="radiogroup" aria-label="Subtitle mode">
        <button
          type="button"
          class:on={mode === ''}
          role="radio"
          aria-checked={mode === ''}
          aria-label="Subtitles off"
          disabled={subtitleChoiceBlocked}
          on:click={() => setMode('')}
        >Off</button>
        <button
          type="button"
          class:on={mode === 'sidecar'}
          class:file={mode === 'sidecar'}
          role="radio"
          aria-checked={mode === 'sidecar'}
          aria-label="Subtitle file"
          disabled={subtitleChoiceBlocked}
          on:click={() => setMode('sidecar')}
        >File</button>
        <button
          type="button"
          class:on={mode === 'embed'}
          class:file={mode === 'embed'}
          role="radio"
          aria-checked={mode === 'embed'}
          aria-label="Embed in video"
          disabled={subtitleChoiceBlocked || embedBlocked}
          title={embedBlocked ? 'Embedding needs FFmpeg' : ''}
          on:click={() => setMode('embed')}
        >Embed</button>
      </div>

      {#if collectionMode}
        <span class="dd static" title="Every video uses English or its first available language.">
          <span class="cat">Language</span>
          English or first available
        </span>
      {:else}
        <div class="ddwrap" bind:this={langRoot}>
          <button
            type="button"
            class="dd"
            class:open={langOpen}
            disabled={langDisabled}
            aria-haspopup="listbox"
            aria-expanded={langOpen}
            aria-label="Subtitle language"
            on:click={() => { if (!langDisabled) { langOpen = !langOpen; fmtOpen = false; } }}
          >
            <span class="cat">Language</span>
            <span class="dd-v">{languageButtonLabel || '—'}</span>
            <span class="chev" aria-hidden="true">▾</span>
          </button>
          {#if langOpen && !langDisabled}
            <div class="fmt-menu" role="listbox" aria-label="Subtitle language" aria-multiselectable="true">
              {#each manualLanguages as language (language.code)}
                <button
                  type="button"
                  class="fmt-opt"
                  class:on={selectedLanguages.has(language.code)}
                  role="option"
                  aria-selected={selectedLanguages.has(language.code)}
                  aria-label={displayName(language)}
                  on:click={() => toggleLanguage(language.code)}
                >
                  <span class="fmt-check" aria-hidden="true">{selectedLanguages.has(language.code) ? '✓' : ''}</span>
                  <span class="fmt-lab">{displayName(language)}</span>
                  {#if isDefaultLanguage(language)}<span class="tag">Default</span>{/if}
                </button>
              {/each}
              {#if split.asr.length}
                <div class="fmt-sep">Auto-generated</div>
                {#each split.asr as language (`${language.code}:auto`)}
                  <button
                    type="button"
                    class="fmt-opt"
                    class:on={selectedLanguages.has(language.code)}
                    role="option"
                    aria-selected={selectedLanguages.has(language.code)}
                    aria-label={`${displayName(language)} (auto-generated)`}
                    on:click={() => toggleLanguage(language.code)}
                  >
                    <span class="fmt-check" aria-hidden="true">{selectedLanguages.has(language.code) ? '✓' : ''}</span>
                    <span class="fmt-lab">{displayName(language)}</span>
                  </button>
                {/each}
              {/if}
              {#if split.translated.length}
                <div class="fmt-sep">Auto-translated · from English</div>
                {#each split.translated.slice(0, TRANSLATED_PREVIEW) as language (`${language.code}:auto-tr`)}
                  <button
                    type="button"
                    class="fmt-opt"
                    class:on={selectedLanguages.has(language.code)}
                    role="option"
                    aria-selected={selectedLanguages.has(language.code)}
                    aria-label={`${displayName(language)} (auto-translated)`}
                    on:click={() => toggleLanguage(language.code)}
                  >
                    <span class="fmt-check" aria-hidden="true">{selectedLanguages.has(language.code) ? '✓' : ''}</span>
                    <span class="fmt-lab">{displayName(language)}</span>
                  </button>
                {/each}
                <div class="fmt-more">…and more on YouTube</div>
              {/if}
            </div>
          {/if}
        </div>
      {/if}

      <div class="ddwrap" bind:this={fmtRoot}>
        <button
          type="button"
          class="dd"
          class:open={fmtOpen}
          disabled={fmtDisabled}
          aria-haspopup="listbox"
          aria-expanded={fmtOpen}
          aria-label="Subtitle format"
          title={!ffmpegAvailable ? 'FFmpeg converts subtitle files; the original format is kept without it.' : ''}
          on:click={() => { if (!fmtDisabled) { fmtOpen = !fmtOpen; langOpen = false; } }}
        >
          <span class="cat">Format</span>
          <span class="dd-v">{formatLabel}</span>
          <span class="chev" aria-hidden="true">▾</span>
        </button>
        {#if fmtOpen && !fmtDisabled}
          <div class="fmt-menu" role="listbox" aria-label="Subtitle format">
            {#each [['', 'Original'], ['srt', 'SRT'], ['vtt', 'VTT']] as [id, label] (id)}
              <button
                type="button"
                class="fmt-opt"
                class:on={formatValue === id}
                role="option"
                aria-selected={formatValue === id}
                on:click={() => setFormat(id)}
              >
                <span class="fmt-check" aria-hidden="true">{formatValue === id ? '✓' : ''}</span>
                <span class="fmt-lab">{label}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>

  <div class="erow">
    <span class="elab">In the file</span>
    <div class="ectl">
      <button
        type="button"
        class="also"
        class:on={!!value.embedMetadata}
        aria-pressed={!!value.embedMetadata}
        aria-label="Title & channel details"
        disabled={embedBlocked}
        on:click={() => setFlag('embedMetadata', !value.embedMetadata)}
      >Title &amp; channel</button>
      <button
        type="button"
        class="also"
        class:on={!!value.embedThumbnail}
        aria-pressed={!!value.embedThumbnail}
        aria-label="Thumbnail artwork"
        disabled={embedBlocked}
        on:click={() => setFlag('embedThumbnail', !value.embedThumbnail)}
      >Artwork</button>
      <button
        type="button"
        class="also"
        class:on={!!value.embedChapters}
        aria-pressed={!!value.embedChapters}
        aria-label="Chapter markers"
        disabled={embedBlocked}
        on:click={() => setFlag('embedChapters', !value.embedChapters)}
      >Chapters</button>
      {#if embedBlocked}
        <button type="button" class="link" on:click={() => dispatch('goto-settings')}>Open Settings</button>
      {/if}
    </div>
  </div>
</div>

<style>
  .opt-sections {
    border-top: 1px solid var(--border-subtle);
    background: var(--surface-subtle);
    margin-top: 8px;
    flex-shrink: 0;
  }
  .erow {
    display: grid;
    grid-template-columns: 88px minmax(0, 1fr);
    gap: 10px;
    align-items: center;
    padding: 10px 14px;
    min-height: 44px;
  }
  .erow + .erow { border-top: 1px solid var(--border-subtle); }
  .erow.disabled { opacity: 0.42; }
  .elab {
    color: var(--text-secondary);
    font-size: 12px;
    font-weight: 550;
    line-height: 1.2;
  }
  .elab .only {
    display: block;
    margin-top: 1px;
    color: var(--text-muted);
    font-size: 10px;
    font-weight: 500;
    letter-spacing: 0;
    line-height: 1.2;
  }
  .ectl {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    min-width: 0;
  }
  .triad {
    display: inline-flex;
    padding: 2px;
    gap: 2px;
    border: 1px solid var(--border-default);
    border-radius: 8px;
    background: var(--surface-base);
  }
  .triad button {
    height: 24px;
    padding: 0 10px;
    border-radius: 6px;
    color: var(--text-muted);
    font-size: 12px;
    font-weight: 550;
    border: 0;
    background: transparent;
  }
  .triad button.on {
    background: var(--surface-raised);
    color: var(--text-primary);
  }
  .triad button.on.file {
    color: #93C5FD;
    background: var(--accent-soft);
  }
  .erow.disabled .triad button.on,
  .erow.disabled .triad button.on.file {
    background: var(--surface-raised);
    color: var(--text-secondary);
  }
  .triad button:disabled { cursor: default; }
  .ddwrap { position: relative; }
  .dd {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 28px;
    padding: 0 10px;
    border: 1px solid var(--border-default);
    border-radius: 7px;
    background: var(--surface-base);
    color: var(--text-primary);
    font-size: 12px;
    cursor: pointer;
  }
  .dd.static {
    cursor: default;
    color: var(--text-secondary);
  }
  .dd:hover:not(:disabled):not(.static) { border-color: var(--border-strong); }
  .dd.open { border-color: var(--accent-500); }
  .dd:disabled { opacity: 0.45; cursor: default; }
  .erow.disabled .dd,
  .erow.disabled .dd:disabled { opacity: 1; }
  .dd .cat { color: var(--text-muted); }
  .dd-v {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .chev { color: var(--text-muted); font-size: 10px; }
  .dd.open .chev { transform: rotate(180deg); }
  .fmt-menu {
    position: absolute;
    z-index: 30;
    top: calc(100% + 4px);
    left: 0;
    min-width: 100%;
    width: max-content;
    max-width: min(320px, calc(100vw - 32px));
    max-height: 280px;
    overflow: auto;
    padding: 4px 0;
    border: 1px solid var(--border-default);
    border-radius: 8px;
    background: var(--surface-raised);
    box-shadow: var(--shadow-card);
  }
  .fmt-opt {
    display: grid;
    grid-template-columns: 16px minmax(0, 1fr);
    align-items: center;
    gap: 8px;
    width: 100%;
    height: 28px;
    padding: 0 12px;
    border: 0;
    background: transparent;
    color: var(--text-secondary);
    font-size: 12px;
    text-align: left;
    cursor: pointer;
  }
  .fmt-opt:has(.tag) { grid-template-columns: 16px minmax(0, 1fr) auto; }
  .fmt-opt:hover,
  .fmt-opt.on { background: var(--surface-hover); color: var(--text-primary); }
  .fmt-check { color: var(--accent-400); font-size: 11px; }
  .fmt-lab {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .fmt-sep {
    margin-top: 4px;
    padding: 8px 12px 3px;
    border-top: 1px solid var(--border-default);
    color: var(--text-muted);
    font-size: 10px;
    font-weight: 650;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .fmt-more {
    padding: 8px 12px 7px;
    color: var(--text-muted);
    font-size: 11px;
    text-align: center;
  }
  .tag {
    font-size: 10px;
    color: var(--text-muted);
    border: 1px solid var(--border-default);
    border-radius: 99px;
    padding: 1px 7px;
    white-space: nowrap;
  }
  .also {
    height: 26px;
    padding: 0 10px;
    border: 1px solid var(--border-default);
    border-radius: 99px;
    background: transparent;
    color: var(--text-muted);
    font-size: 12px;
  }
  .also.on {
    border-color: rgba(59, 130, 246, 0.45);
    background: var(--accent-soft);
    color: #93C5FD;
  }
  .also:disabled { opacity: 0.45; cursor: default; }
  .link {
    padding: 0;
    border: 0;
    background: none;
    color: var(--accent-600);
    font-size: 11px;
    font-weight: 650;
    text-decoration: underline;
  }
</style>
