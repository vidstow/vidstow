import { render, screen, waitFor } from '@testing-library/svelte';
import '@testing-library/jest-dom/vitest';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, test, vi } from 'vitest';

import Following from '../src/pages/Following.svelte';
import { follows, modal, route } from '../src/lib/stores.js';
import { get } from 'svelte/store';
import type { FollowRecord, FollowsView } from '../src/lib/types.js';

function followRecord(overrides: Partial<FollowRecord> = {}): FollowRecord {
  return {
    id: 'fol-1',
    playlistId: 'PLfixture',
    sourceUrl: 'https://www.youtube.com/playlist?list=PLfixture',
    title: 'Fixture playlist',
    channel: 'Fixture channel',
    videoCount: 2,
    output: { quality: '1080p', folder: '/tmp/downloads/Fixture playlist' },
    knownVideoIds: ['fixture0001'],
    pending: [],
    createdAt: '2026-09-14T00:00:00Z',
    updatedAt: '2026-09-14T00:00:00Z',
    checkState: 'idle',
    ...overrides,
  };
}

function view(partial: Partial<FollowsView> = {}): FollowsView {
  return { follows: [], checking: false, checkDone: 0, checkTotal: 0, ...partial };
}

function installBindings() {
  const ListFollows = vi.fn(async () => view());
  const CheckFollow = vi.fn(async () => view());
  const CheckAllFollows = vi.fn(async () => view());
  const StopFollowChecks = vi.fn(async () => view());
  const AdmitFollowReview = vi.fn(async () => view());
  const SkipFollowItems = vi.fn(async () => view());
  const RestoreFollowItems = vi.fn(async () => view());
  const UnfollowPlaylist = vi.fn(async () => view());
  const UpdateFollowOutput = vi.fn(async () => view());
  (window as any).go = {
    main: {
      App: {
        ListFollows, CheckFollow, CheckAllFollows, StopFollowChecks,
        AdmitFollowReview, SkipFollowItems, RestoreFollowItems, UnfollowPlaylist, UpdateFollowOutput,
        PickDownloadFolder: vi.fn(async () => ''),
      },
    },
  };
  return { CheckFollow, CheckAllFollows, StopFollowChecks, AdmitFollowReview, SkipFollowItems, RestoreFollowItems, UnfollowPlaylist };
}

describe('Following page', () => {
  beforeEach(() => {
    modal.set(null);
    route.set('following');
    follows.set(view());
    installBindings();
  });

  test('empty Following shows the rail copy and Go to Home', async () => {
    const onGoto = vi.fn();
    render(Following, { events: { goto: onGoto } });
    expect(screen.getByRole('heading', { name: 'Following' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'You are not following any playlists.' })).toBeInTheDocument();
    expect(screen.getByText(/Head to Home, analyze a public playlist/)).toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole('button', { name: 'Go to Home' }));
    expect(onGoto.mock.calls.some((call) => call[0]?.detail === 'home')).toBe(true);
  });

  test('Stop checking can run while Check all is still in flight', async () => {
    let finish!: (value: FollowsView) => void;
    const hanging = new Promise<FollowsView>((resolve) => { finish = resolve; });
    const { CheckAllFollows, StopFollowChecks } = installBindings();
    CheckAllFollows.mockImplementation(() => hanging);
    follows.set(view({ follows: [followRecord()] }));
    render(Following);
    await userEvent.setup().click(screen.getByRole('button', { name: 'Check all' }));
    await waitFor(() => expect(CheckAllFollows).toHaveBeenCalledOnce());
    follows.set(view({ follows: [followRecord()], checking: true, checkDone: 0, checkTotal: 1 }));
    await userEvent.setup().click(await screen.findByRole('button', { name: 'Stop checking' }));
    await waitFor(() => expect(StopFollowChecks).toHaveBeenCalledOnce());
    finish(view({ follows: [followRecord()], checking: false }));
  });

  test('Check all is available and launch does not check', async () => {
    const { CheckAllFollows, CheckFollow } = installBindings();
    follows.set(view({ follows: [followRecord()] }));
    render(Following);
    expect(CheckAllFollows).not.toHaveBeenCalled();
    expect(CheckFollow).not.toHaveBeenCalled();
    await userEvent.setup().click(screen.getByRole('button', { name: 'Check all' }));
    await waitFor(() => expect(CheckAllFollows).toHaveBeenCalledOnce());
  });

  test('review admits selected videos through the follow API', async () => {
    const { AdmitFollowReview } = installBindings();
    follows.set(view({
      follows: [followRecord({
        pending: [
          { videoId: 'fixture0002', url: 'https://www.youtube.com/watch?v=fixture0002', title: 'New video', available: true, index: 2, duration: '1:00' },
        ],
      })],
    }));
    render(Following);
    await userEvent.setup().click(screen.getByRole('button', { name: 'Review 1 new' }));
    expect(screen.getByText('New video')).toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole('button', { name: 'Download 1 video' }));
    await waitFor(() => expect(AdmitFollowReview).toHaveBeenCalledWith({
      followId: 'fol-1', videoIds: ['fixture0002'],
    }));
  });

  test('Unfollow asks for confirmation and does not mention deleting files', async () => {
    const { UnfollowPlaylist } = installBindings();
    follows.set(view({ follows: [followRecord({ pending: [{ videoId: 'fixture0002', url: 'https://www.youtube.com/watch?v=fixture0002', title: 'New video', available: true, index: 2 }] })] }));
    render(Following);
    await userEvent.setup().click(screen.getByRole('button', { name: 'Manage Fixture playlist' }));
    await userEvent.setup().click(screen.getByRole('button', { name: 'Unfollow playlist…' }));
    const confirm = get(modal);
    expect(confirm?.title).toBe('Unfollow Fixture playlist?');
    expect(confirm?.message).toMatch(/downloaded files and anything already queued stay/i);
    expect(UnfollowPlaylist).not.toHaveBeenCalled();
  });
});
