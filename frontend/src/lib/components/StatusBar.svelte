<script lang="ts">
  import { api } from '../api.js';
  import { counts, jobs, settings, showError } from '../stores.js';
  import { formatSpeed } from '../format.js';

  $: folder = $settings.downloadFolder?.trim() ?? '';
  $: downloadPath = folder || 'Not set';
  $: limit = Math.max(1, $settings.downloadConcurrency || 2);
  $: active = $counts.active;
  $: speedBps = $jobs.reduce((sum, job) => (
    job.status === 'active' && job.speedBps > 0 ? sum + job.speedBps : sum
  ), 0);
  $: occupancy = active > 0
    ? (speedBps > 0 ? `${formatSpeed(speedBps)} · ${active}/${limit} slots` : `${active}/${limit} slots`)
    : 'Idle';

  async function openFolder() {
    if (!folder) return;
    try {
      await api.fs.reveal(folder);
    } catch (err) {
      showError(err, 'That folder could not be opened');
    }
  }
</script>

<footer class="status-bar" aria-label="Application status">
  <button
    type="button"
    class="path"
    title={downloadPath}
    disabled={!folder}
    aria-label={folder ? `Open download folder ${downloadPath}` : 'Download folder not set'}
    on:click={openFolder}
  >
    <strong>{downloadPath}</strong>
  </button>
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
    color: var(--text-secondary);
    font-family: var(--font-mono);
    font-size: 11px;
    line-height: 1;
  }

  .path {
    display: flex;
    min-width: 0;
    align-items: center;
    text-align: left;
    color: inherit;
    font: inherit;
  }
  .path:hover:not(:disabled) strong { color: var(--text-primary); }
  .path:disabled {
    cursor: default;
  }
  .path strong {
    overflow: hidden;
    color: var(--text-secondary);
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
