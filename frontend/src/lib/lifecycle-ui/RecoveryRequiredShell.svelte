<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { DEFAULT_RECOVERY_REQUIRED, type RecoveryRequiredViewModel } from './types.js';

  export interface RecoveryRequiredEvents {
    'copy-diagnostics': void;
    'open-data-folder': void;
  }

  interface Props {
    model?: RecoveryRequiredViewModel;
    title?: string;
    subtitle?: string;
    onCopyDiagnostics?: () => void;
    onOpenDataFolder?: () => void;
  }

  let {
    model = DEFAULT_RECOVERY_REQUIRED,
    title = 'Queue',
    subtitle = 'Saved queue unavailable',
    onCopyDiagnostics,
    onOpenDataFolder,
  }: Props = $props();
  const dispatch = createEventDispatcher<RecoveryRequiredEvents>();

  function copyDiagnostics(): void {
    dispatch('copy-diagnostics');
    onCopyDiagnostics?.();
  }

  function openDataFolder(): void {
    dispatch('open-data-folder');
    onOpenDataFolder?.();
  }
</script>

<section class="recovery-shell" aria-labelledby="lifecycle-recovery-title">
  <header class="page-header">
    <h1>{title}</h1>
    <p>{subtitle}</p>
  </header>

  <section class="recovery-card" aria-label="Recovery required">
    <div class="warning-icon" aria-hidden="true">
      <svg viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2.4">
        <path d="m21.2 7.5-17 29.3a3.4 3.4 0 0 0 3 5.1h33.6a3.4 3.4 0 0 0 3-5.1l-17-29.3a3.4 3.4 0 0 0-5.6 0Z" />
        <path d="M24 17v10" />
        <path d="M24 34h.02" stroke-width="3.8" />
      </svg>
    </div>
    <h2 id="lifecycle-recovery-title">Download state needs recovery</h2>
    <p class="intro">VidStow could not safely read its saved queue. Your media and recovery files were preserved.</p>
    <p class="intro">Downloads are paused until the saved state is reviewed. VidStow will not resume, retry, cancel, or clean up saved work.</p>

    <dl class="status-list">
      <div><dt>State file</dt><dd>{model.stateFileStatus ?? DEFAULT_RECOVERY_REQUIRED.stateFileStatus}</dd></div>
      <div><dt>Automatic cleanup</dt><dd>{model.automaticCleanupStatus ?? DEFAULT_RECOVERY_REQUIRED.automaticCleanupStatus}</dd></div>
      <div><dt>Saved media</dt><dd>{model.savedMediaStatus ?? DEFAULT_RECOVERY_REQUIRED.savedMediaStatus}</dd></div>
    </dl>

    <div class="actions">
      <button type="button" class="app-btn" onclick={copyDiagnostics}>Copy diagnostics</button>
      <button type="button" class="app-btn primary" onclick={openDataFolder}>Open data folder</button>
    </div>

    <p class="footer-message">
      <span class="small-warning" aria-hidden="true">!</span>
      {model.footerMessage ?? DEFAULT_RECOVERY_REQUIRED.footerMessage}
    </p>
  </section>
</section>

<style>
  .recovery-shell { width: min(100%, 780px); margin: 0 auto; padding: var(--sp-6) var(--sp-5) var(--sp-7); font-family: var(--font-sans); }
  .page-header h1 { margin: 0; font-size: var(--fs-2xl); font-weight: 650; letter-spacing: -0.025em; }
  .page-header p { margin: var(--sp-1) 0 0; color: var(--text-muted); font-size: var(--fs-sm); }
  .recovery-card { display: flex; flex-direction: column; align-items: center; margin: var(--sp-5) auto 0; padding: var(--sp-6); border: 1px solid var(--border-strong); border-radius: var(--r-lg); background: var(--surface-raised); box-shadow: none; text-align: center; }
  .warning-icon { width: 44px; height: 44px; color: var(--status-warning); }
  .warning-icon svg { width: 100%; height: 100%; }
  h2 { margin: var(--sp-3) 0 0; font-size: var(--fs-xl); font-weight: 650; letter-spacing: -0.02em; }
  .intro { max-width: 62ch; margin: var(--sp-2) 0 0; color: var(--text-secondary); font-size: var(--fs-sm); line-height: 1.5; }
  .status-list { width: min(100%, 520px); margin: var(--sp-5) 0 0; border: 1px solid var(--border-default); border-radius: var(--r-md); background: var(--surface-sunken); text-align: left; }
  .status-list > div { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-4); padding: var(--sp-2) var(--sp-3); }
  .status-list > div + div { border-top: 1px solid var(--border-default); }
  dt { color: var(--text-primary); font-size: var(--fs-sm); }
  dd { margin: 0; color: var(--text-muted); font-size: var(--fs-xs); text-align: right; }
  .actions { display: flex; gap: var(--sp-2); width: min(100%, 520px); margin-top: var(--sp-4); }
  .actions .app-btn { flex: 1; }
  .footer-message { display: flex; align-items: center; gap: var(--sp-2); margin: var(--sp-4) 0 0; color: var(--text-secondary); font-size: var(--fs-xs); }
  .small-warning { display: inline-grid; width: 18px; height: 18px; place-items: center; border: 1px solid var(--status-warning); border-radius: var(--r-md); color: var(--status-warning); font-size: var(--fs-xs); font-weight: 700; }
  @media (max-width: 700px) { .recovery-shell { padding: var(--sp-5) var(--sp-4); } .recovery-card { padding: var(--sp-5) var(--sp-4); } .actions { flex-direction: column; width: 100%; } .actions .app-btn, .status-list { width: 100%; } .status-list > div { align-items: flex-start; flex-direction: column; gap: var(--sp-1); } dd { text-align: left; } }
</style>
