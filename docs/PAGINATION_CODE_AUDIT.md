# Pagination and continuous-scrolling audit

Audit date: 2026-08-24

Scope: publication, series, audio, video, and document pagination; browser DOM
growth; cursor correctness; loading races; accessibility fallbacks; desktop and
mobile behavior. This is an evidence report, not an implementation change.

## Implementation integrity verdict

**Pass.** Mochil uses one coherent, product-specific catalogue system with
bounded APIs, versioned cursors, lazy artwork, responsive cover/list layouts,
and explicit loading states. The mechanical Impeccable detector reported no
findings, but ran in degraded regex mode because its optional HTML/CSS parser
packages are unavailable.

The pagination architecture is not yet uniform. Books, series, and movies
append efficiently but require a button. Audio and documents have intersection
sentinels, but rebuild all previously loaded markup on each continuation. A
single shared infinite-scroll helper cannot safely be applied until those
rerender paths are corrected.

## Audit health score

| # | Dimension | Score | Key finding |
| --- | --- | ---: | --- |
| 1 | Accessibility | 3/4 | Load buttons provide a keyboard fallback, but an automatic-only design would need an announced loading state and retry control. |
| 2 | Performance | 3/4 | Cursor APIs are fast; audio/document continuation recreates prior DOM and deep scrolling is not windowed. |
| 3 | Responsive design | 4/4 | Existing sentinels and responsive grids work without horizontal overflow. |
| 4 | Theming | 4/4 | Pagination controls and loading states use shared warm-theme tokens. |
| 5 | Implementation integrity | 3/4 | Pagination behavior is duplicated across four controllers and differs in important lifecycle details. |
| **Total** |  | **17/20** | **Good — address DOM lifecycle before universal auto-loading** |

## Executive summary

- Issues: 0 P0, 2 P1, 3 P2, 1 P3.
- The server is not the bottleneck. Loading the second 80-series page took
  3.3 ms locally and transferred 17.9 KiB.
- The first 80 series rendered 1,404 total page nodes. The second page produced
  2,199 nodes: 795 additional nodes for 80 groups.
- Music grew from roughly 3,031 nodes at 72 rows to 5,229 at 144 rows. Its
  first row was replaced rather than preserved during continuation.
- Books already append page markup, lazy-load cover images, and apply
  `content-visibility`; this is the safest first surface for automatic loading.
- Full-library infinite append is not a safe end state for a 12,214-book
  catalogue. A bounded page window with scroll spacers is the durable solution.

## Detailed findings

### [P1] Audio continuation rebuilds every loaded row

- **Location:** `internal/web/static/audio-library.js`, `render()` and `load()`.
- **Category:** Performance / Implementation integrity.
- **Impact:** Every automatic page replaces existing elements, rebinds menus,
  and restarts artwork observation. Focus or an open row menu can disappear.
  Work grows with every page rather than only with the new page.
- **Evidence:** the original first-row DOM identity was not retained after
  loading rows 73–144; total page nodes reached 5,229.
- **Recommendation:** add an append rendering path for ordinary track rows.
  Preserve full rerenders only for sort, search, queue editing, favorites, and
  other views whose order can change.
- **Suggested command:** `$impeccable optimize`.

### [P1] Unlimited append would make deep book browsing memory-bound

- **Location:** `internal/web/static/app.js`, `loadBooks()` and `loadSeries()`.
- **Category:** Performance.
- **Impact:** Automatically appending all 12,214 books could create well over
  100,000 DOM nodes. `content-visibility` reduces layout and paint work, but it
  does not remove DOM objects, listeners, attributes, or retained data.
- **Recommendation:** implement continuous scrolling with a bounded window,
  retaining approximately three pages above and below the viewport. Replace
  removed pages with measured block spacers and restore a page when scrolling
  back. Keep cursor anchors per page.
- **Suggested command:** `$impeccable optimize`.

### [P2] Document continuation recreates the complete result list

- **Location:** `internal/web/static/document-library.js`, `render()`.
- **Category:** Performance / Accessibility.
- **Impact:** Every page assigns `innerHTML` from all loaded documents. This can
  discard focus and menu state and repeats parsing for older rows.
- **Recommendation:** append only the new page when sort/search/layout is
  unchanged. A layout change may intentionally perform one full render.
- **Suggested command:** `$impeccable optimize`.

### [P2] Movies have no loading guard suitable for an observer

- **Location:** `internal/web/static/video-library.js`, `load()`.
- **Category:** Performance / Correctness.
- **Impact:** The current button disables itself before loading, but a sentinel
  can deliver repeated callbacks. Without an explicit `loading` guard, requests
  could overlap and duplicate pages or let a later response replace request
  ownership unexpectedly.
- **Recommendation:** add a loading guard and retain the button as retry/fallback
  before attaching an observer.
- **Suggested command:** `$impeccable harden`.

### [P2] Pagination orchestration is duplicated

- **Location:** `app.js`, `audio-library.js`, `video-library.js`, and
  `document-library.js`.
