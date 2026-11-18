# Web music player technical audit

Impeccable audit of audio-player-template.js, audio-player-chrome.js,
audio-player-queue.js, audio-transport-bindings.js and audio-player.css.
Source review plus a stub-DOM execution of the actual chrome module.
No rendered screenshots, frame timings, contrast ratios or assistive-technology
results are claimed. Existing compact styling remains authoritative; no redesign.

## Implementation integrity

Provisional pass: the player uses shared theme tokens, labeled native controls,
bounded queue rendering and separately owned chrome/transport modules.
The bundled detector returned an empty finding list for the template, CSS and
queue module. That is not evidence of overall visual quality or accessibility.

| Dimension | Provisional score | Evidence / limitation |
| --- | --- | --- |
| Accessibility | 2/4 | Native buttons/range, but sleep purpose and readable seek value are missing after interaction. |
| Performance | 2/4 | Queue capped at 60 rows; timeline redundantly replaces icons every tick. |
| Responsive | 3/4 | Mobile breakpoints and safe-area declarations exist; final cascade and scaled text need rendered checks. |
| Theming | 3/4 | Semantic surface/text/accent tokens dominate; contrast remains unmeasured. |
| Integrity | 3/4 | Product-specific controls and modular ownership; repeated CSS overrides complicate verification. |
| Total | 13/20 | Provisional source assessment, not a visual-quality rating. |

## Verified findings

### P2 — Repeated icon replacement during playback

Location: audio-player-chrome.js, updateTimeline; called on timeupdate in
audio-transport-bindings.js. Both play controls receive new SVG innerHTML even
when playing/paused state is unchanged. A stub-DOM run of 100 playing timeline
updates counted 200 icon replacements. This is unnecessary DOM work; no
measured frame-rate or battery claim is made.

Recommendation: update play-state presentation only when state changes, while
continuing timeline updates. Suggested command: impeccable optimize.

### P2 — Sleep action loses its purpose after activation

Location: audio-player-template.js and audio-player-chrome.js, cycleSleep.
Initially the button contains “Sleep timer”; after the first click, textContent
becomes only “15m”, and when disabled it becomes “Zz”. The button has no
independent accessible name that retains its purpose. This makes the action
less understandable to screen-reader users after interaction.

Recommendation: retain a purpose-bearing accessible name including timer state.
Suggested command: impeccable harden.

### P2 — Seek slider lacks a human-readable time value

Location: audio-player-template.js range input; audio-player-chrome.js timeline.
Its numeric range is 0–1000. updateTimeline changes the raw value but never
sets aria-valuetext. The execution probe confirmed it remains absent.
Users navigating the slider with assistive technology get implementation-scale
progress instead of the displayed elapsed/duration times.

Recommendation: expose readable time and distinguish unavailable duration.
Suggested command: impeccable harden.

## Preserve and verify

- Keep compact controls, theme tokens and artwork; add no permanent explanatory UI.
- Keep native range keyboard behavior, pressed states and queue Escape handling.
- Keep queue DOM bounded to 60 rows and event delegation.
- CSS has several historical overrides, including safe-area versus important
  padding declarations. Check effective styles in a batched desktop/mobile
  browser pass before changing spacing; no rendered overlap is confirmed here.
- No P0/P1 findings were established in this bounded review. That does not prove
  the player is free of blocking bugs.

## Next actions

1. Impeccable optimize: remove redundant play-state DOM updates.
2. Impeccable harden: preserve sleep-action purpose and readable seek values.
3. Impeccable polish: one batched desktop/mobile inspection and at most one
   confirmation pass, without changing the established visual identity.

The existing goal authorizes this focused follow-up. The commands can also be
requested individually or in another order. Re-run the audit after fixes to
reassess the provisional score.

## Focused implementation follow-up

All three verified findings are addressed in source. The same 100-tick probe
now performs two icon replacements rather than 200; state transitions still
update both icons. Seek values expose elapsed/duration text, and sleep labels
retain purpose and reset after expiration. Tests include invalid duration and
timer expiry. All 440 frontend tests pass; module gate and detector pass.

A single batched Chromium component check at 1440px and 390px confirms the
actual template/module expose “1:12 of 5:00”, “Sleep timer: 15 minutes”, and
“Pause”. Screenshots were inspected. This used isolated component CSS and
synthetic theme values/content, not the full authenticated app, so it does not
establish whole-app appearance, contrast, frame-rate or screen-reader quality.
No CSS/layout changes were made.

## Deployment verification

Deployed September 7, 2026 at 15:39 UTC, recreating only the opdshelf service
from the completed server build. Running image:
`sha256:017b517ab7741c2c92c6f31a0e4509d819c4d24e5ccadfdb3677a550cb888fe0`.
The container reports healthy with zero restarts and `/healthz` returns `ok`.
All ten libraries loaded; the existing malformed light-novel EPUB remains a
non-blocking scan warning. No media files were modified.

The mounted Android APK remains 0.16.72, SHA-256
`559d17d7ef60f241471d7547ab6776335beeb9ece55e3d5b655036f9703525a6`.
No Android build or UI change was included in this deployment. Full-app
browser and physical-device accessibility checks remain outstanding.

## Queue interaction follow-up

Impeccable hardening identified that replacing the queue rows during sync
discarded keyboard focus. Rendering now remembers the focused track from the
previous bounded page and restores its replacement by identity after reorder.
If removed, the nearby visible row receives focus; an empty list falls back to
Close. Opening a queue without a visible active track also has a focus fallback.
Updates do not steal focus from controls outside the track list. No layout or
Android UI changes were made.

Four new regression tests cover reorder, removal/empty queues, external focus,
and Escape. All 444 frontend tests and the module-size gate pass. A batched
Chromium DOM interaction check at 1440px and 390px confirmed identity restoration,
the 60-row bound, and empty-list focus. This used a minimal component fixture,
not the authenticated app or a physical assistive-technology test. The changes
were deployed September 7 at 15:55 UTC in the server batch documented in
`AUDIO_QUEUE_INTEGRITY_2026-09-07.md`; the Android APK remains unchanged.

## Playlist form hardening

A delayed rename failure could reset selection after leaving the playlist or
returning to it in a new visit. Selection generations now guard both success
rerenders and failure recovery. Current failures reveal the existing retry form
after rendering, so its error is not hidden. Create and rename submissions no
longer assume a non-null SubmitEvent.submitter. Three regression cases cover
late failure after navigation, visible current errors, and submitter-less
creation. No layout changes or full-browser visual claims accompany this
Impeccable hardening pass.

The repeatable `node scripts/browser-playlist-smoke.mjs` check now exercises
the actual controller and playlist template with native Chromium forms at
1440px and 390px. Its first run caught attempted rename text being lost after
failure; recovery now preserves that input. One confirmation batch passed both
widths, including submitter-less requestSubmit, visible retry text, and late
failure after navigation. All 448 frontend tests pass. The browser fixture uses
synthetic playlists and intercepted local module requests; it does not modify
personal libraries or establish full-app layout/contrast quality.

This browser check is now available as `npm run test:browser:playlists` and is
wired into the frontend CI job with a Chromium installation step. The workflow
has not been executed on GitHub in this session; the command passes locally.
Playlist fixes deployed September 7 at 16:21 UTC in image
`sha256:3a9f9a625923bd82d7da1fdf3ac00134d8e889111523513f7002e3ac93941ff7`.
The container is healthy with zero restarts and `/healthz` returns `ok`.
No Android package was rebuilt.
