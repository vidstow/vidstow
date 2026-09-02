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

1. Analyze one public video or Short, pick a video or audio output, add it to the queue.
2. Review a playlist, pick a subset (checkboxes or a numeric range), apply one output policy, admit the collection.
3. Paste 2 to 20 individual video or Short URLs, drop invalid and duplicate lines, admit the ready ones as one batch.
4. Watch progress, pause, cancel, retry, or recover when something is unsafe to guess about.
5. Find a finished file later, open it, reveal it in the system file manager, or delete it.

Not the audience: people hunting channels, live streams, private videos, other sites, or a yt-dlp replacement with every extractor and flag.

## Product principles

**Analyze before downloading.** Nothing enters the queue until the user has seen identity (title, thumbnail, duration, channel) and chosen an output. A playlist is a collection to review, not "one large URL that starts 137 downloads."

**Readable choices, not format IDs.** The UI offers best available video, capped resolutions (4K, 1440p, 1080p, 720p), original audio, or MP3 bitrates. Codec strings, containers, and approximate size are secondary detail. Selector syntax stays inside the machinery. Video plans prefer MP4, with MKV as fallback when needed. FFmpeg is required to merge most video downloads and to convert to MP3. Original audio that does not need merging can still be queued without FFmpeg.

**The backend authorizes actions.** Queue rows and collection parents render only the actions the application currently allows, plus an opaque command token. The UI must not infer buttons from status text. If persistence is unhealthy, every mutating action is disabled.

**Conservative honesty.** Corrupt, unsafe, contended, unavailable, or indeterminate evidence does not authorize automatic destructive mutation. Pause is cooperative: a transfer may settle as paused, while work already finalizing may complete. Resume does not promise universal byte reuse. Cancel requests session discard; if cleanup cannot be proved, the row stays visible or asks for review. Never display a success state the evidence does not support.

**Local-first and private by default.** Queue, settings, history, and a bounded diagnostic log live in the OS per-user config directory. Automatic diagnostics are off until the user explicitly chooses Send. Reports never include video IDs, URLs, paths, filenames, cookies, tokens, or raw error text.

**Desktop workstation, not a consumer site.** The app is a focused tool: narrow sidebar, dense working surface, persistent destination, live progress. It should feel like a professional utility. It should not feel like a landing page, a media player, or a browser download manager with ads.

## Scope

### Supported

- Public, on-demand YouTube watch URLs, Shorts URLs, and playlist URLs
- Links that contain both a video and a playlist (user chooses which to review)
- Drag-and-drop of a public YouTube URL onto the window
- Single-video analysis with curated output plans
- Playlist review of up to 500 available entries, with All, None, and a bounded index range
- Batch review of 2 to 20 individual video or Short URLs (not playlist URLs in the batch composer)
- One output policy per playlist or batch admission
- FIFO queue with 1 to 10 concurrent downloads (default 2)
- Durable collections: a playlist or batch appears as one expandable parent with independent child jobs
- Persistent download history with search, open, reveal, remove-from-history, and delete-file
- Destination reservations so VidStow does not silently overwrite an unrelated file
- External FFmpeg/FFprobe detection (PATH, Homebrew default prefixes, or a user-selected pair)
- First-launch diagnostic consent, Copy diagnostics, Clear history
- Quit confirmation. Ordinary quit leaves in-progress downloads running, the same as a crash. Pause downloads and quit records paused intent first.
- Cannot-save stop when State v2 cannot be written or the data folder is unsafe. An unreadable queue is quarantined and skipped instead of freezing the app.

### Out of scope

- Channels, site-wide search, live streams, authenticated or private downloads
- Sites other than YouTube
- DRM, cookies, login, or access-control circumvention
- Universal resume or guaranteed byte reuse
- Treating cleanup, publication, or recovery as successful when evidence is uncertain
- Automatic updates
- Feature parity with yt-dlp or the broader `ytdlp-go` CLI
- Cloud sync, accounts, or a hosted library
- Light/dark theme switching. The shipped shell is dark.

If analysis cannot proceed, fail with a clear unsupported-URL dialog. Do not silently try other extractors.

## Core objects

**Job.** One downloadable item with its own lifecycle, attempt, session, reservation, and history record. Children of a playlist or batch are ordinary jobs.

