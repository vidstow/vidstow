import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const read = (path) => readFile(new URL(path, import.meta.url), 'utf8');

test('design tokens match the zinc shell contract', async () => {
  const css = await read('../src/styles/global.css');
  assert.match(css, /--surface-bg:\s+#121212/);
  assert.match(css, /--surface-base:\s+#111113/);
  assert.match(css, /--surface-sidebar:\s+#18181B/);
  assert.match(css, /--text-secondary:\s+#A1A1AA/);
  assert.match(css, /--text-muted:\s+#8A8A93/);
  assert.match(css, /--accent-500:\s+#3B82F6/);
  assert.match(css, /--text-primary:\s+#FAFAFA/);
  assert.match(css, /--sidebar-w:\s+160px/);
  assert.match(css, /--statusbar-h:\s+22px/);
  assert.match(css, /--page-pad-x:\s+24px/);
  assert.match(css, /--rail-pad-top:\s+38px/);
  assert.match(css, /--page-pad-y:\s+calc\(var\(--rail-pad-top\) \+ var\(--nav-first-gap\)\)/);
  assert.match(css, /\.page \{/);
  assert.match(css, /\.page-header h1 \{/);
  assert.match(css, /--r-sm:\s+3px/);
  assert.match(css, /--r-md:\s+4px/);
  assert.match(css, /--font-sans: 'Geist Variable'/);
  assert.match(css, /--font-mono: 'Geist Mono Variable'/);
  assert.match(css, /color-scheme:\s*dark/);
  assert.match(css, /--shadow-card: 0 1px 2px/);
  assert.match(css, /prefers-reduced-motion: reduce/);
  assert.match(css, /:focus-visible/);
  assert.match(css, /\.fieldwrap textarea:focus-visible \{ outline: none/);
  assert.match(css, /\.fieldwrap textarea,\s*\.fieldwrap textarea:focus/);
  assert.match(css, /flex: 1;\s*width: auto;/);
  assert.doesNotMatch(css, /\.fieldwrap \.kbd/);
});

test('Analyze is the light query chip; Download stays the compact blue commit', async () => {
  const home = await read('../src/pages/Home.svelte');
  assert.match(home, /class="dbtn query"/);
  assert.doesNotMatch(home, /class:dim=\{analyzeRecedes\}/);
  assert.match(home, /\.dbtn \{[\s\S]*?height: 24px;/);
  assert.match(home, /\.dbtn \{[\s\S]*?padding: 0 9px;/);
  assert.match(home, /\.dfoot \{[\s\S]*?padding: 8px 10px 8px 16px;/);
  assert.match(home, /\.dbtn\.query \{/);
  assert.match(home, /background: #FAFAFA/);
  assert.match(home, /\.dbtn\.pri \{ background: var\(--accent-600\)/);
  assert.match(home, /\.fieldwrap \.dbtn \{ flex-shrink: 0; align-self: center; \}/);
  assert.match(home, /class="query-spin"/);
  assert.doesNotMatch(home, /class="query-sizer"/);
  assert.match(home, /\.fieldwrap \.dbtn\.query \{ gap: 5px; \}/);
  assert.match(home, /circle cx="11" cy="11" r="7"/);
  assert.match(home, /M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4/);
  assert.doesNotMatch(home, /M12 4v11m0 0-4-4m4 4 4-4M5 19h14/);
});

test('zinc sidebar starts at Home, Settings only, and no FFmpeg footer', async () => {
  const sidebar = await read('../src/lib/components/Sidebar.svelte');
  assert.match(sidebar, /padding: var\(--rail-pad-top\) 6px 12px/);
  assert.doesNotMatch(sidebar, /class="brand"/);
  assert.doesNotMatch(sidebar, /class="brand-mark"/);
  assert.doesNotMatch(sidebar, />VidStow</);
  assert.doesNotMatch(sidebar, /brand-mark\.(?:svg|png)|<img/);
  for (const label of ['Home', 'Queue', 'Downloads', 'Settings']) {
    assert.match(sidebar, new RegExp(`label: '${label}'`));
  }
  assert.doesNotMatch(sidebar, /label: 'About'/);
  assert.doesNotMatch(sidebar, /class="ffmpeg"/);
  assert.doesNotMatch(sidebar, /\$ffmpeg/);
  assert.doesNotMatch(sidebar, /ffmpeg\.org/);
  assert.match(sidebar, /class="nav-badge"/);
  assert.doesNotMatch(sidebar, /data-tip/);
  assert.doesNotMatch(sidebar, /aria-keyshortcuts/);
  assert.doesNotMatch(sidebar, /shortcut/);
});

test('app shell mounts the workstation, status bar, and keeps the About page', async () => {
  const [app, status, stores, main] = await Promise.all([
    read('../src/App.svelte'),
    read('../src/lib/components/StatusBar.svelte'),
    read('../src/lib/stores.ts'),
    read('../../main.go'),
  ]);
  assert.match(app, /class="workstation"/);
  assert.match(app, /<StatusBar \/>/);
  assert.match(app, /import About from '\.\/pages\/About\.svelte'/);
  assert.match(app, /class="home-host"/);
  assert.match(app, /hidden=\{\$route !== 'home'\}/);
  assert.doesNotMatch(app, /\{#if \$route === 'home'\}/);
  assert.match(app, /\$route === 'about'/);
  assert.match(app, /navigate\(target: 'home' \| 'queue' \| 'downloads' \| 'settings' \| 'about'\)/);
  assert.doesNotMatch(app, /pendingHomeFocus/);
  assert.doesNotMatch(app, /key === 'l'/);
  assert.doesNotMatch(app, /key === '2'/);
  assert.doesNotMatch(app, /event\.key === ','/);
  assert.match(stores, /writable<'home' \| 'queue' \| 'downloads' \| 'settings' \| 'about'>/);
  assert.match(status, /'Not set'/);
  assert.match(status, /Idle/);
  assert.match(status, /\$\{active\}\/\$\{limit\} slots/);
  assert.doesNotMatch(status, />Path</);
  assert.doesNotMatch(status, /FFmpeg/);
  assert.doesNotMatch(status, /ytdlp/);
  assert.match(main, /NSAppearanceNameDarkAqua/);
  assert.match(main, /Width:\s+1180/);
  assert.match(main, /Height:\s+760/);
  assert.match(main, /MinWidth:\s+960/);
  assert.match(main, /MinHeight:\s+600/);
});

test('About route renders the Settings page with the About colophon', async () => {
  const [about, settings] = await Promise.all([
    read('../src/pages/About.svelte'),
    read('../src/pages/Settings.svelte'),
  ]);
  assert.match(about, /import Settings from '\.\/Settings\.svelte'/);
  assert.match(about, /<Settings \/>/);
  assert.match(settings, /api\.app\.buildInfo\(\)/);
  assert.match(settings, /engineVersion/);
  assert.match(settings, /license: 'Apache-2\.0'/);
  assert.match(settings, /github\.com\/vidstow\/vidstow/);
  assert.match(settings, /class="colophon"/);
  assert.match(settings, />View source</);
  assert.match(settings, />Read the docs</);
  assert.match(settings, /Built with Go · Wails · Svelte · ytdlp-go · FFmpeg/);
});

test('ui primitives exist and expose the expected API surface', async () => {
  const [button, card, tabs, indicator, progress, dialog, empty, index] = await Promise.all([
    read('../src/lib/components/ui/Button.svelte'),
    read('../src/lib/components/ui/Card.svelte'),
    read('../src/lib/components/ui/Tabs.svelte'),
    read('../src/lib/components/ui/StatusIndicator.svelte'),
    read('../src/lib/components/ui/Progress.svelte'),
    read('../src/lib/components/ui/DialogSurface.svelte'),
    read('../src/lib/components/ui/EmptyState.svelte'),
    read('../src/lib/components/ui/index.ts'),
  ]);
  assert.match(button, /type Variant = 'primary' \| 'secondary' \| 'ghost' \| 'danger'/);
  assert.match(button, /\['btn', `variant-\$\{variant\}`, `size-\$\{size\}`\]/);
  assert.match(button, /aria-busy/);
  assert.match(card, /<section class="card"/);
  assert.match(card, /{@render children\(\)}/);
  assert.match(tabs, /role="tablist"/);
  assert.match(tabs, /aria-selected/);
  assert.match(indicator, /data-tone={tone}/);
  assert.match(progress, /role="progressbar"/);
  assert.match(dialog, /role="dialog"/);
  assert.match(dialog, /aria-modal="true"/);
  assert.match(empty, /class="empty"/);
  for (const name of ['Button', 'Card', 'Tabs', 'StatusIndicator', 'Progress', 'DialogSurface', 'EmptyState']) {
    assert.match(index, new RegExp(`export \\{ default as ${name} \\}`));
  }
});
