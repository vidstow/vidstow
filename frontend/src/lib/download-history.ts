import type { HistoryEntry } from './types.js';
import { formatBytes, formatRelative, qualityLabel } from './format.js';

export type HistoryDayKey = 'today' | 'yesterday' | 'earlier';

export type HistoryCollectionFields = {
  playlistId?: string;
  collectionId?: string;
  playlistTitle?: string;
  collectionTitle?: string;
  collectionIndex?: number;
};

export type HistoryRecord = HistoryEntry & HistoryCollectionFields;

export type HistoryItemRow = {
  kind: 'item';
  entry: HistoryRecord;
};

export type HistoryCollectionRow = {
  kind: 'collection';
  id: string;
  title: string;
  thumbnail: string;
  format: string;
  sizeLabel: string;
  entries: HistoryRecord[];
};

export type HistoryRow = HistoryItemRow | HistoryCollectionRow;

export type HistorySection = {
  key: HistoryDayKey;
  label: string;
  rows: HistoryRow[];
};

const DAY_LABELS: Array<{ key: HistoryDayKey; label: string }> = [
  { key: 'today', label: 'Today' },
  { key: 'yesterday', label: 'Yesterday' },
  { key: 'earlier', label: 'Earlier' },
];

export function collectionIdentity(entry: HistoryRecord): string {
  const playlistId = entry.playlistId?.trim() ?? '';
  if (playlistId) return `playlist:${playlistId}`;
  const collectionId = entry.collectionId?.trim() ?? '';
  if (collectionId) return `collection:${collectionId}`;
  return '';
}

export function historyDayKey(iso: string, now = new Date()): HistoryDayKey {
  const completed = Date.parse(iso);
  if (!Number.isFinite(completed)) return 'earlier';
  const startToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  if (completed >= startToday) return 'today';
  const startYesterday = startToday - 86_400_000;
  if (completed >= startYesterday) return 'yesterday';
  return 'earlier';
}

export function formatHistoryLabel(entry: HistoryRecord): string {
  const quality = qualityLabel(entry.quality);
  return entry.container ? `${quality} · ${entry.container}` : quality;
}

export function historySubtitle(entry: HistoryRecord, now = Date.now()): string {
  const parts = [formatHistoryLabel(entry)];
  if (entry.sizeBytes) parts.push(formatBytes(entry.sizeBytes));
  parts.push(entry.channel || 'YouTube');
  const relative = formatRelative(entry.completedAt, now);
  if (relative) parts.push(relative);
  return parts.join(' · ');
}

export function episodeSubtitle(entry: HistoryRecord, now = Date.now()): string {
  const parts: string[] = [];
  if (entry.durationLabel) parts.push(entry.durationLabel);
  if (entry.sizeBytes) parts.push(formatBytes(entry.sizeBytes));
  const relative = formatRelative(entry.completedAt, now);
  if (relative) parts.push(relative);
  return parts.join(' · ');
}

export function thumbnailFor(entry: HistoryRecord): string {
  if (entry.thumbnail) return entry.thumbnail;
  return entry.videoId
    ? `https://i.ytimg.com/vi/${encodeURIComponent(entry.videoId)}/hqdefault.jpg`
    : '';
}

export function episodeIndex(entry: HistoryRecord, fallback: number): number {
  return Number.isInteger(entry.collectionIndex) && (entry.collectionIndex ?? 0) > 0
    ? entry.collectionIndex as number
    : fallback;
}

export function episodeLabel(index: number): string {
  return `EP ${String(index).padStart(2, '0')}`;
}

function haystack(entry: HistoryRecord): string {
  return [entry.title, entry.channel, entry.filename, entry.quality, entry.container || '', entry.playlistTitle, entry.collectionTitle]
    .join(' ')
    .toLowerCase();
}

function matchesQuery(entry: HistoryRecord, needle: string): boolean {
  return !needle || haystack(entry).includes(needle);
}

function collectionTitle(entries: HistoryRecord[]): string {
  return entries.find((entry) => entry.playlistTitle?.trim())?.playlistTitle?.trim()
    || entries.find((entry) => entry.collectionTitle?.trim())?.collectionTitle?.trim()
    || entries[0]?.title
    || 'Playlist';
}

function collectionFormat(entries: HistoryRecord[]): string {
  const labels = new Set(entries.map((entry) => qualityLabel(entry.quality)).filter(Boolean));
  if (labels.size === 1) return [...labels][0];
  return 'Mixed';
}

function collectionThumbnail(entries: HistoryRecord[]): string {
  return entries.map(thumbnailFor).find(Boolean) || '';
}

function sortEpisodes(entries: HistoryRecord[]): HistoryRecord[] {
  return entries.slice().sort((a, b) => {
    const aIndex = a.collectionIndex ?? Number.MAX_SAFE_INTEGER;
    const bIndex = b.collectionIndex ?? Number.MAX_SAFE_INTEGER;
    if (aIndex !== bIndex) return aIndex - bIndex;
    return 0;
  });
}

function rowsForDay(entries: HistoryRecord[], needle: string): HistoryRow[] {
  const groups = new Map<string, HistoryRecord[]>();
  const singles: HistoryRecord[] = [];

  for (const entry of entries) {
    const id = collectionIdentity(entry);
    if (!id) {
      singles.push(entry);
      continue;
    }
    const bucket = groups.get(id) ?? [];
    bucket.push(entry);
    groups.set(id, bucket);
  }

  const rows: HistoryRow[] = [];
  for (const [id, members] of groups) {
    const title = collectionTitle(members);
    const visible = members.filter((entry) => matchesQuery(entry, needle) || title.toLowerCase().includes(needle));
    if (!visible.length) continue;
    const ordered = sortEpisodes(visible);
    rows.push({
      kind: 'collection',
      id,
      title,
      thumbnail: collectionThumbnail(ordered),
      format: collectionFormat(ordered),
      sizeLabel: formatBytes(ordered.reduce((sum, entry) => sum + (entry.sizeBytes || 0), 0)),
      entries: ordered,
    });
  }

  for (const entry of singles) {
    if (!matchesQuery(entry, needle)) continue;
    rows.push({ kind: 'item', entry });
  }

  return rows;
}

export function buildHistorySections(entries: HistoryRecord[], query = '', now = new Date()): HistorySection[] {
  const needle = query.trim().toLowerCase();
  const buckets: Record<HistoryDayKey, HistoryRecord[]> = { today: [], yesterday: [], earlier: [] };
  for (const entry of entries) {
    buckets[historyDayKey(entry.completedAt, now)].push(entry);
  }
  return DAY_LABELS
    .map(({ key, label }) => ({ key, label, rows: rowsForDay(buckets[key], needle) }))
    .filter((section) => section.rows.length > 0);
}
