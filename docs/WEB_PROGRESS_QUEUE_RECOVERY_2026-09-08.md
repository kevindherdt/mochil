# Web audio/video progress queue recovery

Status: source fix after live 0.16.79, awaiting the next tested deployment batch.

## Failure

The shared `createProgressSync` serialized requests by chaining each save onto `pending.then(...)`. If a transport error escaped, that pending promise stayed rejected. All subsequent submissions inherited the rejection without calling the save function. Catching the error in audio/video callers did not repair the internal queue; even changing items via `reset` left the rejected chain intact.

## Fix

Keep the caller's result promise separate from the serialization tail. Return the original result (including rejection), but retain a caught tail for scheduling the next write. Expected-revision handling, generation checks, one conflict retry and explicit conflict suppression remain intact. This does not blindly retry a failed payload or claim an uncertain write succeeded; a later submission still uses server revision checks.

## Verification

Regression tests exercise network failure followed by an already-queued save, failure during a reconciled conflict retry, reset to a new item after failure, and the real audio-progress wrapper resuming timed saves after offline failure. Existing serialization/passive-client/revision tests remain in the full frontend suite. Both web audio and video use this shared queue; no Android runtime change is claimed.

No layout changes, file mutations or deployment occurred. JavaScript responses already use revalidation, avoiding immutable stale dependency reuse when this batch is deployed.

## In-flight item-switch isolation

Follow-up audit found that revision generation guards ran only after a conflict resolver returned. The resolver could already mutate the shared passive-ownership state using an old item's delayed response. Audio and video resolvers now receive an `isCurrent` predicate and check it immediately after the remote lookup, before applying ownership changes. Successful old-generation saves return false rather than letting callers treat them as current-item success. Audio also guards its post-await seek-claim release and error throttle state by generation.

New regressions delay a real audio conflict lookup across an item reset, verify an old finished response cannot suppress the new item or clear its explicit seek claim, and verify shared queue ownership predicates and late success results. Full frontend suite: 538 tests passed; module gate and video module syntax check passed. No device-level sync acceptance or new deployment is claimed.
