<script lang="ts">
  import { trapModalFocus } from '../lifecycle-ui/modal.js';

  export let open = false;
  export let busy = false;
  export let title = '';
  export let available = 0;
  export let settingsSummary = '';
  export let folder = '';
  export let scope: 'future' | 'all' = 'future';
  export let onClose: () => void = () => {};
  export let onConfirm: () => void = () => {};

  $: allLabel = `Download all ${available} now`;
  $: confirmLabel = scope === 'all'
    ? `Follow and download ${available}`
    : 'Follow playlist';
  $: note = scope === 'all'
    ? `We'll follow the playlist, then download all ${available}.`
    : 'Nothing downloads yet. Home stays as you left it.';
</script>

<svelte:window on:keydown={(event) => event.key === 'Escape' && open && !busy && onClose()} />

{#if open}
  <div class="overlay" role="presentation" on:click|self={() => !busy && onClose()}>
    <div class="dialog" use:trapModalFocus tabindex="-1" role="dialog" aria-modal="true" aria-labelledby="follow-setup-title">
      <header>
        <h2 id="follow-setup-title">Follow this playlist?</h2>
        <button class="close" type="button" on:click={onClose} aria-label="Close" disabled={busy}>×</button>
      </header>
      <strong>{title}</strong>
      <div class="callout">
        <p><b>Manual checks.</b> We'll only look for new videos when you hit Check now.</p>
      </div>
      <div class="dscope" role="radiogroup" aria-label="What to follow">
        <button type="button" class="dscope-opt" class:on={scope === 'future'} aria-pressed={scope === 'future'} on:click={() => (scope = 'future')}>
          <b>Future videos only</b>
          <span>The {available} already here won't show up as new.</span>
        </button>
        <button type="button" class="dscope-opt secondary" class:on={scope === 'all'} aria-pressed={scope === 'all'} on:click={() => (scope = 'all')}>
          <b>{allLabel}</b>
          <span>Start those downloads, then keep following for new ones.</span>
        </button>
      </div>
      <dl>
        <div><dt>Download settings</dt><dd>{settingsSummary}</dd></div>
        <div><dt>Destination</dt><dd class="path">{folder}</dd></div>
      </dl>
      <p class="note">{busy ? 'Preparing to follow…' : note}</p>
      <footer>
        <button type="button" class="app-btn" on:click={onClose} disabled={busy}>Not now</button>
        <button type="button" class="app-btn primary" on:click={onConfirm} disabled={busy}>{busy ? 'Following…' : confirmLabel}</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed; inset: 0; display: grid; place-items: center;
    background: rgba(17, 24, 39, 0.32); z-index: 100; padding: 20px;
  }
  .dialog {
    width: min(440px, 94vw); padding: 20px; background: var(--surface-base);
    border: 1px solid var(--border-default); border-radius: var(--r-lg); box-shadow: var(--shadow-modal);
  }
  header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
  h2 { margin: 0; font-size: 17px; letter-spacing: -0.01em; }
  .close { color: var(--text-muted); font-size: 18px; line-height: 1; padding: 2px 6px; }
  .close:hover { color: var(--text-primary); }
  strong { display: block; margin: 12px 0 0; font-size: 14px; }
  .callout {
    margin: 12px 0 0; padding: 10px 12px; border: 1px solid var(--border-default);
    border-radius: 8px; background: var(--surface-raised);
  }
  .callout p { margin: 0; color: var(--text-secondary); font-size: 13px; line-height: 1.45; }
  .callout b { color: var(--text-primary); }
  .dscope { display: flex; flex-direction: column; gap: 8px; margin: 12px 0 4px; }
  .dscope-opt {
    text-align: left; border: 1px solid var(--border-default); border-radius: 8px;
    padding: 10px 12px 10px 36px; background: var(--surface-base); position: relative;
  }
  .dscope-opt::before {
    content: ""; position: absolute; left: 12px; top: 14px; width: 14px; height: 14px;
    border: 1.5px solid var(--border-strong); border-radius: 50%; background: var(--surface-sunken);
  }
  .dscope-opt:hover { border-color: var(--border-strong); }
  .dscope-opt b { display: block; font-size: 13px; }
  .dscope-opt span { display: block; margin-top: 2px; color: var(--text-secondary); font-size: 12px; }
  .dscope-opt.on { border-color: rgba(59, 130, 246, 0.55); background: var(--accent-soft); }
  .dscope-opt.on::before {
    border-color: var(--accent-500); background: radial-gradient(circle, var(--accent-500) 40%, transparent 45%);
  }
  .dscope-opt.secondary span { color: var(--text-muted); }
  dl { margin: 12px 0 0; }
  dl div { display: grid; grid-template-columns: 140px minmax(0, 1fr); gap: 8px; margin-top: 6px; }
  dt { color: var(--text-muted); font-size: 12px; }
  dd { margin: 0; font-size: 12px; color: var(--text-secondary); }
  dd.path { font-family: var(--font-mono); font-size: 11px; overflow: hidden; text-overflow: ellipsis; }
  .note { margin: 12px 0 0; color: var(--text-muted); font-size: 12px; }
  footer { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
</style>
