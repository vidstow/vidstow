<script lang="ts">
  import { modal } from '../stores.js';
  import { trapModalFocus } from '../lifecycle-ui/modal.js';
  function close() { modal.set(null); }
  $: current = $modal;
  $: choice = current?.kind === 'confirm' || current?.kind === 'ffmpeg-missing' || current?.kind?.startsWith('confirm-');
</script>

<svelte:window on:keydown={(event) => event.key === 'Escape' && close()} />

{#if current}
  <div class="overlay" role="presentation" on:click|self={close}>
    <div
      class:ffmpeg={current.kind === 'ffmpeg-missing'}
      class:confirm={choice}
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      tabindex="-1"
      use:trapModalFocus
    >
      <header>
        <h2 id="modal-title">{current.title}</h2>
        <button class="close" type="button" on:click={close} aria-label="Close">×</button>
      </header>
      <p class="lead">{current.message}</p>
      {#if current.kind === 'ffmpeg-missing'}
        <p class="fine-print">VidStow will continue to offer outputs that do not need FFmpeg.</p>
      {/if}
      {#if current.detail}<pre>{current.detail}</pre>{/if}
      <footer>
        <button type="button" class="app-btn" on:click={close}>Close</button>
        {#each current.actions || [] as action}
          <button class="app-btn" class:primary={action.primary} data-autofocus={action.primary || undefined} type="button" on:click={() => { action.action(); close(); }}>{action.label}</button>
        {/each}
      </footer>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: grid;
    place-items: center;
    padding: var(--sp-4);
    background: rgba(0, 0, 0, 0.72);
    backdrop-filter: blur(3px);
  }
  .dialog {
    width: min(440px, 94vw);
    overflow: hidden;
    padding: var(--sp-4);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-lg);
    background: var(--surface-raised);
    box-shadow: var(--shadow-modal);
    font-family: var(--font-sans);
  }
  header { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: var(--sp-3); }
  h2 { margin: 0; font-size: var(--fs-xl); font-weight: 650; letter-spacing: -0.02em; line-height: 1.25; }
  .close { display: grid; width: 26px; height: 26px; place-items: center; border-radius: var(--r-md); color: var(--text-muted); font-size: 18px; line-height: 1; }
  .close:hover { background: var(--surface-active); color: var(--text-primary); }
  .lead { margin: var(--sp-3) 0; color: var(--text-secondary); font-size: var(--fs-sm); line-height: 1.5; }
  .fine-print { margin: var(--sp-2) 0; color: var(--text-muted); font-size: var(--fs-xs); }
  .dialog pre {
    max-height: 160px;
    overflow: auto;
    margin: var(--sp-3) 0 0;
    padding: var(--sp-3);
    border: 1px solid var(--border-subtle);
    border-radius: var(--r-md);
    background: var(--surface-sunken);
    color: var(--text-secondary);
    font: var(--fs-xs)/1.5 var(--font-mono);
    white-space: pre-wrap;
  }
  footer { display: flex; justify-content: flex-end; gap: var(--sp-2); margin-top: var(--sp-4); }
</style>
