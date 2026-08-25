<script lang="ts">
  import { counts, ffmpeg, history, route } from '../stores.js';

  type Route = 'home' | 'queue' | 'downloads' | 'settings' | 'about';
  type Item = { key: Route; label: string; icon: 'home' | 'queue' | 'downloads' | 'settings' | 'about' };

  $: c = $counts;
  $: saved = $history.length;

  const items: Item[] = [
    { key: 'home', label: 'Home', icon: 'home' },
    { key: 'queue', label: 'Queue', icon: 'queue' },
    { key: 'downloads', label: 'Downloads', icon: 'downloads' },
  ];
  const utility: Item[] = [
    { key: 'settings', label: 'Settings', icon: 'settings' },
    { key: 'about', label: 'About', icon: 'about' },
  ];

  function go(target: Route) {
    route.set(target);
  }

  function openFFmpeg() {
    window.runtime?.BrowserOpenURL?.('https://ffmpeg.org/download.html');
  }
</script>

<aside class="sidebar" aria-label="Primary">
  <button type="button" class="brand" onclick={() => go('home')} aria-label="VidStow home">
    VidStow
  </button>

  <nav class="primary" aria-label="Primary navigation">
    <ul>
      {#each items as item}
        <li>
          <button
            type="button"
            class="nav-item"
            class:active={$route === item.key}
            aria-current={$route === item.key ? 'page' : undefined}
            onclick={() => go(item.key)}
          >
            <span class="nav-icon" aria-hidden="true">
              {#if item.icon === 'home'}
                <svg viewBox="0 0 24 24"><path d="M4 10.5 12 4l8 6.5V20h-5v-6H9v6H4Z" /></svg>
              {:else if item.icon === 'queue'}
                <svg viewBox="0 0 24 24"><path d="M5 6h14M5 12h14M5 18h14" /></svg>
              {:else}
                <svg viewBox="0 0 24 24"><path d="M12 4v11m0 0-4-4m4 4 4-4M5 19h14" /></svg>
              {/if}
            </span>
            <span class="nav-label">{item.label}</span>
            {#if item.key === 'queue' && (c.active + c.pending) > 0}
              <span class="nav-badge" aria-label={`${c.active + c.pending} jobs`}>{c.active + c.pending}</span>
            {/if}
            {#if item.key === 'downloads' && saved > 0}
              <span class="nav-badge" aria-label={`${saved} downloads`}>{saved}</span>
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  </nav>

  <nav class="utility" aria-label="App">
    <ul>
      {#each utility as item}
        <li>
          <button
            type="button"
            class="nav-item"
            class:active={$route === item.key}
            aria-current={$route === item.key ? 'page' : undefined}
            onclick={() => go(item.key)}
          >
            <span class="nav-icon" aria-hidden="true">
              {#if item.icon === 'about'}
                <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8.5" /><path d="M12 11v5" /><path d="M12 8h.01" /></svg>
              {:else}
                <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3" /><path d="M19 13.5v-3l-2-.7a7 7 0 0 0-.8-1.8l.9-1.9L15 4l-1.9.9a7 7 0 0 0-2.1 0L9 4 6.9 6.1 7.8 8a7 7 0 0 0-.8 1.8l-2 .7v3l2 .7a7 7 0 0 0 .8 1.8l-.9 1.9L9 20l2-.9a7 7 0 0 0 2.1 0l1.9.9 2.1-2.1-.9-1.9a7 7 0 0 0 .8-1.8Z" /></svg>
              {/if}
            </span>
            <span class="nav-label">{item.label}</span>
          </button>
        </li>
      {/each}
    </ul>
  </nav>

  <div class="ffmpeg" class:ready={$ffmpeg.available} aria-label={$ffmpeg.available ? 'FFmpeg ready' : 'FFmpeg required'}>
    <span class="dot" aria-hidden="true"></span>
    <span>{$ffmpeg.available ? 'FFmpeg ready' : 'FFmpeg required'}</span>
    {#if !$ffmpeg.available}
      <button type="button" onclick={openFFmpeg} aria-label="Open FFmpeg download page">Get</button>
    {/if}
  </div>
</aside>

<style>
  .sidebar {
    width: var(--sidebar-w);
    min-width: var(--sidebar-w);
    height: 100%;
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    padding: 0 7px 9px;
    border-right: 1px solid var(--border-default);
    background: var(--surface-sidebar);
  }

  .brand {
    height: var(--topbar-h);
    min-height: var(--topbar-h);
    padding: 0 28px;
    color: var(--text-primary);
    font-size: 11px;
    font-weight: 650;
    letter-spacing: -0.01em;
    text-align: left;
  }

  nav.primary {
    padding-top: 16px;
    flex: 1;
  }
  nav.utility { padding-top: 8px; }
  nav ul {
    display: flex;
    margin: 0;
    padding: 0;
    flex-direction: column;
    gap: 4px;
    list-style: none;
  }

  .nav-item {
    width: 100%;
    min-height: 32px;
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 0 8px;
    border-radius: var(--r-md);
    color: var(--text-secondary);
    font-size: 11px;
    font-weight: 450;
    transition: background 100ms ease, color 100ms ease;
  }
  .nav-item:hover {
    background: rgba(255, 255, 255, 0.045);
    color: var(--text-primary);
  }
  .nav-item.active {
    background: var(--surface-active);
    color: var(--text-primary);
    font-weight: 550;
  }

  .nav-icon {
    width: 14px;
    height: 14px;
    display: inline-flex;
    flex: 0 0 auto;
  }
  .nav-icon svg {
    width: 14px;
    height: 14px;
    fill: none;
    stroke: currentColor;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .nav-label { min-width: 0; flex: 1; text-align: left; }
  .nav-badge {
    min-width: 16px;
    height: 16px;
    display: inline-grid;
    padding: 0 4px;
    place-items: center;
    border-radius: var(--r-full);
    background: var(--border-strong);
    color: var(--text-primary);
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 600;
  }

  .ffmpeg {
    min-height: 26px;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 10px;
    color: var(--text-muted);
    font-size: 9px;
    white-space: nowrap;
  }
  .ffmpeg .dot {
    width: 5px;
    height: 5px;
    flex: 0 0 auto;
    border-radius: 50%;
    background: var(--status-danger);
  }
  .ffmpeg.ready .dot { background: var(--status-success); }
  .ffmpeg button {
    margin-left: auto;
    color: var(--accent-400);
    font-size: 9px;
  }
</style>
