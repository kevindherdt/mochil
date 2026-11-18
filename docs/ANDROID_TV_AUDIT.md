# Android TV Experience Audit

Audit date: 2026-08-31

Scope: the native Android TV shell, Home, library browsing, settings, compact playback dock, full Now Playing surface, queue, remote/controller input, focus restoration, edge-to-edge behavior, paging, and bounded performance diagnostics.

## Audit health score

| # | Dimension | Score | Key finding |
|---|---|---:|---|
| 1 | Remote and controller navigation | 4/4 | Every primary route is D-pad reachable; Back closes one hierarchy level; shoulder buttons cycle libraries; long-press actions and explicit transport controls are supported. |
| 2 | Focus visibility and restoration | 4/4 | Focus is visually prominent and restored by stable route/item identity, with recovery when remembered content disappears. |
| 3 | Layout and ten-foot readability | 4/4 | The shell is edge-to-edge, uses an overscan-safe 56 dp content inset, compact media cards, restrained chrome, and individually focusable player actions. |
| 4 | Performance | 4/4 | Large media surfaces and the queue are lazy/virtualized, paging is viewport-driven, artwork is bounded, all route and app first-frame timings are accepted without media identity, and focus-key rebuilding is isolated from unrelated chrome recomposition. |
| 5 | Implementation integrity | 3/4 | TV responsibilities are well separated and no godfile is present, but final overscan, focus traversal, and motion quality still require a rendered pass on representative physical TV hardware. |
| **Total** |  | **19/20** | **Excellent — physical-device verification remains** |

## Implementation integrity verdict

**Pass.** Android TV is a dedicated remote-first surface rather than a stretched handset layout. Route ownership, library state, focus memory, content rendering, playback, settings categories, and queue rendering are separated into focused files. The largest TV source file is 322 lines, and the complete TV UI is divided across 23 files rather than accumulating in one monolith.

Home, library switching, alphabet navigation, Settings categories, and shared
transport controls now use the same conditional focus-visual primitive. Idle
items do not allocate scale/z-index or zero-width border modifier nodes; the
focused item retains the existing visible scale and three-pixel focus ring.

The review found no source-level P0, P1, or P2 defect. The remaining P3 is a verification gap: Compose focus traversal, television overscan, and focus-scale clipping need a rendered pass on real 720p and 1080p devices before the audit can claim a perfect score.

## Executive summary

- Audit health score: **19/20 (Excellent)**
- Issues: **0 P0, 0 P1, 0 P2, 1 P3**
- System bars are hidden again whenever the TV window regains focus, so the app remains edge-to-edge after external overlays.
- Home and library screens remember meaningful entry and item focus rather than resetting to the first card.
- Library focus state is keyed by library, destination, layout, hierarchy, sort, filter, query, and alphabet bucket; changing context does not destroy another context's position.
- When filtered or refreshed content removes the remembered card, focus is deliberately recovered to a stable navigation edge.
- Media grids, home rails, option rails, and the playback queue use lazy containers. Infinite paging is based on the visible viewport rather than a manual “load more” action.
- The compact player exposes Play/Pause, Now Playing, and Close as separate remote targets; full Now Playing progressively discloses its virtualized queue.
- Settings are categorized into Appearance, Libraries, Playback, Updates, Connection, and Diagnostics instead of presenting one long undifferentiated page.

## Detailed finding

### [P3] Final TV rendering and traversal remain unverified on representative hardware

- **Location:** Android TV application as rendered on 720p and 1080p televisions/streamers
- **Category:** Responsive design / focus navigation / motion
- **Impact:** Static source and unit tests can prove ownership and policy, but cannot fully prove directional focus geometry, overscan differences between televisions, focus-scale clipping, animation smoothness, or readability at real viewing distance.
- **Standard:** Android TV quality guidance; this is a release-verification gap, not a confirmed functional defect.
- **Recommendation:** In the next batched APK, run the remote checklist below on the Google streamer and at least one second TV profile. Send bounded performance diagnostics after cold Home, opening a large library, switching destinations, opening/closing Now Playing, and entering/exiting Settings.

## Physical-device checklist

1. Cold-start to Home: initial focus is visible and the first remote action is accepted.
2. Home to library and Back: focus returns to the library that launched the route.
3. Library switching: D-pad edge navigation and L1/R1 work without trapping focus; the prior library position is restored.
4. Large catalog: paging continues before reaching the end and does not create a visible pause.
5. Search/filter/sort/layout changes: each context retains its own scroll and focus position; removing a focused item recovers cleanly.
6. Now Playing: all transport controls are independently focusable; queue opens at the current item and returns focus to Play/Pause when closed.
7. Compact dock: Play/Pause, expand, and Close can each be selected without the entire dock swallowing focus.
8. Settings: category list, content choices, confirmation dialog, and Back are traversable without a dead end.
9. Overscan: focused card scaling and text are not clipped at any edge on both 720p and 1080p output.
10. Performance: submit diagnostics after the test sequence and compare first-frame and request timings with the previous TV build.

## Positive findings

- TV detection uses both UI mode and the Leanback hardware feature.
- The activity re-hides system bars when window focus returns.
- Stable item keys are used throughout lazy media surfaces and queues.
- Grid position memory is bounded to 96 contexts rather than growing indefinitely.
- Paging state has explicit loading, error, and retry presentation.
- Cards expose media progress and completion directly on artwork.
- Remote long-press is a reusable input primitive rather than duplicated screen logic.
- Labels describe icon actions, including active shuffle/repeat state.
- Theme colors come from shared Material roles; focus is communicated with both surface change and a strong border.
- Performance diagnostics deliberately exclude server addresses, searches, titles, and media paths.

## Recommended next actions

1. Complete the physical-device checklist in the next meaningful Android release rather than producing a build solely for this audit.
2. Use the submitted TV timings to target measured bottlenecks; do not add speculative caching or persistent chrome.
3. If the hardware pass exposes a focus or overscan defect, add a policy/unit regression test alongside the fix where possible.
4. Repeat this audit after the physical pass; a clean result can raise implementation integrity to 4/4.
