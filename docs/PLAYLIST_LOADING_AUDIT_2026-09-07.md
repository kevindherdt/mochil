# Playlist loading and single-playlist lookup

Current status: web and Android overview summary adoption shipped with 0.16.74 on September 7, 2026 at 17:44 UTC. See `ANDROID_RELEASE_0.16.74.md` for package hashes and deployment evidence. Sections below record implementation stages; earlier pending-work notes are historical where superseded by the release record. Physical-device acceptance and cross-device protection for replacement playlist writes remain outstanding.

## Verified findings

- `audioPlaylistCollection` copies all stored playlist entries and expands every available music item into an `audioView` on every collection GET.
- Android `MusicPlaylistClient.list` reads that entire response (bounded to 16 MiB). Its item mutation reconciliation also calls this full collection endpoint before editing one playlist.
- Web playlist cards use track arrays for counts and up to four artwork URLs. Detail playback and item actions also depend on complete arrays. Simply truncating `items` would break these consumers or misrepresent playlists as empty.
- Playlist ZIP preparation previously called `List`, copied every playlist's entries, sorted every name, and then selected one playlist.

## Implemented, not yet deployed

`PlaylistStore.Get` returns a detached, account-scoped snapshot of only the requested playlist. Case-insensitive matching is consistent with playlist mutations. ZIP preparation now uses it. Existing download validation, streaming, limits, and archive paths are unchanged. No client UI or APK changes.

Tests cover case-insensitive names, names containing slashes, empty versus missing playlists, account isolation, and mutation of returned snapshots. The audio and web package suites and playlist race tests pass.

## Measured lookup cost

Synthetic in-memory benchmark: 100 entries per playlist, looking up the last stored playlist. Go/linux amd64 on Ryzen 5 PRO 4650G. One 100 ms benchmark run; these are indicative microbenchmarks, not end-to-end download timing. The comparison invokes the previous `List`-then-find path and the new `Get` path against the same fixture. No file reads or network transfer are included.

| Playlists | Previous lookup | New lookup | Previous allocated bytes | New allocated bytes |
| --- | ---: | ---: | ---: | ---: |
| 10 | 8.965 µs | 0.867 µs | 35,384 | 3,456 |
| 100 | 84.398 µs | 1.590 µs | 353,370 | 3,456 |
| 1,000 | 1,560.900 µs | 9.023 µs | 3,529,433 | 3,456 |

The new lookup still searches playlist names linearly; only entry copying and sorting of unrelated playlists are eliminated. This does not speed up media encoding or network throughput.

Reproduce: `go test ./internal/audio -run '^$' -bench BenchmarkPlaylistLookup -benchmem -benchtime=100ms`

## Remaining coordinated work

1. Introduce a compatible overview response with explicit track counts and bounded artwork previews, plus a separate single-playlist detail endpoint.
2. Update web and Android consumers together: overview data must never be mistaken for a complete track list. Load details on open/play, retain cancellation and navigation-generation safeguards, and preserve older-server fallback.
3. Fetch only the target playlist for Android item-edit reconciliation. Avoid overwriting remote edits with incomplete overview data; consider revision-checked mutations separately.
4. Verify empty/missing tracks, removed libraries, rename/delete during requests, account changes, artwork, queue playback, and response sizes with large synthetic fixtures.

The overview over-fetching remains unresolved in the current release. No personal playlist state was modified for this audit.

## Single-playlist API groundwork (not deployed)

`GET /api/audio-playlists/{encoded-name}` now returns one `{name, items}` object. The existing Android forwarding route exposes the same behavior at `/api/android/v1/audio-playlists/{encoded-name}`. Existing collection GET responses are unchanged; `?download=1` continues to stream a ZIP.

- Missing or another account's playlist returns 404. An existing empty playlist returns `items: []`.
- Item order and filtering of unavailable/non-music entries use the same projection as the collection endpoint.
- Playlist names are decoded exactly once. Literal percent signs and percent-escape-looking text previously broke because the handler decoded an already decoded URL path again. Tests also cover slashes, trailing slashes, plus signs, and Unicode through the handler and Android path forwarding. These forwarding tests do not substitute for full bearer-authentication integration tests.
- Audio and web package tests pass, including the existing ZIP download test.

