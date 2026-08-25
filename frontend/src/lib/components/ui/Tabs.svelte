<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface TabOption {
    value: string;
    label: string;
    disabled?: boolean;
    count?: number;
  }

  export interface TabItem<T extends string = string> extends TabOption {
    value: T;
  }

  interface Props {
    options: TabOption[];
    value: string;
    onChange?: (value: string) => void;
    children: Snippet;
    ariaLabel?: string;
  }

  let { options, value, onChange, children, ariaLabel }: Props = $props();
</script>

<div class="tabs" role="tablist" aria-label={ariaLabel}>
  {#each options as option (option.value)}
    <button
      type="button"
      role="tab"
      class="tab"
      class:active={value === option.value}
      aria-selected={value === option.value}
      disabled={option.disabled}
      onclick={() => onChange?.(option.value)}
    >
      {option.label}
      {#if option.count !== undefined}
        <span class="count" aria-hidden="true">{option.count}</span>
      {/if}
    </button>
  {/each}
</div>

<div class="tab-panel">{@render children()}</div>

<style>
  .tabs {
    display: inline-flex;
    gap: 2px;
    padding: 2px;
    background: var(--surface-sunken);
    border: 1px solid var(--border-default);
    border-radius: var(--r-md);
  }
  .tab {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-height: 26px;
    padding: 0 9px;
    border-radius: var(--r-sm);
    color: var(--text-secondary);
    font-size: var(--fs-sm);
    font-weight: 500;
    transition: background 120ms ease, color 120ms ease;
  }
  .tab:hover:not(:disabled):not(.active) { color: var(--text-primary); }
  .tab.active {
    color: var(--text-primary);
    background: var(--surface-active);
    box-shadow: none;
  }
  .tab:disabled { cursor: not-allowed; opacity: 0.5; }
  .count {
    min-width: 15px;
    padding: 0 4px;
    border-radius: var(--r-full);
    background: var(--border-subtle);
    font-size: 11px;
    text-align: center;
  }
  .tab.active .count { background: var(--accent-soft); color: var(--accent-600); }
  .tab-panel { margin-top: var(--sp-2); }
</style>
