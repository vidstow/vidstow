<script lang="ts">
  import { fly } from 'svelte/transition';
  import { banner } from '../stores.js';
  import { MOTION_CLOSE, MOTION_OPEN, motionMs } from '../motion.js';
</script>

{#if $banner}
  <div class="banner-anchor">
    <div
      class="banner {$banner.kind}"
      role="status"
      aria-live="polite"
      in:fly={{ y: 16, duration: motionMs(MOTION_OPEN) }}
      out:fly={{ y: 16, duration: motionMs(MOTION_CLOSE) }}
    >
      <span class="dot" aria-hidden="true"></span>
      <span class="msg">{$banner.message}</span>
    </div>
  </div>
{/if}

<style>
  .banner-anchor {
    position: fixed;
    bottom: calc(var(--statusbar-h) + var(--sp-3));
    left: 50%;
    transform: translateX(-50%);
    z-index: 80;
    max-width: min(560px, 92vw);
  }
  .banner {
    background: var(--surface-raised);
    border: 1px solid var(--border-default);
    border-radius: var(--r-full);
    padding: 10px 16px;
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    box-shadow: var(--shadow-card);
    font-size: var(--fs-sm);
    color: var(--text-primary);
  }
  .dot {
    width: 8px; height: 8px; border-radius: 50%;
  }
  .success .dot { background: var(--status-success); }
  .info .dot    { background: var(--status-info); }
  .warning .dot { background: var(--status-warning); }
  .danger .dot  { background: var(--status-danger); }

  .success { border-color: var(--status-success); }
  .warning { border-color: var(--status-warning); }
  .danger  { border-color: var(--status-danger); }
</style>
