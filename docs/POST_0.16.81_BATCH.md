# Post-0.16.81 stabilization batch

Status: published in 0.16.82/code243 on September 8 at 14:02 UTC. See `ANDROID_RELEASE_0.16.82_PREPARATION.md` for final artifacts, verification and rollback. Historical verification notes below describe the preparation sequence.

## Included

- Native wiki filename matching respects filenames rather than display titles, exact-case ambiguity, and literal repeated `.md` suffixes. Web basename lookup has matching exact-case precedence. Missing-note creation round-trips the requested filename.
- Wiki offline image planning moves parsing/index work off the UI thread, avoids unused heading outlines, and checks cancellation during parsing.
- Playlist rename/delete now support revision-conditional writes in the store, HTTP API, web state and Android. Stale operations fail without automatic mutation replay.

Details: `ANDROID_WIKI_FILENAME_LINKS_2026-09-08.md`, `ANDROID_WIKI_OFFLINE_PLANNING_2026-09-08.md`, `PLAYLIST_STRUCTURE_CONDITIONS_2026-09-08.md`.

## Verified September 8

- Full Go tests and vet passed (`/tmp/bobobooks-post81-go-tests.log`, `/tmp/bobobooks-post81-go-vet.log`).
- All 553 frontend tests passed; module-size gate passed.
- Both Android flavor full unit suites passed after native playlist changes (`/tmp/bobobooks-android-playlist-structure-tests.log`).
- Browser playlist forms, playlist drag, wiki rendering and video progress checks passed. Relevant scripts exercise 1440px/390px fixtures, not comprehensive live-app visual acceptance.
- The expanded `scripts/android-playlist-revision-smoke.mjs` passed against a separately built temporary server on localhost:18104. It exercises real authentication, duplicate-create rejection, stale PUT/PATCH/DELETE rejection, valid conditional rename/delete and final absence. The script removed only its generated synthetic playlist. The temporary server was then stopped; its fixture directory remains `/tmp/bobobooks-post81-http-BmbMvG`.
- Store race tests passed ten repetitions; authenticated Android forwarding tests passed under the race detector.

Both Android lint tasks passed in 4m47s (`/tmp/bobobooks-post81-android-lint.log`); session 26917 is terminal. Subsequent visible-edit client adoption is implemented and unit-tested as recorded in `PLAYLIST_VISIBLE_EDITS_2026-09-08.md`; lint needs refreshing for those later native changes.

## Still required

Finish refreshed lint, release artifact/signature/distribution gates and an intentional batched publication. Physical Android/TV acceptance remains outstanding. The broad project goal is not complete: native/wiki parity, TV usability and real-device performance remain ongoing. Established mobile layout is unchanged by this batch.
