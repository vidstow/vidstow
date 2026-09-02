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
        <button type="button" class="close" aria-label="Close without sending diagnostics" onclick={() => onClose?.()}>×</button>
      </header>

      <div class="body">
        <p id="diagnostic-consent-description">
          When VidStow cannot complete a requested download or encounters an app failure, it can send a small sanitized report.
        </p>
        <p>
          Reports never include video IDs, links, paths, filenames, cookies, tokens, or error text.
        </p>
        <button type="button" class="privacy" onclick={() => onPrivacy?.()}>Diagnostics privacy notice ↗</button>
        <p class="later">You can change this later in Settings. Closing this window leaves sending off.</p>
      </div>

      <footer>
        <button type="button" class="app-btn" disabled={busy} onclick={() => onDisable?.()}>Don’t send</button>
        <button type="button" class="app-btn" disabled={busy} onclick={() => onEnable?.()}>Send diagnostics</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 120;
    display: grid;
    place-items: center;
    padding: var(--sp-5);
    background: rgba(24, 25, 28, 0.45);
    backdrop-filter: blur(2px);
  }
  .dialog {
    width: min(460px, 94vw);
    overflow: hidden;
    border: 1px solid var(--border-default);
    border-radius: 10px;
    background: var(--surface-base);
    box-shadow: var(--shadow-modal);
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-3);
    padding: 18px 20px 0;
  }
  h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 650;
    letter-spacing: -0.02em;
  }
  .close {
    width: 28px;
    height: 28px;
    border-radius: var(--r-sm);
    color: var(--text-muted);
    font-size: 18px;
    line-height: 1;
  }
  .close:hover { background: var(--surface-hover); color: var(--text-primary); }
  .body { padding: 12px 20px 0; }
  p {
    margin: 0 0 10px;
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.5;
  }
  .privacy {
    padding: 0;
    color: var(--accent-primary);
    font-size: 12px;
    text-decoration: underline;
    text-underline-offset: 3px;
  }
  .later {
    margin: 12px 0 0;
    color: var(--text-muted);
    font-size: 12px;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 16px;
    padding: 14px 20px 16px;
    border-top: 1px solid var(--border-subtle);
  }
  footer .app-btn { min-width: 132px; }
  @media (max-width: 520px) {
    footer { flex-direction: column-reverse; }
    footer .app-btn { width: 100%; min-width: 0; }
  }
</style>
