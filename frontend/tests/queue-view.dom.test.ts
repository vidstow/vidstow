import { describe, expect, test } from 'vitest';
import { newestQueueView, queueDisplayItems } from '../src/lib/queue-view.js';
import type {
  DurableLifecycle,
  LifecycleJobViewModel,
  QueueCollectionViewModel,
  QueueView,
} from '../src/lib/lifecycle-ui/types.js';

function view(revision: number): QueueView {
  return {
    revision,
    rows: [],
    summary: { totalJobs: 0, runningJobs: 0, occupiedSlots: 0, slotLimit: 2, waitingJobs: 0, pausedJobs: 0 },
    capabilities: {},
    persistence: { available: true, healthy: true },
  };
}

function job(id: string, lifecycle: DurableLifecycle, collectionId?: string): LifecycleJobViewModel {
  return { id, title: id, lifecycle, collectionId, occupiesSlot: lifecycle === 'active' };
}

function collection(id: string, childJobIds: string[]): QueueCollectionViewModel {
  return {
    id,
    kind: 'playlist',
    title: id,
    policy: 'video:1080p',
    childJobIds,
    total: childJobIds.length,
    completed: 0,
    failed: 0,
    canceled: 0,
    active: 0,
    pending: 0,
    paused: 0,
    progress: 0,
    progressLabel: `0 of ${childJobIds.length} complete`,
  };
}

describe('live QueueView ordering', () => {
  test('does not regress progress/authority for delayed or duplicate bridge events', () => {
    const current = view(4);
    expect(newestQueueView(current, view(3))).toBe(current);
    expect(newestQueueView(current, view(4))?.revision).toBe(4);
    expect(newestQueueView(current, view(5))?.revision).toBe(5);
  });

  test('keeps an event-newer view when an older imperative refresh resolves afterwards', () => {
    const eventView = view(12);
    const staleRefresh = view(11);
    expect(newestQueueView(eventView, staleRefresh)).toBe(eventView);
  });
});

describe('queue display grouping', () => {
  test('keeps an active standalone video above a completed collection', () => {
    const completedCollection = collection('completed-collection', ['completed-child']);
    const items = queueDisplayItems([
      job('active-video', 'active'),
      job('completed-child', 'completed', completedCollection.id),
    ], [completedCollection]);

    expect(items.map((item) => item.key)).toEqual([
      'job:active-video',
      'collection:completed-collection',
    ]);
  });

  test('keeps a queued standalone video above a completed collection', () => {
    const completedCollection = collection('completed-collection', ['completed-child']);
    const items = queueDisplayItems([
      job('queued-video', 'pending'),
      job('completed-child', 'completed', completedCollection.id),
    ], [completedCollection]);

    expect(items.map((item) => item.key)).toEqual([
      'job:queued-video',
      'collection:completed-collection',
    ]);
  });

  test('keeps an active collection above a completed standalone video', () => {
    const activeCollection = collection('active-collection', ['active-child']);
    const items = queueDisplayItems([
      job('active-child', 'active', activeCollection.id),
      job('completed-video', 'completed'),
    ], [activeCollection]);

    expect(items.map((item) => item.key)).toEqual([
      'collection:active-collection',
      'job:completed-video',
    ]);
  });

  test('emits a mixed collection once and keeps its children in backend priority order', () => {
    const mixedCollection = collection('mixed-collection', ['completed-child', 'active-child']);
    const items = queueDisplayItems([
      job('active-child', 'active', mixedCollection.id),
      job('queued-video', 'pending'),
      job('completed-child', 'completed', mixedCollection.id),
    ], [mixedCollection]);

    expect(items.map((item) => item.key)).toEqual([
      'collection:mixed-collection',
      'job:queued-video',
    ]);
    const grouped = items[0];
    expect(grouped.kind).toBe('collection');
    if (grouped.kind === 'collection') {
      expect(grouped.children.map((child) => child.id)).toEqual(['active-child', 'completed-child']);
    }
  });

  test('retains a collection with no live child rows as a final recovery item', () => {
    const orphan = collection('orphan-collection', ['missing-child']);
    const items = queueDisplayItems([job('active-video', 'active')], [orphan]);

    expect(items.map((item) => item.key)).toEqual(['job:active-video', 'collection:orphan-collection']);
  });
});
