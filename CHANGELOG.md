# Changelog

All notable user-visible changes to VidStow are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versions follow semantic versioning where practical for a desktop
application.

`v0.1.0-beta.1` is the first public VidStow prerelease. The current packaged
preview is `v0.1.0-beta.5` for macOS Apple Silicon; a later stable `0.1.0`
does not exist yet.

## [Unreleased]

### Added

- Dark zinc workstation with Home, Queue, Downloads, and Settings. The product
  name lives in the native title bar; the sidebar starts at Home and no longer
  repeats the wordmark or an About item.
- Home Try chips that fill the field with a sample video, playlist, or several
  links. Analyze is still the only submit.
- Analysis shows its current step, a slow-request explanation, retry, and a
  stop-waiting action. Edited links and interrupted requests ignore late results.
- Download review prevents duplicate submissions, preserves choices on start
  errors, and explains missing folders, empty playlist selections, and expired
  batch reviews. A batch with one ready video can continue as a single download.
- Confirmation dialogs trap keyboard focus, restore it on close, and preserve
  follow-up error dialogs. Unavailable artwork embedding is disabled explicitly.
- Queue separates in-progress work, attempts needing attention, and collapsible
  canceled attempts. Rows show output quality and recovery actions; idle headers
  explain failed/canceled counts and hide irrelevant bulk controls. Settled
  cancellations no longer show progress or transfer statistics.
- New download failures preserve safe stage, reason code, HTTP status when
  available, and timestamp across restarts, available under Failure details.
- Downloads rows with relative time, a chevron to expand, Open as the primary
  action, and Reveal as a ghost.
- A clickable status-bar folder path that reveals the download folder. `Not set`
  is not clickable.
- Settings colophon (version, license, platform, source, docs, built-with) in
  place of a separate About page. The About route still renders Settings.
- Cannot-save stop when the data folder is unwritable or unsafe. An unreadable
  queue is quarantined and skipped, with a dismissible workstation notice.
- Home video and playlist docks use inspector rows for Output, Subtitles
  (Off / File / Embed, language, sidecar format), and In the file (title,
  artwork, chapters). Choices seed from Settings Video files defaults, clamp
  when FFmpeg is missing, and ride with video admissions only. Audio greys
  Subtitles in place; captions are not written on audio downloads. Durable
  jobs keep their output choices across retries and relaunch, and the queue
  inspector shows a note for non-default choices.
- Home video chips split 30fps and 60fps when Analyze reports both at the same
  height (`1080p` and `1080p60`). The identity line shows `60 fps` on the high
  rate. Audio is unchanged.

### Changed

- Queue rows use labeled action icons, larger thumbnails, and a separate quality/status
  line. Keyboard-accessible row buttons select details without starting work. Failure
  recovery replaces empty transfer statistics, and narrow windows keep actions visible.

- Home identity puts codec and size on one receipt line. Audio chips use short
  labels (`M4A`, `MP3 128`). Disabled Subtitles shows **Video only** and keeps
  File remembered without the blue selected chrome.
- Ordinary quit leaves in-progress downloads running, the same as a crash. Pause
  downloads and quit is the path that records paused intent first.
- Interrupted jobs are a fixed In progress continues policy. Waiting stays
  waiting. Paused stays paused.
- After Download on Home (single video or playlist), VidStow goes to Queue.
  Batch already did.
- The macOS window uses Dark Aqua so traffic lights and the title sit on the
  zinc canvas.
- Start over from Action required returns the URL to Home.
- Queue inspector actions use the shared app buttons and sit on one wrapping
  row.

### Removed

- **Confirm before starting downloads.** Home **Download** queues immediately.
  A playlist with more than 100 selected videos still asks first.

### Fixed

- Queue rows no longer report **Video unavailable** when a download includes
  subtitles or embedded extras. Those jobs skip the resume session, which cannot
  publish sidecars, and use the classic save path instead. Embedded artwork is
  skipped for now: the engine treats every YouTube thumbnail as the same file
  and aborts the download.
- Failed downloads that never started say **Download could not start**, not
  **Video unavailable**. **Retry** stays the next click. **Open source** and
  **Copy link** are not offered for that failure.
- **Retry** of a subtitles or extras download replaces that row’s leftover
  file after the new attempt finishes, using the options already queued.
  A leftover collision says **File already in the folder**, not **Nothing
  was saved.**
- Action required with no leftover file uses the same inspector as a failed
  start: **Retry** and **Remove**. Review is reserved for leftover saved data.
- A YouTube refusal no longer says **Sign-in required**. It says **Download
  was refused**, and **Retry** is the next click. **Open source** still opens
  the page. VidStow cannot sign in to YouTube.
- Canceled jobs no longer sit on **Cleaning up** after leftover temp data is
  gone. The row says **Canceled**, and the inspector no longer offers **Pause**.
- Queue confirms use the compact dialog chrome. Action required no longer
  uses a red **Action required** eyebrow.

- Active and queued downloads stay above completed work when collections and
  standalone videos are mixed in the queue.
- Action-required downloads can be dismissed from the queue without retrying
  or deleting their preserved temporary data. Rows with pending cleanup remain
  visible until that cleanup settles.

## [0.1.0-beta.5] - 2026-08-22

