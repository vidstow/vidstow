import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const read = (path) => readFile(new URL(path, import.meta.url), 'utf8');
const forbiddenEngine = ['yt', 'dlp'].join('-');

test('design tokens match the locked zinc workstation contract', async () => {
  const css = await read('../src/styles/global.css');
  assert.match(css, /--surface-bg:\s+#09090B/);
  assert.match(css, /--surface-sidebar:\s+#0C0C0E/);
  assert.match(css, /--accent-500:\s+#3B82F6/);
  assert.match(css, /--text-primary:\s+#FAFAFA/);
  assert.match(css, /--text-secondary:\s+#A1A1AA/);
  assert.match(css, /--sidebar-w:\s+140px/);
  assert.match(css, /--topbar-h:\s+32px/);
  assert.match(css, /--statusbar-h:\s+22px/);
  assert.match(css, /--font-sans: 'Geist Variable'/);
  assert.match(css, /--font-mono: 'Geist Mono Variable'/);
  assert.match(css, /color-scheme:\s*dark/);
  assert.match(css, /prefers-reduced-motion: reduce/);
  assert.match(css, /:focus-visible/);
});

test('compact text-brand sidebar keeps badges, app navigation, and FFmpeg status', async () => {
  const sidebar = await read('../src/lib/components/Sidebar.svelte');
  assert.match(sidebar, /class="brand"/);
  assert.match(sidebar, />\s*VidStow\s*<\/button>/);
  assert.doesNotMatch(sidebar, /brand-mark\.(?:svg|png)|<img/);
  for (const label of ['Home', 'Queue', 'Downloads', 'Settings', 'About']) {
    assert.match(sidebar, new RegExp(`label: '${label}'`));
  }
  assert.match(sidebar, /class="nav-badge"/);
  assert.match(sidebar, /class="ffmpeg"/);
  assert.match(sidebar, /'FFmpeg ready' : 'FFmpeg required'/);
  assert.match(sidebar, /BrowserOpenURL\?\.\('https:\/\/ffmpeg\.org\/download\.html'\)/);
});

test('app shell mounts the workstation and exact bottom status copy', async () => {
  const [app, status, stores] = await Promise.all([
    read('../src/App.svelte'),
    read('../src/lib/components/StatusBar.svelte'),
    read('../src/lib/stores.ts'),
  ]);
  assert.match(app, /class="workstation"/);
  assert.match(app, /<StatusBar \/>/);
  assert.match(app, /import About from '\.\/pages\/About\.svelte'/);
  assert.match(app, /\$route === 'about'/);
  assert.match(stores, /writable<'home' \| 'queue' \| 'downloads' \| 'settings' \| 'about'>/);
  assert.match(status, /aria-label="Engine ytdlp-go ready"/);
  assert.match(status, />ytdlp-go ready</);
  assert.match(status, /'FFmpeg ready' : 'FFmpeg required'/);
  assert.doesNotMatch(status, new RegExp(forbiddenEngine));
});

test('About uses backend build info and ytdlp-go naming', async () => {
  const about = await read('../src/pages/About.svelte');
  assert.match(about, /<h1 id="about-title">About<\/h1>/);
  assert.match(about, /name: 'VidStow'/);
  assert.match(about, /api\.app\.buildInfo\(\)/);
  assert.match(about, /ytdlp-go \{formatEngineVersion\(build\.engineVersion\)\}/);
  assert.match(about, /Copy Diagnostics/);
  assert.match(about, /license: 'Apache-2\.0'/);
  assert.match(about, /BrowserOpenURL\?\.\(url\)/);
  assert.match(about, /github\.com\/vidstow\/vidstow/);
  assert.doesNotMatch(about, new RegExp(forbiddenEngine));
  assert.match(about, /not affiliated with, or endorsed by,/);
});

test('ui primitives retain their typed and accessible API surface', async () => {
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
  assert.match(dialog, /use:trapModalFocus/);
  assert.match(empty, /class="empty"/);
  for (const name of ['Button', 'Card', 'Tabs', 'StatusIndicator', 'Progress', 'DialogSurface', 'EmptyState']) {
    assert.match(index, new RegExp(`export \\{ default as ${name} \\}`));
  }
});