- **Category:** Implementation integrity.
- **Impact:** Root margins, loading guards, retry behavior, cursor-conflict
  handling, visibility rules, and button copy drift between media families.
- **Recommendation:** introduce a small `continuous-pager.js` controller that
  owns observer lifecycle and fallback-button state only. Domain controllers
  must continue to own queries, cursors, rendering, and conflict recovery.
- **Suggested command:** `$impeccable harden`.

### [P3] Loading status is visual but not consistently announced

- **Location:** generated series load buttons and media load buttons.
- **Category:** Accessibility.
- **Impact:** Automatic continuation can be silent to screen-reader users,
  especially when the status count is outside their current virtual cursor.
- **Recommendation:** give the shared pager a concise `role=status` message
  such as “Loading more series” and “160 of 2,431 series loaded.” Do not move
  focus when a page appends.
- **Suggested command:** `$impeccable clarify`.

## Recommended implementation sequence

1. **Incremental rendering:** fix audio and documents so a continuation keeps
   existing nodes and adds only the returned page.
2. **Shared observer shell:** add a small controller with an 800 px root margin,
   one in-flight request, explicit disconnect/reset, cursor-conflict recovery,
   and a visible fallback button on failure or without IntersectionObserver.
3. **Safe surfaces first:** enable automatic continuation for books, top-level
   series, movies, documents, and ordinary audio tracks. Keep nested 500-book
   series pages manual until there is real evidence they need automation.
4. **Bound deep browsing:** add page-window recycling for books and series
   before advertising effectively unlimited scrolling.
5. **Verification gates:** preserve focus, open menus, audio playback, scroll
   position, cursor ordering, and back-scroll restoration; test 390×844 and
   1440×900 with slow-network and CPU throttling.

## Positive findings

- Book, series, and movie page rendering already appends new markup.
- Publication, series, audio, and document continuations use snapshot-bound
  cursors and explicitly reject stale generations.
- Cover and artwork loading is viewport-prioritized.
- Book, series, and video items already use `content-visibility` or lazy image
  loading to avoid unnecessary offscreen paint.
- Loading buttons remain at least 44 px and provide a useful no-observer,
  keyboard, and error fallback.
- Race tests, vet, and first-party JavaScript syntax checks pass.

## Recommended actions

1. **[P1] `$impeccable optimize`:** implement incremental audio/document
   rendering and bounded continuous pagination.
2. **[P2] `$impeccable harden`:** unify observer lifecycle, request guards,
   fallback/retry behavior, and cursor-conflict recovery.
3. **[P3] `$impeccable clarify`:** add unobtrusive announced loading status.
4. **[P3] `$impeccable polish`:** verify the final desktop/mobile interaction
   and spacing after the behavior is stable.

## Implementation update — 2026-08-24

- Added the bounded `continuous-pager.js` controller with an 800 px preload
  margin, one request gate, observer plus low-frequency position recovery, and
  a manual fallback when observation is unavailable or a request fails.
- Enabled automatic continuation for books, top-level series, movies,
  documents, and the ordinary music track list. Nested series, grouped
  audiobooks, and podcast episodes remain intentionally manual.
- Changed music tracks and documents to append only the returned page. Earlier
  nodes keep their identity, which also preserves focus and open disclosure
  state.
- A five-page book sample grows linearly at roughly 22 DOM descendants per
  item (2,643 descendants for 120 books; 5,283 for 240). This confirms that
  unlimited accumulation across a 5,038-book shelf is still too expensive.
  Page-window recycling therefore remains the next deep-scroll optimization;
  it was not folded into the shared pager or domain controllers.
- Verification: Go tests and vet pass, all 24 first-party JavaScript tests
  pass, browser checks preserve the first book node across continuation, and
  the Impeccable detector reported no regex-level findings. Its HTML parser was
  unavailable, so the detector result is an undercount rather than a full
  semantic/contrast audit.

## Implementation update — 2026-08-31

- Movie browsing now uses the same measured, bidirectional bounded-window
  primitive as books and music. At most 360 movie cards remain in the DOM;
  removed pages become top or bottom spacers and are fetched again as the user
  scrolls back.
- The existing automatic continuation keeps its keyboard-accessible failure
  fallback, while an independent previous-page sentinel restores trimmed
  results without disturbing the current scroll anchor.
- Television and anime show trees deliberately remain outside this recycling
  path because an expanded season is interactive state, not an interchangeable
  flat movie card.
- The complete 359-test frontend suite and Impeccable detector pass after the
  change. The remaining pagination work is evidence-driven refinement rather
  than an unbounded catalogue defect.
- Ordinary document shelves now use the same three-page measured window and
  bidirectional restoration. Their versioned forward cursor remains the fast
  path; after back-scroll trims the far edge, deterministic offsets restore it,
  and an expired scan cursor restarts the active view safely. Vault trees stay
  outside recycling because their nested open/editor state is not a flat list.
