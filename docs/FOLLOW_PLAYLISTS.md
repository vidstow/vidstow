# Follow playlists

Follow remembers a public YouTube playlist and the download settings chosen on
Home. It is not a Queue collection. Collections are admitted jobs. A follow is
a watch subscription with a known-video baseline, a review list, and saved
output options.

Checks never run on launch. The user starts them with **Check now** or
**Check all**. New videos are reviewed in Following, then admitted through the
existing playlist collection path.

## Persistence

Durable records live in `follows.json` beside State v2 (`state.json`). The
queue document is not extended. Unfollow deletes only the follow record.
Queued jobs, history rows, and files on disk stay.

Schema version `1`:

| Field | Meaning |
| --- | --- |
| `id` | Stable follow id |
| `playlistId`, `sourceUrl`, `title`, `channel`, `thumbnail`, `videoCount` | Playlist identity from Analyze |
| `output.quality`, `output.audioBitrate`, `output.options`, `output.folder` | Saved download policy and playlist folder |
| `knownVideoIds` | Baseline at follow time plus every video seen later |
| `pending[]` | New videos waiting in Following review |
| `skipped[]` | Review items the user skipped; restore returns them to pending |
| `lastCheckedAt`, `lastCheckError` | Last successful check time; presentation error from a failed check |

Pending and skipped items are always in `knownVideoIds`. Admission removes an
id from pending and leaves it known so it does not show up as new again.

## API

Wails methods on `App`. The renderer does not invent child URLs. Review
actions send follow id plus video ids the backend already stored.

| Method | Role |
| --- | --- |
| `ListFollows` | Current follows plus live check progress. No network. |
| `FollowPlaylist` | Save a follow from the Home preview. `scope` is `future` or `all`. |
| `UnfollowPlaylist` | Drop the follow record only |
| `CheckFollow` / `CheckAllFollows` | Re-fetch listings serially; persist finds after each playlist |
| `StopFollowChecks` | Cancel the run; keeps finds already written |
| `AdmitFollowReview` | Map selected pending ids onto the trusted preview, then `StartPlaylistDownload` |
| `SkipFollowItems` / `RestoreFollowItems` | Move between pending and skipped |
| `UpdateFollowOutput` | Change saved quality, options, or folder for later admissions |

`FollowPlaylist` with `future` writes known ids from the current listing and
downloads nothing. `all` does the same, then admits every available entry as
one playlist collection using the follow's folder.

Event: `follows:update` with the same `FollowsView` as `ListFollows`.

## Check rules

1. App open never starts a check.
2. Check all runs one playlist at a time.
3. A check re-runs `AnalyzePlaylist` and diffs against `knownVideoIds`.
4. New available videos append to pending. New unavailable videos are recorded
   as known so they are not rediscovered forever.
5. Skipped ids stay skipped across checks.
6. Stop cancels work in flight and keeps playlists already persisted.
7. A failed check keeps earlier pending items and the last successful
   `lastCheckedAt`. The stored error is a presentation string:
   `Couldn't check for updates.`

## Review and Queue

Following review is not the Queue badge. Download from review admits a normal
playlist collection. Transfer failures stay on Queue. Skip is selected-only and
can be undone. Page size is 10: Download and Skip apply to the on-screen page;
larger leftover pending counts stay for later, with a confirm when that page
is queued.

Edit settings from Following uses the same output editor as Home. Already
queued jobs keep the settings they started with.
