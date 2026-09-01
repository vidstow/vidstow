# Recovery, redesigned: the disk is the referee

**Status:** agreed direction (design review, 2026-09-01)
**Scope:** replaces the two-tier Recovery UI (RecoveryRequiredShell +
ActionRequiredReviewDialog) with a stateless-leaning escalation policy.

---

## The principle

VidStow already does the physically safe thing: downloads are staged in a
hidden temp folder **beside the destination** and published with an
**atomic no-replace rename**. A half-written video can never appear in
`~/Movies`.

All remaining recovery complexity comes from one place: durable records
(the "diary") that can disagree with the disk after a crash. Today the app
stops and asks the user to arbitrate that disagreement.

> **The disk is the referee. When the diary and the disk disagree, the app
> resolves it itself — safely — and never asks the user.**

Safe resolution rules, in order:

1. **Never overwrite** a file the user (or anything else) created.
2. **Never delete** anything without one explicit confirmation.
3. **When unsure whether something finished: assume it didn't** and offer
   resume. Worst case is a re-download, never corruption or duplication.
4. **Never block** on a question the app can answer safely later
   (missing drive, another instance, cleanup).

## The eight bad days

| # | Bad day | Stateless handling |
|---|---------|--------------------|
| 1 | Crash / force-quit mid-download | Restart scan finds temp data → **Resume or Discard** prompt |
| 2 | Quit and reopen mid-download | Same as 1 |
| 3 | Destination drive unplugged | Row state **"Waiting for the drive…"**; resumes on reconnect |
| 4 | Two copies of the app | **Single-instance lock**; second launch says "already running" |
| 5 | Queue/settings file corrupt | "Your files are safe on disk. The queue list was reset." Rebuild from scan. |
| 6 | User deletes files in Finder | Not our data to reconcile. Next scan reflects reality. |
| 7 | App update, old saved data | Journal entries are forward-compatible (URL, format, folder). Temp data gets Resume-or-Discard. |
| 8 | Disk full / permission denied | Ordinary row error. Not a recovery concept. |

## What stays, what goes

**Keep**

- Journal of intent: URL, format, destination, status, history — one JSON
  file, survives restarts, forward-compatible fields only.
- Temp-beside-destination staging + atomic rename (already built; this is
  what makes the disk the referee).
- Checkpoint resume (already built; the temp folder remembers progress).
- Single-instance lock (new, small).

**Drop**

- Action-required review cards and their capability matrices
  (`CanStartOver` / `CanRetryRecovery` / `CanRetryFreshLink` / …).
  Conservative auto-resolution replaces every one of them (table below).
- Session leases as a user-facing concept (single-instance + in-process
  serialization cover it).
- Publication records as a user-facing concept (ownership is derivable:
  file exists in destination → done; artifact in temp → not done).
- Startup reconciliation pass as a UI event. It becomes a silent scan
  that feeds the Resume-or-Discard prompt and row states.

## Auto-resolution table (for the real implementation)

| Old action-required code | Automatic resolution |
|--------------------------|----------------------|
| publication / reconciliation family | Try no-replace re-publish → collision means already published → mark completed |
| session-lease-contended | Single-instance makes this impossible; belt-and-braces: row waits and re-checks |
| output-root-unavailable | Row waits for the volume; resumes on reconnect |
| session-manifest-corrupt / version-unknown | Discard session, restart download automatically |
| migration-reanalysis-required | Journal has the URL → auto re-analyze; row is a normal retry |
| workspace-cleanup-required | Background cleanup retries; row completes meanwhile (tombstone) |
| cleanup-preserved rows | Never block removal on UI; cleanup keeps retrying silently |

## The residual UI (what we design)

1. **Unfinished downloads prompt** — appears once, after a restart that
   found temp data:

   ```
   2 unfinished downloads from last time
   [thumb] Rust vs Go Architecture — 843 MB staged
   [thumb] Concurrency in Go — EP 3 of 6 — 210 MB staged
                                  [Discard all]  [Resume all]  ▸ per-item
   ```

   - Discard asks once, listing exactly what will be deleted.
   - Resume picks up from checkpoints where possible; restarts otherwise.
   - Dismissible ("leave them for now" — items stay listed in Queue).

2. **Waiting for the drive** — a queue row state, not a screen:

   ```
   [thumb] Concurrency in Go — EP 5        Waiting for “Sandisk 64GB”…
   ```

   Amber accent, no error language, auto-resumes. Same state is reused for
   "waiting for the other VidStow to close" (belt-and-braces path).

3. **Corrupt journal notice** — the only true takeover, and it's tiny:

   ```
   Your downloads are safe on disk.
   VidStow couldn't read its saved queue, so the list was reset.
   [Rescan download folders]      [Copy diagnostics]
   ```

   No frozen queue behind it; nothing to arbitrate.

## The promise

> VidStow may re-download when it's unsure. It never overwrites your
> files, never deletes without asking once, and never shows you a screen
> you can't understand.

**Honest cost:** a crash mid-download can lose that download's progress
(if you discard, or the staged data is unusable). In exchange the app
never asks the user to be its auditor, and every claim it makes is
verifiable in Finder.

## Engine notes (what actually changes in code)

- `ytdlp-go` session/lease/publication machinery stays as engine
  internals — it is correct and load-bearing for checkpoint resume. What
  changes is the **escalation policy in vidstow**: `actionRequiredReview`
  and its capability gates are replaced by the auto-resolution table.
- `recovery.Reconcile` degrades from a UI-event producer to a silent
  classifier feeding the prompt and row states.
- Add a single-instance lock at app start (Wails v3 single-instance or a
  flock at the data-dir).
- Journal keeps only intent + history; drop session/publication fields
  from the durable schema over time.
