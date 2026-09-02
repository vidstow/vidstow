# VidStow product and UI brief

This document is the product and interface contract for building or redesigning VidStow's desktop UI. It describes the shipped product, not a wishlist. If a behavior is not listed here, do not invent it.

VidStow is a local desktop application. The UI is a native window, not a website. Design for macOS, Linux, and Windows even when a given release only ships one artifact.

## What it is

VidStow downloads public, on-demand YouTube videos, Shorts, and playlists onto the user's machine.

The user pastes a link (or a small batch of links), inspects what will be saved, chooses a readable output, and sends work to a durable FIFO queue. Analysis and downloads run locally. There is no VidStow account, no cloud library, and no hosted converter in the middle.

It sits between two bad defaults:

- a CLI that exposes every format ID, extractor, and flag
- a website that hides ads, redirects, and a mystery server behind a Download button

The product promise is serious-tool reliability with a small, readable desktop workflow. Complexity stays in the engine and queue. The UI shows decisions a person can make.

Current status: beta. Packaged preview is macOS Apple Silicon. The application still has to be designed as a three-platform desktop product.

## Who it is for

Someone who wants a specific public YouTube video, Short, playlist, or short list of URLs saved as files they own, without signing in and without learning yt-dlp.

Typical jobs:

1. Analyze one public video or Short, pick a complete video file or audio, add it to the queue.
2. Review a playlist, pick a subset (checkboxes or a numeric range), apply one output policy, admit the collection.
3. Paste 2 to 20 individual video or Short URLs, drop invalid and duplicate lines, admit the ready ones as one batch.
4. Watch progress, pause, cancel, retry, or recover when something is unsafe to guess about.
5. Find a finished file later, open it, reveal it in the system file manager, or delete it.

Not the audience: people hunting channels, live streams, private videos, other sites, or a yt-dlp replacement with every extractor and flag.

## Product principles

**Analyze before downloading.** Nothing enters the queue until the user has seen identity (title, thumbnail, duration, channel) and chosen an output. A playlist is a collection to review, not "one large URL that starts 137 downloads."

**Readable choices, not format IDs.** The UI offers best available video, capped resolutions (4K, 1440p, 1080p, 720p), original audio, or MP3 bitrates. Codec strings, containers, and approximate size are secondary detail. Selector syntax stays inside the machinery.

**The backend authorizes actions.** Queue rows and collection parents render only the Pause, Resume, Cancel, Retry, Download again, Start again, Open source, Copy link, Review, Open, or Remove actions the application currently allows, plus an opaque command token. The UI must not infer buttons from status text. If persistence is unhealthy, every mutating action is disabled.

**Conservative honesty.** Corrupt, unsafe, contended, unavailable, or indeterminate evidence does not authorize automatic destructive mutation. Pause is cooperative: a transfer may settle as paused, while work already finalizing may complete. Resume does not promise universal byte reuse. Cancel requests session discard; if cleanup cannot be proved, the row stays visible or asks for review. Never display a success state the evidence does not support.

**Local-first and private by default.** Queue, settings, history, and a bounded diagnostic log live in the OS per-user config directory. Automatic diagnostics are off until the user explicitly chooses Send diagnostics. Reports never include video IDs, URLs, paths, filenames, cookies, tokens, or raw error text.

**One complete video policy.** Video downloads are complete files: thumbnail artwork and chapter markers are included when YouTube provides them. That policy is not a user toggle. Subtitles and title/channel metadata are the optional extras. MP4 is preferred, with MKV as fallback when needed. FFmpeg is required for complete video files and for MP3 conversion. Original audio that does not need merging can still be queued without FFmpeg.

**Desktop workstation, not a consumer site.** The app is a focused tool: narrow sidebar, dense working surface, persistent destination, live progress. It should feel like a professional utility. It should not feel like a landing page, a media player, or a browser download manager with ads.

## Scope

### Supported