**Collection.** A durable parent of type `playlist` or `batch`. It owns title, optional thumbnail/channel, a server-authored output-policy label, ordered child job IDs, and aggregate progress. It is not a second scheduler. Children keep independent lifecycle rules. Removing the last child removes the collection. Batch collections do not persist the pasted text or a collection-level source URL.

**Output plan.** A backend-authored choice for a single video (kind video or audio, label, container, codecs, approximate size, FFmpeg requirement, recommended flag). The user picks a plan. The UI does not invent formats.

**Queue view.** The only source of truth for the Queue page. It includes rows, collections, occupancy summary, queue-level capabilities, a command token, and persistence health. The frontend presents this view. It does not reconstruct authority from job snapshots.

**History entry.** A completed download that survives app launches: title, channel, quality, container, size, path, thumbnail, completion time, and a missing-file flag.

**FFmpeg status.** Available or not, paths, version, and a short message. Most video plans and all MP3 plans require it.

## Application chrome

Default window 1180×760. Minimum 960×600.

Layout, top to bottom:

1. Native title bar. Window title is always `VidStow`. Download progress stays in Queue, not the title.
2. Workstation row: sidebar plus main.
3. Status bar.

### Sidebar

Fixed width. Order:

- Primary nav: Home, Queue, Downloads. The product name lives in the native title bar. The rail does not repeat the wordmark.
- Utility nav: Settings only. About is not in the sidebar. The About route still exists and renders the Settings page, including the colophon at the bottom.

FFmpeg readiness does not live in the rail. It belongs on Settings under Advanced.

Queue shows an accent badge for active + pending/paused jobs. Downloads shows a quieter badge for history count. Active nav uses a soft accent wash and a left accent bar.

### Status bar

Full width, two zones:

- Left: current download folder, truncated, monospace, secondary. Click opens the folder in the file manager. `Not set` when empty, and not clickable.
- Right: `Idle` when no job occupies a slot. While work is running, live occupancy such as `10.7 MB/s · 2/2 slots`, or `1/2 slots` when speed is unknown.

Engine and FFmpeg names do not belong here.

### Main pane

Each route starts on the Home nav row. Home’s header is the analyze bar itself. Queue, Downloads, and Settings titles sit on that same line.

### Global overlays

- Startup: centered spinner, copy `Restoring saved downloads…`. No sidebar until startup resolves.
- Cannot-save: replaces the workstation with a small stop when the data folder is unwritable or unsafe.
- Junk-queue notice: a dismissible banner on the workstation after an unreadable queue was skipped.
- Modal dialogs: dimmed overlay, focus trap, Escape and overlay-click close when safe.
- Toast banner: above the status bar, centered, auto-dismiss. Kinds: success, info, warning, danger. One at a time.
- First-launch diagnostic consent before the FFmpeg-missing dialog, if both apply.
- Window-wide drag-and-drop: a dropped public YouTube URL navigates Home and starts analysis. Anything else shows a warning banner.

Settings hard-codes Show in Finder. Downloads uses Reveal.

## Routes and features

### Home

Home is intake. It is the only place work is analyzed and admitted. A live analysis stays for the rest of this window session. Opening Queue, Downloads, Settings, or About does not discard it. Coming back shows the same field, dock, and selection. An edited URL, a new Analyze, a dropped URL, or Start over from Home replaces it.

**One field.** No page title. No Single URL / Batch URLs switch. Placeholder: `Paste a YouTube URL`. Enter submits Analyze. Paste still accepts several lines, or more than one `http` URL on a line. Under the empty field: `One link, a playlist, or a handful — the link decides.`, then **Try** with compact bordered chips for a video, a playlist, and several links. Those chips fill the field. Analyze is the only submit. A watch URL that also has a list id is not a Try chip. Paste it and Home asks before any dock.

Paste count chooses the path. One URL goes to `ValidateURL`. Two to twenty go to `AnalyzeBatchURLs`. A playlist URL is still a playlist. Playlist URLs do not belong in a batch. The 2 to 20 and 500 caps stay. More than 20 URLs stays on Home with `Too many URLs`.

The URL field stays pinned to the top of Home, empty or after Analyze. The dock, error card, or Try hint occupies the space below it.

**Failed Analyze.** Stay on Home. Show the error on the page. **Paste another link** focuses the field and selects the current URL so the next paste replaces it.

**Single video after analysis.** Three columns: thumbnail, title and meta, then **Type** plus format chips hugging the right. Footer under that.

