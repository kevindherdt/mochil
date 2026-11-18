# Audio progress replay cancellation

## Finding and change

Playback service startup previously iterated the saved outbox using blocking sends wrapped in `runCatching`. Coroutine cancellation did not stop that loop between requests, and cancellation exceptions were treated like ordinary failures.

`replayPendingAudioProgress` now checks cancellation before replay, before each send, and after each blocking send. It propagates cancellation exceptions and acknowledges only successful sends while still active. The journal's existing `removeIfCurrent` comparison remains responsible for protecting newer pending entries. Ordinary network exceptions/false results leave the entry pending and permit later entries to be attempted as before.

An already-running HTTP request may still finish on the server after cancellation; this change cannot roll it back or immediately interrupt `HttpURLConnection`. It prevents later replay requests and cache acknowledgments after cancellation is observed. Any retained entry is subject to the existing passive-progress/revision/seek-age policies on its next replay.

## Verification

Four tests cover mixed success/failure with exact-entry acknowledgment, cancellation during a non-cooperating blocking send, an explicit cancellation exception, and cancellation before replay. Both full Android variant unit suites passed (686 tests each), along with module checks.

No new APK or deployment. This is lifecycle safety, not a complete audit of cross-device progress ownership or concurrent live-save ordering; those remain separate review areas.

## Acknowledge the committed seek record

The durable journal merges an unsent explicit seek into a subsequent passive pause. Previously `put` discarded the merged return value, so the service sent the unmerged pause and later passed that original record to equality-based `removeIfCurrent`. It did not match the saved record, leaving it queued even after a successful send.

Journal commit now returns the exact merged record only when encrypted persistence succeeds. The service uses that record for sending and acknowledgment; failed persistence does not authorize removal of an older journal entry. Ordinary nondurable sends still work without a journal commit. Live sends also use the cancellation-aware delivery helper. This does not interrupt an already-running HTTP call or solve all ordering between concurrent sends.

Four tests cover inherited seek identity and acknowledgment, failed persistence, owner isolation, and newer pending progress surviving an older acknowledgment. Both full Android suites passed (690 tests each); module checks passed. No new APK or deployment.

## Admission of queued progress sends

`AudioProgressSubmissions` assigns bounded per-owner/library/item tickets before live send coroutines start. New submissions invalidate older tickets for that identity; startup replay cannot replace a known live submission. Pending seek intent transfers to newer passive submissions with its original expiry, but acknowledged tickets no longer transfer explicit ownership. The short ticket lock never encloses network work; retention is capped at 256 identities, with evicted tickets invalidated.

`AudioProgressClient.save` checks admission inside its existing serialized lock, after its initial remote read, and before/after conflict reads. The service supplies both cancellation and ticket checks. A stale call therefore cannot proceed merely because it passed an earlier check before waiting on the network lock. Existing revision and passive-position policies still apply.

Tests cover same-item replacement, owner/item independence, replay suppression, pending versus acknowledged seek ownership, bounded ticket eviction, and actual loopback HTTP request counts when admission closes before GET, before PUT, or before conflict retry. Both full Android unit suites passed (695 tests per variant), along with module checks.

Limitations: already-started HTTP mutations cannot be rolled back; ticket admission is not a distributed transaction. Durable outbox entries remain until acknowledged, while nondurable superseded updates are skipped. Work waiting on the client's lock is not itself removed from the dispatcher queue; a bounded delivery worker is still a possible performance improvement. No APK or deployment yet.

## Lock-contention verification

A two-thread loopback test holds the first progress GET open, observes the second sender actually blocked on the client's synchronized lock, then closes admission before releasing the first response. The waiting sender returns false without issuing any HTTP request. Latches/futures have bounded timeouts and server/executor cleanup runs in `finally`.

Both full Android suites passed after this test (696 tests per variant). Full Go tests, all 499 frontend tests, module checks, and isolated playlist browser checks at desktop/mobile widths also passed across the accumulated worktree changes. These are source/debug checks; no new optimized APK, on-device playback acceptance, or deployment occurred.

## Bounded delivery worker

The playback service now routes startup replay and live progress through one IO worker. `AudioProgressDeliveryQueue` retains one pending closure per identity, with a 256-identity cap plus one active operation. Repeated queued updates replace earlier work instead of spawning further coroutines waiting on the network lock. Cancellation clears pending work; ordinary exceptions do not kill later sends. Existing ticket admission still protects the interval before a queued closure reaches the worker.

Durable journal persistence remains separate and happens before enqueueing. Coalesced tickets carry the exact pending journal acknowledgment, including when a later nondurable update replaces durable work; newer journal records remain protected by equality checks. Queue overflow evicts oldest pending identities, not durable journal entries. This is bounded best-effort delivery, not guaranteed transmission of every transient position.

Tests hold the worker busy and submit 1,000 updates for one item: only the latest queued update and independent other-item work execute after release. Other tests verify capacity eviction, continuation after send exceptions, cancellation/rejection, and durable acknowledgment transfer. Both Android variant suites passed (700 tests each), and module checks passed. No device CPU/RAM percentage is claimed; the verified improvement is bounded pending work and coalesced sends. No APK or deployment yet.

## Producer handoff race

Startup replay can obtain a ticket and then be delayed before enqueueing while the main thread submits newer live progress. Merely checking tickets before HTTP would reject the old send but would not prevent its late queue insertion from replacing newer queued work. Queue submission now checks ticket admission inside the short queue lock before replacing any pending closure. The ticket lock does not acquire the queue lock, avoiding reverse lock ordering.

A regression holds the worker busy, creates old replay and newer live tickets, enqueues the new work first, then attempts the delayed old insertion. The old insertion is rejected and the new position is delivered. Both full Android suites passed (701 tests each), and module checks passed. No new APK or deployment.