- Public, on-demand YouTube watch URLs, Shorts URLs, and playlist URLs
- Links that contain both a video and a playlist (user chooses which to review)
- Drag-and-drop of a public YouTube URL onto the window
- Opportunistic clipboard URL on Home (never overwrites work already in progress)
- Single-video analysis with curated output plans
- Playlist review of up to 500 available entries, with search, select all, clear, and a bounded index range
- Batch review of 2 to 20 individual video or Short URLs (not playlist URLs in the batch composer)
- One output policy per playlist or batch admission
- FIFO queue with 1 to 10 concurrent downloads (default 2)
- Durable collections: a playlist or batch appears as one expandable parent with independent child jobs
- Persistent download history with search, open, reveal, remove-from-history, and delete-file
- Destination reservations so VidStow does not silently overwrite an unrelated file
- External FFmpeg/FFprobe detection (PATH, Homebrew default prefixes, or a user-selected pair)
- First-launch diagnostic consent, Copy Diagnostics, Clear history
- Quit confirmation that pauses active work rather than killing it
- Recovery-required shell when State v2 cannot be read safely

### Out of scope

- Channels, search, live streams, authenticated or private downloads
- Sites other than YouTube
- DRM, cookies, login, or access-control circumvention
- Universal resume or guaranteed byte reuse
- Treating cleanup, publication, or recovery as successful when evidence is uncertain
- Automatic updates
- Feature parity with yt-dlp or the broader `ytdlp-go` CLI
- Cloud sync, accounts, or a hosted library
- Light/dark theme switching is not a current product feature. The shipped shell is dark.

If analysis cannot proceed, fail with a clear unsupported-URL dialog. Do not silently try other extractors.

## Core objects

**Job.** One downloadable item with its own lifecycle, attempt, session, reservation, and history record. Children of a playlist or batch are ordinary jobs.

**Collection.** A durable parent of type `playlist` or `batch`. It owns title, optional thumbnail/channel, a server-authored output-policy label, ordered child job IDs, and aggregate progress. It is not a second scheduler. Children keep independent lifecycle rules. Removing the last child removes the collection. Batch collections do not persist the pasted text or a collection-level source URL.

**Output plan.** A backend-authored choice for a single video (kind video or audio, label, container, codecs, approximate size, FFmpeg requirement, recommended flag). The user picks a plan. The UI does not invent formats.

**Output options.** Per-admission extras on complete video files: optional embedded subtitle (one language), optional extra `.srt`, optional title and channel metadata. Artwork and chapters are always on for video. Audio admissions turn extras off.

**Queue view.** The only source of truth for the Queue page. It includes rows, collections, occupancy summary, queue-level capabilities (Pause all, Clear completed), a command token, and persistence health. The frontend presents this view. It does not reconstruct authority from job snapshots.

**History entry.** A completed download that survives app launches: title, channel, quality, container, codecs, size, path, thumbnail, completion time, optional delivery note, and a missing-file flag.

**FFmpeg status.** Available or not, paths, version, and a short message. Complete video and MP3 paths require it.

## Application chrome

Default window 1180×760, matching the p2 frame. Minimum 960×600. Persist size when the user resizes.

Layout, top to bottom:

1. Native title bar. Window title is always `VidStow`. Download progress stays in Queue, not the title.
2. Workstation row: sidebar plus main.
3. Status bar.

### Sidebar

Fixed width, 160px. Slightly raised against the main pane. Order:

- Brand: download-into-tray mark plus the wordmark `VidStow`. Clicking it goes Home. The mark is a small blue tray glyph, not a decorative logo lockup.
- Primary nav: Home, Queue, Downloads.
- Utility nav: Settings only. About is not in the sidebar. The About page still exists as a route until Settings absorbs it.

FFmpeg readiness does not live in the rail. It belongs on Settings under Engine.

Queue shows an accent badge for active + pending/paused jobs. Downloads shows a quieter badge for history count. Active nav uses a soft accent wash and a 2px left accent bar.

### Status bar

About 22px, full width, two zones:

