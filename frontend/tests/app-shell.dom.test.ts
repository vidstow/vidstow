import { render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import App from '../src/App.svelte';
import { ffmpeg, history, jobs, pendingUrl, persistence, queueView, route, settings } from '../src/lib/stores.js';

const firstURL = 'https://www.youtube.com/watch?v=fixture0001';

const emptyQueueView = {
  revision: 1,
  rows: [],
  collections: [],
  summary: {
    totalJobs: 0,
    runningJobs: 0,
    occupiedSlots: 0,
    slotLimit: 2,
    processingOccupied: 0,
    processingLimit: 3,
    waitingJobs: 0,
    pausedJobs: 0,
  },
  capabilities: { pauseAll: false, clearCompleted: false, commandToken: 'queue-token' },
  persistence: { available: true, healthy: true },
};

function installBindings() {
  const ValidateURL = vi.fn(async (raw: string) => ({
    kind: 'single_video', url: raw, videoUrl: raw, playlistUrl: '', videoId: 'fixture0001', playlistId: '',
  }));
  const AnalyzeURL = vi.fn(async (raw: string) => ({
    title: 'Fixture video', channel: 'Fixture channel', duration: '1:00', thumbnail: '', videoId: 'fixture0001', url: raw,
    durationSeconds: 60, viewCount: 1, uploadDate: '', description: '', access: { code: 'public', label: 'Public' },
    plans: [
      { id: 'video', kind: 'video', label: 'Video plan', container: 'mp4', available: true, recommended: true },
      { id: 'audio', kind: 'audio', label: 'Audio plan', container: 'm4a', available: true },
    ],
  }));
  (window as any).go = {
    main: {
      App: {
        GetStartupStatus: vi.fn(async () => ({ mode: 'healthy' })),
        GetSettings: vi.fn(async () => ({
          downloadFolder: '/tmp/downloads',
          ffmpegPath: '',
          windowWidth: 1180,
          windowHeight: 760,
          downloadConcurrency: 2,
          perVideoSubfolder: true,
          confirmBeforeDownload: false,
          automaticDiagnostics: 'disabled',
        })),
        ListJobs: vi.fn(async () => []),
        GetQueueView: vi.fn(async () => emptyQueueView),
        ListDownloads: vi.fn(async () => []),
        GetFFmpegStatus: vi.fn(async () => ({
          available: true, path: '/usr/bin/ffmpeg', version: '7.0', ffprobePath: '/usr/bin/ffprobe', message: '',
        })),
        GetPersistenceStatus: vi.fn(async () => ({ available: true, healthy: true })),
        GetBuildInfo: vi.fn(async () => ({
          version: '0.0.0-test', engineVersion: 'v0.3.0', os: 'linux', architecture: 'amd64', goVersion: 'go1.25',
        })),
        ValidateURL,
        AnalyzeURL,
        AnalyzePlaylist: vi.fn(),
        AnalyzeBatchURLs: vi.fn(),
        RevealInFinder: vi.fn(async () => {}),
      },
    },
  };
  (window as any).runtime = {
    EventsOn: () => () => {},
    EventsOff: () => {},
    EventsEmit: () => {},
    OpenDirectoryDialog: async () => '',
    OpenFileDialog: async () => '',
    ClipboardSetText: async () => {},
    BrowserOpenURL: () => {},
    WindowSetTitle: () => {},
    LogError: () => {},
  };
  return { ValidateURL, AnalyzeURL };
}

async function waitForHome() {
  await screen.findByLabelText('YouTube video, Short, or playlist URL');
}

describe('Home analysis survives navigation', () => {
  beforeEach(() => {
    route.set('home');
    pendingUrl.set('');
    jobs.set([]);
    history.set([]);
    queueView.set(null);
    persistence.set({ available: true, healthy: true });
    ffmpeg.set({ available: true, path: '', version: '', ffprobePath: '', message: '' });
    settings.update((current) => ({
      ...current,
      downloadFolder: '/tmp/downloads',
      automaticDiagnostics: 'disabled',
      confirmBeforeDownload: false,
    }));
    installBindings();
  });

  afterEach(() => {
    route.set('home');
    pendingUrl.set('');
  });

  test('returning from Queue, Downloads, and Settings keeps the analyzed dock', async () => {
    const user = userEvent.setup();
    render(App);
    await waitForHome();

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Queue' }));
    expect(await screen.findByRole('heading', { name: 'Queue' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Nothing here yet' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Go to Home' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Analyze' })).not.toBeInTheDocument();
    expect(document.querySelector('.home-host[hidden]')).toBeTruthy();
    expect(document.querySelector('.home-host[hidden] b')).toHaveTextContent('Fixture video');

    await user.click(screen.getByRole('button', { name: 'Home' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(document.querySelector('.home-host[hidden]')).toBeNull();
    expect(screen.getByLabelText('YouTube video, Short, or playlist URL')).toHaveValue(firstURL);

    await user.click(screen.getByRole('button', { name: 'Downloads' }));
    expect(await screen.findByRole('heading', { name: 'Downloads' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Nothing here yet' })).toBeInTheDocument();
    expect(screen.getByText(/Paste a link on Home/)).toBeInTheDocument();
    expect(screen.getByText(/Finished files show up here/)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Home' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Settings' }));
    expect(await screen.findByRole('heading', { name: 'Settings' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Home' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Download' })).toBeInTheDocument();
  });

  test('the status bar has no Path label', async () => {
    render(App);
    await waitForHome();

    expect(document.querySelector('.status-bar .path')?.textContent?.trim()).toBe('/tmp/downloads');
    expect(document.querySelector('.status-bar .label')).toBeNull();
    expect(screen.getByRole('button', { name: 'Open download folder /tmp/downloads' })).toBeEnabled();

    await userEvent.setup().click(screen.getByRole('button', { name: 'Open download folder /tmp/downloads' }));
    expect((window as any).go.main.App.RevealInFinder).toHaveBeenCalledWith('/tmp/downloads');
    expect(screen.getByRole('button', { name: 'Downloads' }).getAttribute('data-tip')).toBeNull();
    expect(screen.getByRole('button', { name: 'Settings' }).getAttribute('data-tip')).toBeNull();

    await userEvent.setup().click(screen.getByRole('button', { name: 'Downloads' }));
    expect(await screen.findByRole('heading', { name: 'Downloads' })).toBeInTheDocument();
  });

  test('a pending URL handed off from Queue fills Home and analyzes', async () => {
    const { ValidateURL, AnalyzeURL } = installBindings();
    const user = userEvent.setup();
    render(App);
    await waitForHome();

    await user.click(screen.getByRole('button', { name: 'Queue' }));
    expect(await screen.findByRole('heading', { name: 'Nothing here yet' })).toBeInTheDocument();

    pendingUrl.set(firstURL);
    route.set('home');

    const input = await screen.findByLabelText('YouTube video, Short, or playlist URL');
    await waitFor(() => expect(input).toHaveValue(firstURL));
    await waitFor(() => expect(ValidateURL).toHaveBeenCalledWith(firstURL));
    await waitFor(() => expect(AnalyzeURL).toHaveBeenCalledWith(firstURL));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
  });
});

describe('unreadable queue notice', () => {
  beforeEach(() => {
    route.set('home');
    pendingUrl.set('');
    jobs.set([]);
    history.set([]);
    queueView.set(null);
    persistence.set({ available: true, healthy: true });
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: '7.0', ffprobePath: '/usr/bin/ffprobe', message: '' });
    settings.update((current) => ({
      ...current,
      downloadFolder: '/tmp/downloads',
      automaticDiagnostics: 'enabled',
      confirmBeforeDownload: false,
    }));
    installBindings();
    const app = (window as any).go.main.App;
    app.GetStartupStatus = vi.fn(async () => ({ mode: 'healthy', warning: 'queue-reset' }));
    app.GetSettings = vi.fn(async () => ({
      downloadFolder: '/tmp/downloads',
      ffmpegPath: '',
      windowWidth: 1180,
      windowHeight: 760,
      downloadConcurrency: 2,
      perVideoSubfolder: true,
      confirmBeforeDownload: false,
      automaticDiagnostics: 'enabled',
    }));
  });

  afterEach(() => {
    route.set('home');
    pendingUrl.set('');
  });

  test('healthy startup with a reset queue shows a dismissible notice, not the recovery shell', async () => {
    const user = userEvent.setup();
    render(App);
    expect(await screen.findByText('The saved queue could not be read. Files on disk were not touched.')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Download state needs recovery' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Copy diagnostics' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Open data folder' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(screen.queryByText('The saved queue could not be read. Files on disk were not touched.')).not.toBeInTheDocument();
  });
});

describe('cannot-save stop', () => {
  beforeEach(() => {
    route.set('home');
    installBindings();
    const app = (window as any).go.main.App;
    app.GetStartupStatus = vi.fn(async () => ({ mode: 'cannot-save', reason: 'unsafe-permissions' }));
    app.OpenDataFolder = vi.fn(async () => {});
  });

  test('shows a small cannot-save dialog instead of recovery', async () => {
    const user = userEvent.setup();
    render(App);
    expect(await screen.findByRole('heading', { name: 'VidStow cannot save this session' })).toBeInTheDocument();
    expect(screen.getByText(/could not write its data folder/)).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Download state needs recovery' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Home' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Open data folder' }));
    expect((window as any).go.main.App.OpenDataFolder).toHaveBeenCalled();
  });

  test('legacy recovery-required mode uses the cannot-save stop', async () => {
    const app = (window as any).go.main.App;
    app.GetStartupStatus = vi.fn(async () => ({ mode: 'recovery-required', reason: 'indeterminate-commit' }));
    render(App);
    expect(await screen.findByRole('heading', { name: 'VidStow cannot save this session' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Download state needs recovery' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Home' })).not.toBeInTheDocument();
  });
});
