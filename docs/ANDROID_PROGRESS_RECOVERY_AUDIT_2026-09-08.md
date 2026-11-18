# Native audio/video progress recovery audit

Status: regression coverage after 0.16.79; no native runtime change or new APK in this pass.

The browser rejected-promise queue bug does not map directly to Android. `PlaybackProgressDeliveryQueue` catches ordinary send exceptions within its worker, continues with queued work and propagates cancellation. Audio/video submission tickets prevent stale producers from replacing newer intent, with admission checks before sending and after conflict lookups. The revision caches are keyed per device/server/item rather than one active-item global revision.

Added `PlaybackProgressRecoveryTest` against a disposable loopback HTTP server using synthetic credentials. Both real clients encounter a 503 mutation response, then successfully save twice with expected revisions 0, 0, 1. A failed mutation is not acknowledged and does not advance the revision. This is stronger than a mocked queue callback, but is not a physical-device network interruption test.

Direct and Store debug runs executed and passed the new HTTP tests plus delivery-queue, audio admission and video admission suites (Gradle 8s). Those existing tests cover bounded/coalesced pending work, continuation after send failure, lifecycle cancellation, blocked stale producers and supersession during conflict recovery.

Further performance investigation: video checkpoint persistence is deliberately synchronous before enqueueing, preserving its durable-before-send guarantee. Moving that work off the playback thread must retain ordering and shutdown durability; do not casually replace it with fire-and-forget writes. Device timing is needed before claiming this is a demonstrated bottleneck.
