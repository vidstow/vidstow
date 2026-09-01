# Recovery cases: product decisions (1 Sep 2026)

**Status:** Product decisions from a case-by-case review with Tejas. FYI for other agents. Do not treat **RECOVERY.md** or the p2 unfinished-downloads card as the live spec where they disagree with this note.

**Not an implementation order.** This is what the product should do. Build it only when asked, in the playlist-dock app worktree (`feat/p2-playlist-dock`), not in the dirty original checkout.

**Assumption (stated, not asked):** If the user **Pause**s a job, then Quits or the app crashes, that job stays paused. It was not downloading, so it does not start by itself.

---

## What we were deciding

Today, a crash or Quit mid-download restores jobs as paused. Settings says **Interrupted jobs · Restored as paused**. Hard cases open **Action required** (`This download needs your decision`) with **Keep for now**, **Discard saved data**, **Try recovery again**, **Retry with fresh link**, **Start over from Home**.

p2 and **RECOVERY.md** (design review, same day) proposed an **unfinished downloads** card (**Resume all** / **Leave for now** / **Discard**), a quiet amber **waiting for the drive** row, and dropping that dialog. Tejas answered cases one by one. The card is out. Auto-resume of the active job is in. Missing folder is an error, not a quiet wait.

---

## Decisions

### Relaunch (crash, force-quit, or VidStow → Quit)

The job that was **actually downloading** starts by itself. No unfinished card. No paused row waiting for **Resume**. **Confirm before starting downloads** does not appear. The job was already admitted.

Other jobs that were **waiting** stay waiting. Slots and concurrency still apply. No stampede.

A **playlist** (or batch) follows the same rule: the episode that was downloading starts; the rest stay waiting in that collection.

An **app update** with an in-progress queue is a normal relaunch. No migration screen. No “analyze again on Home” for those jobs.

This replaces **Restored as paused** for the in-progress job. Waiting and paused jobs are not that job.

### Save folder gone (USB unplugged, volume missing)

**Queue** shows an **error**, not a quiet wait and not **Action required**. The user must either bring that path back or **Change** the folder.

When the **same path** comes back, the job continues by itself.

If they **Change** to a new folder, the job continues there from where it left off when that is possible. It does not start over unless leftover data cannot be used.

### Two copies of VidStow

The second launch does not open another window. It says VidStow is already running.

### Cannot write (disk full, folder not allowed)

Failed **Queue** row with a clear error. **Retry** or change folder from the inspector. Not a recovery screen.

### Finished file deleted in Finder

**Downloads** keeps the row as a receipt. Open / Show may say the file is no longer at that path. No **File missing** chip, no dimmed library inventory, no recovery attached. VidStow’s job ended when the file was saved.

### Saved queue file unreadable

A small notice: files on disk are safe; the queue list was reset. **Rescan** / **Copy diagnostics** as in the design note. Do not freeze the app behind today’s recovery shell.

### Leftover data that cannot continue (damaged, unreadable)

Paused **Queue** row. Inspector offers **Resume** (fresh attempt) or **Discard**. **Discard** asks once and names the size (`delete 843 MB?`), then removes leftover files. Do not throw those files away on launch. Do not show the unfinished card.

---

## What this replaces

| Source | Out |
| --- | --- |
| Settings / **PRODUCT.md** | **Restored as paused** as the rule for a job that was downloading |
| **RECOVERY.md** / p2 Queue | Unfinished-downloads card; **Resume all** / **Leave for now** as the everyday crash path |
| **RECOVERY.md** | Quiet amber “waiting for the drive” with no error language |
| **RECOVERY.md** | Silent discard-and-restart when leftover data is damaged |
| **RECOVERY.md** vs p2 copy fight | **Discard all** on the card (the card itself is out) |
| App **Downloads** | **File missing** chip and existence-check as recovery |