- Thumbnail, title, channel, optional Short badge, duration, view count
- If the original link also contained a playlist: a swap line `Part of playlist: <title> · N videos →`
- **Type** is a dropdown (Video / Audio) stacked above compact chips (`1440p`, `M4A (Original)`)
- Title uses a two-line clamp. Hover still shows the full name
- Under the channel line: muted container · codecs, then size. Updates with the selected chip
- Footer: **Change**, the path, **Download**. Success: banner `Queued for download`, then Queue. The dock clears; the URL stays

Changing the URL clears analysis. Confirm before starting downloads (off by default) asks `Add this download?` before enqueue.

**Playlist after analysis.** Same dock, adapted:

- Thumbnail, title, Playlist · N videos · duration · channel, unavailable count. Duration is the sum of available entry lengths when every available video reported one. Channel comes from the playlist, or the first child that named one. **Type** is the same dropdown; format chips follow. No invented 1080p coverage or playlist file-size total
- Disclosure: episode count and the chosen policy. Cap note when 500 entries were returned. The list stays collapsed until that disclosure is opened
- Episode review: `N of M selected`, All, None, Range as two position fields plus Apply. All fills the fields with first–last index. None clears them. Rows are checkbox, title, duration. Unavailable rows are visible, dimmed, not selectable
- Range Apply with an invalid span shows `Enter positions from N to M` on the header. Selection is by original index
- Footer: **Change** then the playlist folder path. **Download N videos**. N is selected count. Success navigates to Queue

Confirm before starting downloads, or more than 100 selected videos, asks `Add this playlist?` before enqueue.

**Link that is both video and playlist.** Before any dock, Home shows `This link includes a playlist` / `Choose what to download.` Two cards: the pasted **Video** and **Playlist** (primary). Esc cancels. After a choice, a swap line on the dock switches. The other side is cached so the swap does not re-paste.

**Batch after analysis.** Reached by pasting 2 to 20 lines (or more than one `http` URL on a line), not by flipping a mode. Review list: line number, thumbnail, title or raw input, URL, channel · duration, status chip (Ready, Duplicate, Invalid URL, Analysis failed) plus a reason when not ready. Expired review: `This review expired. Edit the lines and review them again.`

Shared **Type** dropdown and format chips for every ready item. The title list opens from an underlined `{N} titles` hint. Footer: **Change**, the path, **Edit URLs**, **Download N videos**. N is ready count, not pasted count. Disabled unless the token is still valid, ready ≥ 2, and a folder is set. Success navigates to Queue. Edit URLs returns to the field without losing the text.

**FFmpeg.** Most video plans and all MP3 plans require it. Missing FFmpeg opens a dialog with Open Settings, and notes that original audio without merging remains available.

### Queue

Queue is operations. Master-detail: dense rows on the left, inspector on the right.

**Empty.** Header `Queue` stays, with `{occupied} of {limit} slots in use` beside the title. Empty well: `Nothing here yet` / `Paste a link on Home.` / `In-progress downloads show up here.`, and **Go to Home**. Pause all and Resume all wait until there is a list.

**Master (left).** Header `Queue` with the same slot line, plus ghost **Pause all** and **Resume all**. Pause all is enabled only when the queue view authorizes it. Resume all is enabled when any visible job or collection authorizes Resume. Then:

- Standalone incomplete jobs
- Collection parents as a single row: playlist still when the parent or a child has one, otherwise a count badge. A **Playlist** or **Batch** tag sits on the title
- Completed work leaves the list (it lives in Downloads)

A job row is thumbnail, title, progress bar, and a mono tail (speed · ETA, paused, waiting · slot, or the failure heading). Actions do not sit on the row. Selecting a row drives the inspector. Space toggles pause/resume on the focused job.

**Inspector (right).** For a video: title, still, metadata, Speed / ETA / Progress / Status, bar, then authorized actions. Failed jobs add the error card and **Copy details**. For a playlist: title, still, `Playlist · N/M finished`, live children, then authorized parent actions. Batch collections stay count-badge only. Cancel and Remove confirm: completed files remain on disk.

Action buttons are the authorized set for that job or collection. Labels that exist when authorized: Pause, Cancel, Resume / Restart download, Retry, Download again, Start again, Open source, Copy link, Review, Open, Remove. Start again sends the URL back to Home for re-analysis.

