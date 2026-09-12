import { render, screen, waitFor, act } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';

import Home from '../src/pages/Home.svelte';
import { pendingUrl, settings, modal, banner, ffmpeg } from '../src/lib/stores.js';

const firstURL = 'https://www.youtube.com/watch?v=fixture0001';

function videoSummary(raw: string) {
  return {
    title: 'Fixture video', channel: 'Fixture channel', duration: '1:00', thumbnail: '', videoId: 'fixture0001', url: raw,
    durationSeconds: 60, viewCount: 1, uploadDate: '', description: '', access: { code: 'public', label: 'Public' },
    subtitles: [
      { code: 'en', name: 'English' },
      { code: 'de', name: 'German' },
      { code: 'en', name: 'English (auto-generated)', auto: true },
    ],
    plans: [
      { id: 'video', kind: 'video', label: 'Video plan', container: 'MP4', videoCodec: 'H.264', audioCodec: 'AAC', approxBytes: 309248, available: true, recommended: true },
      { id: 'audio', kind: 'audio', label: 'M4A (Original)', container: 'M4A', audioCodec: 'AAC', approxBytes: 309248, available: true },
    ],
  };
}

function playlistSummary(raw: string) {
  return {
    id: 'PLfixture',
    url: raw,
    title: 'Fixture playlist',
    channel: 'Fixture channel',
    duration: '3m',
    durationSeconds: 180,
    thumbnail: '',
    entryCount: 2,
    available: 2,
    unavailable: 0,
    entries: [
      { index: 1, videoId: 'fixture0001', url: firstURL, title: 'First video', available: true, duration: '1:00' },
      { index: 2, videoId: 'fixture0002', url: 'https://www.youtube.com/watch?v=fixture0002', title: 'Second video', available: true, duration: '2:00' },
    ],
  };
}

function installBindings() {
  const ValidateURL = vi.fn(async (raw: string) => ({
    kind: 'single_video', url: raw, videoUrl: raw, playlistUrl: '', videoId: 'fixture0001', playlistId: '',
  }));
  const AnalyzeURL = vi.fn(async (raw: string) => videoSummary(raw));
  const AnalyzePlaylist = vi.fn(async (raw: string) => playlistSummary(raw));
  const AnalyzeBatchURLs = vi.fn();
  const StartBatchDownload = vi.fn();
  const StartDownload = vi.fn(async () => 'job-1');
  const StartPlaylistDownload = vi.fn(async (_req: { selectedItems: number[] }) => ({ collectionId: 'playlist-1', admitted: 2 }));
  (window as any).go = { main: { App: { ValidateURL, AnalyzeURL, AnalyzePlaylist, AnalyzeBatchURLs, StartBatchDownload, StartDownload, StartPlaylistDownload } } };
  return { ValidateURL, AnalyzeURL, AnalyzePlaylist, AnalyzeBatchURLs, StartBatchDownload, StartDownload, StartPlaylistDownload };
}

