# Android response-reader boundary audit

Unpublished source changes; no APK or live deployment.

The inspected audio/video mutation paths put output-stream writes inside their
disconnect `finally` blocks, unlike the wiki setup issue fixed separately.
No equivalent body-upload cleanup change was needed in those inspected paths.

The shared `readBoundedResponseText` helper could consume another entire 8 KiB
chunk before rejecting a response over its configured limit. It now limits each
read to remaining capacity and uses one byte to distinguish exact-limit EOF from
overflow. The accumulator cannot exceed its positive integer ceiling, avoiding
addition overflow. Existing payload limits, UTF-8 behavior, and caller ownership
of stream closing remain unchanged.

Tests cover exact limits, over-limit consumption at several chunk boundaries,
single-byte short reads splitting Unicode characters, empty responses, invalid
limits, and a short response with an Int.MAX_VALUE ceiling. Both Direct and Store
Android debug unit suites pass.

This bounds excess network consumption more tightly; it is not evidence of a
measured steady-state RAM, CPU, or playback-speed improvement. Callers still
allocate the accepted payload and JSON model. Endpoint-specific smaller limits
for tiny progress responses remain a potential follow-up after contract review.

## Single playback-record limit follow-up

Reviewed `internal/audio/progress.go` and `internal/video/progress.go`: the
individual progress endpoints return position, duration, revision, timestamps,
identifiers and flags, not artwork, descriptions or history arrays. Audio and
video single-record clients now share `readPlaybackProgressResponse`, with a
64 KiB ceiling instead of the general 16 MiB ceiling. Collection/history readers
are unchanged. The accepted-payload ceiling is 256 times smaller; this is not a
256-times reduction in normal memory usage.

Tests preserve exact-boundary JSON/Unicode fields and reject oversized valid JSON.
A loopback HTTP test verifies the actual audio client rejects oversized GET and
PUT responses, does not upload after an oversized initial GET, does not acknowledge
an oversized PUT reply as successful, and can make a later successful request.
The bounded shared parser is also wired into both video GET/PUT response paths.
No automatic retries were added and no progress policy or UI changed.

The video client now has its own loopback HTTP regression covering oversized PUT
acknowledgements and oversized GET responses after a 409 conflict. It verifies
that neither replaces the cached revision nor causes an extra upload in that
attempt, and that subsequent valid replies restore ordinary revision-aware
updates. Both Android debug suites pass with these integration tests. This does
not constitute device playback or network-interruption acceptance testing.