Pause all copy on success: `Pause requested for N jobs.` The product does not claim every job paused instantly. Footer: `Jobs are saved automatically.` when persistence is healthy.

Occupancy is a real product fact (N of M active slots). It sits beside the Queue title and in the status bar. Do not invent a second scheduler UI.

**Lifecycle labels** (do not collapse these into generic Downloading): Queued, Preparing, Downloading, Waiting for processing, Finalizing, Ready to publish, Publishing, Pausing, Paused, Canceling, Failed, Canceled, Completed, Action required, Cleaning up.

Progress, speed, and ETA are live presentation. They do not authorize buttons.

Failed rows show a category heading and guidance from the backend. The next step is whichever capability is actually on.

### Downloads

History of completed files. Header `Downloads` and search. Search across title, channel, filename, quality, container, and playlist/collection title.

Rows group under Today / Yesterday / Earlier. Playlist and batch memberships collapse into a parent row (`Playlist · N episodes`) that expands to episodes.

Each item row: thumbnail, title, quality · container · size · channel · relative time. A chevron marks that the row expands. **Open** is the primary button. **Reveal** stays a ghost. Clicking the title area expands details: absolute path, Remove (history only), Delete file (history + disk + matching queue row).

A finished row whose file is gone stays in the list. Open, Reveal, and Delete file stay enabled. If the path is gone, VidStow says that file is no longer at this path.

Empty: header and search stay. Empty well: `Nothing here yet` / `Paste a link on Home.` / `Finished files show up here.`, and **Go to Home**. Search miss: `No matching downloads` and **Clear search**.

Confirmations:

- Remove from history: media file on disk stays untouched.
- Delete file: permanently deletes the file and removes it from Downloads and the queue.

### Settings

Grouped cards. Uppercase micro-headings. Each row is label + control, description in secondary type. Settings save immediately. Do not add a page-level Save button.

**General**

- Default download folder, Show in Finder, **Change Folder**. Show in Finder is hard-coded on every OS.
- Create a subfolder for each download (switch). Playlist admission still uses a playlist folder under that root.
- Confirm before starting downloads (switch, off by default)
- Interrupted jobs are a fixed value, `In progress continues`. The download that was in progress continues on launch. Waiting stays waiting. Paused stays paused.

**Performance**

- Maximum concurrent downloads 1–10, default 2, FIFO. Reducing the limit waits for active jobs. It does not pause them. Warn when concurrency is greater than 4.

**Advanced**

- Engine & dependencies: Ready / Missing, Recheck Dependencies, ytdlp-go version underneath.
- FFmpeg path, **Change Path**.

**Diagnostics**

- Send operational diagnostics: **Send** / **Don’t send**. Neither is a silent default. Disabling deletes anything waiting to send. Privacy notice link.
- Support report: Copy diagnostics, Clear history.

### About

About is the colophon at the bottom of Settings, not a separate sidebar page. Product name, version · Apache-2.0, platform `os/arch`. View source, Read the docs. One built-with line (Go, Wails, Svelte, ytdlp-go, FFmpeg). The About route still mounts this same Settings page.

Never display the string `yt-dlp` as the engine name. The branded engine is `ytdlp-go`.

## Dialogs and system states

**Unsupported URL.** Title `Unsupported URL`. The link must be a valid, publicly accessible YouTube video, Short, or playlist.

**FFmpeg required.** Title `FFmpeg is required for most downloads`. Install it or point VidStow at it in Settings; original audio that does not need merging can still be queued. Primary: Open Settings.

**Diagnostic consent (first launch).** `Help improve VidStow?` Don’t send / Send diagnostics. Closing without a choice leaves automatic sending off. Privacy notice link. No preselected choice in Settings until the user has chosen.

**Quit.** `Quit VidStow?` Active downloads will continue on reopen. Keep working / Pause downloads and quit / Quit. Pause downloads and quit records paused intent first.

**Destination conflict.** `Choose a new filename`. VidStow will not replace the existing file. Authority is the opaque conflict token, never the displayed filename.

**Action required review.** Eyebrow `Action required`, heading and item title from the backend, then only the buttons the review model enables: Remove from queue, Discard saved data, Retry cleanup, Try recovery again, Retry with fresh link, Start over from Home. Closing the dialog leaves the row in the queue. Start over returns the URL to Home.

**Cannot save.** `VidStow cannot save this session`. The data folder is not writable or is unsafe. Open data folder. No sidebar, no queue.

