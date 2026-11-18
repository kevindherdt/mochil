# Android watched-state response audit

Status: published in Android 0.16.83 on September 8, 2026.

## Confirmed defect

`VideoViewModel.setWatched` applied its asynchronous response to whichever show
was selected when the response arrived. Switching shows or libraries during a
save could change an unrelated show's watched count. Reusing an old action
snapshot could also increment an already-updated visible episode twice.

## Change

`VideoWatchedUpdates.kt` owns the small, pure response-to-catalogue update:

- Ignore a result belonging to another library.
- Match the affected show by source identity, including the ungrouped-video bucket.
- Update the matching show card as well as the selected show.
- Derive the count delta from the current episode when available.
- Preserve unrelated items and remove only the affected continue-watching item.

No layout changes, additional requests, media-file writes, or deployment.

## Verification

- Three new regression tests: switch shows, switch libraries, repeated confirmation
  followed by marking unplayed.
- Both full Direct and Store debug unit-test tasks passed.
- Module-size gate passed.

This is not proof of physical Android TV acceptance or concurrent server mutation
ordering. Pagination/search cancellation and delayed mutation errors remain audit
areas; this patch addresses successful watched-state responses only.
