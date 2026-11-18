# Conditional playlist rename and deletion

Status: server, web and Android published in 0.16.82/code243. Final release evidence is in `ANDROID_RELEASE_0.16.82_PREPARATION.md`; historical notes below describe implementation order.

Playlist PATCH and DELETE now accept the detail GET's opaque ETag in `If-Match`, matching replacement PUT. Validation happens under the store mutation lock, including missing-playlist rejection. A stale revision returns HTTP 412 without renaming or removing newer contents. Empty headers preserve legacy behavior. Rename still requires a fresh detail read afterward; it does not return a potentially racy post-mutation ETag.

The web state owner shares one conditional request helper for replacements, renames, and deletion. Failed detail reads send no mutation. A stale request invalidates and reloads detail, retains the playlist, and propagates the conflict without automatic destructive replay. A later deliberate action uses the refreshed revision. No visual controls or layout changed.

Evidence:

- Audio and web Go package suites pass: `/tmp/bobobooks-playlist-structure-tests.log`.
- Stale rename/delete, cross-account rejection, successful current operations, and simultaneous rename-versus-delete are covered. Focused race tests passed ten repetitions: `/tmp/bobobooks-playlist-structure-race.log`.
- HTTP tests verify stale 412 and fresh 204 for both methods.
- All 553 frontend tests pass: `/tmp/bobobooks-playlist-structure-web.log`. New tests cover conflict refresh, no automatic retry, deliberate retry, and failed-read suppression. Existing snapshot/late-response fixtures now supply the newly required detail reads while retaining their original assertions.

## Android adoption

Native rename and delete send `If-Match`. The view model captures the selected detail's revision when available, preserving the user's viewed state; actions from the overview obtain detail before mutating. HTTP 412 uses the existing typed conflict/reload path without retrying the operation. Legacy missing revisions remain compatible.

The native HTTP client now accepts an internal connection factory for testing, retaining its existing default transport. Request tests check PATCH/DELETE methods, escaped names, authorization and revision headers, bodies, redirect refusal, disconnect, success and single-attempt conflicts. These use a controlled connection, not device networking. Both full flavor unit suites pass: `/tmp/bobobooks-android-playlist-structure-tests.log`.

The authenticated Android API regression now checks stale PATCH/DELETE 412, current PATCH/DELETE 204 and revoked-token rejection, in addition to PUT. The PATCH success is an intentional same-name operation; actual name changes are covered by the store and ordinary HTTP tests. Focused API tests pass under the race detector: `/tmp/bobobooks-android-playlist-structure-api.log`. The module-size gate passes.

Remaining: release/browser gates and publication. Older clients and servers without revisions remain unprotected. This does not solve unavailable-track preservation in full playlist replacements.