describe('Home analysis authority', () => {
  test('empty Home shows try chips that fill the field', async () => {
    render(Home);
    expect(document.querySelector('.fieldwrap .kbd')).toBeNull();
    const video = screen.getByRole('button', { name: 'a video' });
    const playlist = screen.getByRole('button', { name: 'a playlist' });
    const batch = screen.getByRole('button', { name: 'several links' });
    expect(video).toHaveClass('try');
    expect(playlist).toHaveClass('try');
    expect(batch).toHaveClass('try');
    expect(screen.queryByRole('button', { name: 'a private link' })).not.toBeInTheDocument();
    await userEvent.setup().click(video);
    expect(screen.getByLabelText('YouTube video, Short, or playlist URL')).toHaveValue(
      'https://www.youtube.com/watch?v=jNQXAC9IVRw',
    );
  });

  test('a watch URL with a list chooses video or playlist and can swap', async () => {
    const user = userEvent.setup();
    const watch = 'https://www.youtube.com/watch?v=Xpr8D6LeAtw&list=PLPTV0NXA_ZSgsLAr8YCgCwhPIJNNtexWu';
    (window as any).go.main.App.ValidateURL = vi.fn(async () => ({
      kind: 'video_playlist',
      url: watch,
      videoUrl: 'https://www.youtube.com/watch?v=Xpr8D6LeAtw',
      playlistUrl: 'https://www.youtube.com/playlist?list=PLPTV0NXA_ZSgsLAr8YCgCwhPIJNNtexWu',
      videoId: 'Xpr8D6LeAtw',
      playlistId: 'PLPTV0NXA_ZSgsLAr8YCgCwhPIJNNtexWu',
    }));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), watch);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('This link includes a playlist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Fixture video/ })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Fixture video/ }));
    expect(await screen.findByRole('button', { name: /Part of playlist/ })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Part of playlist/ }));
    expect(await screen.findByText('Fixture playlist')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Pasted video/ }));
    expect(await screen.findByRole('button', { name: /Part of playlist/ })).toBeInTheDocument();
  });

  test('Enter in the URL field analyzes instead of inserting a newline', async () => {
    const user = userEvent.setup();
    const { AnalyzeURL } = installBindings();
    render(Home);
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.click(field);
    await user.paste(firstURL);
    await user.keyboard('{Enter}');
    await waitFor(() => expect(AnalyzeURL).toHaveBeenCalled());
    expect(field).toHaveValue(firstURL);
  });

  beforeEach(() => {
    pendingUrl.set('');
    modal.set(null);
    banner.set(null);
    settings.update((current) => ({ ...current, downloadFolder: '/tmp/downloads', outputOptions: {} }));
    installBindings();
  });

  test('invalidates an analyzed result when the URL input changes', async () => {
    const user = userEvent.setup();
    render(Home);

    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.type(input, firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.clear(input);
    await user.type(input, 'https://www.youtube.com/watch?v=fixture0002');
    await waitFor(() => expect(screen.queryByText('Fixture video')).not.toBeInTheDocument());
    expect(screen.getByText(/the link decides/)).toBeInTheDocument();
  });

  test('re-analyzes a URL dropped while that URL is already in the field', async () => {
    const { AnalyzeURL } = installBindings();
    render(Home);

    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await userEvent.setup().type(input, firstURL);
    pendingUrl.set(firstURL);

    await waitFor(() => expect(AnalyzeURL).toHaveBeenCalledWith(firstURL));
    expect(input).toHaveValue(firstURL);
    expect(get(pendingUrl)).toBe('');
  });

  test('a pending URL fills an empty field and analyzes it', async () => {
    const { ValidateURL, AnalyzeURL } = installBindings();
    render(Home);

    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    expect(input).toHaveValue('');
    pendingUrl.set(firstURL);

    await waitFor(() => expect(input).toHaveValue(firstURL));
    await waitFor(() => expect(ValidateURL).toHaveBeenCalledWith(firstURL));
    await waitFor(() => expect(AnalyzeURL).toHaveBeenCalledWith(firstURL));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(get(pendingUrl)).toBe('');
  });

  test('a pending URL replaces a different field value before analyzing', async () => {
    const { AnalyzeURL } = installBindings();
    render(Home);

    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await userEvent.setup().type(input, 'https://www.youtube.com/watch?v=oldvideo01');
    pendingUrl.set(firstURL);

    await waitFor(() => expect(input).toHaveValue(firstURL));
    await waitFor(() => expect(AnalyzeURL).toHaveBeenCalledWith(firstURL));
    expect(AnalyzeURL).not.toHaveBeenCalledWith('https://www.youtube.com/watch?v=oldvideo01');
  });

  test('Analyze becomes Stop in the same slot while the request is in flight', async () => {
    const user = userEvent.setup();
    (window as any).go.main.App.ValidateURL = vi.fn(() => new Promise(() => {}));
    render(Home);

    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.type(input, firstURL);
    const idle = screen.getByRole('button', { name: 'Analyze' });
    expect(idle).toHaveClass('dbtn', 'query');
    expect(idle.querySelector('svg')).toBeTruthy();
    expect(idle.querySelector('.query-spin')).toBeNull();
    const idleWidth = idle.getBoundingClientRect().width;

    await user.click(idle);
    const stop = await screen.findByRole('button', { name: 'Stop' });
    expect(stop).toHaveClass('dbtn', 'query', 'stop-slot');
    expect(stop.querySelector('.stop-mark')).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Analyze' })).not.toBeInTheDocument();
    expect(document.querySelector('.hud-strip')).toBeTruthy();
    expect(await screen.findByText(/Checking link/)).toBeInTheDocument();
    expect(stop.getBoundingClientRect().width).toBe(idleWidth);
  });

  test('does not publish an in-flight result after the URL changes', async () => {
    const user = userEvent.setup();
    let finishAnalysis!: (result: ReturnType<typeof videoSummary>) => void;
    const AnalyzeURL = vi.fn(() => new Promise<ReturnType<typeof videoSummary>>((resolve) => { finishAnalysis = resolve; }));
    (window as any).go.main.App.AnalyzeURL = AnalyzeURL;
    render(Home);

    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.type(input, firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await waitFor(() => expect(AnalyzeURL).toHaveBeenCalledOnce());

    const secondURL = 'https://www.youtube.com/watch?v=fixture0002';
    await user.clear(input);
    await user.type(input, secondURL);
    finishAnalysis(videoSummary(firstURL));

    await waitFor(() => expect(screen.queryByText('Fixture video')).not.toBeInTheDocument());
    expect(input).toHaveValue(secondURL);
    expect(screen.getByRole('button', { name: 'Analyze' })).toBeEnabled();
  });

  test('does not enable playlist admission without a destination', async () => {
    const user = userEvent.setup();
    const playlistURL = 'https://www.youtube.com/playlist?list=PLfixture';
    settings.update((current) => ({ ...current, downloadFolder: '' }));
    (window as any).go.main.App.ValidateURL = vi.fn(async () => ({
      kind: 'playlist', url: playlistURL, playlistUrl: playlistURL, playlistId: 'PLfixture',
    }));
    (window as any).go.main.App.AnalyzePlaylist = vi.fn(async () => ({
      id: 'PLfixture', url: playlistURL, title: 'Fixture playlist', channel: 'Fixture channel', duration: '3m',
      durationSeconds: 180, thumbnail: '',
      entryCount: 1, available: 1, unavailable: 0,
      entries: [{ index: 1, videoId: 'fixture0001', url: firstURL, title: 'First video', available: true }],
    }));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), playlistURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));

    expect(await screen.findByText('Fixture playlist')).toBeInTheDocument();
    expect(screen.getByText('Playlist · 1 videos · 3m · Fixture channel')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Download 1 video' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'All' })).not.toBeInTheDocument();
  });

  test('keeps playlist episodes behind the disclosure and shows Range without Search', async () => {
    const user = userEvent.setup();
    const playlistURL = 'https://www.youtube.com/playlist?list=PLfixture';
    (window as any).go.main.App.ValidateURL = vi.fn(async () => ({
      kind: 'playlist', url: playlistURL, playlistUrl: playlistURL, playlistId: 'PLfixture',
    }));
    (window as any).go.main.App.AnalyzePlaylist = vi.fn(async (raw: string) => playlistSummary(raw));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), playlistURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));

    expect(await screen.findByRole('button', { name: 'Download 2 videos' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Download 2 videos' }).querySelector('svg')).toBeTruthy();
    expect(screen.queryByText('First video')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /2 episodes/ }));
    expect(await screen.findByText('First video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'All' })).toBeInTheDocument();
    expect(screen.getByLabelText('Range start')).toBeInTheDocument();
    expect(screen.queryByLabelText('Search playlist')).not.toBeInTheDocument();
    expect(screen.getByText('2 selected · up to 1080p')).toBeInTheDocument();
  });

  test('playlist Download starts the collection path, not a single video', async () => {
    const user = userEvent.setup();
    const playlistURL = 'https://www.youtube.com/playlist?list=PLfixture';
    const { StartDownload, StartPlaylistDownload } = installBindings();
    StartPlaylistDownload.mockImplementation(async (req: { selectedItems: number[] }) => ({
      collectionId: 'playlist-1', admitted: req.selectedItems.length,
    }));
    (window as any).go.main.App.ValidateURL = vi.fn(async () => ({
      kind: 'playlist', url: playlistURL, playlistUrl: playlistURL, playlistId: 'PLfixture',
    }));
    (window as any).go.main.App.AnalyzePlaylist = vi.fn(async (raw: string) => playlistSummary(raw));
    const onGoto = vi.fn();
    render(Home, { events: { goto: onGoto } });

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), playlistURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await user.click(await screen.findByRole('button', { name: /2 episodes/ }));
    await user.click(screen.getByRole('button', { name: /First video/ }));
    expect(await screen.findByRole('button', { name: 'Download 1 video' })).toBeEnabled();
    await user.click(screen.getByRole('button', { name: 'Download 1 video' }));

    await waitFor(() => expect(StartPlaylistDownload).toHaveBeenCalledWith({
      url: playlistURL, playlistId: 'PLfixture', quality: '1080p', audioBitrate: 0, selectedItems: [2], options: {},
    }));
    expect(StartDownload).not.toHaveBeenCalled();
    expect(get(banner)).toMatchObject({ kind: 'success', message: 'Added 1 video to queue' });
    expect(onGoto.mock.calls.some((call) => call[0]?.detail === 'queue')).toBe(true);
  });

  test('playlist start errors keep Playlist could not start without single-video copy from Home', async () => {
    const user = userEvent.setup();
    const playlistURL = 'https://www.youtube.com/playlist?list=PLfixture';
    const { StartDownload, StartPlaylistDownload } = installBindings();
    StartPlaylistDownload.mockRejectedValue(new Error('A selected video requires a channel membership and is not available in this version.'));
    (window as any).go.main.App.ValidateURL = vi.fn(async () => ({
      kind: 'playlist', url: playlistURL, playlistUrl: playlistURL, playlistId: 'PLfixture',
    }));
    (window as any).go.main.App.AnalyzePlaylist = vi.fn(async (raw: string) => playlistSummary(raw));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), playlistURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await user.click(await screen.findByRole('button', { name: 'Download 2 videos' }));

    await waitFor(() => expect(StartPlaylistDownload).toHaveBeenCalledOnce());
    expect(StartDownload).not.toHaveBeenCalled();
    expect(await screen.findByRole('alert')).toHaveTextContent('Playlist could not start');
    expect(get(modal)).toBeNull();
    expect(screen.getByText('Fixture playlist')).toBeInTheDocument();
  });

  test('All and None write the range fields', async () => {
    const user = userEvent.setup();
    const playlistURL = 'https://www.youtube.com/playlist?list=PLfixture';
    (window as any).go.main.App.ValidateURL = vi.fn(async () => ({
      kind: 'playlist', url: playlistURL, playlistUrl: playlistURL, playlistId: 'PLfixture',
    }));
    (window as any).go.main.App.AnalyzePlaylist = vi.fn(async (raw: string) => ({
      ...playlistSummary(raw),
      entryCount: 4,
      available: 4,
      entries: [
        { index: 1, videoId: 'a', url: firstURL, title: 'First video', available: true, duration: '1:00' },
        { index: 2, videoId: 'b', url: firstURL, title: 'Second video', available: true, duration: '2:00' },
        { index: 3, videoId: 'c', url: firstURL, title: 'Third video', available: true, duration: '3:00' },
        { index: 4, videoId: 'd', url: firstURL, title: 'Fourth video', available: true, duration: '4:00' },
      ],
    }));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), playlistURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await user.click(await screen.findByRole('button', { name: /4 episodes/ }));

    const start = screen.getByLabelText('Range start');
    const end = screen.getByLabelText('Range end');
    expect(start).toHaveValue(1);
    expect(end).toHaveValue(4);

    await user.clear(end);
    await user.type(end, '2');
    await user.click(screen.getByRole('button', { name: 'Apply' }));
    expect(screen.getByText('2 of 4 selected')).toBeInTheDocument();
    expect(end).toHaveValue(2);

    await user.click(screen.getByRole('button', { name: 'All' }));
    expect(start).toHaveValue(1);
    expect(end).toHaveValue(4);
    expect(screen.getByText('4 of 4 selected')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'None' }));
    expect(start).toHaveValue(null);
    expect(end).toHaveValue(null);
    expect(screen.getByText('0 of 4 selected')).toBeInTheDocument();
  });

  test('badges a Short from extracted media type, not the submitted URL', async () => {
    const user = userEvent.setup();
    const { AnalyzeURL } = installBindings();
    AnalyzeURL.mockImplementation(async (raw: string) => ({ ...videoSummary(raw), mediaType: 'short' }));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));

    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(screen.getByText((content, el) => el?.tagName === 'EM' && content === 'Short')).toBeInTheDocument();
  });

  test('does not badge a Shorts URL without extracted media type', async () => {
    const user = userEvent.setup();
    const shortsURL = 'https://www.youtube.com/shorts/fixture0001';
    const watchURL = 'https://www.youtube.com/watch?v=fixture0001';
    const { ValidateURL } = installBindings();
    ValidateURL.mockImplementation(async () => ({
      kind: 'single_video', url: watchURL, videoUrl: watchURL, playlistUrl: '', videoId: 'fixture0001', playlistId: '',
    }));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), shortsURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));

    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(screen.queryByText('Short')).not.toBeInTheDocument();
  });

  test('two pasted lines go to batch review without a Batch URLs mode', async () => {
    const user = userEvent.setup();
    const { ValidateURL, AnalyzeBatchURLs } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'batch-token', expiresAt: '2099-08-22T12:00:00Z',
      counts: { pasted: 2, ready: 2, duplicate: 0, invalid: 0, analysisFailed: 0 },
      items: [
        { lineNumber: 1, input: 'one', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'First' },
        { lineNumber: 2, input: 'two', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Second' },
      ],
    });
    render(Home);
    expect(screen.queryByRole('button', { name: 'Batch URLs' })).not.toBeInTheDocument();
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await waitFor(() => expect(AnalyzeBatchURLs).toHaveBeenCalledWith('one\ntwo'));
    expect(ValidateURL).not.toHaveBeenCalled();
    expect(await screen.findByText('Batch of public videos')).toBeInTheDocument();
  });

  test('does not publish an in-flight batch review after the lines change', async () => {
    const user = userEvent.setup();
    let finishReview!: (result: any) => void;
    const { AnalyzeBatchURLs } = installBindings();
    AnalyzeBatchURLs.mockImplementation(() => new Promise((resolve) => { finishReview = resolve; }));
    render(Home);
    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.type(input, 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await waitFor(() => expect(AnalyzeBatchURLs).toHaveBeenCalledOnce());
    await user.type(input, '\nthree');
    finishReview({
      token: 'stale-token', expiresAt: '2099-01-01T00:00:00Z',
      counts: { pasted: 2, ready: 2, duplicate: 0, invalid: 0, analysisFailed: 0 }, items: [],
    });
    await waitFor(() => expect(screen.queryByText('Batch of public videos')).not.toBeInTheDocument());
    expect(screen.getByRole('button', { name: 'Analyze' })).toBeEnabled();
  });

  test('reviews mixed batch lines on Home and starts every ready item', async () => {
    const user = userEvent.setup();
    const { AnalyzeBatchURLs, StartBatchDownload } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'batch-token', expiresAt: '2099-08-22T12:00:00Z',
      counts: { pasted: 5, ready: 3, duplicate: 1, invalid: 1, analysisFailed: 0 },
      items: [
        { lineNumber: 1, input: 'https://youtu.be/fixture0001', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'First', thumbnail: 'https://i.ytimg.com/vi/fixture0001/hqdefault.jpg' },
        { lineNumber: 2, input: 'https://youtu.be/fixture0002', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Second' },
        { lineNumber: 3, input: 'https://www.youtube.com/watch?v=fixture0001', status: 'duplicate', messageKey: 'batch.duplicate', message: 'Duplicate of line 1', duplicateOfLine: 1 },
        { lineNumber: 4, input: 'not-a-url', status: 'invalid', messageKey: 'batch.invalid_url', message: 'Only YouTube links are supported.' },
        { lineNumber: 5, input: 'https://youtu.be/fixture0003', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Third' },
      ],
    });
    StartBatchDownload.mockResolvedValue({ collectionId: 'batch-1', admitted: 3 });
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo\nthree\nfour\nfive');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));

    expect(await screen.findByText('Duplicate of line 1')).toBeInTheDocument();
    expect(screen.getByText('Only YouTube links are supported.')).toBeInTheDocument();
    expect(screen.getByText('3 ready · 1 duplicate · 1 invalid')).toBeInTheDocument();
    expect(screen.queryByLabelText('Batch review counts')).not.toBeInTheDocument();
    expect(document.querySelector('.batch-thumbnail img')).toHaveAttribute('src', 'https://i.ytimg.com/vi/fixture0001/hqdefault.jpg');
    const start = screen.getByRole('button', { name: 'Download 3 videos' });
    expect(start).toBeEnabled();
    expect(start.querySelector('svg')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Subtitle language' })).toHaveTextContent('English or first available');
    expect(screen.getByRole('radiogroup', { name: 'Subtitle mode' })).toBeInTheDocument();
    await user.click(start);
    await waitFor(() => expect(StartBatchDownload).toHaveBeenCalledWith({ token: 'batch-token', quality: '1080p', audioBitrate: 0, options: {} }));
  });

  test('batch Download sends the playlist inspector choices', async () => {
    const user = userEvent.setup();
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    const { AnalyzeBatchURLs, StartBatchDownload } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'batch-token', expiresAt: '2099-08-22T12:00:00Z',
      counts: { pasted: 2, ready: 2, duplicate: 0, invalid: 0, analysisFailed: 0 },
      items: [
        { lineNumber: 1, input: 'one', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'First' },
        { lineNumber: 2, input: 'two', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Second' },
      ],
    });
    StartBatchDownload.mockResolvedValue({ collectionId: 'batch-1', admitted: 2 });
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Subtitle language' })).toHaveTextContent('English or first available');
    await user.click(screen.getByRole('radio', { name: 'Subtitle file' }));
    await user.click(screen.getByRole('button', { name: 'Subtitle language' }));
    await user.click(screen.getByRole('option', { name: 'Spanish' }));
    await user.click(screen.getByLabelText('Title & channel details'));
    await user.click(screen.getByRole('button', { name: 'Download 2 videos' }));

    await waitFor(() => expect(StartBatchDownload).toHaveBeenCalledTimes(1));
    const request = StartBatchDownload.mock.calls[0][0];
    expect(request.options.subtitleMode).toBe('sidecar');
    expect(request.options.subtitleFormat).toBe('srt');
    expect(request.options.subtitleLanguages).toEqual(['es']);
    expect(request.options.embedMetadata).toBe(true);
  });

  test('batch Language seeds from the Settings default subtitle language', async () => {
    const user = userEvent.setup();
    settings.update((current) => ({
      ...current,
      outputOptions: { subtitleMode: 'sidecar', subtitleLanguages: ['es'] },
    }));
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    const { AnalyzeBatchURLs, StartBatchDownload } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'batch-token', expiresAt: '2099-08-22T12:00:00Z',
      counts: { pasted: 2, ready: 2, duplicate: 0, invalid: 0, analysisFailed: 0 },
      items: [
        { lineNumber: 1, input: 'one', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'First' },
        { lineNumber: 2, input: 'two', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Second' },
      ],
    });
    StartBatchDownload.mockResolvedValue({ collectionId: 'batch-1', admitted: 2 });
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Subtitle language' })).toHaveTextContent('Spanish');
    await user.click(screen.getByRole('button', { name: 'Download 2 videos' }));

    await waitFor(() => expect(StartBatchDownload).toHaveBeenCalledTimes(1));
    expect(StartBatchDownload.mock.calls[0][0].options.subtitleLanguages).toEqual(['es']);
  });

  test('batch audio outputs do not carry subtitle choices', async () => {
    const user = userEvent.setup();
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    const { AnalyzeBatchURLs, StartBatchDownload } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'batch-token', expiresAt: '2099-08-22T12:00:00Z',
      counts: { pasted: 2, ready: 2, duplicate: 0, invalid: 0, analysisFailed: 0 },
      items: [
        { lineNumber: 1, input: 'one', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'First' },
        { lineNumber: 2, input: 'two', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Second' },
      ],
    });
    StartBatchDownload.mockResolvedValue({ collectionId: 'batch-1', admitted: 2 });
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Download 2 videos' })).toBeEnabled();
    await user.click(screen.getByRole('radio', { name: 'Subtitle file' }));
    await user.click(screen.getByRole('button', { name: 'Audio' }));
    expect(screen.getByRole('radio', { name: 'Subtitle file' })).toBeDisabled();
    expect(screen.getByText('Video only')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Download 2 videos' }));

    await waitFor(() => expect(StartBatchDownload).toHaveBeenCalledTimes(1));
    const request = StartBatchDownload.mock.calls[0][0];
    expect(request.quality).toBe('audio');
    expect(request.options.subtitleMode).toBe('');
    expect(request.options.subtitleLanguages).toBeUndefined();
  });

  test('invalidates a batch review when the user edits the lines', async () => {
    const user = userEvent.setup();
    const { AnalyzeBatchURLs } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'batch-token', expiresAt: '2099-08-22T12:00:00Z', counts: { pasted: 2, ready: 2, duplicate: 0, invalid: 0, analysisFailed: 0 },
      items: [
        { lineNumber: 1, input: 'one', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'First' },
        { lineNumber: 2, input: 'two', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Second' },
      ],
    });
    render(Home);

    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.type(input, 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Download 2 videos' })).toBeEnabled();
    expect(screen.getByText('2 videos ready to download')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Edit URLs' }));
    expect(screen.queryByRole('button', { name: 'Download 2 videos' })).not.toBeInTheDocument();
    expect(screen.getByLabelText('YouTube video, Short, or playlist URL')).toHaveValue('one\ntwo');
  });

  test('disables admission when the backend batch review is expired', async () => {
    const user = userEvent.setup();
    const { AnalyzeBatchURLs } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'expired-token', expiresAt: '2000-01-01T00:00:00Z',
      counts: { pasted: 2, ready: 2, duplicate: 0, invalid: 0, analysisFailed: 0 },
      items: [
        { lineNumber: 1, input: 'one', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'First' },
        { lineNumber: 2, input: 'two', status: 'ready', messageKey: 'batch.ready', message: 'Ready', title: 'Second' },
      ],
    });
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('This review expired');
    expect(screen.getByRole('button', { name: 'Download 2 videos' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'Review again' }));
    await waitFor(() => expect(AnalyzeBatchURLs).toHaveBeenCalledTimes(2));
  });

  test('switching output types selects a visible compatible plan', async () => {
    const user = userEvent.setup();
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Download' })).toBeEnabled();
    expect(screen.getByRole('radio', { name: 'Video plan' })).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByText('MP4 · H.264 · AAC')).toBeInTheDocument();
    expect(screen.getByText('302 KB')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Audio' }));
    expect(screen.getByRole('button', { name: 'Audio' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('radio', { name: 'M4A' })).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByRole('radio', { name: 'M4A' })).toHaveAttribute('title', 'Original');
    expect(screen.getByText('M4A · AAC')).toBeInTheDocument();
    expect(screen.queryByText('Video only')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Download' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Download' }).querySelector('svg')).toBeTruthy();
  });

  test('a failed Analyze selects the URL so the next paste replaces it', async () => {
    const user = userEvent.setup();
    const { ValidateURL } = installBindings();
    ValidateURL.mockRejectedValue(new Error('This video is private.'));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Check this link')).toBeInTheDocument();
    expect(screen.getByText('This video is private.')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Paste another link' })).not.toBeInTheDocument();
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL') as HTMLTextAreaElement;
    await waitFor(() => expect(field).toHaveFocus());
    expect(field.selectionStart).toBe(0);
    expect(field.selectionEnd).toBe(field.value.length);
  });

  test('successful Download clears the analysed dock, keeps the URL, and goes to Queue', async () => {
    const user = userEvent.setup();
    const { StartDownload } = installBindings();
    const onGoto = vi.fn();
    render(Home, { events: { goto: onGoto } });

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Download' }));
    await waitFor(() => expect(StartDownload).toHaveBeenCalledOnce());
    await waitFor(() => expect(screen.queryByText('Fixture video')).not.toBeInTheDocument());
    expect(screen.queryByRole('button', { name: 'Download' })).not.toBeInTheDocument();
    expect(screen.getByText(/the link decides/)).toBeInTheDocument();
    expect(screen.getByLabelText('YouTube video, Short, or playlist URL')).toHaveValue(firstURL);
    expect(get(banner)).toMatchObject({ kind: 'success', message: 'Queued for download' });
    expect(onGoto.mock.calls.some((call) => call[0]?.detail === 'queue')).toBe(true);
  });

  test('failed Download keeps the analysed dock', async () => {
    const user = userEvent.setup();
    const { StartDownload } = installBindings();
    StartDownload.mockRejectedValue(new Error('Could not start this download.'));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Download' }));
    await waitFor(() => expect(StartDownload).toHaveBeenCalledOnce());
    expect(screen.getByText('Fixture video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Download' })).toBeEnabled();
    expect(await screen.findByRole('alert')).toHaveTextContent('Download could not start');
    expect(get(modal)).toBeNull();
  });

  test('stale format errors keep Analyze again copy, not the jobs prefix', async () => {
    const user = userEvent.setup();
    const { StartDownload } = installBindings();
    StartDownload.mockRejectedValue(new Error('jobs: output options expired; analyze the video again'));
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Download' }));
    await waitFor(() => expect(StartDownload).toHaveBeenCalledOnce());
    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('Download could not start');
    expect(alert).toHaveTextContent('The formats from Analyze went stale. Click Analyze again, then Download.');
    expect(alert).not.toHaveTextContent('jobs:');
    expect(screen.getByRole('button', { name: 'Analyze again' })).toBeEnabled();
  });

  test('Analyze submits the live field when the DOM and Svelte state diverge', async () => {
    const { ValidateURL, AnalyzeURL } = installBindings();
    render(Home);
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL') as HTMLTextAreaElement;
    await userEvent.setup().type(field, firstURL);
    const later = 'https://www.youtube.com/watch?v=fixture9999';
    field.value = later;
    await userEvent.setup().click(screen.getByRole('button', { name: 'Analyze' }));
    await waitFor(() => expect(ValidateURL).toHaveBeenCalledWith(later));
    expect(AnalyzeURL).toHaveBeenCalledWith(later);
  });

  test('passes subtitle and detail choices to StartDownload', async () => {
    const user = userEvent.setup();
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    const StartDownload = vi.fn(async (_request: { options: Record<string, unknown> }) => 'job-1');
    (window as any).go.main.App.StartDownload = StartDownload;
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('radio', { name: 'Subtitle file' }));
    expect(screen.getByRole('button', { name: 'Subtitle language' })).toHaveTextContent('English');
    await user.click(screen.getByRole('button', { name: 'Subtitle language' }));
    expect(screen.getByRole('option', { name: 'English' })).toHaveAttribute('aria-selected', 'true');
    await user.click(screen.getByLabelText('Title & channel details'));
    await user.click(screen.getByRole('button', { name: 'Download' }));

    await waitFor(() => expect(StartDownload).toHaveBeenCalledTimes(1));
    const request = StartDownload.mock.calls[0][0];
    expect(request.options.subtitleMode).toBe('sidecar');
    expect(request.options.subtitleLanguages).toEqual(['en']);
    expect(request.options.subtitleFormat).toBe('srt');
    expect(request.options.embedMetadata).toBe(true);
  });

  test('single video uses the Settings language when that video offers it', async () => {
    const user = userEvent.setup();
    settings.update((current) => ({
      ...current,
      outputOptions: { subtitleMode: 'sidecar', subtitleLanguages: ['de'] },
    }));
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Subtitle language' })).toHaveTextContent('German');
  });

  test('single video falls back to English when the Settings language is missing', async () => {
    const user = userEvent.setup();
    settings.update((current) => ({
      ...current,
      outputOptions: { subtitleMode: 'sidecar', subtitleLanguages: ['es'] },
    }));
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Subtitle language' })).toHaveTextContent('English');
  });

  test('audio outputs do not carry subtitle choices', async () => {
    const user = userEvent.setup();
    ffmpeg.set({ available: true, path: '/usr/bin/ffmpeg', version: 'ffmpeg version 7', ffprobePath: '', message: '' });
    const StartDownload = vi.fn(async (_request: { options: Record<string, unknown> }) => 'job-1');
    (window as any).go.main.App.StartDownload = StartDownload;
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();

    await user.click(screen.getByRole('radio', { name: 'Subtitle file' }));
    await user.click(screen.getByRole('button', { name: 'Subtitle language' }));
    await user.click(screen.getByRole('option', { name: 'English' }));
    expect(screen.queryByText('Video only')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Audio' }));
    expect(screen.getByRole('radio', { name: 'Subtitle file' })).toBeDisabled();
    expect(screen.getByRole('radio', { name: 'Subtitle file' })).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByText(/Subtitles/)).toBeInTheDocument();
    expect(screen.getByText('Video only')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Download' }));

    await waitFor(() => expect(StartDownload).toHaveBeenCalledTimes(1));
    const request = StartDownload.mock.calls[0][0];
    expect(request.options.subtitleMode).toBe('');
    expect(request.options.subtitleLanguages).toBeUndefined();
    expect(request.options.embedMetadata).toBeUndefined();
  });
  test('locks a pending download against repeated clicks and preserves choices on failure', async () => {
    const user = userEvent.setup();
    let reject!: (error: Error) => void;
    const { StartDownload } = installBindings();
    StartDownload.mockImplementation(() => new Promise<string>((_, fail) => { reject = fail; }));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await user.dblClick(await screen.findByRole('button', { name: 'Download' }));
    expect(StartDownload).toHaveBeenCalledOnce();
    expect(screen.getByRole('button', { name: 'Adding…' })).toBeDisabled();
    expect(screen.getByLabelText('YouTube video, Short, or playlist URL')).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Audio' })).toBeDisabled();
    await act(() => reject(new Error('Destination is not writable.')));
    expect(await screen.findByRole('alert')).toHaveTextContent('Destination is not writable.');
    expect(screen.getByRole('button', { name: 'Download' })).toBeEnabled();
    expect(screen.getByText('Fixture video')).toBeInTheDocument();
  });

  test('stopping analysis ignores its late result and allows a fresh attempt', async () => {
    const user = userEvent.setup();
    let finish!: (value: ReturnType<typeof videoSummary>) => void;
    const { AnalyzeURL } = installBindings();
    AnalyzeURL.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Stop' })).toBeInTheDocument();
    expect(await screen.findByText(/Reading video details/)).toBeInTheDocument();
    expect(document.querySelector('.hud-strip')).toBeTruthy();
    expect(screen.getByText(/One video, a playlist, or up to 20 links/)).toBeInTheDocument();
    expect(screen.queryByText(/Nothing is downloaded yet/)).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Stop waiting' })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Stop' }));
    await act(() => finish(videoSummary(firstURL)));
    expect(screen.queryByText('Fixture video')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Analyze' })).toBeEnabled();
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
    expect(AnalyzeURL).toHaveBeenCalledTimes(2);
  });

  test('valid-link extraction errors offer retry and clear the previous review', async () => {
    const user = userEvent.setup();
    const { AnalyzeURL } = installBindings();
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await screen.findByText('Fixture video');
    AnalyzeURL.mockRejectedValueOnce(new Error('Check your connection and try again.'));
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Could not read this link');
    expect(screen.queryByText('Fixture video')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Try again' }));
    expect(await screen.findByText('Fixture video')).toBeInTheDocument();
  });

  test('a cleared default folder disables download and explains how to continue', async () => {
    const user = userEvent.setup();
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await screen.findByText('Fixture video');
    await act(() => settings.update((current) => ({ ...current, downloadFolder: '' })));
    expect(screen.getByRole('button', { name: 'Download' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Choose folder' })).toBeEnabled();
    expect(screen.getByText(/Choose a download folder/)).toBeInTheDocument();
  });

  test('editing a batch in flight prevents the old review from appearing', async () => {
    const user = userEvent.setup();
    let finish!: (value: unknown) => void;
    const { AnalyzeBatchURLs } = installBindings();
    AnalyzeBatchURLs.mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
    render(Home);
    const input = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.type(input, `${firstURL}{Shift>}{Enter}{/Shift}https://youtu.be/fixture0002`);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await waitFor(() => expect(AnalyzeBatchURLs).toHaveBeenCalledOnce());
    await user.clear(input);
    await user.type(input, firstURL);
    await act(() => finish({ token: 'late', expiresAt: new Date(Date.now() + 60000).toISOString(), counts: { ready: 2 }, items: [] }));
    expect(screen.queryByText('Batch review')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Download 2/ })).not.toBeInTheDocument();
  });

  test('playlist waiting uses the same Stop HUD as a single video', async () => {
    const user = userEvent.setup();
    let finish!: (value: ReturnType<typeof playlistSummary>) => void;
    const { ValidateURL, AnalyzePlaylist } = installBindings();
    const playlistURL = 'https://www.youtube.com/playlist?list=PLfixture';
    ValidateURL.mockResolvedValue({
      kind: 'playlist', url: playlistURL, playlistUrl: playlistURL, playlistId: 'PLfixture', videoUrl: '', videoId: '',
    });
    AnalyzePlaylist.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), playlistURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Stop' })).toBeInTheDocument();
    expect(await screen.findByText(/Reading playlist/)).toBeInTheDocument();
    expect(document.querySelector('.hud-strip')).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Stop' }));
    await act(() => finish(playlistSummary(playlistURL)));
    expect(screen.queryByText('Fixture playlist')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Analyze' })).toBeEnabled();
  });

  test('batch waiting uses the same Stop HUD as a single video', async () => {
    const user = userEvent.setup();
    let finish!: (value: unknown) => void;
    const { AnalyzeBatchURLs } = installBindings();
    AnalyzeBatchURLs.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByRole('button', { name: 'Stop' })).toBeInTheDocument();
    expect(await screen.findByText(/Reviewing 2 links/)).toBeInTheDocument();
    expect(document.querySelector('.hud-strip')).toBeTruthy();
    await user.click(screen.getByRole('button', { name: 'Stop' }));
    await act(() => finish({ token: 'late', expiresAt: '2099-01-01T00:00:00Z', counts: { ready: 2 }, items: [] }));
    expect(screen.queryByText('Batch of public videos')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Analyze' })).toBeEnabled();
  });

  test('a failed playlist Analyze selects the URL so the next paste replaces it', async () => {
    const user = userEvent.setup();
    const { ValidateURL, AnalyzePlaylist } = installBindings();
    const playlistURL = 'https://www.youtube.com/playlist?list=PLfixture';
    ValidateURL.mockResolvedValue({
      kind: 'playlist', url: playlistURL, playlistUrl: playlistURL, playlistId: 'PLfixture', videoUrl: '', videoId: '',
    });
    AnalyzePlaylist.mockRejectedValue(new Error('Check your connection and try again.'));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), playlistURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Could not read this link')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Paste another link' })).not.toBeInTheDocument();
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL') as HTMLTextAreaElement;
    await waitFor(() => expect(field).toHaveFocus());
    expect(field.selectionStart).toBe(0);
    expect(field.selectionEnd).toBe(field.value.length);
  });

  test('a failed batch Analyze selects the URLs so the next paste replaces them', async () => {
    const user = userEvent.setup();
    const { AnalyzeBatchURLs } = installBindings();
    AnalyzeBatchURLs.mockRejectedValue(new Error('Check your connection and try again.'));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Batch could not be reviewed')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Paste another link' })).not.toBeInTheDocument();
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL') as HTMLTextAreaElement;
    await waitFor(() => expect(field).toHaveFocus());
    expect(field.selectionStart).toBe(0);
    expect(field.selectionEnd).toBe(field.value.length);
  });

  test('opening a mixed playlist after a failed preview offers Use the video', async () => {
    const user = userEvent.setup();
    const { ValidateURL, AnalyzePlaylist } = installBindings();
    ValidateURL.mockResolvedValue({
      kind: 'video_playlist', url: firstURL, videoUrl: firstURL,
      playlistUrl: 'https://www.youtube.com/playlist?list=PLfixture', videoId: 'fixture0001', playlistId: 'PLfixture',
    });
    AnalyzePlaylist.mockRejectedValue(new Error('Connection lost'));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await screen.findByText('Preview unavailable. Choose playlist to try again.');
    await user.click(screen.getByRole('button', { name: /Full playlist/ }));
    expect(await screen.findByText('Could not read the playlist')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Use the video' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Paste another link' })).not.toBeInTheDocument();
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL') as HTMLTextAreaElement;
    await waitFor(() => expect(field).toHaveFocus());
    expect(field.selectionStart).toBe(0);
    expect(field.selectionEnd).toBe(field.value.length);
    await user.click(screen.getByRole('button', { name: 'Use the video' }));
    expect(await screen.findByRole('button', { name: 'Download' })).toBeEnabled();
  });

  test('failed playlist prefetch still lets the user choose the video', async () => {
    const user = userEvent.setup();
    const { ValidateURL, AnalyzePlaylist } = installBindings();
    ValidateURL.mockResolvedValue({ kind: 'video_playlist', url: firstURL, videoUrl: firstURL, playlistUrl: 'https://www.youtube.com/playlist?list=PLfixture', videoId: 'fixture0001', playlistId: 'PLfixture' });
    AnalyzePlaylist.mockRejectedValue(new Error('Connection lost'));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await screen.findByText('Preview unavailable. Choose playlist to try again.');
    await user.click(screen.getByRole('button', { name: /Video Fixture video/ }));
    expect(await screen.findByRole('button', { name: 'Download' })).toBeEnabled();
  });

  test('one ready batch item can continue through single-video review', async () => {
    const user = userEvent.setup();
    const { AnalyzeBatchURLs, AnalyzeURL, StartBatchDownload } = installBindings();
    AnalyzeBatchURLs.mockResolvedValue({
      token: 'one-ready', expiresAt: '2099-01-01T00:00:00Z',
      counts: { pasted: 2, ready: 1, duplicate: 0, invalid: 1, analysisFailed: 0 },
      items: [{ lineNumber: 1, input: firstURL, status: 'ready', title: 'First' }, { lineNumber: 2, input: 'invalid', status: 'invalid', message: 'Invalid link' }],
    });
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), 'one\ntwo');
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await user.click(await screen.findByRole('button', { name: 'Review ready video' }));
    expect(await screen.findByRole('button', { name: 'Download' })).toBeEnabled();
    expect(AnalyzeURL).toHaveBeenCalledWith(firstURL);
    expect(StartBatchDownload).not.toHaveBeenCalled();
  });

  test('pending playlist admission locks selection and clears its review on success', async () => {
    const user = userEvent.setup();
    const { ValidateURL, StartPlaylistDownload } = installBindings();
    const url = 'https://www.youtube.com/playlist?list=PLfixture';
    ValidateURL.mockResolvedValue({ kind: 'playlist', url, playlistUrl: url, playlistId: 'PLfixture', videoUrl: '', videoId: '' });
    let finish!: (value: { collectionId: string; admitted: number }) => void;
    StartPlaylistDownload.mockImplementation(() => new Promise((resolve) => { finish = resolve; }));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), url);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    await user.dblClick(await screen.findByRole('button', { name: 'Download 2 videos' }));
    expect(StartPlaylistDownload).toHaveBeenCalledOnce();
    expect(screen.getByRole('button', { name: 'Audio' })).toBeDisabled();
    await act(() => finish({ collectionId: 'collection-1', admitted: 2 }));
    expect(screen.queryByText('Fixture playlist')).not.toBeInTheDocument();
  });

});
