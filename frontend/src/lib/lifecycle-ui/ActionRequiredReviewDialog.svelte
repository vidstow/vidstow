<script lang="ts">
  import { trapModalFocus } from './modal.js';
  import type { ActionRequiredReviewViewModel } from './types.js';

  interface Props {
    open: boolean;
    review: ActionRequiredReviewViewModel | null;
    busy?: boolean;
    onClose?: () => void;
    onRemove?: () => void;
    onStartOver?: () => void;
    onRetryRecovery?: () => void;
    onRetryFreshLink?: () => void;
    onDiscard?: () => void;
    onRetryCleanup?: () => void;
  }

  let { open, review, busy = false, onClose, onRemove, onStartOver, onRetryRecovery, onRetryFreshLink, onDiscard, onRetryCleanup }: Props = $props();

  function close(): void {
    if (!busy) onClose?.();
  }

  function onKeydown(event: KeyboardEvent): void {
    if (open && event.key === 'Escape') close();
  }

  function onOverlayClick(event: MouseEvent): void {
    if (event.target === event.currentTarget) close();
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open && review}
  <div class="overlay" role="presentation" onclick={onOverlayClick}>
    <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="action-required-title" tabindex="-1" use:trapModalFocus>
      <header class="dialog-header">
        <div>
          <span class="eyebrow">Action required</span>
          <h2 id="action-required-title">{review.heading}</h2>
          <p class="item-title">{review.title}</p>
        </div>
        <button type="button" class="close-button" aria-label="Close" disabled={busy} onclick={close}>	imes</button>
      </header>

      <div class="dialog-body">
        <p>{review.message}</p>
        <div class="preservation">
          <strong>What happens to the saved data?</strong>
          <p>{review.preservationNotice}</p>
        </div>
        {#if review.canStartOver}
          <p class="next-step">Starting over takes you to Home for fresh analysis and a new destination reservation, avoiding conflicts with the preserved attempt.</p>
        {:else if !review.canRetryCleanup}
          <p class="next-step">Starting over is unavailable for this item. You can keep the row for review{review.canRemove ? ' or remove it without deleting saved temporary data' : ''}.</p>
        {/if}
      </div>

      <footer class="dialog-footer">
        <button type="button" class="app-btn" disabled={busy} onclick={close}>Keep for now</button>
        {#if review.canRemove}
          <button type="button" class="app-btn" disabled={busy} onclick={() => onRemove?.()}>Remove from queue</button>
        {/if}
        {#if review.canDiscard}
          <button type="button" class="app-btn danger" disabled={busy} onclick={() => onDiscard?.()}>Discard saved data</button>
        {/if}
        {#if review.canRetryCleanup}
          <button type="button" class="app-btn primary" data-autofocus disabled={busy} onclick={() => onRetryCleanup?.()}>Retry cleanup</button>
        {:else if review.canRetryRecovery}
          <button type="button" class="app-btn" disabled={busy} onclick={() => onRetryRecovery?.()}>Try recovery again</button>
        {/if}
        {#if review.canRetryFreshLink}
          <button type="button" class="app-btn" disabled={busy} onclick={() => onRetryFreshLink?.()}>Retry with fresh link</button>
        {/if}
        {#if review.canStartOver}
          <button type="button" class="app-btn primary" data-autofocus disabled={busy} onclick={() => onStartOver?.()}>
            {busy ? 'Working…' : 'Start over from Home'}
          </button>
        {/if}
      </footer>
    </div>
  </div>
{/if}

<style>
  .overlay { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; padding: var(--sp-4); background: rgba(0, 0, 0, 0.74); backdrop-filter: blur(3px); }
  .dialog { width: min(560px, 94vw); overflow: hidden; border: 1px solid var(--border-strong); border-radius: var(--r-lg); background: var(--surface-raised); box-shadow: var(--shadow-modal); font-family: var(--font-sans); }
  .dialog-header { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--sp-3); padding: var(--sp-4); border-bottom: 1px solid var(--border-default); }
  .eyebrow { color: var(--status-danger); font-size: var(--fs-xs); font-weight: 650; letter-spacing: 0.07em; text-transform: uppercase; }
  h2 { margin: 2px 0 0; font-size: var(--fs-xl); font-weight: 650; letter-spacing: -0.02em; }
  .item-title { max-width: 52ch; margin: 2px 0 0; overflow: hidden; color: var(--text-secondary); font-size: var(--fs-sm); text-overflow: ellipsis; white-space: nowrap; }
  .close-button { display: grid; width: 26px; height: 26px; flex: 0 0 auto; place-items: center; border-radius: var(--r-md); color: var(--text-muted); font-size: 18px; line-height: 1; }
  .close-button:hover { background: var(--surface-active); color: var(--text-primary); }
  .dialog-body { display: flex; flex-direction: column; gap: var(--sp-3); padding: var(--sp-4); }
  .dialog-body > p { margin: 0; color: var(--text-primary); font-size: var(--fs-sm); line-height: 1.5; }
  .preservation { padding: var(--sp-3); border: 1px solid var(--border-default); border-radius: var(--r-md); background: var(--surface-sunken); }
  .preservation strong { font-size: var(--fs-sm); font-weight: 600; }
  .preservation p, .next-step { margin: var(--sp-1) 0 0; color: var(--text-secondary) !important; font-size: var(--fs-sm); line-height: 1.5; }
  .dialog-footer { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: var(--sp-2); padding: var(--sp-3) var(--sp-4); border-top: 1px solid var(--border-default); background: var(--surface-base); }
  @media (max-width: 560px) { .dialog-footer { flex-direction: column-reverse; } .dialog-footer .app-btn { width: 100%; } }
</style>
