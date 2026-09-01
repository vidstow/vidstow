import { render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { get } from 'svelte/store';

import Home from '../src/pages/Home.svelte';
import { pendingUrl, pendingHomeFocus, settings, modal } from '../src/lib/stores.js';

const firstURL = 'https://www.youtube.com/watch?v=fixture0001';

function videoSummary(raw: string) {
  return {
    title: 'Fixture video', channel: 'Fixture channel', duration: '1:00', thumbnail: '', videoId: 'fixture0001', url: raw,
    durationSeconds: 60, viewCount: 1, uploadDate: '', description: '', access: { code: 'public', label: 'Public' },
    plans: [
      { id: 'video', kind: 'video', label: 'Video plan', container: 'mp4', available: true, recommended: true },
      { id: 'audio', kind: 'audio', label: 'Audio plan', container: 'm4a', available: true },
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
  (window as any).go = { main: { App: { ValidateURL, AnalyzeURL, AnalyzePlaylist, AnalyzeBatchURLs, StartBatchDownload } } };
  return { ValidateURL, AnalyzeURL, AnalyzePlaylist, AnalyzeBatchURLs, StartBatchDownload };
}

describe('Home analysis authority', () => {
  test('empty Home shows the field shortcut and dotted try chips', async () => {
    render(Home);
    const kbd = document.querySelector('.fieldwrap .kbd');
    expect(kbd).toBeTruthy();
    expect(kbd?.textContent).toMatch(/L|Ctrl\+L/);
    await userEvent.setup().click(screen.getByRole('button', { name: 'a video' }));
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

  test('pending home focus selects the field', async () => {
    render(Home);
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL');
    pendingHomeFocus.set(true);
    await waitFor(() => expect(field).toHaveFocus());
  });

  beforeEach(() => {
    pendingUrl.set('');
    pendingHomeFocus.set(false);
    modal.set(null);
    settings.update((current) => ({ ...current, downloadFolder: '/tmp/downloads', confirmBeforeDownload: false }));
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
    expect(screen.queryByText('First video')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /2 episodes/ }));
    expect(await screen.findByText('First video')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'All' })).toBeInTheDocument();
    expect(screen.getByLabelText('Range start')).toBeInTheDocument();
    expect(screen.queryByLabelText('Search playlist')).not.toBeInTheDocument();
    expect(screen.getByText('2 selected · up to 1080p')).toBeInTheDocument();
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
    await user.click(start);
    await waitFor(() => expect(StartBatchDownload).toHaveBeenCalledWith({ token: 'batch-token', quality: '1080p', audioBitrate: 0 }));
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
  });

  test('switching output types selects a visible compatible plan', async () => {
    const user = userEvent.setup();
    render(Home);

    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Video plan')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Audio' }));
    expect(screen.getByText('Audio plan')).toBeInTheDocument();
    expect(screen.queryByText('Video plan')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Download' })).toBeEnabled();
  });

  test('a failed Analyze offers Paste another link instead of only a dead modal', async () => {
    const user = userEvent.setup();
    const { ValidateURL } = installBindings();
    ValidateURL.mockRejectedValue(new Error('This video is private.'));
    render(Home);
    await user.type(screen.getByLabelText('YouTube video, Short, or playlist URL'), firstURL);
    await user.click(screen.getByRole('button', { name: 'Analyze' }));
    expect(await screen.findByText('Unsupported URL')).toBeInTheDocument();
    expect(screen.getByText('This video is private.')).toBeInTheDocument();
    const recover = screen.getByRole('button', { name: 'Paste another link' });
    const field = screen.getByLabelText('YouTube video, Short, or playlist URL');
    await user.click(recover);
    await waitFor(() => expect(screen.queryByText('Unsupported URL')).not.toBeInTheDocument());
    expect(field).toHaveFocus();
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
});
