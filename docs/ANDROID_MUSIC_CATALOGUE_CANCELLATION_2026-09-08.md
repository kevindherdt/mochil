# Music catalogue request lifetime

Status: implemented after 0.16.83; pending next APK batch.

The generation guard previously discarded obsolete music responses but did not
cancel their coroutine work. `MusicCatalogueRequests` now owns one supervisor job
per browsing context and cancels its children before advancing to a new context.
Album/artist detail, discovery, recent tracks, root pages, alphabet navigation and
artist queue preparation use this owner. Generation checks remain as a second
boundary against stale results.

Favorites/account writes and downloads remain in their separate owners and are
not cancelled by browsing. Coroutine cancellation is excluded from ordinary
catalogue error/fallback handling.

Tests verify cancellation of all old read children, preservation of unrelated
download work, successful loading in the new context, prevention of undispatched
obsolete work, and shutdown cancellation. Both full Android flavor unit suites
and module checks passed (`/tmp/bobobooks-music-cancellation-tests.log`, 13 seconds).

This is cooperative cancellation. A blocking HTTP call already executing may run
until its existing timeout; no forced thread interruption or cancellation of
server-side work is claimed. Physical TV performance remains unmeasured.