Consumers still use the old full collection response. This is the detail API foundation for step 1, not completion of lazy overview loading. A future client must distinguish an unsupported detail endpoint on an older server from a missing playlist on a newer server before falling back, and must never treat an error as an empty playlist to overwrite.

## Android edit-path adoption (not packaged)

`MusicPlaylistViewModel.updateTracks` now calls `MusicPlaylistClient.get` before moving/removing a track. The existing serialized mutation/reconciliation state remains unchanged. On a new server this pre-edit read retrieves one playlist. The post-edit reconciliation still retrieves the full collection; overview and detail-screen lazy loading remain outstanding.

The client falls back to collection GET only on HTTP 405, the previous server's explicit unsupported-method response. A 404 is a missing playlist, not a reason to recreate it. Authentication errors, redirects, server errors, invalid JSON, missing/null detail `items`, and a mismatched returned name fail the edit. An explicit empty array remains valid. Connections close before the fallback request.

Local HTTP-server tests cover encoded names, one-request detail loading, ordered tracks, legacy fallback, error isolation, malformed responses, and empty playlists. No production account data is used. Direct Android playlist unit tests pass. This does not add cross-device compare-and-swap: a remote mutation between GET and PUT is still a separate concurrency concern.

## Summary API foundation (not deployed)

`GET /api/audio-playlists?view=summary` (also forwarded through the Android API) returns:

```json
{"version":1,"playlists":[{"name":"Saved","item_count":20,"artwork_urls":["..."]}]}
```

The explicit versioned object differs from the legacy full-list array. This lets clients distinguish an older server ignoring the query from a supported summary response without treating a preview as complete playlist contents. There is no `items` field. Artwork contains at most four distinct URLs in playlist order. Counts skip unavailable/non-music entries just like the full-detail projection. Empty accounts and empty artwork use arrays, never null.

The implementation still copies stored queue-entry references and checks every entry to derive accurate available-track counts. It avoids projecting full track metadata and generating full playback/download URLs for overview items; this is not an indexed constant-time count cache.

Synthetic handler test with one 20-track playlist plus one empty playlist: legacy JSON 13,797 bytes, summary JSON 536 bytes (about 96% smaller). This is an uncompressed fixture-specific payload comparison, not production latency, browser memory, or server RSS measurement. Tests cover collection/detail count and artwork agreement, ordering, missing entries, another account's playlist exclusion, empty arrays, and Android forwarding. Client overview adoption remains required before users benefit.

## Web single-item additions (not deployed)

Before enabling summaries, the web add-to-playlist controller needed a correctness change: it previously PUT the local playlist plus the new track. That can discard additions made on another device since the local list was loaded. It now uses the existing single-entry POST, followed by collection reconciliation. The server performs the append under its playlist lock and treats existing entries idempotently. The existing visible menu is unchanged.

This also removes the requirement for a complete overview track array when submitting an addition. Reconciliation is still a full collection GET until summary adoption; this change trades that read for safe atomic append, not a claim of lower total bandwidth for this action yet. Reorder/removal still use replacement writes and need separate concurrency handling.

New tests simulate a remote addition between initial load and local append, verify that POST contains only the new identity, preserve local state on rejected writes, and ensure the controller no longer submits a replacement array for additions. Full frontend suite: 451 passing tests. Impeccable optimization guidance was used to preserve incumbent controls and prioritize correctness before reducing overview payloads.

## Web summary adoption (not deployed)

The web requests `?view=summary`, accepts legacy arrays from older servers, and distinguishes summary models from complete details with `detailLoaded: false`. Overview and discovery cards retain counts and up to four artwork previews. Opening a summary, including from discovery, requests only that playlist and displays loading feedback with a Back control; failures retain the overview metadata and offer Retry. Playback and reorder controls are only rendered once full detail is available.

