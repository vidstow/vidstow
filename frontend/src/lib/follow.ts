import type { FollowRecord, FollowsView, Quality } from './types.js';

export const REVIEW_PAGE_SIZE = 10;

export const emptyFollowsView = (): FollowsView => ({
  follows: [], checking: false, checkDone: 0, checkTotal: 0,
});

export function pendingCount(view: FollowsView): number {
  return view.follows.reduce((sum, follow) => sum + (follow.pending?.length ?? 0), 0);
}

export function word(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`;
}

export function followOutputSummary(follow: FollowRecord): string {
  const quality = follow.output?.quality as Quality | undefined;
  if (quality === 'audio') {
    return follow.output?.audioBitrate ? `MP3 ${follow.output.audioBitrate}` : 'Original audio';
  }
  const labels: Record<string, string> = {
    best: 'Best available', '4k': '4K', '1440p': '1440p', '1080p': '1080p', '720p': '720p',
  };
  return quality ? `Up to ${labels[quality] ?? quality}` : 'Saved download settings';
}

export function formatLastChecked(iso?: string): string {
  if (!iso) return 'Not checked yet';
  const then = Date.parse(iso);
  if (!Number.isFinite(then)) return 'Not checked yet';
  const delta = Date.now() - then;
  if (delta < 45_000) return 'Just now';
  if (delta < 90_000) return '1 minute ago';
  if (delta < 3_600_000) return `${Math.round(delta / 60_000)} minutes ago`;
  if (delta < 48 * 3_600_000) return `${Math.round(delta / 3_600_000)} hours ago`;
  return new Date(then).toLocaleString();
}

export function applyFollowsView(current: FollowsView | null | undefined, next: FollowsView | null | undefined): FollowsView {
  if (!next) return current ?? emptyFollowsView();
  return {
    follows: next.follows ?? [],
    checking: !!next.checking,
    checkDone: next.checkDone ?? 0,
    checkTotal: next.checkTotal ?? 0,
    currentFollowId: next.currentFollowId,
  };
}
