# Android TV navigation audit — September 7, 2026

Scope: Now Playing, its queue and dock, Settings categories, and shared TV
controls. Impeccable native audit; source review only. `adb devices` returned
no connected devices. No screenshots, contrast measurements, frame timings,
or actual remote-navigation results are claimed. No mobile layout changes.

## Platform conformance

Source-level pass: native Compose/Material controls, Back handling in Now
Playing, separately focusable dock buttons, and TV-specific settings categories.
Device acceptance remains unverified. Current PRODUCT/DESIGN documents describe
older web behavior; the user's current minimal, content-first requirements and
incumbent native implementation take precedence. No document migration performed.

## Provisional source score

| Dimension | Score | Evidence / limitation |
| --- | --- | --- |
| Accessibility | 2/4 | Icon labels exist; some active states and speed purpose lack semantics. |
| Performance | 3/4 | Queue/categories virtualized; lifecycle-aware collection; no device timings. |
| Appearance | 3/4 | Semantic Material colors and shared focus borders; contrast unmeasured. |
| Platform conformance | 3/4 | Native controls, Back and focus groups; remote behavior unverified. |
| Adaptivity | 2/4 | Fixed category/card dimensions need scaled-text validation. |
| Total | 13/20 | Provisional; not a device quality rating. |

## Findings

### P2 — Queue removal has no explicit focus handoff

`TvPlaybackQueue.kt`: Remove calls `removeQueueItem` and dismisses the popup,
but leaves the position store's focused identity pointing to the removed item.
`TvWideMediaCard` restores only a matching identity. No remaining card can use
that explicit restore target. Compose may choose a fallback, but the app does
not ensure which one. This can make repeated queue editing unpredictable.

Recommendation: select the next surviving item (or previous at the end),
update focus identity, and restore it after list recomposition. Test removal
at the beginning/middle/end with the popup open. Command: impeccable harden.

### P2 — Active icon controls rely on color for state

`TvControls.kt`, `TvIconButton`: `active` changes tint but supplies no selected
or state description semantics. Shuffle/repeat labels communicate state in
some call sites, but the shared component does not consistently expose it.
Keep those labels and provide a shared semantic state where applicable, without
announcing ordinary push buttons as toggles. Command: impeccable harden.

### P2 — Speed value lacks an accessible purpose

`TvNowPlayingScreen.kt`: the speed action renders a numeric multiplier through
`TvTextButton`, whose semantics expose selection but no purpose such as
"Playback speed, 1.5 times". A screen-reader user hears a value without enough
context. Add an explicit accessibility label at this call site. Command:
impeccable harden.

### P2 — Scaled-text fit needs a device check

`TvSettingsCategories.kt`: 56dp category cards have two text rows and 12dp
vertical padding. `TvMediaGrid.kt`: wide cards have fixed 70dp height with
multi-line titles. These fixed bounds do not grow with font scaling. Actual
clipping is not confirmed; test 1.3x font scale before choosing between bounded
text and adaptive minimum heights. Command: impeccable adapt.

## Preserve

- Separate player controls; do not turn the whole dock into a single focus target.
- Lifecycle-aware queue/transport subscriptions and isolated clock composition.
- Stable keys and lazy lists; no new permanent menus or instructions.
- Material semantic colors, visible shared focus borders, categorized settings.

## Next pass

1. Harden queue focus handoff and accessible state/purpose labels in one batch.
2. Validate on TV: popup dismissal/removal, Back, queue replacement, long titles,
   large font scale, light/dark Ubuntu themes, and focus after changing categories.
3. Adapt only confirmed layout failures; keep mobile unchanged.
4. Impeccable polish as the final bounded confirmation, not an open-ended redesign.

The broad goal already authorizes implementation; this audit does not require
a new approval. Hardware verification remains pending the next available TV.

## Implementation follow-up

Queue removal now records the next surviving item (previous at the end) as
the explicit focus target before the next queue composition. Unit tests cover
first/middle/last removal and empty results. Shuffle/repeat opt into selected
semantics on the shared TV icon button; ordinary buttons retain their existing
semantics. Playback speed has a resource-backed descriptive accessibility label.
Direct/Store TV unit suites and the module-size gate pass. No rendered or
TalkBack verification is claimed without a connected TV. Scaled-text checks
remain pending; mobile layout and visual TV styling were not changed.

## Follow-up: Now Playing interaction paths

September 7 source review of `TvNowPlayingScreen.kt`, `TvPlaybackQueue.kt`,
`TvInputNavigation.kt`, the shared grid focus store, and playback queue mutations.
ADB still lists no device. This is an Impeccable native audit, not an implementation
or device acceptance pass; no updated whole-screen score is justified.

### P2 — On-screen Back and remote Back disagree inside Queue

In `TvNowPlayingScreen.kt`, `BackHandler` first clears `showingQueue`, but the
visible header Back button invokes `onBack` directly. With Queue open, selecting
that button leaves Now Playing rather than returning to the transport. This
is a verified difference between callbacks, not an inferred Compose focus bug.
It makes the navigation hierarchy depend on which Back control the user chooses.

Recommendation: share one back action for both paths: close Queue first, otherwise
return to browsing. Keep Close player separate so ordinary Back does not stop
audio. Add tests for both queue states and verify remote focus restoration.
Suggested command: impeccable harden.

### P2 — Seek semantics do not convey purpose or readable playback time

`TvNowPlayingTimeline` gives Material Slider a millisecond range and a focus
border, with no purpose label or time-valued state description. Adjacent elapsed
and duration text does not explicitly label the slider. Material supplies range
semantics, but source cannot prove how a particular TalkBack version announces
them; do not claim it necessarily reads raw milliseconds.

Recommendation: expose a resource-backed playback-position label and elapsed /
duration state description, distinguishing unavailable duration. Keep Material's
range actions and the separate timeline subscription. Test missing duration,
long audiobooks, track changes, and remote seeking. Suggested command: impeccable
harden.

### Preserve and remaining uncertainty

- Transport focus is intentionally requested only on initial entry and Queue close,
  not every track transition; existing unit coverage protects that policy.
- Queue uses lazy keyed rows and the earlier removal handoff; this review does
  not claim an observed failure of that fix.
- Close player explicitly stops playback before leaving; ordinary navigation
  must retain background listening.
- Popup focus, queue-boundary scrolling, scaled text, and TalkBack still require
  a connected TV. Finish the implementation batch with a bounded impeccable
  polish pass when that hardware is available.

### Follow-up implementation (pending APK batch)

The header and system Back now share a queue-first callback. If playback has
ended and the empty screen is visible, Back returns directly to browsing rather
than consuming a hidden queue state. Close player remains a separate stop action.
The seek slider now exposes resource-backed purpose and elapsed/duration state
text, including unavailable duration, without replacing Material range actions.

Direct and Store TV unit suites pass after Kotlin/resource compilation. New
tests exercise queue-first callbacks, shared source wiring, negative time, and
25-hour audiobook formatting. The module-size gate passes. These checks do not
prove TalkBack announcement behavior or actual remote focus; those remain device
acceptance items. No APK was rebuilt or published, and no layout was changed.
