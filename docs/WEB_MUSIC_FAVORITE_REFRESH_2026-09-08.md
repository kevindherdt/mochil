# Favorites refresh ownership

Status: worktree after 0.16.76, not deployed.

Favorites reads now have generation ownership: an older overlapping response cannot replace a newer result. Each active read records successful per-track toggles made while it is pending, then merges those changes into the returned list. This preserves unrelated server favorites even during the initial load, while preventing a stale response from undoing a recent add/remove. Per-read change maps are released on completion, not retained for the session lifetime.

Failed or malformed responses retain the last successful list and report an error rather than treating failure as an empty collection. The existing null-as-empty response contract is preserved; the initial regression run caught that compatibility requirement. Tests cover initial-load add, pending-refresh removal, unrelated server items, overlapping reads, failed refreshes, and malformed objects. No layout, APK, or running server changed.

## Shared refresh lifetime

The music collection loader now gives its completion/finalizer ownership checks too. An earlier refresh cannot mark a newer one complete or clear its pending promise. Normal callers join an active refresh, including when a previous collection was already loaded, instead of returning prematurely with stale data. Two deferred-response tests verify old-first completion and joining a forced refresh after an initial successful load, with no redundant reads. Full frontend tests and module checks pass after this change. This is request coordination, not a measured rendering-performance claim.

## Playlist submission snapshots

Serialized playlist saves now capture a shallow track snapshot and request body when submitted, before waiting behind earlier mutations. Append captures the selected library/item identity at submission too. Later caller changes cannot silently change the queued request or the top-level metadata published after save. Nested metadata is not deep-cloned because the payload needs only stable IDs; this is not general deep immutability. Deferred tests cover changed array contents and changed track IDs/titles while another write is pending. Full frontend tests and module checks passed. These post-0.16.76 changes remain undeployed.

## Favorites responsibility extracted

`music-favorites-state.js` now owns favorite items/identity lookup, read generations, in-flight reconciliation maps, per-item serialized writes, and load errors. `music-state.js` delegates through its existing public interface and continues coordinating aggregate loading alongside playlists. No queue or playlist state enters the favorites owner. This separates a real state/lifecycle responsibility rather than splitting to meet a line threshold.

Existing full frontend tests and the isolated browser playlist fixture validate the retained facade behavior and module loading. No UI or release change is part of this extraction; runtime performance improvements are not claimed for modularization alone.

## Failed playlist refreshes

A failed/malformed playlist overview refresh now preserves the last usable collection and reports its error; a genuine successful empty response still clears it. After a confirmed append, the affected cached detail is marked unverified before reconciliation, so retaining the overview on network failure does not imply the old track list is current. Opening it requires a fresh detail load. Tests cover failure, malformed overview, recovery to empty, and a confirmed append followed by failed refresh. All 485 frontend tests and module checks passed. Changes remain undeployed.

## Shared playlist detail reads

Concurrent opens now share a pending detail request for the same name and playlist generation. Different names load independently. Refreshes and mutations retain their generation invalidation; an older request's cleanup cannot remove a newer pending request. Completed requests leave the in-flight map, including failures, so retry remains possible. This reduces two concurrent same-playlist reads to one in the deterministic test; no production latency or memory percentage is claimed.

Three deferred-response tests cover sharing, independent names, refresh replacement and cleanup ownership, failure/retry, and edits invalidating pending details. All 488 frontend tests, module checks, and the isolated playlist browser checks at 1440px and 390px passed. No visual changes, Android build, or deployment were made.

## Favorite submission identity

Favorite toggles now take a shallow selected-track snapshot immediately at submission. Previously the serialization key was captured immediately, but the network identity and published item still referenced the caller's mutable object. Changes while waiting could therefore send a request for a different track while updating the original key. Both request and local reconciliation now use the same captured selection. Nested metadata remains shared; this change guarantees top-level identity/metadata stability, not general deep immutability.

Three tests cover mutation before the first request, mutation while waiting behind another toggle, and failure isolation with another track and queued retry. All 491 frontend tests and module checks passed; the isolated playlist browser checks passed at 1440px and 390px. This remains an undeployed state-layer change with no layout change or new APK.

## Queue selection snapshots

Add-to-queue and play-next capture selected track identity before awaiting initial queue loading; play-next also captures its supplied current-track anchor. Queue replacement takes shallow item copies so later caller mutations cannot diverge local identities from the serialized payload, including on subsequent reorder/save. Null selections are ignored without fetching or writing. Nested metadata is not deep-cloned. Copying queue item objects adds linear allocation during edits; this is a correctness change, not a claimed performance improvement.

Four regressions cover delayed append, delayed play-next anchoring, caller mutation followed by reorder, and two simultaneous additions sharing one initial read without losing either item. All 495 frontend tests, module checks, and isolated playlist browser checks at 1440px and 390px passed. No deployment or APK build was made.

## Explicit queue replacement supersedes pending actions

Four new regressions first failed on the previous code: add/play-next waiting for the initial GET restored a track after a newer explicit clear or replacement. Queue replacement now has its own generation, separate from save generations. Waiting additions check it before committing. Relative additions do not advance it, preserving simultaneous additions; a save retry also does not invalidate pending actions. Existing direct remove/reorder operations use the explicit replacement path.

The regressions also verify that newly requested additions after replacement still work and superseded actions send no extra PUT. All 499 frontend tests, module checks, and isolated playlist browser checks at 1440px and 390px passed. This is local request ownership, not a claim of cross-device queue conflict resolution. Changes remain undeployed.

## Queue responsibility extracted

`music-queue-state.js` now owns queue data, lazy load coordination, save/replacement generations, serialized writes, edits, retry, and error state. It contains no favorites or playlist state. The existing music facade delegates the same public methods and re-exports `queueWithNext`, preserving existing imports. This separates a complete lifecycle responsibility rather than chasing a line-count threshold; no runtime speedup is claimed for the extraction.

All 499 frontend tests, module checks, and isolated playlist browser checks at 1440px and 390px passed after extraction. Queue edits were also expanded from dense one-line functions for readability. No UI change, APK build, or deployment is included.
