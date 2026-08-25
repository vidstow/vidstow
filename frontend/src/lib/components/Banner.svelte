<script lang="ts">
  import { banner } from '../stores.js';
</script>

{#if $banner}
  <div class="banner {$banner.kind}" role="status" aria-live="polite">
    <span class="dot" aria-hidden="true"></span>
    <span class="msg">{$banner.message}</span>
  </div>
{/if}

<style>
  .banner {
    position: fixed;
    bottom: calc(var(--statusbar-h) + var(--sp-3));
    left: 50%;
    z-index: 80;
    display: flex;
    max-width: min(560px, 92vw);
    min-height: 32px;
    padding: 7px 11px;
    transform: translateX(-50%);
    align-items: center;
    gap: var(--sp-2);
    border: 1px solid var(--border-strong);
    border-radius: var(--r-md);
    background: var(--surface-raised);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
    color: var(--text-primary);
    font-family: var(--font-sans);
    font-size: var(--fs-sm);
    line-height: 1.4;
  }
  .dot { width: 6px; height: 6px; flex: 0 0 auto; border-radius: 50%; }
  .success .dot { background: var(--status-success); }
  .info .dot { background: var(--status-info); }
  .warning .dot { background: var(--status-warning); }
  .danger .dot { background: var(--status-danger); }
  .success { border-color: rgba(34, 197, 94, 0.4); }
  .warning { border-color: rgba(245, 158, 11, 0.4); }
  .danger { border-color: rgba(239, 68, 68, 0.4); }
</style>