| Source | Keep |
| --- | --- |
| App today | Write to a hidden folder next to the save path; move into the real name only if that name is free. Never overwrite a file the user already has. |
| App today | Checkpoint resume when leftover data is usable |
| App today | Master-detail **Queue**; failed rows; **Retry** on ordinary failures |
| This review | Never delete leftover files without one confirm that names the size |
| This review | Launch still does not start **waiting** or **paused** jobs by itself |

**Action required** as a catch-all quiz is not the everyday path. Do not rip the dialog and its codes in the same change as auto-resume unless a case above has nowhere else to go. Everyday relaunch, missing folder, disk full, receipts, and unusable leftovers already have a window in this note.

---

## Changes to make (when implementing)

Work in `/Users/tejas/projects/vidstow-p2-playlist-dock` on **feat/p2-playlist-dock**. Do not mix **p2-shell** or the original **feat/landing-page** checkout. Product still targets macOS, Linux, and Windows.

### Queue and startup

- On startup, jobs whose lifecycle was **active downloading** (or equivalent in-progress transfer) become running again, up to concurrency. Do not wait for **Resume**.
- Jobs that were **pending / waiting** stay waiting.
- Jobs that were **paused** stay paused (assumption above).
- Skip the confirm-before-download dialog on this relaunch path.
- Playlist and batch children use the same rule as standalone jobs.
- Settings copy must stop saying every interrupted job is restored paused. Say what is true: the in-progress download continues; waiting stays waiting; paused stays paused.

### Missing folder

- Surface as a **Queue** error on that job: bring the path back, or **Change**.
- Watch the original path; when it returns, continue without a click.
- **Change** retargets the job and continues from leftover data when possible.

### Single instance

- Second process does not start a second queue. Tell the user VidStow is already running (bring the existing window forward if the platform allows, but the decision was: no second copy).

### Ordinary write failure

- Disk full and permission denied stay **failed** with **Retry** / change folder. Do not reuse the missing-folder error.

### Downloads receipts

- Stop treating a missing file as a recovery state. Drop the **File missing** chip and the “inventory” behavior. Open / Show may fail plainly (`no longer at that path`). **Remove from history** still only forgets the receipt.

### Unusable leftover data

- Paused row, not auto-start, not a top-of-list card.
- Inspector **Discard** → one confirm with size → then delete leftover files.
- Inspector **Resume** → start again (from checkpoint if it becomes usable, otherwise a fresh attempt).

### Dead queue file

- Tiny notice, files-on-disk-are-safe, list reset. Align copy with **RECOVERY.md**’s journal notice, not **RecoveryRequiredShell**.

### Do not

- Do not ship the p2 unfinished card as the crash/Quit path.
- Do not auto-start waiting jobs or the rest of a playlist.
- Do not auto-delete leftover files on launch.
- Do not overwrite a finished file that already exists at the destination name.
- Do not implement this in **youtube_dlp** or by publishing a local `go.mod` replace.
- Do not add Cursor attribution on commits or PRs.

### Suggested slice order (if building later)

1. Single-instance lock.
2. Startup: resume the in-progress job; leave waiting/paused; skip confirm; Settings copy.
3. Missing-folder error, auto-continue when the path returns, **Change** and continue.
4. Unusable leftover: paused + sized Discard confirm.
5. Downloads receipt (drop **File missing**).
6. Dead queue-file notice.

---

## Context the other agent may need

- Canonical p2 mock: `penpot_designs/jpg-workstation/p2.html` (vidstow-design). Queue still *draws* the unfinished card; that draw is not the product decision.
- Older design writeup: `penpot_designs/jpg-workstation/RECOVERY.md`. Principle (do not overwrite; delete only with a confirm) still holds. Everyday UI in that file does not.
- Shipped Queue UI: `frontend/src/lib/lifecycle-ui/QueueOverview.svelte`, **Action required** in `ActionRequiredReviewDialog.svelte`, Settings in `QueueSettingsCard.svelte`.
- Voice for user-facing copy: window first, named buttons, no engine glossary. Project rules: `.cursorrules` and `.cursor/rules/adaptive-hybrid-persona.mdc`.
