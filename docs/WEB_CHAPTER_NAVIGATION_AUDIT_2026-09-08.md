# Web audio chapter navigation audit

Follow-up fixes deployed September 8 at 14:21 UTC; see `WEB_CHAPTER_RELEASE_2026-09-08.md`. Source-only notes below are chronological.

Impeccable technical audit after release 0.16.82. Scope: shared destination rail and audio chapter picker/template/styles. Report only; no production UI changes. The strongest reproducible findings are in the chapter picker, shared with spoken-audio playback.

## Implementation integrity verdict

Provisional pass: the implementation is product-specific, uses native buttons, explicit playback labels, shared theme tokens and a separate chapter controller. The detector returned `[]` for the inspected rail CSS, rail JavaScript, chapter controller and player template when run with JSON output. This is not proof of visual quality or accessibility. DESIGN.md still describes older product/layout rules; current code and the user's latest compact-layout instructions take precedence. No design artifacts were repaired.

## Audit health

| Dimension | Provisional score | Evidence / limit |
| --- | --- | --- |
| Accessibility | 2/4 | Labeled native controls, but chapter open/close fails focus handoff and Escape. |
| Performance | 3/4 | Chapter DOM updates gated by active index; no device timing or large-list benchmark. |
| Responsive | 3/4 | Existing mobile insets, 44px chapter close and 52px row minimums; full text-scaling acceptance unmeasured. |
| Theming | 3/4 | Chapter controls use text/muted/accent/surface tokens; contrast not measured. |
| Integrity | 3/4 | Focused shared components, no detector findings; stale design documentation limits documentary authority. |
| Total | 14/20 | Provisional good, with specific navigation gaps—not a visual-quality or WCAG certification. |

## Findings

### P1 — Chapter picker loses keyboard context

Location: `internal/web/static/audiobook-chapters.js`, toggle/close callbacks; `audio-player-template.js`, chapter panel.

Opening closes the More disclosure containing the focused trigger without focusing the chapter list or Close. Chromium at 1440px and 390px reports focus outside the panel after keyboard activation. Focusing Close and pressing Enter leaves the focused node inside the now-hidden panel. The controller has no Escape handler; Escape left the panel open in both checks. Keyboard users must rediscover their place rather than continuing the task.

Recommendation: focus the active chapter (or Close when empty), restore a visible player control on close, and add scoped Escape dismissal without swallowing unrelated keyboard controls. Preserve existing player expansion and playback behavior. Suggested command: impeccable harden. Actual screen-reader announcements remain unverified; no specific conformance claim is made.

### P2 — Opening does not reveal the current chapter

Location: `internal/web/static/audiobook-chapters.js`, `update()` early return and toggle callback.

`openItem()` computes the active index while the panel is hidden. Opening calls `update()` with the same index, which returns before `scrollIntoView`. The browser fixture at both widths used 80 chapters and playback in chapter 61: the active row was outside the panel and scrollTop remained zero. Users have to search down a long list despite the app knowing their chapter.

Recommendation: separate chapter-state updates from the explicit reveal-on-open action. Scroll only the picker viewport and do not shift the underlying library. Suggested command: impeccable harden.

## Evidence and positives

`/tmp/bobobooks-chapters-audit.mjs` loads actual player template, chapter controller and app/theme/player CSS through a synthetic Chromium fixture at 1440x900 and 390x900. It measures focus, active-row bounds, scroll position and Escape behavior. It does not use real audio decoding, personal data, screenshots or assistive technology. No production files changed.

Preserve the borderless compact destination rail, current visual identity, labeled buttons, native range behavior, chapter timestamps, and existing per-index rendering guard. The rail uses roving tab stops and arrow/Home/End navigation; its buttons have 44px minimum dimensions. No new navigation bars or persistent explanatory text are warranted by these findings.

## Next actions

1. Impeccable harden: chapter focus restoration, scoped Escape and current-chapter reveal; include behavior regressions.
2. Impeccable polish: one bounded desktop/mobile confirmation, preserving the established layout.

These can be run individually, together, or in a preferred order; repeat the audit after fixes. The existing goal authorizes the focused follow-up without a redesign.

## Focused hardening follow-up

Implemented after 0.16.82, source-only. Opening reveals the current row within the chapter list and focuses it, or focuses Close if no chapter matches. Close restores a visible More summary only when focus was inside the panel; background calls do not steal outside focus. Escape is scoped to the panel. The trigger now exposes expanded/controls state and the section has a Chapters label. Existing styling and playback behavior are preserved.

`npm run test:browser:audio-chapters` is a repeatable Chromium regression using actual modules/styles. It passes at 1440px and 390px for active-row visibility/focus, unchanged document scroll, Escape, close restoration, missing-current fallback and outside-focus preservation. The initial test caught a statement-boundary error in the new code, corrected before the successful run. The deferred-controller unit stub was updated to represent the newly used DOM interfaces. All 555 frontend tests and the module-size gate pass (`/tmp/bobobooks-chapter-focus-tests.log`). No screenshot/contrast, physical assistive-technology or full live-app acceptance claim is made. No APK or deployment.

## Loader recovery and CI

The browser chapter regression is now in the frontend CI job and selects a configured/system Chromium when available, otherwise Playwright's installed browser. It passed locally; the GitHub workflow itself has not been executed here.

Deferred loading no longer retains a rejected promise or leaves its rejection unhandled. A later open retries the module loader; only the latest active chaptered item receives initialization or an existing-status-area error. Closing or switching to music suppresses obsolete failures. Tests inject the loader to verify failure/retry, shared in-flight loading and stale-failure suppression; these are not a browser module-cache/network-recovery guarantee. Playback is not stopped and no popup is added.

All 558 frontend tests, the module-size gate and both-width chapter browser checks pass (`/tmp/bobobooks-chapter-loader-tests.log`). These changes remain source-only after 0.16.82.
