# Music favorite request ownership

Status: source implementation after 0.16.83; not yet published.

## Findings

`MusicViewModel` previously issued concurrent favorite writes and admitted every
favorites refresh. An older list could overwrite an optimistic toggle, and rapid
toggle requests could reach the server in the wrong order. Reversing a failed
click alone also fails when an earlier pending click failed too.

## Implementation

`MusicFavoriteRequests` owns this responsibility outside the catalogue view model.
It serializes writes, invalidates older list responses when an action begins,
defers refresh while actions remain pending, and tracks confirmed state separately
for each pending track. Only the latest action for a track reconciles its optimistic
state and reports the outcome. Refresh failures preserve the available list;
coroutine cancellation is not reported as an ordinary failed save.

The existing mobile/TV UI, favorite identity and account API are unchanged. This
does not provide offline durable favorite writes or server-side cross-device
conditional mutation. An ambiguous network failure still requires a later
successful refresh to discover the server's actual state.

## Tests

Controlled coroutine tests cover stale read admission, ordered/nonoverlapping
writes, two failed toggles, and a failed undo after a confirmed successful write.
An initial test declaration accidentally returned Boolean to JUnit; it was fixed
to return Unit before rerunning. Both full Direct and Store unit suites passed
(`/tmp/bobobooks-favorites-requests-tests.log`, 12 seconds), as did the module gate.

## Library switch follow-up

Opening another music library now preserves the account-scoped favorite snapshot
while refreshing it. The request owner also applies a confirmed successful result
to the current state rather than relying solely on the optimistic click-time state.
This covers navigation that replaced that state before the save completed, including
a subsequent failed refresh. The current library and its loading state stay intact.

A fifth coroutine regression covers this sequence. Both full flavor unit suites
and the module gate passed again (`/tmp/bobobooks-favorites-library-tests.log`,
13 seconds). These changes remain unpublished after 0.16.83.
