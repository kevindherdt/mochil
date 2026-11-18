# Stabilization and UI audit

Audit date: 2026-08-24

This pass reviewed the Go services, persisted state, catalogue/query hot paths,
and the rendered web application at 1440×900 and 390×844 in dark and light
themes. It also exercised race detection, shuffled test repetitions, vet,
coverage, JavaScript syntax checks, large-catalogue benchmarks, a production
container rebuild, and authenticated browser smoke tests.

## Outcome

| Dimension | Score | Result |
| --- | ---: | --- |
| Accessibility | 4/4 | Visible controls are labelled, focus is explicit, reduced motion is supported, and the closed search control no longer remains in the tab order. |
| Performance | 4/4 | Catalogue grouping allocations were reduced substantially; scroll handling no longer reads layout on every event; podcast episodes load one bounded page at a time. |
| Responsive design | 4/4 | No page-level overflow at 390 px; navigation, browse modes, alphabet controls, secondary media actions, and reader controls meet the 44 px target. |
| Theming | 4/4 | Dark/light tokens consistently cover native selects and their options, focus, selection, surfaces, and text. |
| Implementation integrity | 3/4 | The product-specific, cover-led editorial system is coherent. The detector was degraded because optional parser modules were unavailable, so its empty result is an undercount rather than proof of no issues. |
| **Total** | **19/20** | **Excellent — minor polish only** |

Implementation integrity passes. The interface has a consistent Mochil
visual language: Newsreader provides the editorial hierarchy, Source Sans owns
controls and supporting copy, warm espresso/oat surfaces work in both themes,
and apricot is reserved for active state and progress. Library and settings
pages use content-led rows and shelves rather than generic dashboard cards.

## Fixed findings

- **P1 — failed persistence remained visible in memory.** Audio progress,
  video progress/preferences, reading state, collections, and KoSync now restore
  their previous state when an atomic save fails. Regression tests inject real
  filesystem failures.
- **P1 — legacy state loaders accepted unsafe input shapes.** Catalogue
  overrides, metadata jobs, KoSync, and video preferences now use the shared
  bounded, regular-file-only, single-JSON-value decoder.
- **P1 — media and reading snapshots exposed mutable nested state.** Book
  authors/series indexes, audio chapters, video stream lists, and reading
  anchors/locator maps now cross store boundaries as deep copies. The audit
  also caught and fixed a clone-helper defect that discarded Readium/Foliate
  locator maps, protecting exact cross-reader progress during rollback and
  synchronization.
- **P2 — series grouping repeated expensive normalization and path work.** The
  query index now caches normalized title keys and path depth. Natural series
  ordering also correctly places `Series 2` before `Series 10`.
- **P2 — scroll handling forced layout reads on every scroll event.** Sticky
  measurements now run on resize/content changes, while scroll state is
  animation-frame throttled and layout-free.
- **P2 — podcast focus eagerly fetched every episode.** It now fetches one
  bounded page and uses the existing “Load more episodes” control.
- **P2 — undersized browse targets.** Header icons, browse-mode buttons, and
  alphabet controls now use at least 44 px targets.
- **P2 — hidden search remained keyboard-focusable.** The collapsed search
  panel is now inert and removed from the accessibility tree until opened.
- **P2 — secondary touch targets were undersized.** Media overflow actions,
  podcast management, document controls, wiki actions, metadata dialogs, and
  reader controls now meet the 44 px minimum without increasing visual weight.
- **P2 — long flat lists accumulated DOM indefinitely.** Books and music now
  use measured bidirectional recycling windows with bounded node counts and
  logical range reporting.

## Verification

- `go test -race ./...`: pass
- `go test -shuffle=on -count=3 ./...`: pass
- `go vet ./...`: pass
- JavaScript syntax checks for every first-party module: pass
- Statement coverage: 60.3% overall
- Permanent authenticated Chromium smoke test: pass in dark and light themes
  at 1440×900 and 390×844, with no runtime errors, page-level overflow,
  undersized visible mobile controls, or audio loss across navigation
- Production image rebuild and container start: pass

Coverage is strongest around stateful domain packages and deliberately lower
around the large HTTP composition package. `cmd/opdshelf-manifest` is a build
helper and `internal/jellyfin` currently contains protocol models, so neither
has direct statement coverage. External Seafile, Calibre, feed-provider, codec,
and browser-decoder behavior still requires integration environments; unit and
HTTP tests cover the local boundaries and failure handling.

## Remaining low-risk opportunities

- The frontend boundary review retained the cohesive reader, video, audio,
  metadata, and settings coordinators. Their storage, rendering, paging,
  compatibility, and persistence concerns are already delegated; split a
  coordinator only when a new feature creates an independently testable
  boundary rather than moving code to hide line count.
- Add integration fixtures for external storage and media tools when those
  services can run reliably in CI.
- Impeccable's optional HTML/CSS parser dependencies are installed and its full
  detector currently reports no findings.

## Android follow-up — 2026-08-27

- `go vet ./...` and `go test -race ./...`: pass.
- Direct and store Android unit suites: pass.
- Full direct and store Android lint: 0 errors. Remaining warnings are
  deliberate synchronous durability writes, conservative free-space checks,
  and dependency-update notices rather than runtime defects.
- Removed the suspicious mixed indentation and Compose primitive boxing hints.
- Normalized optional `Modifier` placement in the affected reader and music
  components.
- Moved EPUB highlight/range and cross-reader position matching into
  `EpubReaderMatching.kt`; its interoperability tests continue to exercise the
  same internal API without routing matching policy through the Activity.
- Moved EPUB remote revision, pending-state, initial resolution, conflict, and
  save-result policy into `EpubProgressCoordinator.kt` behind a narrow gateway
  and state-store contract. The reader Activity no longer owns server revision
  or conflict-blocking state.
- Moved EPUB annotation mutation, persistence, synchronization, anchor repair,
  and Notes export into `EpubAnnotationController.kt`. Notes export retains the
  annotation mutex through synchronization and snapshot creation so the split
  does not introduce an edit/export race.
- Music browsing now uses a dedicated request generation owner. Switching
  library, grouping, search, artist, album, or alphabet bucket invalidates
  older asynchronous work before it can replace the visible content or surface
  a stale error. Disconnected shelves now fall back to offline tracks or stop
  loading with an actionable connection message.
- Publication shelves, grouped publication views, podcasts, and video now use
  request identity/generation guards, preventing late responses from replacing
  a newer destination or surfacing an obsolete failure.
- Books and series expose a long-press collection action backed by the existing
  synchronized server collections. The picker, transport, state owner, and
  models are separate bounded modules rather than additions to the app shell.
- The mobile destination rail has mutually exclusive selection semantics and
  stable label color. Manga chapter-only Home labels include their series name.
- Normal light and dark palettes use softer paper and charcoal surfaces; the
  high-contrast e-ink palettes remain unchanged.
- The configured Wiki library now uses the vault media type, activating the
  existing nested, collapsible folder/page tree and split reading workspace.
- Browser module tests: 47/47 pass. The rendered authenticated smoke test was
  not rerun because this shell was intentionally not given production account
  credentials.
