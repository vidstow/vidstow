<script lang="ts">
  import type { QueueSummaryViewModel } from './types.js';

  interface Props {
    summary: QueueSummaryViewModel;
  }

  let { summary }: Props = $props();

</script>

<div class="summary" aria-label="Queue summary">
  <div class="occupancy" role="status" aria-label={`${summary.occupiedSlots} of ${summary.slotLimit} active slots, ${summary.waitingJobs} waiting, ${summary.pausedJobs} paused`}>
    <span><strong>{summary.occupiedSlots} of {summary.slotLimit} active slots</strong></span>
    <span aria-hidden="true" class="divider"></span>
    <span>{summary.waitingJobs} waiting</span>
    <span aria-hidden="true" class="divider"></span>
    <span>{summary.pausedJobs} paused</span>
    {#if summary.processingLimit !== undefined}
      <span aria-hidden="true" class="divider"></span>
      <span>{summary.processingOccupied ?? 0} of {summary.processingLimit} processing</span>
    {/if}
  </div>
</div>

<style>
  .summary {
    display: flex;
    flex-direction: column;
  }

  .occupancy {
    display: inline-flex;
    align-items: center;
    align-self: flex-start;
    gap: var(--sp-3);
    min-height: 30px;
    padding: 0 var(--sp-3);
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
    color: var(--text-secondary);
    background: var(--surface-raised);
    font-size: var(--fs-sm);
    white-space: nowrap;
  }

  .occupancy strong {
    color: var(--text-secondary);
    font-weight: 600;
  }

  .divider {
    width: 1px;
    height: 14px;
    background: var(--border-default);
  }

  @media (max-width: 560px) {
    .occupancy {
      align-items: flex-start;
      flex-wrap: wrap;
      gap: var(--sp-2) var(--sp-3);
      padding: var(--sp-2) var(--sp-3);
      white-space: normal;
    }

    .divider { display: none; }
  }
</style>
