# Authenticated browser-session design walkthrough

These PNGs are direct exports of the ten final boards on the Penpot page
`Authenticated Browser Cookies · Final UX`
(`8e6408d8-b028-80e2-8008-86817ce6ef5a`). They are review artifacts rather
than runtime assets.

| # | File | State and contract illustrated |
| --- | --- | --- |
| 01 | `01-public-default.png` | Every new URL starts in **Public only** mode; browser access is never selected silently. |
| 02 | `02-browser-setup-consent.png` | Backend-discovered source selection, explicit consent, macOS prompt warning, and a disabled preflight until consent. |
| 03 | `03-source-check-remediation.png` | A denied/unreadable source fails visibly without profile substitution or a public retry. |
| 04 | `04-video-review-session-supplied.png` | Single-media review identifies request mode and exact source without claiming that YouTube required or accepted the session. |
| 05 | `05-playlist-explicit-outcomes.png` | Stable per-occurrence Ready/Auth required/Unavailable/Invalid outcomes, independent duplicate occurrences, approved count, source order, and the 500-occurrence safety bound. |
| 06 | `06-queue-collection-recovery.png` | Exact parent counts and bound source remain visible; one child can require isolated recovery without recreating completed siblings. |
| 07 | `07-settings-browser-access.png` | Configure, check, and forget controls; a successful local check does not attest YouTube login. |
| 08 | `08-forget-blocked-active.png` | Forgetting cannot remove authority underneath an active operation; the user must pause first. |
| 09 | `09-action-required-decision.png` | A temporarily unusable bound source offers explicit retry, removal, cleanup, or public restart decisions. |
| 10 | `10-bound-source-unavailable.png` | A missing source is never replaced by the currently selected Settings profile or by anonymous access. |

All boards are 1180 × 760. The design page remains the editable source of truth;
these checked-in exports make the reviewed states durable and visible in the
pull request and repository history.
