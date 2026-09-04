<script lang="ts">
  // Always-visible Design D extras: Subtitles column + full-width Metadata.
  // Output (Video | Audio + format chips) is passed in as a slot so video,
  // playlist, and batch can keep their own plan lists. FFmpeg-dependent
  // choices stay disabled and are clamped away while FFmpeg is missing.
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

  let lastDelivery: 'sidecar' | 'embed' = 'sidecar';
  let langOpen = false;
  let fmtOpen = false;
  let langRoot: HTMLDivElement | undefined;
  let fmtRoot: HTMLDivElement | undefined;

  $: mode = value.subtitleMode ?? '';
  $: if (mode === 'sidecar' || mode === 'embed') lastDelivery = mode;
  $: selectedLanguages = new Set(value.subtitleLanguages ?? []);
  $: manualLanguages = uniqueLanguages(languages.filter((language) => !language.auto));
  $: autoLanguages = uniqueLanguages(languages.filter((language) => language.auto));
  $: split = splitAutoLanguages(manualLanguages, autoLanguages);
  $: noLanguagesReported = !collectionMode && languages.length === 0;
  $: subtitleChoiceBlocked = !allowSubtitles || noLanguagesReported;
  $: embedBlocked = !ffmpegAvailable;
  $: subtitlesOff = mode === '' && !subtitleChoiceBlocked;
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
    if (subtitleChoiceBlocked && next !== '') return;
    if (next === 'embed' && embedBlocked) return;
    const nextValue: OutputOptions = { ...value, subtitleMode: next, subtitleAutoCaptions: next !== '' };
    if (next === '') {
      langOpen = false;
      fmtOpen = false;
    } else if (next === 'sidecar' && !nextValue.subtitleFormat && ffmpegAvailable) {
      nextValue.subtitleFormat = 'srt';
    }
    if (next === 'sidecar' || next === 'embed') lastDelivery = next;
    if (next !== '' && !collectionMode && !nextValue.subtitleLanguages?.length) {
      const preferred = defaultLanguage()?.code;
      if (preferred) nextValue.subtitleLanguages = [preferred];
    }
    value = nextValue;
  }

  function toggleSubtitles() {
    if (subtitleChoiceBlocked) return;
    if (mode === '') {
      const restore = lastDelivery === 'embed' && embedBlocked ? 'sidecar' : lastDelivery;
      setMode(restore);
      return;
    }
    setMode('');
  }

  function toggleLanguage(code: string) {
    const next = new Set(value.subtitleLanguages ?? []);
    if (next.has(code)) next.delete(code);
    else if (next.size < MAX_LANGUAGES) next.add(code);
    const fallback = defaultLanguage()?.code;
    const languages = next.size ? [...next] : fallback ? [fallback] : undefined;
    value = { ...value, subtitleLanguages: languages, subtitleAutoCaptions: true };
  }

  function setFormat(next: string) {
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
  <div class="cols">
    <div class="col">
      <slot name="output" />
    </div>
    <div class="col">
      <div class="opt-sec">
        <h3>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><rect x="3" y="5" width="18" height="14" rx="2"/><path d="M7 12h4M7 15.5h7M14.5 12H17"/></svg>
          Subtitles
          <span class="hswitch">
            <button
              type="button"
              class="switch"
              class:on={mode !== ''}
              role="switch"
              aria-checked={mode !== ''}
              aria-label="Subtitles on or off"
              disabled={subtitleChoiceBlocked}
              on:click={toggleSubtitles}
            ><span class="knob"></span></button>
          </span>
        </h3>
        {#if !allowSubtitles}
          <p class="note">Subtitles need a video output.</p>
        {:else if noLanguagesReported}
          <p class="note">No subtitles were reported for this video.</p>
        {:else}
          {#if collectionMode}
            <p class="note">Every video uses English or its first available language.</p>
          {/if}
          <div class="subblock" class:off={subtitlesOff}>
            <div class="chips" role="radiogroup" aria-label="Subtitle mode">
              <button
                type="button"
                class="seg"
                class:on={mode === 'sidecar'}
                role="radio"
                aria-checked={mode === 'sidecar'}
                disabled={subtitleChoiceBlocked}
                on:click={() => setMode('sidecar')}
              >Subtitle file</button>
              <button
                type="button"
                class="seg"
                class:on={mode === 'embed'}
                role="radio"
                aria-checked={mode === 'embed'}
                disabled={subtitleChoiceBlocked || embedBlocked}
                title={embedBlocked ? 'Embedding needs FFmpeg' : ''}
                on:click={() => setMode('embed')}
              >Embed in video</button>
            </div>
            {#if !collectionMode && languages.length}
              <div class="langrow">
                <span class="inline-label">Language</span>
                <div class="fmt hug" bind:this={langRoot}>
                  <button
                    type="button"
                    class="fmt-btn"
                    class:open={langOpen}
                    disabled={subtitlesOff}
                    aria-haspopup="listbox"
                    aria-expanded={langOpen}
                    aria-label="Subtitle language"
                    on:click={() => { if (!subtitlesOff) { langOpen = !langOpen; fmtOpen = false; } }}
                  >
                    <span class="fmt-v">{languageButtonLabel}</span>
                    <span class="fmt-chev" aria-hidden="true">▾</span>
                  </button>
                  {#if langOpen && !subtitlesOff}
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
                            <span class="auto-badge" title="Auto-translated" aria-hidden="true">
                              <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor"><path d="M12 3l1.9 5.6 5.6 1.4-5.6 1.9L12 17.5l-1.9-5.6L4.5 10l5.6-1.4z"/><path d="M19 15l.9 2.6 2.6.9-2.6.9-.9 2.6-.9-2.6-2.6-.9 2.6-.9z"/></svg>
                            </span>
                          </button>
                        {/each}
                        <div class="fmt-more">…and more on YouTube</div>
                      {/if}
                    </div>
                  {/if}
                </div>
              </div>
            {/if}
            {#if mode !== 'embed'}
              <div class="langrow">
                <span class="inline-label">Format</span>
                <div class="fmt hug" bind:this={fmtRoot}>
                  <button
                    type="button"
                    class="fmt-btn"
                    class:open={fmtOpen}
                    disabled={subtitlesOff || !ffmpegAvailable}
                    aria-haspopup="listbox"
                    aria-expanded={fmtOpen}
                    aria-label="Subtitle format"
                    on:click={() => { if (!subtitlesOff && ffmpegAvailable) { fmtOpen = !fmtOpen; langOpen = false; } }}
                  >
                    <span class="fmt-v">{formatLabel}</span>
                    <span class="fmt-chev" aria-hidden="true">▾</span>
                  </button>
                  {#if fmtOpen && !subtitlesOff}
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
                {#if !ffmpegAvailable}
                  <p class="hint">FFmpeg converts subtitle files; the original format is kept without it.</p>
                {/if}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
  <div class="opt-sec meta-full-row">
    <div class="meta-inline-head">
      <h3>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 4h7l9 9-7 7-9-9Z"/><circle cx="9" cy="9" r="1.4"/></svg>
        Metadata
      </h3>
      <div class="meta-checks">
        <label class="check">
          <input type="checkbox" disabled={embedBlocked} checked={!!value.embedMetadata} on:change={(event) => setFlag('embedMetadata', event.currentTarget.checked)} />
          Title &amp; channel details
        </label>
        <label class="check">
          <input type="checkbox" disabled={embedBlocked} checked={!!value.embedThumbnail} on:change={(event) => setFlag('embedThumbnail', event.currentTarget.checked)} />
          Thumbnail artwork
        </label>
        <label class="check">
          <input type="checkbox" disabled={embedBlocked} checked={!!value.embedChapters} on:change={(event) => setFlag('embedChapters', event.currentTarget.checked)} />
          Chapter markers
        </label>
      </div>
    </div>
    {#if embedBlocked}
      <p class="hint">
        Embedding needs FFmpeg.
        <button type="button" class="link" on:click={() => dispatch('goto-settings')}>Open Settings</button>
      </p>
    {/if}
  </div>
</div>

<style>
  .opt-sections {
    border-top: 1px solid var(--border-subtle);
    background: var(--surface-subtle);
    margin-top: 8px;
    flex-shrink: 0;
  }
  .cols {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }
  .cols > .col { min-width: 0; }
  .cols > .col + .col { border-left: 1px solid var(--border-subtle); }
  .opt-sec {
    padding: 10px 12px 12px;
    min-width: 0;
  }
  .cols .opt-sec { border-top: 0; }
  .opt-sec h3 {
    margin: 0 0 8px;
    color: var(--text-muted);
    font-size: 10px;
    font-weight: 650;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    display: flex;
    align-items: center;
    gap: 7px;
  }
  .opt-sec h3 svg,
  .meta-inline-head h3 svg {
    width: 12px;
    height: 12px;
    flex-shrink: 0;
  }
  .hswitch { margin-left: 8px; display: inline-flex; }
  .switch {
    position: relative;
    width: 34px;
    height: 18px;
    border-radius: 999px;
    flex-shrink: 0;
    border: 1px solid var(--border-strong);
    background: var(--surface-base);
    cursor: pointer;
  }
  .switch .knob {
    position: absolute;
    top: 1px;
    left: 1px;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--text-muted);
    transition: transform 120ms ease, background 120ms ease;
  }
  .switch.on {
    background: var(--accent-soft);
    border-color: rgba(59, 130, 246, 0.5);
  }
  .switch.on .knob {
    transform: translateX(16px);
    background: var(--accent-400);
  }
  .switch:disabled { cursor: default; opacity: 0.45; }
  .note, .hint {
    margin: 0;
    color: var(--text-muted);
    font-size: 11px;
    line-height: 1.45;
  }
  .subblock {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 10px;
    align-items: flex-end;
  }
  .subblock > .chips { flex: 1 1 100%; }
  .subblock.off { opacity: 0.55; }
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    min-width: 0;
  }
  .seg {
    height: 26px;
    padding: 0 8px;
    border: 1px solid var(--border-default);
    border-radius: 7px;
    background: var(--surface-base);
    color: var(--text-secondary);
    font-size: 12px;
    white-space: nowrap;
    transition: border-color 120ms ease, color 120ms ease, background 120ms ease;
  }
  .seg:hover:not(:disabled) { border-color: var(--border-strong); color: var(--text-primary); }
  .seg.on {
    border-color: rgba(59, 130, 246, 0.55);
    background: var(--accent-soft);
    color: #93C5FD;
  }
  .seg:disabled { opacity: 0.45; cursor: default; }
  .langrow {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .inline-label {
    color: var(--text-secondary);
    font-size: var(--fs-xs);
    font-weight: 600;
    flex-shrink: 0;
  }
  .fmt { position: relative; min-width: 0; }
  .fmt.hug { width: fit-content; min-width: 0; }
  .fmt-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    height: 28px;
    padding: 0 10px;
    border: 1px solid var(--border-default);
    border-radius: 8px;
    background: var(--surface-base);
    color: var(--text-primary);
    text-align: left;
    cursor: pointer;
  }
  .fmt-btn:hover:not(:disabled) { border-color: var(--border-strong); }
  .fmt-btn.open { border-color: var(--accent-500); }
  .fmt-btn:disabled { opacity: 0.45; cursor: default; }
  .fmt-v {
    min-width: 0;
    flex: 1;
    overflow: hidden;
    color: var(--text-primary);
    font-size: 12px;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .fmt-chev { flex-shrink: 0; color: var(--text-muted); font-size: 10px; }
  .fmt-btn.open .fmt-chev { transform: rotate(180deg); }
  .fmt-menu {
    position: absolute;
    z-index: 30;
    top: calc(100% + 4px);
    left: 0;
    right: auto;
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
  .fmt-opt:has(.tag),
  .fmt-opt:has(.auto-badge) { grid-template-columns: 16px minmax(0, 1fr) auto; }
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
  .auto-badge {
    display: inline-flex;
    padding: 3px;
    border-radius: 6px;
    background: rgba(34, 197, 94, 0.14);
    color: #4ade80;
    flex-shrink: 0;
  }
  .meta-full-row {
    border-top: 1px solid var(--border-subtle);
    background: var(--surface-subtle);
    padding: 10px 14px 11px;
  }
  .meta-inline-head {
    display: flex;
    align-items: center;
    gap: 20px;
    flex-wrap: wrap;
  }
  .meta-inline-head h3 {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 7px;
  }
  .meta-checks {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }
  .check {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: var(--fs-xs);
    font-weight: 550;
    color: var(--text-primary);
    cursor: pointer;
  }
  .check input { flex-shrink: 0; }
  .check input:disabled { cursor: default; }
  .link {
    padding: 0;
    border: 0;
    background: none;
    color: var(--accent-600);
    font-size: 11px;
    font-weight: 650;
    text-decoration: underline;
  }
  @media (max-width: 760px) {
    .cols { grid-template-columns: minmax(0, 1fr); }
    .cols > .col + .col { border-left: 0; border-top: 1px solid var(--border-subtle); }
  }
</style>