### Added

- Home can review 2–20 pasted public YouTube video or Short URLs, identify
  invalid and duplicate lines, and, when at least two items are ready,
  atomically admit all ready items under one durable, expandable queue parent.
  Child order, completed-sibling outcomes, and aggregate completion progress
  are preserved across failures and app restarts.
- VidStow keeps a bounded, owner-only history of sanitized terminal failures for
  Copy Diagnostics and Clear history. A separate explicit consent choice can
  send those allowlisted reports in the background; neither choice is
  preselected, and reports exclude video identities, links, paths, filenames,
  cookies, tokens, and error text.

### Changed

- The packaged application version is `0.1.0-beta.5`.
- Failed queue items show category-specific guidance and capability-backed next
  steps, including Start again for disk-space and folder-permission failures.
- The macOS window uses a light native title bar so its title remains legible
  above VidStow's light application shell.
- Migrated the engine dependency and package imports to the published
  `github.com/tejasa97/ytdlp-go v0.3.0` module path.

### Fixed

- Startup waits for the saved queue check instead of briefly reporting recovery,
  ignores stale workspace identities for terminal jobs, and clears a leftover
  recovery marker only when the committed state is provably intact.
- Completed queue rows no longer show stale transfer speed or time-remaining
  values and consistently report full progress.

## [0.1.0-beta.4] - 2026-08-18

### Changed

- The packaged application version is `0.1.0-beta.4`.
- Action-required Review can try recovery again, retry with a fresh link, or
  discard saved data. Uncertain session evidence is not reused.
- Queue rows with pending or quarantined cleanup stay visible until that
  cleanup settles. Retry cleanup is available, and Remove appears only after
  cleanup is done.

### Fixed

- Deleting a file from Downloads also removes the matching completed item from
  the queue.
- Canceled rows that still carried a leftover cleanup phase show as Canceled
  and can be removed once cleanup has settled.

## [0.1.0-beta.3] - 2026-08-18

### Changed

- The packaged application version is `0.1.0-beta.3`.
- The desktop engine pin is `github.com/tejasa97/youtube_dlp v0.2.3`.
- Repeated downloads using the same YouTube player can reuse completed EJS
  preprocessing while each job retains its isolated JavaScript helper.

### Fixed

- Action-required queue rows provide a bounded review flow and can start over
  through normal analysis without deleting or reusing uncertain saved data.

## [0.1.0-beta.2] - 2026-08-17

### Changed

- README and release packaging recommend the
  [`vidstow/tap`](https://github.com/vidstow/homebrew-tap) Homebrew cask as the
  supported install path. The GitHub ZIP remains a checksum-pinned technical
  artifact.
- The packaged application version is `0.1.0-beta.2`.
- The desktop engine pin is `github.com/tejasa97/youtube_dlp v0.2.2`.

### Fixed

- Retrying a download that failed mid-transfer because its saved YouTube link
  expired no longer reuses the dead link forever: the first retry still
  resumes, and once a resume makes no progress (or the server rejects the link
  with HTTP 403) the next retry discards the saved data and restarts with a
  fresh link. After two such restarts the row explains that the item must be
  removed and downloaded again from Home.
- Mid-transfer failures now explain what Retry will do instead of surfacing
  the raw engine error.
- The macOS app icon uses Apple's 824/1024 icon grid with transparent
  padding so the Dock draws it at the same size as neighboring icons.

## [0.1.0-beta.1] - 2026-08-15

### Added

- Analysis and downloading of public, on-demand YouTube Shorts URLs.
- Automatic discovery of Homebrew FFmpeg and FFprobe when they are not on the
  application PATH.
- Initial desktop workflow for public single-video YouTube URLs.
- Analyze, quality presets, download queue, history, and FFmpeg diagnostics.
- Durable State v2 queue lifecycle, destination reservation, startup recovery,
  and bounded shutdown foundations.
- Backend-authored lifecycle queue controls and restored thumbnails.
- Packaged `ytdlp-js-helper` runtime for the macOS application bundle.
- Manual Apple-Silicon beta-candidate workflow with checksums, release
  metadata, and build-provenance attestation.

### Changed

- App icon, sidebar mark, and docs logos now use the simpler arrow-into-tray
  download glyph.
- The packaged application version is `0.1.0-beta.1`; no stable `0.1.0` release
  exists.
- Candidate packaging currently targets macOS on Apple Silicon only.
- Release automation creates a draft prerelease and cannot publish it.
- The desktop engine pin is `github.com/tejasa97/youtube_dlp v0.2.1`.

[Unreleased]: https://github.com/vidstow/vidstow/compare/v0.1.0-beta.5...HEAD
[0.1.0-beta.5]: https://github.com/vidstow/vidstow/releases/tag/v0.1.0-beta.5
[0.1.0-beta.4]: https://github.com/vidstow/vidstow/releases/tag/v0.1.0-beta.4
[0.1.0-beta.3]: https://github.com/vidstow/vidstow/releases/tag/v0.1.0-beta.3
[0.1.0-beta.2]: https://github.com/vidstow/vidstow/releases/tag/v0.1.0-beta.2
[0.1.0-beta.1]: https://github.com/vidstow/vidstow/releases/tag/v0.1.0-beta.1