**Junk queue.** Unreadable or untrusted `state.json` is quarantined. Settings are salvaged when possible. The workstation starts healthy with a dismissible notice: `The saved queue could not be read. Files on disk were not touched.`

**Collection cancel/remove.** Confirm. Completed files remain on disk.

## Visual design

The shipped look is a zinc-dark workstation with a blue accent. Precise, quiet, technical, not playful and not skeuomorphic. Palette and type tokens live in `frontend/src/styles/global.css`.

Sans: Geist, then system UI. Mono: Geist Mono, then the system mono, for paths, URLs, versions, occupancy, speed, and ETA. Body type is small on purpose. Do not jump to marketing sizes.

Tight radii. Dense lists, not large cards. Native window chrome: Dark Aqua on macOS so traffic lights sit on the zinc canvas. Linux and Windows keep native window controls.

Stroke icons. The product glyph is an arrow into a tray, not a play button and not a YouTube clone.

Honor `prefers-reduced-motion`. No page-route animation, no skeleton shimmer, no confetti.

## Interaction patterns

**Switches** for binary Settings rows. Do not use Off/On pills.

**Type** on Home is a closed select. Plans for that kind are compact chips. Container, codecs, and size sit under the channel line, not next to **Download**.

**Primary button** is blue (**Download**, **Download N videos**, **Open**, Quit). **Analyze** is a light query chip in the URL field, not a second blue. Destructive actions are outline/danger text.

**Lists are the product.** Rows are the scan unit. Do not turn jobs into large cards.

**Busy and idempotence.** Analyze, add, and batch start disable while in flight. After add, the same identity should not be re-queued until something changes.

**Toasts vs dialogs.** Success and short info go to the banner. Failures that need a decision go to a modal.

**Confirm only when the action is destructive or ambiguous.** Adding to the queue does not confirm unless Confirm before starting downloads is on, or a playlist has more than 100 selected videos. Canceling a collection does. Deleting a file does. Discarding saved session data does.

**Capability + token.** Every mutating queue control is disabled unless both the capability flag is true and the command token is valid. Persistence failure strips capabilities.

## Copy voice

Plain, specific, slightly dry. Sentence case. No exclamation marks. No emoji. No "Oops". No "Your download journey".

Good: `Added 12 downloads to the queue`. `FFmpeg is required for most downloads`. `VidStow will not replace the existing file.`

Bad: `Let's get started!` `Powerful downloads made simple.` `We couldn't process your request.`

Use "download" as the user-facing noun, "queue" for in-flight work, "Downloads" for history. Use "video" and "audio", not "asset" or "media item".

Engine name in the UI: `ytdlp-go`. Product name: `VidStow` (capital V and S).

Errors shown to the user are bounded presentation strings from the backend. Do not dump stack traces, raw paths unless the user is in a path field or history detail, or engine stderr.

## Accessibility

- Every icon-only control has an `aria-label`.
- Dialogs are `role="dialog"` with labelled title, focus trap, Escape to close when not busy.
- Analyze field, search fields, and range inputs have visible or visually-hidden labels.
- Progress bars expose min/max/now.
- Active sidebar item is `aria-current="page"`. Selected queue row is `aria-selected`.
- Batch review uses a live summary.
- Keyboard: buttons and inputs are native. Visible accent focus rings.
- Thumbnails are empty `alt`. Status is in adjacent text.

## Platform behavior

Design the same IA on macOS, Linux, and Windows. Differences:

- File manager verb (Finder / Explorer / folder). Settings currently says Show in Finder on every OS. Downloads says Reveal.
- Path separators and path truncation in the status bar and history.
- Window controls stay native. Do not draw fake traffic lights.
- Drag-and-drop URLs onto the window.

This is a desktop app. Do not pretend to be a phone layout. Below ~760px the queue inspector stacks under the list.

## What a UI rebuild may change

- Spacing, type scale, icon set, and visual polish
- Platform-correct file-manager labels
- Internal component structure

## What a UI rebuild must not change

- Analyze-then-admit workflow
- Supported URL kinds and the 2–20 / 500 caps
- Backend-authored plans, capabilities, and tokens
- Conservative recovery and action-required honesty
- No-account, local-first data story and diagnostic consent
- Route set: Home, Queue, Downloads, Settings, About
- Cross-platform product intent

Build a beautiful professional desktop tool around those constraints. Do not add features to fill the window.
