<script lang="ts">
  import { counts, jobs, settings } from '../stores.js';
  import { formatSpeed } from '../format.js';

  $: downloadPath = $settings.downloadFolder || 'Not set';
  $: limit = Math.max(1, $settings.downloadConcurrency || 2);
  $: active = $counts.active;
  $: speedBps = $jobs.reduce((sum, job) => (
    job.status === 'active' && job.speedBps > 0 ? sum + job.speedBps : sum
  ), 0);
  $: occupancy = active > 0
    ? (speedBps > 0 ? `${formatSpeed(speedBps)} · ${active}/${limit} slots` : `${active}/${limit} slots`)
    : 'Idle';
</script>

<footer class="status-bar" aria-label="Application status">
  <div class="path" title={downloadPath}>
    <span class="label">Path</span>
    <strong>{downloadPath}</strong>
  </div>
  <div class="occupancy" class:live={active > 0} aria-label={occupancy}>
    {occupancy}
  </div>
</footer>

<style>
  .status-bar {
    z-index: 20;
    display: flex;
    width: 100%;
    height: var(--statusbar-h);
    min-height: var(--statusbar-h);
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 0 14px;
    border-top: 1px solid var(--border-default);
    background: #0A0A0C;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 10px;
    line-height: 1;
  }

  .path {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 8px;
  }
  .path .label { flex-shrink: 0; }
  .path strong {
    overflow: hidden;
    color: var(--text-muted);
    font-family: inherit;
    font-size: inherit;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .occupancy {
    flex-shrink: 0;
    font-variant-numeric: tabular-nums;
  }
  .occupancy.live { color: var(--accent-400); }
</style>
