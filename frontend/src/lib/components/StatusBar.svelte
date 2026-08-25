<script lang="ts">
  import { ffmpeg, settings } from '../stores.js';

  $: downloadPath = $settings.downloadFolder || 'Not set';
</script>

<footer class="status-bar" aria-label="Application status">
  <div class="path" title={downloadPath}>
    <span>Path</span>
    <strong>{downloadPath}</strong>
  </div>
  <div class="engine" aria-label="Engine ytdlp-go ready">
    <span>Engine</span>
    <strong>ytdlp-go ready</strong>
  </div>
  <div class="ffmpeg" class:ready={$ffmpeg.available}>
    {$ffmpeg.available ? 'FFmpeg ready' : 'FFmpeg required'}
  </div>
</footer>

<style>
  .status-bar {
    z-index: 20;
    display: grid;
    width: 100%;
    height: var(--statusbar-h);
    min-height: var(--statusbar-h);
    grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
    align-items: center;
    padding: 0 14px;
    border-top: 1px solid var(--border-default);
    background: var(--surface-sunken);
    color: var(--text-muted);
    font-size: 9px;
    line-height: 1;
  }

  .path,
  .engine {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 8px;
  }

  .path strong,
  .engine strong {
    overflow: hidden;
    color: var(--text-secondary);
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .engine { justify-self: center; }
  .ffmpeg { justify-self: end; color: var(--status-danger); }
  .ffmpeg.ready { color: var(--status-success); }
</style>
