<script lang="ts">
  import type { Snippet } from 'svelte';

  type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
  type Size = 'sm' | 'md' | 'lg';

  interface Props {
    variant?: Variant;
    size?: Size;
    disabled?: boolean;
    loading?: boolean;
    fullWidth?: boolean;
    label: string;
    children?: Snippet;
  }

  let {
    variant = 'secondary',
    size = 'md',
    disabled = false,
    loading = false,
    fullWidth = false,
    label,
    children,
    ...rest
  }: Props & Record<string, unknown> = $props();

  const classes = $derived(['btn', `variant-${variant}`, `size-${size}`].filter(Boolean).join(' '));
</script>

<button
  type="button"
  class={classes}
  class:disabled={disabled || loading}
  class:full={fullWidth}
  {disabled}
  aria-busy={loading || undefined}
  aria-label={loading ? `${label}…` : undefined}
  {...rest}
>
  {#if loading}
    <span class="spinner" aria-hidden="true"></span>
  {/if}
  {#if children}{@render children()}{:else}{label}{/if}
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--sp-2);
    border-radius: var(--r-md);
    font-family: var(--font-sans);
    font-weight: 550;
    line-height: 1;
    white-space: nowrap;
    cursor: pointer;
    transition: background 120ms ease, border-color 120ms ease, color 120ms ease, box-shadow 120ms ease;
  }
  .btn:focus-visible { outline: 2px solid var(--accent-400); outline-offset: 2px; }
  .btn:disabled { cursor: not-allowed; opacity: 0.42; }
  .size-sm { min-height: 24px; padding: 0 var(--sp-2); font-size: var(--fs-xs); }
  .size-md { min-height: 28px; padding: 0 10px; font-size: var(--fs-sm); }
  .size-lg { min-height: 32px; padding: 0 var(--sp-3); font-size: var(--fs-md); }
  .full { width: 100%; }
  .variant-primary { border: 1px solid var(--accent-500); background: var(--accent-500); color: var(--text-on-accent); }
  .variant-primary:hover:not(:disabled) { border-color: var(--accent-400); background: var(--accent-400); }
  .variant-secondary { border: 1px solid var(--border-default); background: var(--surface-raised); color: var(--text-secondary); }
  .variant-secondary:hover:not(:disabled) { border-color: var(--border-strong); background: var(--surface-hover); color: var(--text-primary); }
  .variant-ghost { border: 1px solid transparent; background: transparent; color: var(--text-secondary); }
  .variant-ghost:hover:not(:disabled) { background: var(--surface-active); color: var(--text-primary); }
  .variant-danger { border: 1px solid var(--border-default); background: var(--surface-raised); color: var(--status-danger); }
  .variant-danger:hover:not(:disabled) { border-color: rgba(239, 68, 68, 0.45); background: var(--status-danger-soft); }
  .spinner { width: 12px; height: 12px; border: 2px solid currentColor; border-top-color: transparent; border-radius: 50%; animation: spin 700ms linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
