<script lang="ts">
  import { onMount } from 'svelte';

  export type FormatOption = {
    id: string;
    label: string;
    size?: string;
  };

  interface Props {
    label: string;
    options: FormatOption[];
    value: string;
    disabled?: boolean;
    onChange: (id: string) => void;
  }

  let { label, options, value, disabled = false, onChange }: Props = $props();

  let open = $state(false);
  let root: HTMLDivElement | undefined;

  const selected = $derived(options.find((option) => option.id === value));

  function toggle(): void {
    if (disabled || !options.length) return;
    open = !open;
  }

  function choose(id: string): void {
    onChange(id);
    open = false;
  }

  onMount(() => {
    const onPtr = (event: PointerEvent) => {
      if (!open || !root) return;
      if (!root.contains(event.target as Node)) open = false;
    };
    const onDocKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && open) open = false;
    };
    document.addEventListener('pointerdown', onPtr);
    document.addEventListener('keydown', onDocKey);
    return () => {
      document.removeEventListener('pointerdown', onPtr);
      document.removeEventListener('keydown', onDocKey);
    };
  });
</script>

<div class="fmt" bind:this={root}>
  <button
    type="button"
    class="fmt-btn"
    class:open
    {disabled}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-label={label}
    onclick={toggle}
  >
    <span class="fmt-k">{label}</span>
    <span class="fmt-v">{selected?.label ?? 'Choose'}</span>
    <span class="fmt-chev" aria-hidden="true">▾</span>
  </button>
  {#if open}
    <div class="fmt-menu" role="listbox" aria-label={label}>
      {#each options as option (option.id)}
        <button
          type="button"
          class="fmt-opt"
          class:on={option.id === value}
          role="option"
          aria-selected={option.id === value}
          onclick={() => choose(option.id)}
        >
          <span class="fmt-check" aria-hidden="true">{option.id === value ? '✓' : ''}</span>
          <span class="fmt-lab">{option.label}</span>
          {#if option.size}<span class="fmt-size">{option.size}</span>{/if}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .fmt { position: relative; min-width: 0; width: 100%; }
  .fmt-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    height: 28px;
    padding: 0 10px;
    border: 1px solid var(--border-default);
    border-radius: 8px;
    background: var(--surface-base);
    color: var(--text-primary);
    text-align: left;
    cursor: pointer;
  }
  .fmt-btn:hover:not(:disabled) { border-color: var(--border-strong); }
  .fmt-btn.open { border-color: var(--accent-500); }
  .fmt-btn:disabled { opacity: 0.45; cursor: default; }
  .fmt-k {
    flex-shrink: 0;
    color: var(--text-muted);
    font-size: 11px;
    font-weight: 500;
  }
  .fmt-v {
    min-width: 0;
    flex: 1;
    overflow: hidden;
    color: var(--text-primary);
    font-size: 12px;
    font-weight: 500;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .fmt-chev {
    flex-shrink: 0;
    color: var(--text-muted);
    font-size: 10px;
  }
  .fmt-btn.open .fmt-chev { transform: rotate(180deg); }
  .fmt-menu {
    position: absolute;
    z-index: 30;
    top: calc(100% + 4px);
    right: 0;
    left: 0;
    max-height: 280px;
    overflow: auto;
    padding: 4px 0;
    border: 1px solid var(--border-default);
    border-radius: 8px;
    background: var(--surface-raised);
    box-shadow: var(--shadow-card);
  }
  .fmt-opt {
    display: grid;
    grid-template-columns: 16px minmax(0, 1fr);
    align-items: center;
    gap: 8px;
    width: 100%;
    height: 28px;
    padding: 0 12px;
    border: 0;
    background: transparent;
    color: var(--text-secondary);
    font-size: 12px;
    text-align: left;
    cursor: pointer;
  }
  .fmt-opt:has(.fmt-size) { grid-template-columns: 16px minmax(0, 1fr) 10ch; }
  .fmt-opt:hover,
  .fmt-opt.on { background: var(--surface-hover); color: var(--text-primary); }
  .fmt-check {
    color: var(--accent-400);
    font-size: 11px;
  }
  .fmt-lab {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .fmt-size {
    overflow: hidden;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
    text-align: right;
    white-space: nowrap;
  }
</style>