Detail responses are generation-checked against collection refreshes and mutations. A late detail response cannot resurrect a deleted playlist. The controller deduplicates its in-flight selected-playlist fetch and does not render a completed response after navigating away. Ordinary re-renders of the same selection no longer increment navigation identity. The new decoder is a separate module; no CSS/layout redesign was performed.

Unit coverage includes no eager detail requests, cached loaded details, late responses after deletion, explicit loading/retry instead of an empty playlist, old-server arrays, and malformed details. The isolated Chromium fixture now exercises summary selection → loading → detail playback at 1440px and 390px in addition to rename recovery. This is a component integration check, not a full authenticated app performance profile or physical-phone acceptance test.

Remaining: Android overview adoption, cross-device revision protection for replacement writes, release-wide verification and deployment, and production-scale timing/memory measurements. The 96% payload reduction above remains fixture-specific.

## Android summary transport (not activated in screens)

`MusicPlaylistClient.overview` now supports the versioned summary response and legacy full-array servers in one request. It returns a separate `MusicPlaylistOverview` type containing only name, count, and artwork URLs. It cannot be passed as a `MusicPlaylist` or a `List<AudioTrack>` to playback/replacement APIs. Response bytes, playlist count, item count, and artwork count remain bounded. Empty summary arrays are supported; malformed and unsupported summary structures fail.

Before activating it, `MusicPlaylistLibrary` artwork/count rendering and detail selection must use the new type. `MusicPlaylistsPanel` currently plays complete `playlist.tracks` directly, so it needs a name-based detail-loading action with pending/error handling instead. Request ownership must preserve selection-generation and mutation serialization guarantees. No existing screen has been switched to empty placeholder track lists.

New local HTTP and parser tests cover negotiated query, one-request older-server compatibility, deduplicated legacy artwork, counts, empty responses, unsupported versions, malformed structures, and limits. Both Direct and Store Android playlist test tasks pass. Existing layouts and APKs remain unchanged by this transport-only stage.

## Android screen adoption (not packaged)

Android playlist state now stores summaries separately from the selected full playlist. Opening one loads detail through the serialized request owner; selection generations discard late responses after leaving. Collection reconciliation reloads only selected detail, rather than expanding every playlist. The UI retains overview counts/artwork and existing detail controls, with Back/loading/Retry when detail is unavailable. Relative server artwork paths are resolved against the configured server; arbitrary absolute preview hosts are not supplied the authorization header.

The expanded-player playlist panel now uses a suspend name-based playback action. It fetches the full playlist before replacing the queue, disables repeated play requests while loading, and reports failure without switching panels. Its composition-owned coroutine is cancelled when the panel leaves composition; cancellation is rethrown rather than converted into playback. No APK was rebuilt. Physical-device visual, remote, and playback acceptance remain pending.

New request tests cover no detail fetching during overview refresh, leaving before a delayed response, and retry after failed detail. Existing rename/navigation, mutation serialization, and superseded-refresh tests were migrated to summary fixtures and continue passing. Full Direct and Store unit-test tasks passed after integration; release-wide lint and optimized APK validation are still required.

## Batch verification

- Full Go package suite passes (`go test ./...`).
- Full Android unit-test reports: 622 tests per Direct/Store variant; zero failures, errors, or skipped tests.
- Web: 456 frontend tests pass; isolated playlist browser flows pass at 1440px and 390px.
- Module-size and source store-readiness checks pass.
- Live Mochil remains healthy with zero container restarts. Worktree changes are not deployed and no new optimized APK has been built for this batch.

Android lint, full audio/web race checks, optimized release validation, and device acceptance are recorded separately as they finish. Green unit tests alone do not prove TV remote behavior, absence of visual regressions, or production latency improvements.

Follow-up: `go vet ./...` and complete `go test -race ./internal/audio ./internal/web` both passed. Direct Android lint completed successfully; Store lint remains running at this checkpoint. Do not interpret the prior Store HTML/XML report as a result for the current run before its Gradle task finishes.
