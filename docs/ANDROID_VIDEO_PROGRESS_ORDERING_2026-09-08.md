# Video progress submission ordering

Unpublished changes; no APK/deployment or device acceptance claim.

VideoPlayerActivity previously launched IO saves independently, while the client
serialized network work but had no way to reject an older waiting request after
newer intent arrived. The activity now registers a per-item ticket before launch.
VideoProgressClient checks ticket admission inside its synchronized entry point,
before fetching conflict state, and again before retrying with that state.
Superseded requests return false rather than continuing the conflict sequence.

The bounded 256-item ticket owner uses server/device/library/item identity.
Replacing a pending explicit seek with a passive checkpoint retains seek intent
for up to ten minutes, using monotonic elapsed time. Successful acknowledgement
clears retained intent; an older completion cannot acknowledge a newer ticket.
Eviction invalidates older tickets. Non-finite positions/durations are rejected
before networking.

Both Android debug test suites pass. Tests cover ticket supersession, retained
seek intent, expiry, identity isolation, bounded eviction, and loopback HTTP
admission before PUT, before conflict GET, and after GET before retry. Existing
oversized-response/revision-cache tests continue passing.

Already transmitted HTTP writes cannot be recalled; this is not a cross-device
transaction or durable video journal. Activity shutdown and final-checkpoint
lifecycle are unchanged. IO jobs still launch individually: bounded delivery and
durable final-position recovery remain follow-up work, not claims of this patch.

## Bounded delivery follow-up

`VideoProgressReporter` now borrows one lifecycle-owned worker rather than launching
one IO coroutine per checkpoint. The existing audio delivery primitive has been
renamed `PlaybackProgressDeliveryQueue` and reused without changing its semantics:
one active operation, at most 256 pending item keys, replacement of queued work
for the same key, admission checked under the queue lock, and cleanup on scope
cancellation. Audio continues using the same implementation and journal policy.

A loopback slow-server reporter test sends an active update, queues a seek for
another item, then replaces it with 1,000 passive checkpoints. Only the active
request and latest queued checkpoint reach the server; the latter retains seek
intent. Both Android debug suites and the module-size gate pass. This verifies
bounded pending work, not measured device CPU/RAM savings. Video durable final-
position recovery and physical-device shutdown tests remain outstanding.

## Durable checkpoint format groundwork

Added `PendingVideoProgress` and its bounded codec/merge policy. A checkpoint
contains a hashed server/device owner, library/item IDs, position/duration,
revision, completion flag and timestamped seek intent. It cannot contain stream
URLs, subtitle payloads, titles or authentication tokens. The format accepts at
most 64 checkpoints and 256 KiB, rejects invalid numeric/identity fields, and
merges newer positions without losing an unsent seek timestamp. Replay seek
authority expires after ten minutes and rejects future-dated timestamps.

Both Android debug suites pass with round-trip, privacy-field, identity isolation,
size/count/invalid-data, retention and seek-expiry tests. This is format groundwork
only: it is not wired into encrypted storage, playback restoration or replay yet.
Do not advertise durable video progress until those integrations and tests exist.

## Encrypted journal storage groundwork

`VideoProgressJournal` now provides bounded owner-filtered reads, synchronous
commit with explicit failure reporting, and exact-value acknowledgement removal.
Its Android factory uses the existing AES-GCM/Keystore-backed payload store with
the 256 KiB plaintext bound and server/device-partitioned preference names.
Existing backup/cloud/device-transfer rules exclude all these preferences.
A shared process lock protects overlapping journal instances during recreation.

Tests reopen the journal over retained fixture bytes, preserve pending seek intent,
reject foreign owners, retain newer checkpoints after older acknowledgements, and
exercise false/throwing persistence failures. Both Android debug suites pass.
The tests establish storage orchestration, not physical Android Keystore operation.
Playback capture, startup replay, and local resume remain unwired; this is not yet
an end-to-end durable progress feature. No APK/deployment.

## Playback capture integration

The activity now lazily supplies an encrypted journal to its progress reporter.
Each checkpoint commits synchronously before delivery admission, so cancelling
the network scope cannot discard an already committed final position. Successful
delivery acknowledges only the exact stored record; queued replacements carry
the prior durable acknowledgement when a newer storage attempt fails. Pending
journal seek timestamps are translated to monotonic age without renewing their
original expiry. Keystore initialization/write failures record privacy-safe
diagnostic categories and do not prevent ordinary playback/network delivery.

Reporter tests now exercise the journal during a 1,000-checkpoint slow-server
burst, verify eventual acknowledgement removal, and retain a merged checkpoint
when the delivery scope was already cancelled. Both debug suites and module
checks pass. Startup replay and local resume integration remain next; capture
alone does not yet restore progress after reopening. No APK/deployment.

## Startup replay integration

Server-connected player startup now schedules journal replay through the bounded
delivery worker. Entries are owner-filtered, retain their recorded revision, and
do not need stream URLs. Replay admission cannot replace a live ticket for the
same item. Seek authority expires by original timestamp and is checked again at
delivery, rather than remaining valid indefinitely while queued. Successful
synchronization removes only the exact checkpoint; failures leave it available.

Loopback tests cover failed replay, retained storage, a new reporter successfully
replaying the checkpoint, revision preservation and expired seek intent. Ticket
tests cover live-over-replay ownership and expiry during queuing. Both Android
debug suites and module checks pass. Physical process-death/Keystore acceptance
is not available here. Local starting-position selection remains incomplete:
startup must choose a matching unsent checkpoint before playback can emit a
checkpoint from an older server-provided starting position. No APK/deployment.

## Local starting-position integration

The activity now resolves the selected queue entry against the encrypted journal
before starting replay or initializing Media3. Matching requires owner, library,
item and media revision. Only the selected entry's position/completion state can
change; titles, URLs, subtitles and other entries remain untouched. Ordinary
pending progress cannot move behind the incoming remote position or reopen a
remote-finished item; a recent explicit seek retains the same bounded authority
as replay. Finished pending checkpoints are replay work, not an instruction to
open the item at EOF. Known duration bounds the restored position.

Checkpoint encoding now includes bounded `media_revision`; a replacement file
cannot inherit an earlier file's pending seek timestamp. Tests cover identity and
media mismatch, remote-ahead/finished state, recent/expired seeks, selected-entry
isolation and duration bounds. Both Android debug suites and module checks pass.
This establishes the local startup integration in source; offline reopening,
process-death and Keystore behavior still need physical-device testing. Remote
replay media-revision validation remains a separate server-contract concern.
No APK/deployment.

## Indexed media-revision precondition

Android live writes and journal replay now include `expected_media_revision`
when known. The server compares it with the current indexed video revision before
changing progress; a mismatch returns HTTP 412 with `video_media_changed`, not
the HTTP 409 progress conflict that prompts a retry. Older clients without the
optional field retain existing behavior. Rejected journal entries are retained,
not falsely acknowledged or silently deleted.

A Go regression replaces and rescans a temporary video at the same catalogue ID,
rejects the old revision even for an explicit seek, verifies stored progress did
not change, and accepts the new revision. Android payload/replay tests verify the
field survives delivery. Both Android suites and targeted Go progress tests pass.
This checks the indexed revision, not an atomic live filesystem/content-hash
transaction; unscanned changes remain outside the guarantee. No APK/deployment.
