# Audio journal read optimization

Status: source-only after 0.16.80; no APK build or deployment.

Replay already filters audio entries to the paired device identity; video journal creation and replay likewise use their owner identity. This inspection does not prove every account-transition lifecycle, but it found no path here that intentionally replays foreign-owner rows.

Audio `entries()` previously encrypted and persisted its filtered list on every call, even if unchanged (and cleared storage again for an already empty list). It now persists only when filtering removes rows. Since filtering only removes entries without transforming them, comparing sizes is sufficient; no extra full-list equality pass is needed. Same-owner repeated reads avoid an encryption/commit; mixed-owner and legacy pruning remain intact. A failed prune never makes foreign rows eligible for replay.

The policy regression counts zero persistence calls for repeated same-owner/empty reads, verifies foreign-owner pruning, and checks replay isolation after persistence failure. Both Android flavor targeted tests run through `/tmp/bobobooks-audio-journal-read-tests.log` (session 70978). This is a reduced-write-count improvement, not a measured end-to-end latency or RAM percentage claim.
