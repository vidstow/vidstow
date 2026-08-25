import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const read = (path) => readFile(new URL(path, import.meta.url), 'utf8');
const forbiddenEngine = ['yt', 'dlp'].join('-');

test('approved navigation and window branding are used', async () => {
  const [sidebar, main] = await Promise.all([
    read('../src/lib/components/Sidebar.svelte'),
    read('../../main.go'),
  ]);
  for (const label of ['Home', 'Queue', 'Downloads', 'Settings']) {
    assert.match(sidebar, new RegExp(`label: '${label}'`));
  }
  assert.doesNotMatch(sidebar, /brand-mark\.(?:svg|png)/);
  assert.match(main, /Title:\s+"VidStow"/);
  assert.match(await read('../index.html'), /<title>VidStow<\/title>/);
});

test('Home uses one persistent URL strip while preserving batch and playlist review', async () => {
  const home = await read('../src/pages/Home.svelte');
  assert.match(home, /class="analyze-bar"/);
  assert.match(home, /Paste a YouTube URL to analyze/);
  assert.match(home, /api\.clipboard\.getText\(\)/);
  assert.match(home, /inputMode = 'single';/);
  assert.doesNotMatch(home, />Single URL<\/button>/);
  assert.doesNotMatch(home, />Batch URLs<\/button>/);
  assert.match(home, /aria-label=\{inputMode === 'batch' \? 'Close batch URL composer' : 'Batch URLs'\}/);
  assert.match(home, /Start \$\{batchReadyCount\} downloads/);
  assert.match(home, /<small>\{item\.message\}<\/small>/);
  assert.match(home, /This link includes a playlist/);
  assert.match(home, /Review the playlist instead/);
  assert.match(home, /Every selected video uses this format/);
  assert.match(home, /PLAYLIST_ADMIT_CAP = 500/);
  assert.match(home, /All available/);
  assert.match(home, /Search playlist…/);
  assert.match(home, />Choose Download</);
  assert.match(home, /playlistQueueBusy/);
  assert.match(home, /videoQueueBusy/);
});

test('Queue is a locked list and inspector with only approved ordinary actions', async () => {
  const [queue, overview, row, collection] = await Promise.all([
    read('../src/pages/Queue.svelte'),
    read('../src/lib/lifecycle-ui/QueueOverview.svelte'),
    read('../src/lib/lifecycle-ui/LifecycleJobRow.svelte'),
    read('../src/lib/lifecycle-ui/CollectionRow.svelte'),
  ]);
  assert.match(queue, /<QueueOverview/);
  assert.match(queue, /api\.queue\.pauseAll/);
  assert.match(queue, /api\.queue\.clearCompleted/);
  assert.match(queue, /api\.queue\.pause/);
  assert.match(queue, /api\.queue\.cancel/);
  assert.match(queue, /Completed files remain on disk\./);
  assert.match(overview, /class="queue-master"/);
  assert.match(overview, /class="inspector"/);
  assert.match(overview, />Pause all<\/span>/);
  assert.match(overview, />Clear completed<\/span>/);
  assert.match(overview, /white-space:\s*nowrap/);
  assert.doesNotMatch(overview, /repeat\(2,\s*110px\)/);
  assert.match(overview, /Completed · \{completedJobs\.length\}/);
  assert.match(row, /aria-label="Pause download"/);
  assert.match(row, /aria-label="Cancel download"/);
  assert.match(collection, />Pause<\/button>/);
  assert.match(collection, />Cancel<\/button>/);
  for (const rejected of ['Resume', 'Retry failed', 'Download again', 'Start again', 'Open source', 'Copy link', 'Remove']) {
    assert.doesNotMatch(overview + row + collection, new RegExp(`>${rejected}(?:<|\\s)`));
  }
});

test('Downloads, Settings, and About retain their existing capabilities', async () => {
  const [downloads, settings, about] = await Promise.all([
    read('../src/pages/Downloads.svelte'),
    read('../src/pages/Settings.svelte'),
    read('../src/pages/About.svelte'),
  ]);
  assert.match(downloads, /placeholder="Search downloads…"/);
  assert.match(downloads, /aria-label="Open downloaded file"/);
  assert.match(downloads, /aria-label="Show in Finder"/);
  assert.match(downloads, /aria-label="Remove from history"/);
  assert.match(downloads, /aria-label="Delete downloaded file"/);
  assert.match(downloads, /api\.downloads\.remove\(entry\.id\)/);
  assert.match(downloads, /api\.downloads\.deleteFile\(entry\.id\)/);
  assert.match(settings, />Default download folder</);
  assert.match(settings, />FFmpeg path</);
  assert.match(settings, />Diagnostics</);
  assert.match(settings, />Copy Diagnostics</);
  assert.match(settings, />Clear history</);
  assert.match(settings, /api\.settings\.setAutomaticDiagnostics\(value\)/);
  assert.match(settings, /await api\.diagnostics\.copy\(\)/);
  assert.match(settings, /await api\.diagnostics\.clear\(\)/);
  assert.match(about, /ytdlp-go/);
  assert.doesNotMatch(about, new RegExp(forbiddenEngine));
});

