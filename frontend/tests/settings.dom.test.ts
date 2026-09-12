import { render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';

import Settings from '../src/pages/Settings.svelte';
import About from '../src/pages/About.svelte';
import { ffmpeg, settings } from '../src/lib/stores.js';

function installBindings(overrides: Record<string, unknown> = {}) {
  const App = {
    GetBuildInfo: vi.fn(async () => ({
      version: '0.1.0-test', engineVersion: 'v0.3.0', os: 'linux', architecture: 'amd64', goVersion: 'go1.25',
    })),
    UpdateSettings: vi.fn(async (next: unknown) => next),
    SetAutomaticDiagnostics: vi.fn(async (value: string) => ({
      downloadFolder: '/tmp/downloads',
      ffmpegPath: '/usr/bin/ffmpeg',
      windowWidth: 1180,
      windowHeight: 760,
      downloadConcurrency: 2,
      perVideoSubfolder: true,
      automaticDiagnostics: value,
    })),
    PickDownloadFolder: vi.fn(async () => '/tmp/chosen'),
    RevealInFinder: vi.fn(async () => {}),
    ProbeFFmpeg: vi.fn(async () => ({
      available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7.1', ffprobePath: '/usr/bin/ffprobe', message: '',
    })),
    PickFFmpegPath: vi.fn(async () => '/opt/ffmpeg'),
    ConfigureFFmpeg: vi.fn(async () => ({
      available: true, path: '/opt/ffmpeg', version: 'ffmpeg version 7.1', ffprobePath: '/opt/ffprobe', message: '',
    })),
    CopyDiagnostics: vi.fn(async () => 'diag'),
    ClearDiagnostics: vi.fn(async () => {}),
    ...overrides,
  };
  (window as any).go = { main: { App } };
  (window as any).runtime = {
    BrowserOpenURL: vi.fn(),
    EventsOn: () => () => {},
    EventsOff: () => {},
    EventsEmit: () => {},
    OpenDirectoryDialog: async () => '',
    OpenFileDialog: async () => '',
    ClipboardSetText: async () => {},
    WindowSetTitle: () => {},
    LogError: () => {},
  };
  return App;
}

describe('Settings page', () => {
  beforeEach(() => {
    settings.set({
      downloadFolder: '/tmp/downloads',
      ffmpegPath: '/usr/bin/ffmpeg',
      windowWidth: 1180,
      windowHeight: 760,
      downloadConcurrency: 2,
      perVideoSubfolder: true,
      automaticDiagnostics: 'disabled',
      outputOptions: {},
    });
    ffmpeg.set({
      available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7.1', ffprobePath: '/usr/bin/ffprobe', message: '',
    });
    installBindings();
  });

  test('renders grouped cards, live engine info, and the About colophon', async () => {
    render(Settings);
    expect(screen.getByRole('heading', { name: 'Settings' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'General' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Performance' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Advanced' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Diagnostics' })).toBeInTheDocument();
    expect(screen.getByText('Default download folder')).toBeInTheDocument();
    expect(screen.getByText('/tmp/downloads')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Show in Finder' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Change Folder' })).toBeInTheDocument();
    expect(screen.getByText('Create a subfolder for each download')).toBeInTheDocument();
    expect(screen.getByRole('switch', { name: 'Create a subfolder for each download' })).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByText('Interrupted jobs')).toBeInTheDocument();
    expect(screen.getByText('In progress continues')).toBeInTheDocument();
    expect(screen.getByText('The download that was in progress continues when VidStow opens. Waiting stays waiting. Paused stays paused.')).toBeInTheDocument();
    expect(screen.getByText('Maximum concurrent downloads')).toBeInTheDocument();
    expect(screen.getByText(/Range: 1–10; Recommended: 2–4/)).toBeInTheDocument();
    expect(screen.queryByText(/HTTP 429/)).not.toBeInTheDocument();

    await waitFor(() => expect(screen.getByText('ytdlp-go v0.3.0')).toBeInTheDocument());
    expect(screen.getByText(/FFmpeg version 7\.1 · ready for merging/)).toBeInTheDocument();
    expect(screen.getByText('/usr/bin/ffmpeg')).toBeInTheDocument();
    expect(screen.getByText('Ready')).toBeInTheDocument();

    expect(screen.getByText('When VidStow cannot complete a requested download, send a small sanitized report. Links, paths, and error text stay private.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Diagnostics privacy notice ↗' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copy diagnostics' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Clear history' })).toBeInTheDocument();

    expect(screen.getByText('VidStow')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText('0.1.0-test · Apache-2.0 · linux/amd64')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'View source' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Read the docs' })).toBeInTheDocument();
    expect(screen.getByText(/Built with Go · Wails · Svelte · ytdlp-go · FFmpeg/)).toBeInTheDocument();
  });

  test('subfolder switch talks to UpdateSettings', async () => {
    const user = userEvent.setup();
    const App = installBindings();
    render(Settings);
    await user.click(screen.getByRole('switch', { name: 'Create a subfolder for each download' }));
    await waitFor(() => expect(App.UpdateSettings).toHaveBeenCalledWith(expect.objectContaining({ perVideoSubfolder: false })));
  });

  test('video files defaults talk to UpdateSettings through outputOptions', async () => {
    const user = userEvent.setup();
    const App = installBindings();
    render(Settings);
    expect(screen.getByRole('heading', { name: 'Video files' })).toBeInTheDocument();
    expect(screen.getByRole('switch', { name: 'Embed thumbnail artwork' })).toBeDisabled();
    expect(screen.queryByLabelText('Default subtitle language')).not.toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText('Default subtitle mode'), 'sidecar');
    await waitFor(() => expect(App.UpdateSettings).toHaveBeenCalledWith(expect.objectContaining({
      outputOptions: expect.objectContaining({ subtitleMode: 'sidecar' }),
    })));
    expect(screen.getByText('Saves a caption file next to the video.')).toBeInTheDocument();
    expect(await screen.findByLabelText('Default subtitle file format')).toBeInTheDocument();
    expect(screen.getByLabelText('Default subtitle language')).toHaveValue('');

    await user.selectOptions(screen.getByLabelText('Default subtitle language'), 'es');
    await waitFor(() => expect(App.UpdateSettings).toHaveBeenCalledWith(expect.objectContaining({
      outputOptions: expect.objectContaining({ subtitleMode: 'sidecar', subtitleLanguages: ['es'] }),
    })));

    await user.click(screen.getByRole('switch', { name: 'Embed title and channel details' }));
    await waitFor(() => expect(App.UpdateSettings).toHaveBeenCalledWith(expect.objectContaining({
      outputOptions: expect.objectContaining({ embedMetadata: true }),
    })));

    await user.selectOptions(screen.getByLabelText('Default subtitle mode'), 'embed');
    await waitFor(() => expect(App.UpdateSettings).toHaveBeenCalledWith(expect.objectContaining({
      outputOptions: expect.objectContaining({ subtitleMode: 'embed' }),
    })));
    expect(screen.getByText('Writes captions inside the video. The folder still shows one MP4.')).toBeInTheDocument();
    expect(screen.queryByLabelText('Default subtitle file format')).not.toBeInTheDocument();
  });

  test('concurrency stepper warns above 4 and talks to UpdateSettings', async () => {
    const user = userEvent.setup();
    const App = installBindings();
    render(Settings);
    await user.click(screen.getByRole('button', { name: 'Increase concurrent downloads' }));
    await waitFor(() => expect(App.UpdateSettings).toHaveBeenCalledWith(expect.objectContaining({ downloadConcurrency: 3 })));
    settings.update((current) => ({ ...current, downloadConcurrency: 5 }));
    await waitFor(() => expect(screen.getByText(/YouTube rate limits \(HTTP 429\)/)).toBeInTheDocument());
  });

  test('Send and Don’t send call the existing automatic diagnostics API', async () => {
    const user = userEvent.setup();
    const App = installBindings();
    render(Settings);
    await user.click(screen.getByRole('button', { name: 'Send' }));
    expect(App.SetAutomaticDiagnostics).toHaveBeenCalledWith('enabled');
    await user.click(screen.getByRole('button', { name: /Don.t send/ }));
    expect(App.SetAutomaticDiagnostics).toHaveBeenCalledWith('disabled');
  });

  test('Show in Finder, Change Folder, Recheck Dependencies, and diagnostics buttons use existing APIs', async () => {
    const user = userEvent.setup();
    const App = installBindings();
    render(Settings);
    await user.click(screen.getByRole('button', { name: 'Show in Finder' }));
    expect(App.RevealInFinder).toHaveBeenCalledWith('/tmp/downloads');

    await user.click(screen.getByRole('button', { name: 'Recheck Dependencies' }));
    expect(App.ProbeFFmpeg).toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Copy diagnostics' }));
    expect(App.CopyDiagnostics).toHaveBeenCalled();
    await user.click(screen.getByRole('button', { name: 'Clear history' }));
    expect(App.ClearDiagnostics).toHaveBeenCalled();
  });

  test('the About route renders the same Settings page', async () => {
    render(About);
    expect(await screen.findByRole('heading', { name: 'Settings' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'View source' })).toBeInTheDocument();
  });
});
