import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const read = (path) => readFile(new URL(path, import.meta.url), 'utf8');

test('approved navigation and window branding are used', async () => {
  const [sidebar, main] = await Promise.all([
    read('../src/lib/components/Sidebar.svelte'),
    read('../../main.go'),
  ]);
  for (const label of ['Home', 'Queue', 'Downloads', 'Settings']) assert.match(sidebar, new RegExp(`label: '${label}'`));
  assert.match(sidebar, /class="brand-mark"/);
  assert.doesNotMatch(sidebar, /brand-mark\.svg/);
  assert.doesNotMatch(sidebar, /label: 'About'/);
  for (const rejected of ['v0 · single video', 'Single public YouTube videos only', 'brand-name', 'class="logo"', 'logo-universal']) assert.doesNotMatch(sidebar, new RegExp(rejected));
  assert.match(main, /Title:\s+"VidStow"/);
  assert.match(main, /NSAppearanceNameDarkAqua/);
  assert.match(await read('../index.html'), /<title>VidStow<\/title>/);
});

test('page titles and controls match the approved redesign', async () => {
  const [home, queue, downloads, settings] = await Promise.all([
    read('../src/pages/Home.svelte'), read('../src/pages/Queue.svelte'),
    read('../src/pages/Downloads.svelte'), read('../src/pages/Settings.svelte'),
  ]);
  assert.doesNotMatch(home, />Single URL<\/button>/);
  assert.doesNotMatch(home, />Batch URLs<\/button>/);
  assert.doesNotMatch(home, /Download from YouTube/);
  assert.doesNotMatch(home, /Add a YouTube link/);
  assert.doesNotMatch(home, /Start \$\{batchReadyCount\}/);
  assert.doesNotMatch(home, /Starting…/);
  assert.match(home, /downloadVideosLabel\(selectedItems\.size\)/);
  assert.match(home, /downloadVideosLabel\(batchReadyCount\)/);
  assert.match(home, /<small>\{item\.message\}<\/small>/);
  assert.match(home, /the link decides\.<br>/);
  assert.match(home, /class="sep"/);
  assert.match(home, /word-break: break-all/);
  assert.match(home, /class="kbd"/);
  assert.match(home, /class="cmdkey"/);
  assert.match(home, /textarea:focus-visible/);
  assert.doesNotMatch(home, /a private link/);
  assert.doesNotMatch(home, /a video in a playlist/);
  assert.match(home, /-webkit-text-decoration: underline dotted/);
  assert.match(home, /This link includes a playlist/);
  assert.match(home, /Choose what to download/);
  assert.match(home, /class="sdialog"/);
  assert.match(home, /class="swapline"/);
  assert.match(home, /playlist\.duration/);
  assert.match(home, /playlist\.channel/);
  assert.doesNotMatch(home, /reach 1080p/);
  assert.doesNotMatch(home, /Review the playlist instead/);
  assert.match(home, /PLAYLIST_ADMIT_CAP = 500/);
  assert.match(home, /VidStow can review up to \{PLAYLIST_ADMIT_CAP\} videos from a playlist\./);
  assert.doesNotMatch(home, /Admit another batch/);
  assert.doesNotMatch(home, />Admit /);
  assert.match(home, />All</);
  assert.doesNotMatch(home, />Select all</);
  assert.doesNotMatch(home, /aria-label="Search playlist"/);
  assert.match(home, /aria-label="Range start"/);
  assert.match(home, /class="eprow"/);
  assert.match(home, />Change</);
  assert.doesNotMatch(home, /Save to /);
  assert.match(home, /Paste another link/);
  assert.match(home, /<footer class="dfoot">[\s\S]*?>Download</);
  assert.match(home, />Download<\/button>/);
  assert.doesNotMatch(home, />Add to Queue</);
  assert.match(home, /class="drow drow2 v2"/);
  assert.match(home, /\.drow2 \.thumb \{[^}]*max-height: 90px/);
  assert.match(home, /\.drow2\.v2 \.thumb \{[^}]*max-height: 90px/);
  assert.doesNotMatch(home, /\.drow2\.v2 \.thumb \{[^}]*height: auto/);
  assert.match(home, /grid-template-columns: 160px minmax\(0, 1fr\) minmax\(168px, 220px\)/);
  assert.match(home, /-webkit-line-clamp: 2/);
  assert.match(home, /\.modeseg button \{[\s\S]*?font-size: 11px;/);
  assert.match(home, /\.seg \{[\s\S]*?font-size: 12px;/);
  assert.match(home, /\.modeseg button\.on \{[\s\S]*?background: var\(--surface-raised\)/);
  assert.match(home, /class="dchoice stack"/);
  assert.match(home, /class="dcost"/);
  assert.doesNotMatch(home, /Best available up to/);
  assert.doesNotMatch(home, /doutcome"><b>/);
  assert.match(home, /'Analyze'/);
  assert.match(home, /class="dbtn query"/);
  assert.doesNotMatch(home, /class="dbtn pri" type="submit"/);
  assert.doesNotMatch(home, /12vh/);
  assert.doesNotMatch(home, /class:empty=/);
  assert.match(home, /\.composer \{[\s\S]*?width: 100%;/);
  assert.match(home, /\.fieldwrap textarea \{[\s\S]*?font-family: var\(--font-mono\)/);
  assert.match(home, /\.fieldwrap textarea \{[\s\S]*?font-size: 13px/);
  assert.match(queue, /<QueueOverview/);
  assert.match(queue, /api\.queue\.pauseAll/);
  assert.match(queue, /onResumeAll=\{resumeAll\}/);
  assert.match(await read('../src/lib/lifecycle-ui/QueueOverview.svelte'), /Go to Home/);
  assert.match(await read('../src/lib/lifecycle-ui/QueueOverview.svelte'), /Nothing in the queue/);
  assert.match(await read('../src/lib/lifecycle-ui/QueueOverview.svelte'), /grid-template-columns: minmax\(0, 1fr\) 300px/);
  assert.match(await read('../src/lib/lifecycle-ui/QueueOverview.svelte'), /height: 36px/);
  assert.match(queue, /api\.queue\.clearCompleted/);
  assert.match(queue, /collectionLabel = collection\?\.kind === 'batch' \? 'batch' : 'playlist'/);
  assert.match(queue, /Completed files remain on disk\./);
  assert.match(queue, /Jobs are saved automatically\./);
  assert.match(downloads, /View your recently downloaded items\./);
  assert.match(downloads, /placeholder="Search downloads…"/);
  assert.match(settings, /Configure downloads, queue behavior, and external tools\./);
  assert.match(settings, />Default download folder</);
  assert.match(settings, />FFmpeg path</);
  assert.match(settings, />Diagnostics</);
  assert.match(settings, />Copy Diagnostics</);
  assert.match(settings, />Clear history</);
  assert.match(settings, /> Send diagnostics<\/label>/);
  assert.match(settings, /> Don’t send<\/label>/);
  assert.match(settings, /automaticDiagnostics === 'enabled'/);
  assert.match(settings, /automaticDiagnostics === 'disabled'/);
  assert.match(settings, /Disabling this immediately deletes anything waiting to be sent/);
  assert.match(settings, /api\.settings\.setAutomaticDiagnostics\(value\)/);
  assert.match(settings, /await api\.diagnostics\.copy\(\)/);
  assert.match(settings, /await api\.diagnostics\.clear\(\)/);
});

test('first launch asks for explicit diagnostic consent without a default', async () => {
  const [app, dialog] = await Promise.all([
    read('../src/App.svelte'),
    read('../src/lib/lifecycle-ui/DiagnosticConsentDialog.svelte'),
  ]);
  assert.match(app, /if \(!savedSettings\.automaticDiagnostics\)/);
  assert.match(app, /chooseAutomaticDiagnostics\('enabled'\)/);
  assert.match(app, /chooseAutomaticDiagnostics\('disabled'\)/);
  assert.match(app, /api\.settings\.setAutomaticDiagnostics\(value\)/);
  assert.match(app, /https:\/\/diagnostics\.vidstow\.workers\.dev\/privacy/);
  assert.match(dialog, />Send diagnostics<\/button>/);
  assert.match(dialog, />Don’t send<\/button>/);
  assert.match(dialog, /cannot complete a requested download or encounters an app failure/);
  assert.match(dialog, /never include video IDs, links, paths, filenames, cookies, tokens, or error text/);
  assert.doesNotMatch(dialog, /checked|selected/);
});

test('analysis failures stay on Home with a paste-another-link recovery', async () => {
  const home = await read('../src/pages/Home.svelte');
  assert.match(home, /title: 'Unsupported URL'/);
  assert.match(home, /analyzeError = \{/);
  assert.match(home, /Paste another link/);
  assert.match(home, /urlField\?\.select\(\)/);
  assert.doesNotMatch(home, /catch \(err\) \{\s*unsupported = \{\s*url: result\.url/);
});

test('unsupported and FFmpeg-required states retain accurate product copy', async () => {
  const [home, modal] = await Promise.all([
    read('../src/pages/Home.svelte'), read('../src/lib/components/Modal.svelte'),
  ]);
  assert.match(home, /valid, publicly accessible YouTube video, Short, or playlist/);
  assert.doesNotMatch(home, /supports YouTube videos and playlists/);
  assert.match(home, /title: 'FFmpeg Required'/);
  assert.match(home, /choose an original audio option/);
  assert.match(modal, /VidStow will continue to offer outputs that do not need FFmpeg\./);
  assert.match(modal, /class:primary=\{action\.primary\}/);
});

test('startup performs one FFmpeg fetch and normalizes binding errors', async () => {
  const [app, stores] = await Promise.all([
    read('../src/App.svelte'),
    read('../src/lib/stores.ts'),
  ]);
  assert.equal((app.match(/api\.ffmpeg\.status\(\)/g) || []).length, 1);
  assert.match(app, /title: 'The app could not finish starting'/);
  assert.match(app, /Your downloads are safe on disk\. VidStow could not read its saved queue, so the Queue was reset!/);
  assert.match(app, /class="btn sm ghost" onclick=\{copyRecoveryDiagnostics\}/);
  assert.match(app, /class="btn sm" onclick=\{openRecoveryDataFolder\}/);
  assert.match(app, /class="btn sm ghost quiet"/);
  assert.doesNotMatch(app, /class="notice-btn"/);
  assert.doesNotMatch(app, /so the Queue was reset\.</);
  assert.match(app, /api\.events\.onJobUpdate\(updateJobInList\)/);
  assert.match(app, /if \(idx === -1\) return \[updated, \.\.\.list\]/);
  assert.match(stores, /'message' in err/);
  assert.match(stores, /return fallback/);
});

test('terminal queue rows expose recovery and removal actions', async () => {
  const row = await read('../src/lib/components/ProgressRow.svelte');
  assert.match(row, /<article class="job-card"/);
  assert.doesNotMatch(row, /role="cell"/);
  assert.doesNotMatch(row, /<t[rd][^>]*>/);
  for (const action of ['Cancel download', 'Open downloaded file', 'Retry download', 'Remove download']) {
    assert.match(row, new RegExp(`<button[^>]+type="button"[^>]+aria-label="${action}"`));
  }
  assert.match(row, /aria-label="Pause download"/);
  assert.match(row, /aria-label="Resume download"/);
  for (const method of ['cancel', 'retry', 'remove', 'pause', 'resume']) assert.match(row, new RegExp(`api\\.jobs\\.${method}\\(job\\.id\\)`));
  assert.match(row, />Retry</);
  assert.match(row, /job\.status === 'failed'/);
});

test('download history actions remain native accessible buttons', async () => {
  const downloads = await read('../src/pages/Downloads.svelte');
  assert.doesNotMatch(downloads, /role="(?:table|row|cell)"/);
  assert.match(downloads, /<button[^>]+aria-label="Open downloaded file"/);
  assert.match(downloads, /<button[^>]+aria-label="Show in Finder"/);
  assert.match(downloads, /aria-label="Remove from history"/);
  assert.match(downloads, /aria-label="Delete downloaded file"/);
  assert.match(downloads, /await api\.fs\.open\(entry\.absolutePath\)/);
  assert.match(downloads, /await api\.fs\.reveal\(entry\.absolutePath\)/);
  assert.match(downloads, /api\.downloads\.remove\(entry\.id\)/);
  assert.match(downloads, /api\.downloads\.deleteFile\(entry\.id\)/);
  assert.match(downloads, /That file is no longer at this path/);
  assert.doesNotMatch(downloads, /fileMissing/);
  assert.doesNotMatch(downloads, /File missing/);
  assert.match(downloads, /formatLabel\(entry\)/);
  assert.match(downloads, /container/);
});

test('queue distinguishes temporary in-memory storage from durable automatic saving', async () => {
  const [queue, stores] = await Promise.all([
    read('../src/pages/Queue.svelte'),
    read('../src/lib/stores.ts'),
  ]);
  assert.match(queue, /view\?\.persistence/);
  assert.match(queue, /Queue actions are disabled/);
  assert.match(queue, /Jobs are saved automatically/);
  assert.match(stores, /persistence = writable<PersistenceStatus>/);
});