test('complete video policy is fixed, truthful, and shared by single and playlist admission', async () => {
  const [home, editor, settings, types, downloads, stores] = await Promise.all([
    read('../src/pages/Home.svelte'),
    read('../src/lib/components/OutputOptionsEditor.svelte'),
    read('../src/pages/Settings.svelte'),
    read('../src/lib/types.ts'),
    read('../src/pages/Downloads.svelte'),
    read('../src/lib/stores.ts'),
  ]);
  assert.match(types, /subtitleSidecar\?: boolean/);
  assert.match(types, /language: string/);
  assert.match(types, /chapterCount: number/);
  assert.match(stores, /subtitleMode: ''/);
  assert.match(home, /subtitleMode: subtitlesOn \? 'embed' : ''/);
  assert.match(home, /subtitleLanguages: subtitlesOn \? options\.subtitleLanguages\?\.slice\(0, 1\) : undefined/);
  assert.match(home, /subtitleAutoCaptions: subtitlesOn/);
  assert.match(home, /embedThumbnail: true/);
  assert.match(home, /embedChapters: true/);
  assert.match(home, /seedOutputOptions\(summary\.subtitles \?\? \[\], summary\.language\)/);
  assert.match(home, /api\.jobs\.start\(\{/);
  assert.match(home, /api\.jobs\.startPlaylist\(\{/);
  assert.match(home, /likely MP4, with MKV fallback/);
  assert.match(home, /FFmpeg is required to create this complete file/);
  assert.doesNotMatch(home, /\$settings\.confirmBeforeDownload|title: 'Add this download\?'|title: 'Add this playlist\?'/);
  assert.match(editor, /Subtitles \{subtitlesOn \? 'On' : 'Off'\}/);
  assert.match(editor, /None available/);
  assert.match(editor, /type="radio" name="subtitle-language"/);
  assert.match(editor, /\(auto\/transcribed\)/);
  assert.doesNotMatch(editor, /class="disclosure"|type="checkbox" checked=\{selectedLanguages/);
  assert.match(settings, /Include subtitles on new adds/);
  assert.match(settings, /Each video’s own language/);
  assert.match(settings, /Also save an \.srt file/);
  assert.doesNotMatch(settings, /Confirm before starting downloads|Include auto-generated captions|Embed thumbnail artwork|Embed chapter markers/);
  assert.match(downloads, /actualContainer\(entry\)/);
  assert.match(downloads, /filename\.match\(\/\\\.\(\[a-z0-9\]\+\)\$\/i\)/);
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
  assert.match(dialog, />Send diagnostics<\/button>/);
  assert.match(dialog, />Don’t send<\/button>/);
  assert.match(dialog, /cannot complete a requested download or encounters an app failure/);
  assert.match(dialog, /never include video IDs, links, paths, filenames, cookies, tokens, or error text/);
  assert.doesNotMatch(dialog, /checked|selected/);
});

test('analysis and FFmpeg failures retain accurate product copy', async () => {
  const [home, modal] = await Promise.all([
    read('../src/pages/Home.svelte'),
    read('../src/lib/components/Modal.svelte'),
  ]);
  assert.match(home, /title: 'Unsupported URL'/);
  assert.match(home, /valid, publicly accessible YouTube video, Short, or playlist/);
  assert.match(home, /title: 'FFmpeg Required'/);
  assert.match(home, /choose an original audio option/);
  assert.match(modal, /VidStow will continue to offer outputs that do not need FFmpeg\./);
  assert.match(modal, /use:trapModalFocus/);
});

test('startup performs one FFmpeg fetch and normalizes binding errors', async () => {
  const [app, stores] = await Promise.all([
    read('../src/App.svelte'),
    read('../src/lib/stores.ts'),
  ]);
  assert.equal((app.match(/api\.ffmpeg\.status\(\)/g) || []).length, 1);
  assert.match(app, /title: 'The app could not finish starting'/);
  assert.match(app, /api\.events\.onJobUpdate\(updateJobInList\)/);
  assert.match(app, /if \(idx === -1\) return \[updated, \.\.\.list\]/);
  assert.match(stores, /'message' in err/);
  assert.match(stores, /return fallback/);
});

test('queue persistence remains fail-closed without altering QueueView capabilities', async () => {
  const [queue, stores, types] = await Promise.all([
    read('../src/pages/Queue.svelte'),
    read('../src/lib/stores.ts'),
    read('../src/lib/lifecycle-ui/types.ts'),
  ]);
  assert.match(queue, /view\?\.persistence/);
  assert.match(queue, /Queue actions are disabled/);
  assert.match(queue, /capabilities: \{\}, commandToken: undefined/);
  assert.match(stores, /persistence = writable<PersistenceStatus>/);
  assert.match(types, /resume\?: boolean/);
  assert.match(types, /retry\?: boolean/);
});

test('Queue and Downloads page titles share the workstation --fs-lg token', async () => {
  const [overview, downloads, styles] = await Promise.all([
    read('../src/lib/lifecycle-ui/QueueOverview.svelte'),
    read('../src/pages/Downloads.svelte'),
    read('../src/styles/global.css'),
  ]);
  assert.match(overview, /h1 \{[^}]*font-size: var\(--fs-lg\)/);
  assert.match(downloads, /\.page-header h1 \{[^}]*font-size: var\(--fs-lg\)/);
  assert.match(styles, /\.page-header h1 \{[^}]*font-size: var\(--fs-lg\)/);
});
