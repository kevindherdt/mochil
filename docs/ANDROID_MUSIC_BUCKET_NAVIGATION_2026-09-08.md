# Music letter-jump state ownership

Status: source changes after 0.16.84; pending a later APK batch.

Letter jumps now accept server offsets only for root music tracks/albums/artists.
Detail, downloaded-only, offline-only and favorite-filtered projections reject
those offsets before cancelling requests or clearing content. Unknown letters
and negative offsets are also rejected. Search and navigation data are preserved.

A new jump clears obsolete loading-more/cursor state as it replaces the page.
Selecting the already selected letter resets the bucket and loads offset zero
within the same mode/query rather than repeating the previous jump.

Pure transition regressions cover valid offsets, repeated selection, state
preservation and invalid contexts. Both complete Android debug unit suites passed
in 14 seconds; module gate passed. Log: `/tmp/bobobooks-music-bucket-tests.log`.
No UI layout changes. The separate server group-offset correction is still
unpublished and is needed for correct duplicate-edition/album-artist positions.
