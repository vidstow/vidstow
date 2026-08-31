import { render, screen } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import App from '../src/App.svelte';
import { ffmpeg, history, jobs, pendingHomeFocus, pendingUrl, persistence, queueView, route, settings } from '../src/lib/stores.js';

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
        ValidateURL,
        AnalyzeURL,
        AnalyzePlaylist: vi.fn(),
        AnalyzeBatchURLs: vi.fn(),
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
    pendingHomeFocus.set(false);
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
    pendingHomeFocus.set(false);
  });

  test('returning from Queue, Downloads, and Settings keeps the analyzed dock', async () => {
    const user = userEvent.setup();
    render(App);
    await waitForHome();

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Queue' }));
    expect(await screen.findByText('Nothing in the queue')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Analyze' })).not.toBeInTheDocument();
    expect(document.querySelector('.home-host[hidden]')).toBeTruthy();
    expect(document.querySelector('.home-host[hidden] b')).toHaveTextContent('Fixture video');

    await user.click(screen.getByRole('button', { name: 'Home' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(document.querySelector('.home-host[hidden]')).toBeNull();
    expect(screen.getByLabelText('YouTube video, Short, or playlist URL')).toHaveValue(firstURL);

    await user.click(screen.getByRole('button', { name: 'Downloads' }));
    expect(await screen.findByText('No downloads yet.')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Home' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Settings' }));
    expect(await screen.findByRole('heading', { name: 'Settings' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Home' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Download' })).toBeInTheDocument();
  });
});
