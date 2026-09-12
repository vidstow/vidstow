# VidStow Workstation design

This document describes the shipped desktop chrome and interaction design for
VidStow after the zinc workstation landed on `main` (#92 / #93). It is a design
contract for coding agents and UI work, not a roadmap. If a behavior is not in
`PRODUCT.md`, do not invent it here either.

Tokens live in `frontend/src/styles/global.css`. When this file and the CSS
disagree, trust the CSS and update this document.

## Purpose

VidStow is a dense, quiet desktop downloader for public YouTube videos, Shorts,
playlists, and small batches. The shell should feel like a professional utility,
not a marketing site and not a consumer media player.

One zinc-dark canvas, one blue accent, Geist for UI and Geist Mono for paths and
occupancy. Complexity stays in the engine and queue. The UI shows decisions a
person can make, and only the actions the backend currently authorizes.

## Layout and chrome

Default window 1180×760. Minimum 960×600. Layout, top to bottom:

1. Native title bar. Window title is always `VidStow`. Progress stays in Queue,
   not the title.
2. Workstation row: 160px sidebar (`--sidebar-w`) + main pane.
3. 22px status bar (`--statusbar-h`).

### Sidebar

- Primary nav: Home, Queue, Downloads. No wordmark in the rail; the product
  name lives in the title bar.
- Utility nav: Settings only. About is not a sidebar item; the About route
  still mounts the Settings page (colophon at the bottom).
- Queue badge: accent count for active + pending/paused jobs.
- Downloads badge: quieter history count.
- Active nav: soft accent wash and a left accent bar.
- FFmpeg readiness does **not** live in the rail. It belongs under Settings →
  Advanced.

### Status bar

Full width, monospace, two zones:

- Left: current download folder (truncated). Click opens it. `Not set` when
  empty, and not clickable.
- Right: `Idle`, or live occupancy such as `10.7 MB/s · 2/2 slots`.

Engine and FFmpeg names do not belong here. Never display `yt-dlp` as the
engine name; the branded engine is `ytdlp-go`.

### Surfaces and type

| Role | Token / value |
| --- | --- |
| Canvas | `--surface-bg` `#121212` |
| Base | `--surface-base` `#111113` |
| Sidebar / raised | `--surface-sidebar` / `--surface-raised` `#18181B` |
| Sunken | `--surface-sunken` `#0C0C0E` |
| Input | `--surface-input` `#1F1F23` |
| Active / selected | `--surface-active` `#27272A` |
| Borders | `--border-subtle` `#202023` · `--border-default` `#27272A` · `--border-strong` `#3F3F46` |
| Text | `--text-primary` `#FAFAFA` · `--text-secondary` `#A1A1AA` · `--text-muted` `#8A8A93` |
| Accent | `--accent-500` `#3B82F6`, lighter `--accent-400` `#60A5FA` |
| Success / danger / warning | `#22C55E` / `#EF4444` / `#F59E0B` |
| Type | Geist Variable (UI), Geist Mono Variable (paths, occupancy) |
| Radii | `--r-sm` 3px … `--r-xl` 8px; progress may use `--r-full` |
| Spacing | 4px scale (`--sp-1`…); `--page-pad-x` 24px; `--rail-pad-top` 38px |

Almost flat. Hierarchy comes from surface steps and 1px zinc borders, not drop
shadows. Modals may use `--shadow-modal`. No light mode. No second brand color.

## Screens

### Home

Intake only. One paste field at the top (`Paste a YouTube URL`). No page title.
No Single URL / Batch URLs tabs. The link decides: one URL, a playlist, or 2–20
batch lines. Empty state hint plus compact **Try** chips; **Analyze** is the
only submit.

After analysis, the dock shows identity, **Output** (Video / Audio pills and
format chips), **Subtitles** (**Off** / **File** / **Embed**, language, sidecar
format), **In the file** (title & channel, artwork, chapters), and a footer with
folder **Change** + **Download**. A live analysis survives navigation to Queue,
Downloads, or Settings for the rest of the window session. Audio greys
**Subtitles** in place (`Video only`) and does not write captions.

### Queue

Operations. Master-detail: dense rows on the left, inspector on the right.
Completed work leaves the list and lives in Downloads.

Header: `Queue`, slot line (`{occupied} of {limit} slots in use` / idle copy),
ghost **Pause all** and **Resume all** when authorized. Empty well points back
to Home.

Row selection drives the inspector. Action buttons are the authorized set for
that job or collection (Pause, Cancel, Resume / Restart download, Retry,
Download again, Start again, Open source, Copy link, Review, Open, Remove, and
the matching collection actions). The UI must not invent toolbar items or infer
buttons from status text. Pause all success copy: `Pause requested for N jobs.`
Footer: `Jobs are saved automatically.` when persistence is healthy.

### Downloads

History of finished files. Header + search. Rows group under Today / Yesterday /
Earlier. Playlist and batch parents collapse episodes. Primary **Open**, ghost
**Reveal**. Expanded details: path, Remove (history only), Delete file (history
+ disk). Empty well points back to Home.

### Settings

Grouped cards, uppercase micro-headings, immediate save (no page-level Save).

- **General** — default folder, Show in Finder, Change Folder, per-download
  subfolder switch
- **Performance** — concurrent downloads 1–10 (default 2); warn above 4
- **Video files** — subtitle defaults and embed switches (see Captions)
- **Advanced** — engine / FFmpeg readiness and paths
- **Diagnostics** — Send / Don’t send, Copy diagnostics, Clear history

About is the colophon at the bottom of this page, not a separate rail item.

## Captions: Off / File / Embed

On Home (and seeded from Settings → Video files), subtitle mode is a three-way
control:

| Control | Meaning |
| --- | --- |
| **Off** | No captions with this download. |
| **File** | Save a caption sidecar next to the video (Original / SRT / VTT). |
| **Embed** | Write captions inside the video so the folder still shows one MP4. Needs FFmpeg; disabled without it. |

Language appears when captions are on. Single videos expose tracks reported by
analysis (auto-generated marked). Playlists and batches use one caption policy
(**English or first available**, or a named language) with Settings fallback,
then English or first available.

If the user chose Embed (or File) and the video has no captions VidStow can put
in the file, the download still completes. Queue reports that honesty — for
example `Saved without captions. This video had none VidStow could put in the
file.` and an options note such as `no captions in file` — instead of pretending
captions landed.

## Queue honesty

- HTTP **429** heading is **YouTube asked VidStow to slow down**, not “Download
  interrupted.” **Retry** stays the next click after a short wait.
- Missing embeddable captions: complete the file and say so (see above). Do not
  claim caption success the evidence does not support.
- Pause all success copy: `Pause requested for N jobs.` Do not claim every job
  paused instantly.
- Failure copy comes from structured backend headings and guidance. Raw engine
  errors, signed URLs, and credentials never appear in durable UI state.
- Lifecycle labels stay specific (Queued, Preparing, Downloading, Finalizing,
  Paused, Failed, Action required, Cleaning up, …). Do not collapse them into a
  generic “Downloading.”

## Constraints

- Public, on-demand YouTube only. No channels, live, cookies, other sites, or
  light/dark theme switching.
- Never write `yt-dlp` in the UI. Brand the engine as `ytdlp-go`.
- Do not add Home Single/Batch tabs. Batch is paste-count, not a mode switch.
- Do not invent Queue actions. Render only what `QueueView` authorizes, plus the
  opaque command token.
- Do not put FFmpeg or engine status in the sidebar or status bar.
- Do not restyle by cloning another product’s DESIGN.md over these tokens.
- Chrome and copy changes stay in the frontend (and this doc). Do not change Go,
  ytdlp-go, jobs, reservationfs, recovery, or Wails bindings for visual work
  unless the product contract in `PRODUCT.md` requires it.

## Related docs

- `PRODUCT.md` — full product and UI contract
- `README.md` — shipped user-facing overview
- `docs/ARCHITECTURE.md` — system boundary and queue authority
- `frontend/src/styles/global.css` — source of truth for tokens
