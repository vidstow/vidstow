import type {
  LifecycleJobViewModel,
  QueueCollectionViewModel,
  QueueView,
} from './lifecycle-ui/types.js';

export type QueueDisplayItem =
  | {
      kind: 'collection';
      key: string;
      collection: QueueCollectionViewModel;
      children: LifecycleJobViewModel[];
    }
  | {
      kind: 'job';
      key: string;
      job: LifecycleJobViewModel;
    };

// Preserve the backend-authored job priority while grouping collection children.
// A collection is inserted where its highest-priority child first appears, so
// active or pending collections cannot be displaced by completed standalone
// jobs (and completed collections cannot displace active standalone jobs).
export function queueDisplayItems(
  jobs: LifecycleJobViewModel[],
  collections: QueueCollectionViewModel[] = [],
): QueueDisplayItem[] {
  const collectionById = new Map(collections.map((collection) => [collection.id, collection]));
  const childrenByCollection = new Map<string, LifecycleJobViewModel[]>();

  for (const job of jobs) {
    if (!job.collectionId || !collectionById.has(job.collectionId)) continue;
    const children = childrenByCollection.get(job.collectionId) ?? [];
    children.push(job);
    childrenByCollection.set(job.collectionId, children);
  }

  const emittedCollections = new Set<string>();
  const items: QueueDisplayItem[] = [];
  for (const job of jobs) {
    const collection = job.collectionId ? collectionById.get(job.collectionId) : undefined;
    if (!collection) {
      items.push({ kind: 'job', key: `job:${job.id}`, job });
      continue;
    }
    if (emittedCollections.has(collection.id)) continue;
    emittedCollections.add(collection.id);
    items.push({
      kind: 'collection',
      key: `collection:${collection.id}`,
      collection,
      children: childrenByCollection.get(collection.id) ?? [],
    });
  }

  // A collection without a live child row should be unusual, but retaining it
  // at the end is safer than silently hiding backend-authored recovery data.
  for (const collection of collections) {
    if (emittedCollections.has(collection.id)) continue;
    items.push({
      kind: 'collection',
      key: `collection:${collection.id}`,
      collection,
      children: childrenByCollection.get(collection.id) ?? [],
    });
  }

  return items;
}

// Queue events are ordered by the backend, but bridge delivery can still
// duplicate or delay a payload during startup. Never let an older snapshot
// overwrite newer progress or authority in the live UI.
export function newestQueueView(current: QueueView | null, next: QueueView): QueueView | null {
  if (current && Number.isFinite(current.revision) && Number.isFinite(next?.revision) && next.revision < current.revision) {
    return current;
  }
  return next;
}