- Left: current download folder, truncated, monospace. `Not set` when empty.
- Right: `Idle` when no job occupies a slot. While work is running, live occupancy such as `10.7 MB/s · 2/2 slots`, or `1/2 slots` when speed is unknown.

Engine and FFmpeg names do not belong here.

### Main pane

Each route starts on the **Home** nav row: 24px from the rail. Home’s header is the analyze bar itself. Queue, Downloads, and Settings titles sit on that same line.

### Global overlays

- Startup: centered spinner, copy `Restoring saved downloads…`. No sidebar until startup resolves.
- Recovery-required: replaces the workstation. No queue mutation.
- Modal dialogs: dimmed overlay, focus trap, Escape and overlay-click close when safe.
- Toast banner: above the status bar, centered, auto-dismiss. Kinds: success, info, warning, danger. One at a time.
- First-launch diagnostic consent before the FFmpeg-missing dialog, if both apply.
- Window-wide drag-and-drop: a dropped public YouTube URL navigates Home and starts analysis. Anything else shows a warning banner.

File-manager labels must be platform-aware: Show in Finder on macOS, Show in Explorer on Windows, Show in folder (or the desktop environment's name) on Linux. The current UI hard-codes Finder. A rebuild should not.

## Routes and features

### Home

Home is intake. It is the only place work is analyzed and admitted. A live analysis stays for the rest of this window session. Opening Queue, Downloads, Settings, or About does not discard it. Coming back shows the same field, dock, and selection. An edited URL, a new Analyze, a dropped URL, or Start over from Home replaces it.

**One field.** No page title. No Single URL / Batch URLs switch. Placeholder: `Paste a YouTube URL`. Enter submits Analyze. Paste still accepts several lines. Under the empty field, two lines: the paste sentence, then **Try:** with dotted chips for a video, a playlist, and several links. Those chips fill the field. Analyze is the only submit. A watch URL that also has a list id is not a Try chip. Paste it and Home asks before any dock.

Paste count chooses the path. One URL goes to `ValidateURL`. Two to twenty go to `AnalyzeBatchURLs`. A playlist URL is still a playlist. Playlist URLs do not belong in a batch. The 2 to 20 and 500 caps stay.

The URL field stays pinned to the top of Home, empty or after Analyze. The dock, error card, or Try hint occupies the space below it. The field does not jump from a centered landing into the working layout. Pasted URLs use Geist Mono at 13px in secondary; they go primary while the field is focused.

**Failed Analyze.** Stay on Home. Show the error on the page. **Paste another link** focuses the field and selects the current URL so the next paste replaces it. Do not leave Analyze bound to a dead URL as the only next step.

**Single video after analysis.**

The dock is three columns: thumbnail, title and meta, then **Type** plus format chips hugging the right. Leftover width goes to the title. Footer under that. Subtitles and extras are not on the dock yet. When they ship they sit in a row under this header, not beside the quality chips.

- Thumbnail is a 160×90 cover crop so a Short cannot stretch the dock. Title, channel, optional Short badge, duration, view count
- If the original link also contained a playlist: a swap line `Part of playlist: <title> · N videos →`
- **Type** is a dropdown (Video / Audio) stacked above compact bluish chips (`1440p`, `M4A (Original)`). Chips hug their labels and wrap inside that right column; they do not stretch across the card.
- Title uses a two-line clamp. Hover still shows the full name. Same clamp on playlist titles
- Under the channel line: muted container · codecs, then size, in the UI sans. No chip, no highlight, no mono. Updates with the selected chip.
- Footer: **Change**, the path, **Download**. No Save to label. Success: banner `Queued for download`. The dock clears; the URL stays.

Changing the URL clears analysis.

**Playlist after analysis.**

Same dock, adapted:

- Thumbnail, title, Playlist · N videos · duration · channel, unavailable count. Duration is the sum of available entry lengths when every available video reported one. Channel comes from the playlist, or the first child that named one. **Type** is the same dropdown; **Format** is chips (`1080p`, `720p`, original audio, MP3). No invented 1080p coverage or playlist file-size total.
- Disclosure: episode count and the chosen policy. Cap note when 500 entries were returned. The list stays collapsed until that disclosure is opened
- Episode review: `N of M selected`, All, None, Range as two position fields plus Apply. All fills the fields with first–last index. None clears them. Rows are checkbox, title, duration. Unavailable rows are visible, dimmed, not selectable. Search is not on this list yet
- Range Apply with an invalid span shows `Enter positions from N to M` on the header. Selection is by original index
- Footer: **Change** then the playlist folder path. **Download N videos** while the list is collapsed. N is selected count. With the list open, that commit sits under the rows as `N selected · policy` plus **Download N videos**

**Link that is both video and playlist.**

Before any dock, Home shows `This link includes a playlist` / `Choose what to download.` Two cards: the pasted **Video** (thumb, title, channel · duration · size when the backend reported it) and **Playlist** (primary). Esc cancels. Enter and the playlist card open the full list. After a choice, a swap line on the dock switches: `Part of playlist: … →` on the video, `Pasted video: … →` on the playlist. The other side is cached so the swap does not re-paste.

**Batch after analysis.**

Reached by pasting 2 to 20 lines or comma-separated watch URLs, not by flipping a mode. Review list: line number, thumbnail, title or raw input, URL, channel · duration, status chip (Ready, Duplicate, Invalid URL, Analysis failed) plus a reason when not ready. Summary such as `12 videos ready to download` or `10 ready · 2 duplicate`. Expired review (30 minutes): `This review expired. Edit the lines and review them again.`

Shared **Type** dropdown and format chips for every ready item. The title list opens from an underlined `{N} titles` hint, not from invented coverage or a size total. Footer: **Change**, the path, **Edit URLs**, **Download N videos**. N is ready count, not pasted count. Disabled unless the token is still valid, ready ≥ 2, and a folder is set. Success navigates to Queue. Edit URLs returns to the field without losing the text.

**Output option rules.**

Artwork and chapters stay on for video. Audio tab hides extras. Video admission and MP3 conversion require FFmpeg. Missing FFmpeg opens a dialog with Open Settings, and notes that original audio without merging remains available.

### Queue

Queue is operations. Master-detail, matching **p2.html**: 36px rows on the left, a 300px inspector on the right.

**Empty.** Header `Queue` stays. Empty well: list mark, left-aligned `Nothing here yet` / `Paste a link on Home.` / `In-progress downloads show up here.`, and **Go to Home** with a home mark. Pause all and Resume all wait until there is a list.

**Master (left).** Header `Queue` with ghost tool buttons **Pause all** and **Resume all**. Pause all is enabled only when the queue view authorizes it. Resume all is enabled when any visible job or collection authorizes Resume. Optional persistence warning banner. Then:

- Standalone incomplete jobs, densest first
- Collection parents as a single 36px row: playlist artwork at 34×22 when the parent or a child has a still, otherwise the count badge. A **Playlist** or **Batch** tag sits on the title. Then a 3px bar and `N/M · speed` or `N/M · paused`
- Completed work leaves the list (it lives in Downloads)

A job row is thumbnail (34×22), title, progress bar, and a mono tail (speed · ETA, paused, waiting · slot, or the failure heading). Failed and action-required rows get a 2px red inset edge. Actions do not sit on the row.

Selecting a row drives the inspector. Selected row uses the raised fill. Space toggles pause/resume on the focused job, matching p2.

**Inspector (right).** For a video: title, 16:9 still, metadata, Speed / ETA / Progress / Status, bar, then authorized actions. Failed jobs add the error card and **Copy details**. For a playlist: title, playlist still, `Playlist · N/M finished · speed`, bar, up to two live children with 34×22 thumbs and bars, waiting titles with thumbs, a finished count, then **Pause collection** / **Resume collection**, Cancel, and any other authorized parent action. Batch collections stay count-badge only, with no synthetic parent still. Cancel and Remove still confirm: `Completed files remain on disk.`

Action buttons are the full authorized set for that job or collection. Labels:

| Action | When it exists | Meaning |
| --- | --- | --- |
| Pause | Authorized | Cooperative pause |
| Cancel | Authorized | Discard the engine session |
| Resume / Restart download | Authorized | Resume, or restart from byte zero when `restartOnly` |
| Retry | Authorized | Enqueue another attempt |
| Download again | Authorized | Fresh copy in the queue |
| Start again | Authorized | Sends the URL back to Home for re-analysis (disk/permission class failures) |
| Open source | Authorized | Open the YouTube page |
| Copy link | Authorized | Copy the canonical watch URL |
| Review | Action-required | Opens the review dialog |
| Open | Completed, file present | Open the file |
| Remove | Authorized | Remove the queue row. Files on disk stay unless a dedicated delete flow says otherwise |

Pause all copy on success: `Pause requested for N jobs.` The product does not claim every job paused instantly. Footer-level honesty: `Jobs are saved automatically.` when persistence is healthy.

Occupancy is a real product fact (N of M active slots, waiting, paused, optional processing slots). Surface it in the queue header or inspector. Do not invent a second scheduler UI.

**Lifecycle labels** (do not collapse these into generic Downloading):

| Durable lifecycle | Phase (when active) | Label |
| --- | --- | --- |
| pending | | Queued |
| active | preparing | Preparing |
| active | downloading (default) | Downloading |
| active | waiting-for-processing | Waiting for processing |
| active | finalizing | Finalizing |
| active | ready-to-publish | Ready to publish |
| active | publishing | Publishing |
| pausing | | Pausing |
| paused | | Paused |
| canceling | | Canceling |
| failed | | Failed |
| canceled | | Canceled |
| completed | | Completed |
| action-required | | Action required |
| any | cleaning-up | Cleaning up |

Progress, speed, and ETA are live presentation. They do not authorize buttons. Completed rows must not keep a stale speed or ETA.

Failed rows show a category heading and guidance from the backend (`failure.heading`, `failure.message`, `recommendedAction`). The next step is whichever capability is actually on.

### Downloads

History of completed files. Header `Downloads` / `View your recently downloaded items.`

Search across title, channel, filename, quality, container, delivery note. When more than 10 entries exist, Recent (10) and All tabs, plus `Show all N downloads` under the recent list.

Each row: thumbnail with duration, title, channel · quality · container · size · relative time. Open and Show in `<file manager>` on the row. Clicking the main area expands details: delivery note, codecs and absolute path (monospace), Remove (history only), Delete file (history + disk + matching queue row).

A finished row whose file is gone stays in the list. Open, Reveal, and Delete file stay enabled. If the path is gone, VidStow says that file is no longer at this path.

Empty: header `Downloads` and search stay. Empty well matches Queue: tray mark, `Nothing here yet` / `Paste a link on Home.` / `Finished files show up here.`, and **Go to Home** with a home mark. Search miss: `No matching downloads` / `Try a different title, channel, or filename.` and **Clear search**.

Confirmations:

- Remove from history: media file on disk stays untouched.
- Delete file: permanently deletes the file and removes it from Downloads and the queue.

### Settings

Grouped cards. Uppercase micro-headings. Each row is label + control, description in secondary type.

**General**

- Default download folder, Show in file manager, **Change Folder**
- Create a subfolder for each download (all artifacts for one video together). Playlist admission still uses a playlist folder under that root. Batch children may use distinct roots when this is on. Binary rows use a switch, not On/Off pills.
- Confirm before starting downloads
- Interrupted jobs are a fixed value, `In progress continues`. The download that was in progress continues on launch. Waiting stays waiting. Paused stays paused.

**Performance**

- Maximum concurrent downloads 1-10, default 2, FIFO, with range and recommended 2–4 on the row. Reducing the limit waits for active jobs. It does not pause them. Warn when concurrency is greater than 4: stability and rate limits.

**Video files**

- Complete video files: Always on. Thumbnail artwork and chapter markers when YouTube provides them.
- Include subtitles on new adds: Off / On. Fresh install is Off. Home can override per analysis.
- Preferred subtitle language, shown when On: each video's own language, or a common language list.
- Also save an `.srt` file (in addition to the embedded track, when subtitles are On).
- Embed title and channel details.

**Advanced**

- Engine and dependencies as one status row: FFmpeg Ready / Missing, Recheck Dependencies, ytdlp-go version underneath.
- FFmpeg path, **Change Path** to pick a binary (matching ffprobe beside it).
- If missing: install hint plus link to ffmpeg.org/download.html.

**Diagnostics**

- Send operational diagnostics: Send diagnostics / Don't send. Neither is a silent default. Copy must say reports exclude video IDs, links, paths, filenames, cookies, tokens, and error text. Disabling deletes anything waiting to send. Privacy notice link.
- Support report: Clear history, Copy Diagnostics.

Settings save immediately (optimistic), with a success or error banner. Do not add a page-level Save button.

### About

Product name, one-line tagline, short description. Facts: version · Apache-2.0, engine `ytdlp-go <version>`, platform `os/arch`. View source, Read the docs, Copy Diagnostics. Built-with chips (Go, Wails, Svelte, ytdlp-go, FFmpeg) that open their sites. Legal: not affiliated with YouTube or Google. User is responsible for rights and local law.

Never display the string `yt-dlp` as the engine name. The branded engine is `ytdlp-go`.

## Dialogs and system states

**Unsupported URL.** Title `Unsupported URL`. Explain that the link must be a valid, publicly accessible YouTube video, Short, or playlist.

**FFmpeg required.** Title and a specific reason (complete video vs MP3). Fine print: outputs that do not need FFmpeg remain available. Primary: Open Settings.

**Diagnostic consent (first launch).** `Help improve VidStow?` Don't send / Send diagnostics. Closing without a choice leaves automatic sending off. Privacy notice link. No preselected radio in Settings until the user has chosen.

**Quit.** `Quit VidStow?` Active downloads will continue on reopen. Summary list: active (will continue on reopen), waiting or paused (already safe). `The download that was in progress will continue the next time VidStow opens. Waiting and paused jobs stay as they are.` Keep working / Pause downloads and quit / Quit. Pause downloads and quit records paused intent first, so those jobs stay paused.

**Destination conflict.** `Choose a new filename`. VidStow will not replace the existing file. Read-only unavailable name and proposed name, Available chip when the proposal is free. Cancel download / Use new name. Authority is the opaque conflict token, never the displayed filename.

**Action required review.** Eyebrow `Action required`, heading and item title from the backend, preservation notice, then only the buttons the review model enables: Remove from queue, Discard saved data, Retry cleanup, Try recovery again, Retry with fresh link, Start over from Home. Closing the dialog leaves the row in the queue. Start over returns the URL to Home and leaves preserved evidence untouched. Discard confirms that only data VidStow can identify safely will be removed.

**Cannot save (small stop).** `VidStow cannot save this session`. The data folder is not writable. Open data folder. Not the full-screen recovery shell.

**Junk queue.** Unreadable or untrusted `state.json` is quarantined. Settings are salvaged when possible. The workstation starts healthy with a dismissible notice: `The saved queue could not be read. Files on disk were not touched.` Files on disk are not touched.

**Recovery required (full screen).** Triangle warning, `Download state needs recovery`. Media and recovery files preserved. Downloads paused. VidStow will not resume, retry, cancel, or clean up. Status list: State file, Automatic cleanup (Disabled), Saved media (Preserved). Copy diagnostics / Open data folder. Footer: `No recovery files have been changed.` No sidebar, no queue. Unwritable data uses the cannot-save stop instead.

**Collection cancel/remove.** Confirm. Completed files remain on disk.

## Visual design

The shipped look is a zinc-dark workstation with a blue accent. A redesign may refine spacing, type, and hierarchy. It should keep this character: precise, quiet, technical, not playful and not skeuomorphic.

### Color

| Token | Value | Role |
| --- | --- | --- |
| Surface background | `#121212` | App canvas |
| Surface sidebar | `#18181B` | Nav |
| Surface base | `#111113` | Cards, lists |
| Surface raised | `#18181B` | Dialogs, hover |
| Surface sunken | `#0C0C0E` / `#0C0C0E` | Inputs, header strips |
| Border subtle | `#202023` | Row rules |
| Border default | `#27272A` | Cards, controls |
| Border strong | `#3F3F46` | Emphasis, dialogs |
| Text primary | `#FAFAFA` | Titles, values |
| Text secondary | `#A1A1AA` | Supporting |
| Text muted | `#8A8A93` | Meta, placeholders |
| Accent | `#3B82F6` / `#60A5FA` | Primary buttons, selection, progress |
| Accent soft | `rgba(59, 130, 246, 0.13)` | Selected plan, active tab count |
| Success | `#22C55E` | Ready, completed |
| Warning | `#F59E0B` | FFmpeg missing, action required, persistence |
| Danger | `#EF4444` | Failed, destructive |

Selection in lists is a 2px inset accent bar plus a slightly lighter row, not a loud fill. Progress bars are 3px accent on a dark track. Status chips use the matching soft background, not saturated pills.

Native window chrome should not fight the shell. On macOS the title bar uses Dark Aqua so traffic lights and the window title sit on the zinc canvas. Linux and Windows keep native window controls and inherit the dark webview background.

### Type

Sans: Geist (variable), then system UI. Mono: Geist Mono, then SF Mono / Consolas, for paths, URLs, versions, occupancy numbers, speed, ETA.

Shipped sizes are small on purpose (10-14px body, 9px status bar). A rebuild may lift body type slightly for readability, but this is a dense tool. Do not jump to 16-18px marketing type. Page titles around 14-16px, weight ~650, slight negative tracking. Uppercase section labels in Settings/About: 10-11px, wide tracking, muted.

Tabular numbers for percent, size, duration, indexes.

### Shape and density

Radii are tight: 3-8px. Almost no drop shadow on cards. Modals use a heavy dark shadow and a 3px backdrop blur.

Control height: 24-32px. Icon buttons 24×24. Sidebar items 32px. Do not use large rounded consumer buttons.

Thumbnails: 16:9, 2-4px radius, sunken fill if missing. Video analyze header ~72px wide. Playlist entries ~64px. Queue rows ~36×28. History ~80px.

Spacing scale: 4 / 8 / 12 / 16 / 20 / 24 / 32. The workspace on every route starts on the Home nav row, 24px from the rail, so the Home field, Queue, Downloads, and Settings titles share one line. Home workspace is a single bordered card inset 10-12px from that canvas, not a stack of marketing cards.

### Motion

120ms color/background transitions. Collection chevron rotates 90°. Startup spinner. Occupied-slot dots may pulse. Honor `prefers-reduced-motion`: no animation.

No page-route animation, no skeleton shimmer theater, no confetti on complete.

### Iconography

Stroke icons, 1.5-1.8 width, round caps. 13-14px in chrome, 16px in search fields. The product glyph is an arrow into a tray, not a play button and not a YouTube clone.

## Interaction patterns

**Segmented controls** for binary modes (Off/On subtitles, Recent/All). Active segment is a raised inner chip, not a filled primary button.

**Select menus** for binary lists that should stay closed until chosen. On Home, **Type** chooses Video or Audio and sits with the format chips on the right of the dock. Plans for that kind are compact bluish chips. Container, codecs, and size sit under the channel line, not next to **Download**.

**Primary button** is blue, used once per working surface (**Download**, **Download N videos**, Pause downloads and quit). **Analyze** is the light query chip in the URL field: same 24px body, cream fill, dark type. It is not a second blue. The playlist footer hugs **Change** and **Download N videos** (`8px 10px 8px 16px`); it does not keep a padded empty band around them. Destructive actions are outline/danger text, never the only primary in a footer unless the dialog is itself destructive.

**Lists are the product.** Rows are the scan unit. Keep thumbnail + title + one meta line + trailing status/actions. Do not turn jobs into large cards.

**Busy and idempotence.** Analyze, add, and batch start disable while in flight and ignore stale responses via generation counters. After add, the same identity (video/plan/folder/options, or playlist selection/policy/folder) should not be re-queued until something changes.

**Toasts vs dialogs.** Success and short info go to the banner. Failures that need a decision or a long explanation go to a modal. Persistence problems can be a warning banner plus disabled actions.

**Confirm only when the action is destructive or ambiguous.** Adding to the queue does not confirm. Canceling a collection does. Deleting a file does. Discarding saved session data does.

**Capability + token.** Every mutating queue control is disabled unless both the capability flag is true and the command token is a non-empty Unicode scalar string of at most 256 UTF-8 bytes with no C0/C1 controls. Persistence failure strips capabilities in the view adapter.

## Copy voice

Plain, specific, slightly dry. Sentence case. No exclamation marks. No emoji. No "Oops". No "Your download journey".

Good: `Added 12 downloads to the queue`. `FFmpeg is required to create complete video files with artwork and chapters.` `VidStow will not replace the existing file.`

Bad: `Let's get started!` `Powerful downloads made simple.` `We couldn't process your request.`

Use "download" as the user-facing noun, "queue" for in-flight work, "Downloads" for history. Use "video" and "audio", not "asset" or "media item". Use "complete video file" when explaining the FFmpeg policy.

Engine name in the UI: `ytdlp-go`. Product name: `VidStow` (capital V and S).

Errors shown to the user are bounded presentation strings from the backend. Do not dump stack traces, raw paths unless the user is in a path field or history detail, or engine stderr.

## Accessibility

- Every icon-only control has an `aria-label`.
- Dialogs are `role="dialog"` with labelled title, focus trap, Escape to close when not busy.
- Analyze field, search fields, and range inputs have visible or visually-hidden labels.
- Progress bars expose min/max/now.
- Selected queue row is `aria-current`.
- Batch review uses a live summary.
- Contrast: primary text on zinc-900, secondary on zinc-400. Do not use accent-on-accent.
- Keyboard: buttons and inputs are native. Do not build a custom focus system unless you keep tab order obvious. Visible 2px accent focus rings.
- Thumbnails are empty `alt`. Status is in adjacent text.

## Platform behavior

Design the same IA on macOS, Linux, and Windows. Differences:

- File manager verb (Finder / Explorer / folder).
- Path separators and path truncation in the status bar and history.
- Window controls stay native. Do not draw fake traffic lights.
- Drag-and-drop URLs onto the window.
- Clipboard read is best-effort and may be denied.

Minimum usable width 1000px. Below ~760px, stack the queue inspector under the list, collapse Home identity to two columns, and let the save bar wrap. This is a desktop app. Do not pretend to be a phone layout.

## What a UI rebuild may change

- Spacing, type scale, icon set, and visual polish
- Whether occupancy lives only in the status bar or is also repeated in the queue header / inspector
- Platform-correct file-manager labels
- Native title-bar appearance so it matches the dark shell
- Internal component structure

## What a UI rebuild must not change

- Analyze-then-admit workflow
- Supported URL kinds and the 2-20 / 500 caps
- Backend-authored plans, capabilities, and tokens
- Complete-video policy (artwork and chapters always on)
- Conservative recovery and action-required honesty
- No-account, local-first data story and diagnostic consent
- Route set: Home, Queue, Downloads, Settings, About
- Cross-platform product intent

Build a beautiful professional desktop tool around those constraints. Do not add features to fill the window.
