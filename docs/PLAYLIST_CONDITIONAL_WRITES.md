# Conditional playlist replacements

Status: server, web and Android adoption published in 0.16.81/code242. The chronological sections below describe implementation order; rename/delete conditions, older-client protection and missing-media preservation remain outside this completed replacement-write slice.

Update: web adoption is now implemented in source as described below; Android adoption is still pending. Nothing is deployed.

## Contract

Playlist detail GET returns a quoted strong ETag calculated from its stored name and full ordered entries, including entries whose media is currently unavailable. Nil and empty entry lists share a revision. No persistent schema migration is needed; revisions are content-based, not historical counters.

Playlist PUT accepts that opaque token in `If-Match`. Under the store's mutation lock it checks the current stored playlist, validates the token and either applies the replacement or returns HTTP 412. A successful PUT returns the resulting ETag. A stale or missing playlist cannot be recreated by a conditional replacement. Legacy requests without this header retain their existing behavior during client migration.

The comparison and mutation share one transaction lock; checking at the HTTP layer then calling ordinary Put would remain racy. Account scoping remains unchanged. Create-only behavior separately prevents duplicate POSTs from erasing existing lists.

## Evidence and remaining work

Full audio/web package tests pass (`/tmp/bobobooks-playlist-revision-tests.log`). Store and HTTP regressions demonstrate that a concurrent append survives a stale replacement and an up-to-date token permits an intentional replacement. The revision uses full stored contents, not the filtered display list.

Next: web detail reads must retain ETag, serialized edits must send it and retain the returned token, and conflict handling must reload without replaying a stale whole-list snapshot. Android MusicPlaylistClient and its editing state need the same adoption. Existing clients without a token remain unprotected against stale replacements; do not claim cross-device editing is solved yet. Rename/delete conditional behavior and missing-media preservation also need deliberate coverage. Tests should exercise both clients and API adapters before release.

## Web adoption

Detail reads retain the response ETag. Serialized replacements send `If-Match` when available and retain the successful response's ETag. New/renamed playlists require fresh details before replacement. A 412 invalidates the loaded detail, attempts a reload, notifies subscribers, and rethrows the error without resending the rejected snapshot. Failed refresh leaves details unloaded so another edit cannot immediately replace them. Legacy servers without ETags remain compatible, without claiming stale-write protection.

A simulated second-device append regression verifies the stale PUT is rejected, the appended track appears after reload, and subsequent deliberate edits use updated tokens. Full frontend suite: 549 passed (`/tmp/bobobooks-web-playlist-revisions.log`). The detail-loading regression now explicitly resolves the fresh read required before replacement, while confirming a preceding late read cannot overwrite the saved result. Android migration remains the next dependency.

## Android adoption

MusicPlaylist now retains the detail response ETag. Remove/reorder operations send the token from their serialized fresh read. Saving a queue reads the existing playlist or creates a missing one before obtaining its token and replacing it. A concurrent duplicate create is rejected by the server rather than erasing tracks. Older servers without ETags retain legacy compatibility.

HTTP 412 becomes a typed conflict. The request owner reloads selected details and preserves an explanatory error, without replaying the failed mutation. The client does not retain a successful PUT's ETag because it already reloads authoritative detail before every subsequent edit.

Both flavor targeted playlist suites passed (`/tmp/bobobooks-android-playlist-revision-tests.log`). A loopback HTTP test verifies ETag reception, If-Match transmission and exactly one rejected PUT; a request-state test verifies conflict detail refresh without mutation replay. Full batch verification and end-to-end Android API adapter coverage remain before release. Missing-media preservation and rename/delete conditions remain separate gaps. Nothing deployed.

## Authenticated adapter verification

`TestAndroidPlaylistRevisionSurvivesAuthenticatedForwarding` exercises the actual `androidAPI` entry point with a temporary issued device token. It verifies unauthenticated GET rejection, detail ETag forwarding, stale If-Match rejection after another writer appends, unchanged stored contents, successful fresh-token replacement and revoked-token PUT rejection. The ordinary targeted run passed. This complements the native HTTP-client and web state tests; it is not a physical Android device test.

The HTTP revision/create/Android adapter tests also passed under the Go race detector. `TestOnlyOneConcurrentPlaylistReplacementWins` starts two different replacements against one revision and verifies exactly one succeeds and one conflicts; it and the conditional-store regression passed ten repetitions under the race detector.

## Failed detail-read guard

The web whole-list save method also requires an existing successfully loaded playlist. A failed detail loader returns undefined, so checking only `detailLoaded === false` was insufficient and could fall through to an unconditional PUT. It now rejects both missing and failed reads with no request sent. Creation stays a separate operation. The queued-snapshot test now starts with an existing playlist while preserving its original identity/order assertions. Full frontend verification: `/tmp/bobobooks-playlist-failed-refresh-tests.log`.
