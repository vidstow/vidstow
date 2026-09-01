<script lang="ts">
  interface Props {
    title: string;
    message: string;
    icon: 'queue' | 'downloads' | 'search';
    action: string;
    actionKind?: 'home' | 'ghost';
    onAction: () => void;
  }

  let { title, message, icon, action, actionKind = 'home', onAction }: Props = $props();
</script>

<div class="page-empty" role="status">
  <div class="stack">
    <div class="mark" aria-hidden="true">
      {#if icon === 'queue'}
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 6h16M4 12h16M4 18h10"/></svg>
      {:else if icon === 'downloads'}
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M12 4v12m0 0l-4-4m4 4l4-4M5 20h14"/></svg>
      {:else}
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>
      {/if}
    </div>
    <h2 class="t">{title}</h2>
    <p class="s">{message}</p>
    <button type="button" class="cta" class:ghost={actionKind === 'ghost'} onclick={onAction}>
      {#if actionKind === 'home'}
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 11l9-7 9 7v9a2 2 0 0 1-2 2h-4v-7H9v7H5a2 2 0 0 1-2-2z"/></svg>
      {/if}
      {action}
    </button>
  </div>
</div>

<style>
  .page-empty {
    flex: 1;
    min-height: 0;
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding: 56px var(--page-pad-x) 40px;
  }
  .stack {
    display: flex;
    flex-direction: column;
    align-items: center;
    max-width: 36ch;
    text-align: center;
  }
  .mark {
    display: grid;
    place-items: center;
    width: 40px;
    height: 40px;
    margin-bottom: 14px;
    border-radius: 10px;
    border: 1px solid var(--border-default);
    background: var(--surface-raised);
    color: var(--text-muted);
  }
  .t {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
    letter-spacing: -0.015em;
    line-height: 1.3;
    color: var(--text-primary);
  }
  .s {
    margin: 6px 0 0;
    color: var(--text-muted);
    font-size: 13px;
    line-height: 1.45;
    white-space: pre-line;
  }
  .cta {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 7px;
    height: 32px;
    margin-top: 18px;
    padding: 0 14px 0 12px;
    border: 1px solid var(--accent-600);
    border-radius: 8px;
    background: var(--accent-600);
    color: #fff;
    font-family: inherit;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: background-color 120ms ease, border-color 120ms ease;
  }
  .cta:hover { background: #1D4ED8; border-color: #1D4ED8; }
  .cta:focus-visible {
    outline: 2px solid var(--accent-400);
    outline-offset: 2px;
  }
  .cta svg { flex-shrink: 0; }
  .cta.ghost {
    background: transparent;
    border-color: var(--border-default);
    color: var(--text-primary);
    font-weight: 500;
  }
  .cta.ghost:hover {
    background: var(--surface-hover);
    border-color: var(--border-default);
  }
</style>
