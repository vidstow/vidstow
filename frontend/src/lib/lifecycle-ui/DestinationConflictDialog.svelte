<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { trapModalFocus } from './modal.js';
  import {
    isValidConflictToken,
    type DestinationConflictEventDetail,
    type DestinationConflictViewModel,
  } from './types.js';

  export interface DestinationConflictEvents {
    close: void;
    'cancel-download': DestinationConflictEventDetail;
    'use-new-name': DestinationConflictEventDetail;
  }

  interface Props {
    open: boolean;
    conflict: DestinationConflictViewModel;
  }

  let { open, conflict }: Props = $props();
  const dispatch = createEventDispatcher<DestinationConflictEvents>();
  const proposedNameAvailable = $derived(conflict.proposedNameAvailable === true);
  const hasConflictAuthority = $derived(isValidConflictToken(conflict.conflictToken));

  function onKeydown(event: KeyboardEvent): void {
    if (open && event.key === 'Escape') dispatch('close');
  }

  function onOverlayClick(event: MouseEvent): void {
    if (event.target === event.currentTarget) dispatch('close');
  }

  function useNewName(): void {
    if (proposedNameAvailable && hasConflictAuthority) {
      dispatch('use-new-name', { conflictToken: conflict.conflictToken });
    }
  }

  function cancelDownload(): void {
    if (hasConflictAuthority) {
      dispatch('cancel-download', { conflictToken: conflict.conflictToken });
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <div class="overlay" role="presentation" onclick={onOverlayClick}>
    <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="lifecycle-conflict-title" tabindex="-1" use:trapModalFocus>
      <header class="dialog-header">
        <div>
          <h2 id="lifecycle-conflict-title">Choose a new filename</h2>
          <p>The reserved filename is no longer available. VidStow will not replace the existing file.</p>
        </div>
        <button type="button" class="close-button" aria-label="Close" onclick={() => dispatch('close')}>×</button>
      </header>

      <div class="dialog-body">
        <div class="field">
          <label for="lifecycle-unavailable-name">Unavailable name</label>
          <input id="lifecycle-unavailable-name" value={conflict.unavailableName} readonly aria-readonly="true" />
        </div>

        <div class="field">
          <div class="field-label">
            <label for="lifecycle-proposed-name">New reserved name</label>
            {#if proposedNameAvailable}<span class="available"><span aria-hidden="true">✓</span> Available</span>{/if}
          </div>
          <input id="lifecycle-proposed-name" value={conflict.proposedName} readonly aria-readonly="true" />
        </div>

        <p class="unchanged-copy">The existing file will remain unchanged.</p>
      </div>

      <footer class="dialog-footer">
        <button type="button" class="app-btn" disabled={!hasConflictAuthority} onclick={cancelDownload}>Cancel download</button>
        <button type="button" class="app-btn primary" data-autofocus disabled={!proposedNameAvailable || !hasConflictAuthority} onclick={useNewName}>Use new name</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .overlay { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; padding: var(--sp-4); background: rgba(0, 0, 0, 0.74); backdrop-filter: blur(3px); }
  .dialog { width: min(540px, 94vw); overflow: hidden; border: 1px solid var(--border-strong); border-radius: var(--r-lg); background: var(--surface-raised); box-shadow: var(--shadow-modal); font-family: var(--font-sans); }
  .dialog-header { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--sp-3); padding: var(--sp-4); border-bottom: 1px solid var(--border-default); }
  h2 { margin: 0; font-size: var(--fs-xl); font-weight: 650; letter-spacing: -0.02em; }
  .dialog-header p { max-width: 56ch; margin: var(--sp-1) 0 0; color: var(--text-secondary); font-size: var(--fs-sm); line-height: 1.45; }
  .close-button { display: grid; width: 26px; height: 26px; flex: 0 0 auto; place-items: center; border-radius: var(--r-md); color: var(--text-muted); font-size: 18px; line-height: 1; }
  .close-button:hover { background: var(--surface-active); color: var(--text-primary); }
  .dialog-body { display: flex; flex-direction: column; gap: var(--sp-3); padding: var(--sp-4); }
  .field { display: flex; flex-direction: column; gap: 6px; }
  .field-label { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-3); }
  label { color: var(--text-secondary); font-size: var(--fs-xs); font-weight: 600; }
  input { height: 32px; border-color: var(--border-default); border-radius: var(--r-md); background: var(--surface-sunken); color: var(--text-secondary); font-family: var(--font-mono); font-size: var(--fs-xs); }
  .available { display: inline-flex; align-items: center; gap: 5px; color: var(--status-success); font-size: var(--fs-xs); font-weight: 600; }
  .available span { display: inline-grid; width: 15px; height: 15px; place-items: center; border: 1px solid currentColor; border-radius: 50%; }
  .unchanged-copy { margin: 0; color: var(--text-muted); font-size: var(--fs-xs); }
  .dialog-footer { display: flex; justify-content: flex-end; gap: var(--sp-2); padding: var(--sp-3) var(--sp-4); border-top: 1px solid var(--border-default); background: var(--surface-base); }
  @media (max-width: 560px) { .dialog-footer { flex-direction: column-reverse; } .dialog-footer .app-btn { width: 100%; } }
</style>
