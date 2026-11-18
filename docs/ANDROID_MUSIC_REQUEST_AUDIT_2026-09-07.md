# Android music request ordering

Source audit: `MusicQueueViewModel.kt`, `MusicPlaylistViewModel.kt`, and their
HTTP clients. These are verified unguarded execution paths, not a claim that a
particular user log reproduced each race. No UI or APK changed in this audit.

## Queue: failure recovery can overwrite a newer local edit

`persist` updates visible tracks optimistically, then schedules a mutex-serialized
write. On failure it starts `refresh()` outside that mutex. Refresh has no
generation check and replaces the complete state on success. A possible order is:

1. Queue A starts saving.
2. The user edits to queue B; its save waits behind A.
3. A fails and starts an independent GET.
4. That GET returns old server queue S and replaces visible B.
5. A waiting save reads `mutableState.value.tracks` and can persist S instead of B.

Even where step 5 reads B first, the delayed GET can still replace its visible
state. Failure recovery also clears its own save-error message when starting
refresh. An older completed write can set `saving=false` while a newer write
is still pending. Cancellation is caught as an ordinary failure by runCatching.

## Playlists: refreshes and transformations lack a shared transaction boundary

Refresh GETs are neither serialized nor generation-checked. Mutation success
launches another independent refresh instead of completing its reconciliation
inside the mutation boundary. `updateTracks` captures the playlist before
entering the write mutex; two quick edits can therefore both transform the same
old list. Rename changes selectedName before success and does not restore it
on failure. These issues require preserving user navigation as well as data.

## Preserve

Clients already bound response parsing, use finite timeouts, disable redirects,
and disconnect HTTP connections in finally blocks. Nullable playlist items are
normalized to an empty array. Keep those guarantees and existing UI surfaces.
Credentials are captured at ViewModel construction; this audit does not establish
whether server switches always replace that ViewModel scope, so no cross-server
leak is claimed from that observation alone.

## Implementation order

1. Give queue request ordering one independently testable owner: immutable
   mutation snapshots, freshness-guarded reads and acknowledgements, durable
   error state for the newest failed write, and cancellation propagation.
2. Add delayed-response tests for edit-during-save, old-refresh-after-new-edit,
   failed-old-write recovery, repeated queued writes, and scope cancellation.
3. Serialize playlist transformations against the latest reconciled state;
   guard selection separately so completion cannot navigate the user backward.
4. Run Direct and Store tests; batch these with the pending TV/Wiki fixes before
   producing the next update. Do not claim cross-device transactional queue
   editing: the server currently accepts last-writer-wins replacements.

## Queue implementation follow-up

`MusicQueueRequests.kt` now owns request generations, immutable mutation
snapshots, serialization, and acknowledgements. The Android ViewModel keeps
credential/IO wiring and queue-edit policy. Old GETs cannot replace a new edit;
refresh is ignored during an active save. Already superseded queued writes are
skipped, while an in-flight older write cannot clear the newest saving/error
state. The latest failure retains its local edit and visible error without an
automatic destructive refresh. Cancellation propagates, including a transport
exception arriving after cancellation. This retains unsaved edits in memory,
not across process death; no offline durable outbox is claimed.

Five coroutine tests exercise delayed reads, failed older writes with a waiting
edit, immutable failed snapshots, queued-write coalescing, and cancellation.
Direct and Store queue suites and the module-size gate pass. No UI or APK changes
were published. Playlist transaction/selection fixes remain the next step.

## Playlist implementation follow-up

`MusicPlaylistRequests.kt` now serializes reads, writes, and post-write reloads.
Older refresh responses are discarded after a mutation is requested, and saving
remains true while another mutation is pending. Renames retain the old selection
on failure and update it after success only if the selection visit has not changed.
Successful writes followed by a failed reload are reported distinctly.

Track edits capture the tapped identity, fetch the latest playlist inside the
serialized operation, then transform that snapshot. This adds a pre-write GET
for track edits rather than risking a stale replacement; it is not atomic against
another device's concurrent replacement. Save-queue captures its explicit input
before enqueueing. Credential/HTTP dispatch remains in the ViewModel while request
state has its own small owner.

Direct/Store playlist suites and module-size checks pass. New coroutine cases
cover failed/delayed rename selection, sequential reconciliation with pending
saving state, and a stale refresh preceding a mutation. These source changes
remain pending the next APK batch; no mobile layout was changed.
