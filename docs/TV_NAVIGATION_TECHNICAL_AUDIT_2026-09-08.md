# Android TV navigation technical audit

Impeccable native audit, source-only. No connected device was available (`adb
devices` returned none). No screenshot, TalkBack, overscan, contrast or frame-time
acceptance is claimed. Mobile layout was outside scope and no UI was changed.

## Platform verdict

Source-level pass: native Compose/Material controls, lazy grids, remote key
handling and individually focusable playback actions—not a web port. Actual TV
interaction acceptance remains unverified. The established compact TV navigation
takes precedence over generic phone navigation recommendations.

## Provisional scoped health score

| Dimension | Score | Evidence / limitation |
| --- | --- | --- |
| Accessibility | 2/4 | Labeled 52dp transport controls; alphabet selection lacks semantics |
| Performance | 3/4 | Lazy keyed content, bounded discovery, isolated playback clock |
| Appearance & theming | 3/4 | Semantic Material color roles; contrast not measured |
| Platform conformance | 3/4 | Focus groups/restoration and guarded long-press menus |
| Adaptivity | 2/4 | Adaptive grid, but fixed search/dock widths need device checks |
| Total | 13/20 | Acceptable; incomplete physical evidence, not a product-wide score |

## Findings

Three P2 findings; no verified P0/P1 in this bounded inspection.

1. **P2: focused letter disagrees with catalogue ordering.**
   `ui/TvFocusDetail.kt`, `tvNavigationBucket`, takes the first character only.
   Server `catalog.SortKey` strips leading The/An/A and trims whitespace.
   Focusing “The Album” therefore highlights T although the server groups it in
   A. Use matching normalization, with shared contract cases for articles,
   numbers and Unicode. Suggested command: `$impeccable harden`.

2. **P2: alphabet availability and jump validity have different rules.**
   `ui/TvLibraryScreen.kt` exposes music navigation at the root with an empty
   query whenever navigation data exists. `audio/MusicBucketNavigation.kt`
   rejects downloaded-only, favorites-only, offline-only and non-root-grouping
   offsets. With retained navigation under those conditions the rail can offer
   actions the handler rejects. Share eligibility policy, retaining the TV's
   deliberate query/detail visibility restrictions. Suggested command:
   `$impeccable harden`.

3. **P2: selected alphabet bucket is only visual.**
   `ui/TvAlphabetGrid.kt`, `TvAlphabetButton`, draws selected border/color but
   does not expose selected semantics. Unlike `TvTextButton`, assistive
   technology cannot distinguish the active bucket through this node's explicit
   semantics. Add selected state without extra visible text or icons.
   Suggested command: `$impeccable harden`.

## Patterns and positives

The main gap is duplicated navigation state policy, not a need for a new shell.
Keep the current lazy grids and stable item keys, 96-route bounded position
store, discovery cap of 80 tracks, long-press opening-release guard, semantic
focus borders and separate dock clock state collection. The dock exposes Play,
Now Playing and Close as separate controls within a focus group; source does
not show the old whole-player-only selection problem.

## Next actions

1. `$impeccable harden`: fix the three navigation/state findings as one batch.
2. Verify native tests and compile; then test remote selection, Back and
   long-press cancellation on TV when hardware is available.
3. `$impeccable polish`: one bounded physical pass for focused readability,
   font scaling, search/dock fit and edge clipping. Do not infer these from a
   browser rendering or JVM test.

You can ask me to run these one at a time, all at once, or in any order you prefer.
Re-run `$impeccable audit` after fixes to see your score improve.

## Hardening follow-up (source-only after 0.16.85)

The three source findings are addressed together:

- TV letter calculation now trims and removes the same single leading article
  as the server before choosing a bucket. Regression cases include article
  names, non-article lookalikes, numbers and non-Latin names.
- MusicState owns `supportsMusicBucketNavigation`, used both by the jump
  transition and the TV rail. Filtered/offline/detail views no longer offer
  retained root offsets; negative offsets are excluded from the rail.
- Alphabet cards explicitly publish selected semantics without new visual UI.

Both full Android debug unit suites and the module gate passed. Build completed
in 25s; log `/tmp/bobobooks-tv-nav-hardening-tests.log`. No new APK was built.
Impeccable hardening kept the established layout and scoped changes to state
correctness/accessibility. A physical `$impeccable polish` pass remains pending
connected TV hardware; the original provisional score is not upgraded on the
basis of source tests alone.

## Shared server/client contract

`contracts/navigation_buckets.json` is now consumed by both Go catalogue tests
and Android TV tests. Sixteen cases pin article handling, single-prefix removal,
whitespace, numeric titles, accented/non-Latin titles, emoji and empty input.
This prevents the two implementations from passing only separately authored
expectations while disagreeing with one another. It is not an exhaustive Unicode
normalization specification.

The Go contract test, both full Android debug suites and module gate passed.
Android build/test completed in 12s; log
`/tmp/bobobooks-navigation-contract-android.log`. Runtime changes remain queued
for the next Android batch; this follow-up adds shared verification only.
