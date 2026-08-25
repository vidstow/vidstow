<script lang="ts">
  import { trapModalFocus } from './modal.js';

  interface Props {
    open: boolean;
    busy?: boolean;
    onClose?: () => void;
    onEnable?: () => void;
    onDisable?: () => void;
    onPrivacy?: () => void;
  }

  let { open, busy = false, onClose, onEnable, onDisable, onPrivacy }: Props = $props();

  function onKeydown(event: KeyboardEvent): void {
    if (open && event.key === 'Escape') onClose?.();
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <div class="overlay" role="presentation" onclick={(event) => event.target === event.currentTarget && onClose?.()}>
    <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="diagnostic-consent-title" aria-describedby="diagnostic-consent-description" tabindex="-1" use:trapModalFocus>
      <header>
        <h2 id="diagnostic-consent-title">Help improve VidStow?</h2>
        <button type="button" class="close" aria-label="Close" onclick={() => onClose?.()}>×</button>
      </header>
      <p id="diagnostic-consent-description">
        Send a small, sanitized report only when VidStow cannot complete a requested download or encounters an app failure. Reports never include video IDs, links, paths, filenames, cookies, tokens, or error text.
      </p>
      <button type="button" class="privacy" onclick={() => onPrivacy?.()}>Read the diagnostics privacy notice ↗</button>
      <footer>
        <button type="button" class="app-btn" disabled={busy} onclick={() => onDisable?.()}>Don’t send</button>
        <button type="button" class="app-btn primary" disabled={busy} onclick={() => onEnable?.()}>Send diagnostics</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .overlay { position: fixed; inset: 0; z-index: 120; display: grid; place-items: center; padding: var(--sp-4); background: rgba(0, 0, 0, 0.74); backdrop-filter: blur(3px); }
  .dialog { width: min(480px, 94vw); overflow: hidden; padding: var(--sp-4); border: 1px solid var(--border-strong); border-radius: var(--r-lg); background: var(--surface-raised); box-shadow: var(--shadow-modal); font-family: var(--font-sans); }
  header { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: var(--sp-3); }
  h2 { margin: 0; font-size: var(--fs-xl); font-weight: 650; letter-spacing: -0.02em; }
  .close { display: grid; width: 26px; height: 26px; place-items: center; border-radius: var(--r-md); color: var(--text-muted); font-size: 18px; line-height: 1; }
  .close:hover { background: var(--surface-active); color: var(--text-primary); }
  p { margin: var(--sp-3) 0 var(--sp-2); color: var(--text-secondary); font-size: var(--fs-sm); line-height: 1.55; }
  .privacy { padding: 0; color: var(--accent-400); font-size: var(--fs-xs); text-decoration: underline; text-underline-offset: 3px; }
  footer { display: flex; justify-content: flex-end; gap: var(--sp-2); margin: var(--sp-4) calc(-1 * var(--sp-4)) calc(-1 * var(--sp-4)); padding: var(--sp-3) var(--sp-4); border-top: 1px solid var(--border-default); background: var(--surface-base); }
</style>
