# Android Mobile Navigation Audit

Audit date: 2026-08-31

Scope: the connected native Android shell, Home/library destination rail, Saved, Sources, Settings, system Back behavior, active destination styling, compact/expanded adaptation, touch targets, and state ownership.

## Audit health score

| # | Dimension | Score | Key finding |
|---|---|---:|---|
| 1 | Accessibility | 4/4 | Material controls provide semantics and minimum touch enforcement; continuation actions also expose TalkBack alternatives. |
| 2 | Performance | 4/4 | The destination rail, catalogue surfaces, and dashboard are lazy; route-specific state subscriptions stay outside the shell. |
| 3 | Appearance & theming | 4/4 | Active Home and library chips share one semantic Material color contract across theme families. |
| 4 | Platform conformance | 4/4 | Auxiliary roots return to their opener and share the shell's active destination state. |
| 5 | Adaptivity | 3/4 | Compact windows use the lazy destination rail; widths of 840 dp and above use a permanent Material drawer. Physical-device verification remains. |
| **Total** |  | **19/20** | **Excellent source-level result — verify the adaptive shell on hardware** |

## Platform conformance verdict

**Pass.** The app reads as native Compose/Material rather than a web wrapper: it uses Material controls, system Back handling, edge-to-edge Scaffold insets, lazy content, native dialogs and sheets, semantic color roles, and platform text scaling. Compact and expanded windows now share one destination model and preserve the primary route that opened an auxiliary root.

## Executive summary

- Audit health score: **19/20 (Excellent, source-level)**
- Open issues: **0 P0, 0 P1, 0 P2, 1 physical-verification gate**
- The Home/library rail no longer allows two active chips and uses identical selected styling for both destinations.
- Dashboard and library headers share the same 56 dp rail ownership and 16 dp content gutter.
- Saved, Sources, and Settings retain their opener and display active destination state in compact and expanded shells.
- Windows at 840 dp and above use a restrained 220 dp permanent Material drawer; narrower and split-screen windows retain the compact lazy rail.

## Detailed findings

### [Resolved P1] Auxiliary destinations discarded their originating route

- **Location:** `android/app/src/main/java/net/kdhmedia/bobobooks/ui/AppShell.kt`
- **Category:** Platform conformance
- **Impact:** A user opening Settings, Saved, or Sources from a library loses their place in the navigation hierarchy. Back unexpectedly jumps to Home, adding taps and making the system Back action disagree with native history semantics.
- **Guideline:** Android system Back should return to the previous destination and must not be replaced by a hard-coded unrelated root.
- **Recommendation:** Store the last primary destination (Home or Library) when opening an auxiliary root. Both `BackHandler` and visible Back callbacks should return there. A forced offline-library transition must also update the return destination safely.
- **Suggested command:** `$impeccable harden`

### [Resolved P2] Saved, Sources, and Settings could not express active navigation state

- **Location:** `AppShell.kt`, `MobileLibraryRail.kt`, `ExternalLibraryScreen.kt`, `SourceHubScreen.kt`, `DeviceSettingsScreen.kt`
- **Category:** Appearance / navigation hierarchy
- **Impact:** The shared rail visually identifies Home or the selected server library, but disappears on auxiliary roots. Users lose the compact spatial context established on the two primary destinations.
- **Guideline:** Navigation state should remain perceivable and consistent across peer destinations.
- **Recommendation:** During a later layout pass, separate shell-level destination context from each screen's content header. Preserve the content-first height budget; do not simply stack the existing rail above Settings' category rail.
- **Suggested command:** `$impeccable layout`

### [Resolved P2] Expanded widths retained compact phone navigation

- **Location:** `MobileLibraryRail.kt`, connected non-TV shell
- **Category:** Adaptivity
- **Impact:** Tablets, foldables, landscape phones, and desktop-windowed Android receive a stretched horizontal chip strip instead of using available width for a Material navigation rail or drawer. It remains functional but is not a first-class expanded layout.
- **Guideline:** Material navigation should adapt by window size rather than stretching a compact pattern unchanged.
- **Recommendation:** Introduce a window-size-owned shell strategy: horizontal destination viewport on compact width, navigation rail or modal/permanent drawer on expanded width, while retaining identical destination state and visibility preferences.
- **Suggested command:** `$impeccable adapt`

## Positive findings

- Home and a selected library are mutually exclusive by policy and tests.
- Active chip colors use Material semantic roles and are identical across Home and library destinations.
- The horizontally overflowing rail is lazy, snapping, bounded to one row, and provides a trailing overflow fade.
- Selecting a library from Home performs both state selection and destination navigation in one action.
- Hidden Saved or Sources destinations force a safe return to Home rather than leaving an unreachable screen mounted.
- Dashboard continuation rows support swipe, long press, and TalkBack custom actions without permanent trailing clutter.
- Media subscriptions remain route-owned, preventing unrelated playback clocks or catalogues from invalidating the shell.

## Completed actions

1. **[Resolved P1] `$impeccable harden`:** Preserve the primary opener for auxiliary destinations and cover the Back policy with unit tests.
2. **[Resolved P2] `$impeccable layout`:** Give auxiliary destinations restrained active shell context without stacking competing rails.
3. **[Resolved P2] `$impeccable adapt`:** Use a Material expanded-width navigation strategy for tablets, foldables, landscape, and windowed Android.
4. **[Verification gate] `$impeccable polish`:** Verify compact/expanded light/dark rendering, font scale, gestures, and insets on emulator or hardware.

Re-run the native audit after fixes and physical-device verification.

## Remediation status

The P1 Back-history defect is fixed. `AppShell` now records only the primary
Home or Library destination that opened an auxiliary root. System Back and each
visible Back action return to that origin; invalid or auxiliary return values
fall back safely to Home. Forced offline-library entry also updates the return
destination. Focused policy tests cover all auxiliary roots and malformed
history states, and both Direct and Store unit/lint suites pass.

The two P2 adaptivity findings are also fixed. Auxiliary roots now retain the
shared compact rail and its active state. At 840 dp and above, `AppShell` owns a
virtualized 220 dp Material permanent drawer and suppresses the duplicate
screen-level rail. The width policy naturally returns split-screen windows to
the compact layout. Direct and Store unit tests and lint pass after the change,
and the module-size architecture gate remains clean.

The current source-level score is **19/20**. The final point remains reserved
for rendered verification on a tablet or foldable across compact/expanded
widths, light/dark themes, larger font scales, gestures, and system insets; no
ADB device was available during this audit.
