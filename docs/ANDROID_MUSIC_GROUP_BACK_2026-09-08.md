# Music group Back state

Status: implemented after 0.16.83; pending the next batch.

Closing an album/artist previously invalidated the pending response but left its
`loading` state true. The discarded response could then never clear that state.
A pending root pagination flag could also remain set after detail navigation.

Pure group-exit transitions now clear abandoned loading/error state while retaining
the cached parent groups, artist albums (when leaving an album), query and root
continuation cursor. A redundant close without a selected detail is a no-op and
does not invalidate a legitimate root request. No new fetches or layout changes.

Tests exercise Back during album loading, Back during artist loading and redundant
Back at a loading root. These test state transitions; physical remote focus and
scroll restoration still require device acceptance.

Both full Direct and Store unit suites passed (13 seconds,
`/tmp/bobobooks-music-back-tests.log`), along with the module-size gate.

## Root pagination isolation

The adjacent `loadMore` entry point also lacked detail, initial-loading and local
filter guards. A late scroll callback could use a retained root cursor from an
album/artist or downloaded/favorites view, fetching unrelated pages and changing
loading state in that context. `MusicPaginationPolicy` now admits continuation only
for a ready online root with its matching cursor type. Cached parent cursors remain
available after leaving detail; this does not discard browsing position or add UI.

Four policy tests cover loading state, detail navigation, local/favorite filters,
and track-vs-group cursors, including returning from an abandoned detail load.

Both complete Android flavor unit suites and module-size checks passed after the
pagination follow-up (`/tmp/bobobooks-music-pagination-tests.log`). Not yet shipped.
