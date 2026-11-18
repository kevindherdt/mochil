# Preserve unseen playlist references

Status: server, web and native adoption published in 0.16.82/code243. See `ANDROID_RELEASE_0.16.82_PREPARATION.md` for final release gates; historical notes below describe implementation order.

`playlistView` omits unavailable libraries/items. Whole-array replacement can therefore erase unseen references even with a valid revision, since availability is not part of that revision.

`PUT .../audio-playlists/{name}?edit=visible` now requires `If-Match` and an object with `visible_entries` (original visible identities) and `entries` (their requested replacement). Both must be arrays. Missing condition returns 428; stale or missing playlist returns 412.

The merge retains occurrences outside the original visible snapshot in their existing order. Replacement entries fill former visible positions, with surplus entries appended. Unknown base identities or more occurrences than stored, and replacement overlap with wholly unseen identities, are rejected. Repeated tracks remain supported. Existing references may survive files disappearing since GET; genuinely new references must resolve to current music items. The final write condition is checked atomically against the original stored revision.

Ordinary PUT remains explicit whole-playlist replacement. Clients should use visible-edit for remove/reorder, and ordinary PUT for explicit save-queue/replace. Capability signaling is needed before adoption; ETag support alone does not establish support for this new contract.

Audio/web package tests pass (`/tmp/bobobooks-playlist-visible-tests.log`), covering merge reorder/removal/clear, invalid bases, retained unavailable references, missing/stale conditions, rejected unknown insert and actual stored ETag. Module-size gate passes.

## Client adoption

Detail GET advertises `X-Mochil-Playlist-Edit: visible-v1`. Web and Android retain that explicit capability; the presence of an ETag alone never opts an older server into the new body format. Advertised support without a revision rejects the edit locally rather than falling back to whole replacement.

Web serialized transforms send their original visible identities and retain capability after subsequent writes. Native remove/reorder uses `editVisible` with the loaded playlist; explicit save-queue keeps ordinary PUT. Older servers retain legacy behavior without a claim of unavailable-track protection.

All 555 frontend tests passed (`/tmp/bobobooks-visible-edit-web.log`), including repeated edits, original identities, capability retention after explicit replacement and malformed-capability rejection. Both native full unit suites passed (`/tmp/bobobooks-visible-edit-android.log`), including non-empty original/replacement request payloads and legacy fallback. Go audio/web packages pass (`/tmp/bobobooks-visible-edit-go.log`); focused merge/HTTP/authenticated Android forwarding tests pass under the race detector (`/tmp/bobobooks-visible-edit-race.log`). The adapter test proves the capability header and query mode survive authentication/forwarding and preserve an unseen stored reference. Module-size gate passes.

Remaining: fresh lint for the native adoption, release/browser gates and publication. No personal playlist or media changes.
