# Android TV music surface audit

Scope: Impeccable native technical audit of TV library content, shared TV controls,
grid virtualization and playback dock. No UI changes in this audit. `adb devices`
returned no connected devices; no screenshots or physical performance measurements.
Incumbent semantic Material roles and compact content-first layout are authoritative.
Earlier context setup reported stale PRODUCT/DESIGN guidance; it was not rewritten.

## Platform conformance verdict

Source-level pass for native Compose controls, focus groups, Back actions and
dedicated TV layout rather than a stretched phone screen. Runtime conformance is
unverified: this audit cannot certify controller traversal, inset behavior, contrast
at viewing distance or large-font rendering without a TV/emulator.

## Provisional source health score

| Dimension | Score | Evidence / limitation |
| --- | ---: | --- |
| Accessibility | 2/4 | Labeled 52dp controls; stale focus membership finding below |
| Performance | 3/4 | Lazy keyed grids, isolated clock updates; no device frame trace |
| Appearance / theming | 3/4 | Shared semantic roles and focus border; rendered contrast unverified |
| Platform conformance | 3/4 | Native controls and dedicated edge navigation; remote acceptance pending |
| Adaptivity | 2/4 | Dedicated TV layouts; fixed-width dock/text controls need large-font checks |
| Total | 13/20 | Acceptable source baseline; significant verification remains |

These are provisional code-review scores, not measured visual-quality ratings.

## Findings

Two P2 issues; no confirmed P0/P1/P3 issues in this bounded scope.

### P2: empty-state ownership includes hidden cached views

Location: `ui/TvLibraryScreen.kt`, `activeItemsEmpty` music/audiobook branch.
Category: accessibility / task feedback.

The expression counts root groups, discovery collections and all favorite tracks
regardless of the selected artist, album or browse mode. An empty current result
can therefore bypass the empty-state message because hidden cached content exists.
The video branch similarly includes root shows while inside a selected show.
Users can see an empty content area without the intended explanatory feedback.

Recommendation: determine emptiness from exactly the content branch rendered by
the grid, using a shared pure policy covered by empty-detail and filtered-view
tests. Do not clear caches just to make the check pass.
Suggested command: `$impeccable harden`.

### P2: focus-membership memoization omits artist detail dependencies

Location: `ui/TvLibraryScreen.kt`, `rememberedFocusVisible` remember keys;
`ui/TvLibraryContent.kt`, `tvHasVisibleFocusKey` and `tvAudioGroups`.
Category: accessibility / navigation.

The remembered calculation reads the artist album list and selected artist through
music state, but those values are absent from its dependency keys. When the album
list arrives without changing the listed keys, an earlier false membership result
can remain cached. The fallback can retain edge-rail focus instead of recognizing
the remembered album. Exact remote symptoms require device reproduction.

Recommendation: include every data dependency used by membership, or compute a
small explicit visible-content input shared by the grid and focus policy. Preserve
the existing bounded lookup instead of rebuilding a full identity set per frame.
Suggested command: `$impeccable harden`.

## Positive findings to preserve

- Lazy grids retain stable item identities and bounded route-position storage.
- Shared TV focus modifiers avoid transforms and border nodes on idle controls.
- Playback dock clock observes timeline state separately from artwork/title controls.
- TV controls use Material semantic colors, labels and 52dp targets.
- Loading, retry and remote long-press handling have dedicated components.

## Unverified, not reported as confirmed defects

Long-duration clock strings inside the fixed-width dock, 1.3x font scaling,
Ubuntu/dark theme viewing-distance contrast, overscan and real remote/controller
behavior still require device screenshots and interaction. Do not redesign mobile
or enlarge all TV cards based solely on these risks.

## Recommended actions

1. `$impeccable harden`: align empty-state and focus dependency ownership with visible content.
2. `$impeccable adapt`: verify TV insets and font scaling on a connected TV/emulator.
3. `$impeccable polish`: one bounded confirmation pass after the functional fixes.

You can ask me to run these one at a time, all at once, or in any order you prefer.
Re-run `$impeccable audit` after fixes to see your score improve.
