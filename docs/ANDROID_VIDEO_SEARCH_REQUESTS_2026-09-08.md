# Android video search request ownership

Status: published in Android 0.16.83 on September 8, 2026. No layout changes.

## Evidence and fixes

The previous search callback changed the query immediately but only advanced the
request generation after its 250 ms delay. A response for the old query could
therefore still update the catalogue during that delay. A delayed callback also
read whichever library was current, triggering a redundant load after navigation.

`VideoSearchDebounce` now owns the delayed request trigger, using the existing
catalogue generation to invalidate old results immediately and suppress a pending
callback after navigation. Rapid typing still coalesces into one request.

`VideoViewModel.search` cancels the old load and clears old pagination cursors while
the new query is pending. A cancellation checkpoint between the root catalogue
request and the follow-up continue-watching request avoids that second request
when the root load has already been cancelled.

This does not interrupt a blocking HTTP request already executing. No measured
latency or memory reduction is claimed.

## Follow-up: pagination and mutation failures

The view model now retains and cancels its page-loading job when searching or
loading another root/show/season. Calls for the wrong pagination kind are ignored,
so an obsolete show-list callback cannot fetch show pages inside an episode view.
Existing generation and offset checks still guard successful responses.

Watched/dismiss mutations capture their initiating catalogue generation. A failure
is displayed only in that generation; coroutine cancellation is not presented as
a save error. Reporting a mutation error no longer clears catalogue loading state.
The server-side mutation itself is not cancelled just because the user navigates.

The failure-policy regression test covers current errors, cancellation, and stale
errors after navigation. Pagination cancellation wiring was code-reviewed; physical
TV navigation under delayed networking remains unverified.

Both full flavor unit suites and the module-size gate passed after this follow-up
(`/tmp/bobobooks-video-request-final-tests.log`, 12 seconds). No APK was built or
published for these changes alone.

## Verification

Three deterministic coroutine tests cover immediate invalidation, navigation
during debounce, and rapid typing. They use a controlled suspended delay rather
than timing-dependent sleeps. Both complete Direct and Store debug unit-test
suites passed again with the final cancellation checkpoint included (12 seconds;
`/tmp/bobobooks-video-search-final-tests.log`). The module-size gate passed.
