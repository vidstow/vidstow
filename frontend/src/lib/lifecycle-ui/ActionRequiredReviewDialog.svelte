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

  const compactRetry = $derived(
    Boolean(
      review?.canRetryFreshLink &&
        !review.canRetryRecovery &&
        !review.canDiscard &&
        !review.canStartOver &&
        !review.canRetryCleanup,
    ),
  );

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
  <div class="overlay" class:compact={compactRetry} role="presentation" onclick={onOverlayClick}>
    <div
      class="dialog"
      class:compact={compactRetry}
      role="dialog"
      aria-modal="true"
      aria-labelledby="action-required-title"
      tabindex="-1"
      use:trapModalFocus
    >
      <header class="dialog-header">
        <h2 id="action-required-title">{review.heading}</h2>
        <button type="button" class="close-button" aria-label="Close" disabled={busy} onclick={close}>×</button>
      </header>

      <div class="dialog-body">
        {#if !compactRetry && review.title}
          <p class="item-title">{review.title}</p>
        {/if}
        <p class="lead">{review.message}</p>
        {#if review.preservationNotice}
          <div class="preservation">
            <strong>What happens to the saved data?</strong>
            <p>{review.preservationNotice}</p>
          </div>
        {/if}
        {#if review.canStartOver}
          <p class="next-step">Starting over takes you to Home for fresh analysis and a new destination reservation, avoiding conflicts with the preserved attempt.</p>
        {:else if !compactRetry && !review.canRetryCleanup}
          <p class="next-step">Starting over is unavailable for this item. You can keep the row for review{review.canRemove ? ' or remove it without deleting saved temporary data' : ''}.</p>
        {/if}
      </div>

      <footer class="dialog-footer">
        {#if review.canRemove}
          <button type="button" class="app-btn" disabled={busy} onclick={() => onRemove?.()}>{compactRetry ? 'Remove' : 'Remove from queue'}</button>
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
          <button
            type="button"
            class="app-btn"
            class:primary={compactRetry}
            data-autofocus={compactRetry ? true : undefined}
            disabled={busy}
            onclick={() => onRetryFreshLink?.()}
          >{compactRetry ? 'Retry' : 'Retry with fresh link'}</button>
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
  .overlay { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; padding: 20px; background: rgba(17, 24, 39, 0.32); }
  .overlay:not(.compact) { padding: var(--sp-5); background: rgba(24, 25, 28, 0.45); backdrop-filter: blur(2px); }
  .dialog { width: min(548px, 94vw); overflow: hidden; border: 1px solid var(--border-default); border-radius: var(--r-lg); background: var(--surface-base); box-shadow: var(--shadow-modal); }
  .dialog.compact { width: min(320px, 94vw); }
  .dialog-header { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-3); padding: var(--sp-5) var(--sp-6) var(--sp-3); }
  .dialog.compact .dialog-header { padding: 14px 16px 0; }
  h2 { margin: 0; font-size: 17px; font-weight: 700; letter-spacing: -0.01em; }
  .dialog.compact h2 { font-size: 15px; font-weight: 650; letter-spacing: -0.02em; }
  .item-title { max-width: 52ch; margin: 0; overflow: hidden; color: var(--text-muted); font-size: var(--fs-sm); text-overflow: ellipsis; white-space: nowrap; }
  .close-button { width: 32px; height: 32px; flex: 0 0 auto; border-radius: var(--r-sm); color: var(--text-muted); font-size: 20px; line-height: 1; }
  .close-button:hover { background: var(--surface-hover); }
  .dialog-body { display: flex; flex-direction: column; gap: var(--sp-4); padding: 0 var(--sp-6) var(--sp-4); }
  .dialog.compact .dialog-body { gap: 0; padding: 8px 16px 0; }
  .lead { margin: 0; color: var(--text-secondary); font-size: 13px; line-height: 1.55; }
  .dialog.compact .lead { line-height: 1.45; }
  .preservation { padding: var(--sp-4); border: 1px solid var(--border-default); border-radius: var(--r-md); background: var(--surface-sunken); }
  .preservation strong { font-size: var(--fs-sm); }
  .preservation p, .next-step { margin: var(--sp-1) 0 0; color: var(--text-secondary) !important; font-size: var(--fs-sm); line-height: 1.5; }
  .dialog-footer { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; padding: var(--sp-3) var(--sp-6) var(--sp-5); }
  .dialog.compact .dialog-footer { margin-top: 0; padding: 14px 16px 12px; }
  @media (max-width: 560px) {
    .dialog-header, .dialog-body { padding-left: var(--sp-4); padding-right: var(--sp-4); }
    .dialog.compact .dialog-header { padding: 14px 16px 0; }
    .dialog.compact .dialog-body { padding: 8px 16px 0; }
    .dialog-footer { flex-direction: column-reverse; padding-left: var(--sp-4); padding-right: var(--sp-4); }
    .dialog.compact .dialog-footer { padding: 14px 16px 12px; }
    .dialog-footer .app-btn { width: 100%; }
  }
</style>
