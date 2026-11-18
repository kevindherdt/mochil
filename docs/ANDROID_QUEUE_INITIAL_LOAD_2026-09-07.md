# Android queue initial-load safety

Status: shipped in Android 0.16.75, published September 7 at 18:47 UTC. Chronological unbundled notes below describe the preparation stage.

## Confirmed issue

Queue loading is lazy, but `MusicQueueViewModel.append`, `playNext`, and their multi-track variants previously transformed the default local empty list immediately. Using them before a successful initial read could replace the saved server queue with only the newly selected tracks. `MusicQueueClient.get` also treated HTTP 404 as a successful empty queue.

## Fix

The request owner accepts queued transformations while the initial queue is unknown. Concurrent pending edits share one read and are applied in order to the successfully loaded queue, then persisted. A failed read clears those pending edits, reports the existing load error, and never writes a replacement. Recovery does not silently replay failed user actions. An explicit replacement/clear supersedes older pending edits and makes delayed reads stale. Bulk additions capture their input list before queuing the transform.

Queue GET now requires a successful array response. A 404, server error, or malformed body does not establish an empty queue. No mobile layout or APK changes accompany this fix.

## Verification

New tests cover shared initial reads, preservation of existing track order, failed reads with no writes/replay, and an explicit replacement superseding a delayed append. Local HTTP tests distinguish successful empty arrays from 404/503/null responses. Existing queue request-ordering tests continue passing.

These are local-client safeguards. They do not add cross-device revisions or a durable offline edit journal. Queue action notification timing and explicit retry semantics need a further review so UI wording does not claim success before a queued operation finishes.

## Failed-save refresh protection

Further inspection found that an explicit refresh after a failed PUT could replace retained edits with the older server queue. The request owner now tracks unsaved changes independently of loading/error text. Refresh retries the retained snapshot until its latest write succeeds; only subsequent refreshes resume reading the server. An intentionally cleared queue receives the same protection. Older successful writes cannot clear this flag for newer edits because completion remains generation-guarded.

`MusicQueueRetryTest` covers failed-save retry without GET, normal GET after successful retry, and repeated failed retries of an empty queue. Both complete Direct and Store debug unit-test tasks passed on September 7. This remains an unbundled worktree change; no APK or deployment was changed. A visible retry action and queue notification timing remain separate follow-up work.

## Mobile recovery surface

The expanded queue now wires its conditional retry action through the playback host to the protected request owner's refresh method. Loaded queues offer “Retry queue sync”; unknown queues offer “Retry loading queue”. Errors use a polite accessibility live region. Initial loading receives text feedback, and failed/unknown loads no longer render the misleading successful empty state. Normal queue controls and layout remain unchanged. Impeccable's hardening guidance informed this narrowly scoped recovery UI.

Direct and Store unit-test suites and the module boundary gate passed after wiring the UI. No device is attached to ADB, so rendered touch/TalkBack acceptance remains unverified and the change is not packaged.

Notification review also confirmed that the existing “Added to queue” and “Playing next” actions immediately update the local playback queue as well as requesting persistence. Those messages are not proof of a successful server save, but are not simply premature network-success messages either. Any future wording change should preserve that distinction rather than delaying local playback until a server responds.

The Direct and Store lint tasks subsequently completed successfully (4m48s) for the queue recovery UI. The source-contract regression test checks retry wiring and empty-state gating; it is not a rendered interaction test.

## Playlist response boundary follow-up

The playlist model already rejects saving an empty queue. Separately, summary decoding used `JSONObject.getInt`, which can truncate fractional numbers and wrap large values. Summary versions and item counts now require actual numeric, finite, integral values within their bounds before conversion. Tests reject strings, booleans, null, fractions, overflowing integers, and huge exponents; zero, the 10,000-track limit, and integral decimal representations remain supported. Both full Android unit-test tasks passed after this parser change. The preceding lint run covered the queue UI, not this subsequent parser edit.
