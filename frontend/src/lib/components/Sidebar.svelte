<script lang="ts">
  import { route, counts, history } from '../stores.js';
  type Route = 'home' | 'queue' | 'downloads' | 'settings' | 'about';

  $: c = $counts;
  $: saved = $history.length;

  const items: Array<{ key: Route; label: string; icon: string }> = [
    { key: 'home',      label: 'Home',      icon: 'home' },
    { key: 'queue',     label: 'Queue',     icon: 'queue' },
    { key: 'downloads', label: 'Downloads', icon: 'downloads' },
  ];
  const utility: Array<{ key: Route; label: string; icon: string }> = [
    { key: 'settings', label: 'Settings', icon: 'settings' },
  ];

  function go(target: Route) {
    route.set(target);
  }
</script>

<aside class="sidebar" aria-label="Primary">
  <button type="button" class="brand" onclick={() => go('home')} aria-label="VidStow home">
    <span class="brand-mark" aria-hidden="true">
      <svg viewBox="0 0 24 24" width="12" height="12">
        <path fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" d="M12 4v11m0 0-4-4m4 4 4-4M5 19h14"/>
      </svg>
    </span>
    <span class="brand-title">VidStow</span>
  </button>

  <nav aria-label="Primary navigation">
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
                <svg viewBox="0 0 24 24" width="16" height="16"><path fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" d="M3 11l9-7 9 7v9a2 2 0 0 1-2 2h-4v-7H9v7H5a2 2 0 0 1-2-2z"/></svg>
              {:else if item.icon === 'queue'}
                <svg viewBox="0 0 24 24" width="16" height="16"><path fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h10"/></svg>
              {:else}
                <svg viewBox="0 0 24 24" width="16" height="16"><path fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" d="M12 4v12m0 0l-4-4m4 4l4-4M5 20h14"/></svg>
              {/if}
            </span>
            <span class="nav-label">{item.label}</span>
            {#if item.key === 'queue' && (c.active + c.pending) > 0}
              <span class="nav-badge" aria-label={`${c.active + c.pending} jobs`}>{c.active + c.pending}</span>
            {/if}
            {#if item.key === 'downloads' && saved > 0}
              <span class="nav-badge subtle" aria-label={`${saved} downloads`}>{saved}</span>
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
              <svg viewBox="0 0 24 24" width="16" height="16"><path fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" d="M12 9.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5zM19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 0 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 0 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 0 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 0 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></svg>
            </span>
            <span class="nav-label">{item.label}</span>
          </button>
        </li>
      {/each}
    </ul>
  </nav>
</aside>

<style>
  .sidebar {
    width: var(--sidebar-w);
    flex-shrink: 0;
    background: var(--surface-sidebar);
    border-right: 1px solid var(--border-default);
    display: flex;
    flex-direction: column;
    /* Keep navigation below the native macOS traffic-light/titlebar region. */
    padding: 38px 6px 12px;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 2px 6px 14px;
    color: var(--text-primary);
    text-decoration: none;
    border-radius: var(--r-sm);
  }
  .brand-mark {
    display: inline-grid;
    place-items: center;
    width: 20px;
    height: 20px;
    border-radius: 6px;
    background: linear-gradient(180deg, #3B82F6, #1D4ED8);
    color: #fff;
    flex-shrink: 0;
  }
  .brand-title {
    font-size: 13px;
    font-weight: 650;
    letter-spacing: -0.01em;
  }

  nav { flex: 1; padding-top: 2px; }
  nav.utility {
    flex: 0;
    padding-top: 8px;
    border-top: 1px solid var(--border-default);
  }
  nav.utility .nav-item {
    background: var(--surface-active);
    color: var(--text-primary);
  }
  nav.utility .nav-item:hover {
    background: #3F3F46;
  }
  nav ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .nav-item {
    position: relative;
    width: 100%;
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 30px;
    padding: 0 8px 0 10px;
    border-radius: 7px;
    color: var(--text-secondary);
    font-size: 13px;
    transition: background 120ms ease, color 120ms ease;
  }

  .nav-item:hover {
    background: var(--surface-raised);
    color: var(--text-primary);
  }

  .nav-item.active {
    background: var(--accent-soft);
    color: var(--text-primary);
  }
  nav.utility .nav-item.active {
    background: var(--accent-soft);
  }
  .nav-item.active::before {
    content: "";
    position: absolute;
    left: 0;
    top: 7px;
    bottom: 7px;
    width: 2px;
    border-radius: 99px;
    background: var(--accent-500);
  }

  .nav-icon { display: inline-flex; }

  .nav-label { flex: 1; text-align: left; }

  .nav-badge {
    background: var(--accent-600);
    color: var(--text-on-accent);
    font-size: 10px;
    font-weight: 600;
    padding: 0 5px;
    height: 17px;
    min-width: 17px;
    display: inline-grid;
    place-items: center;
    border-radius: var(--r-full);
    font-variant-numeric: tabular-nums;
  }
  .nav-badge.subtle {
    background: var(--border-default);
    color: var(--text-secondary);
  }
</style>
