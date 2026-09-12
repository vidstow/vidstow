<script lang="ts">
  import { trapModalFocus } from './modal.js';

  interface Props {
    onOpenDataFolder?: () => void;
  }

  let { onOpenDataFolder }: Props = $props();
</script>

<div class="overlay" role="presentation">
  <div
    class="dialog"
    role="dialog"
    aria-modal="true"
    aria-labelledby="cannot-save-title"
    aria-describedby="cannot-save-description"
    tabindex="-1"
    use:trapModalFocus
  >
    <header>
      <h2 id="cannot-save-title">VidStow cannot save this session</h2>
    </header>
    <div class="body">
      <p id="cannot-save-description">
        VidStow could not write its data folder. Settings and the queue will not persist until this folder is writable.
      </p>
    </div>
    <footer>
      <button type="button" class="app-btn" onclick={() => onOpenDataFolder?.()}>Open data folder</button>
    </footer>
  </div>
</div>

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
    padding: 18px 20px 0;
  }
  h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 650;
    letter-spacing: -0.02em;
  }
  .body { padding: 12px 20px 0; }
  p {
    margin: 0;
    color: var(--text-secondary);
    font-size: 13px;
    line-height: 1.5;
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
