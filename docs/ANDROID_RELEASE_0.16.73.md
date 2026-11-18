# Android 0.16.73 release

Published September 7, 2026 at 16:50 UTC: 0.16.73 / code 234.
The preparation chronology below records the staged checks; the publication
section is authoritative for current deployment status.

Batch: music queue request ordering, playlist serialized edits and rename
selection preservation, Wiki combined image cache budget and early download
stopping, TV queue-first Back, and descriptive seek semantics. No layout redesign.
Server release notes contain only these Android changes, not earlier releases
or unrelated server/CI maintenance.

The live Compose override now mounts the archived 0.16.72 Direct APK instead of
Gradle's mutable output path. Its verified SHA-256 is
`559d17d7ef60f241471d7547ab6776335beeb9ece55e3d5b655036f9703525a6`.
The current server remains healthy with zero restarts. Do not deploy new server
version metadata until the matching APK has completed verification.

Validation passed for both distributions: 612 unit tests each, full lint (zero
errors), and the persistent-key signing gate, in a 4m 58s combined Gradle run.
All 448 frontend tests, web version/notes tests, and module/readiness gates pass.
Optimized Direct and Store packaging is now running with the existing key.
After packaging, check version and certificate,
archive the artifacts, then point the live mount to the new archived Direct APK
and deploy the matching server metadata together. Do not use CI's ephemeral key
for installed-user updates.

Matching server image is staged as `opdshelf:android-0.16.73`, image ID
`sha256:33d6ad7e11c0afcd5032e1da14e90684c95f0b1c83dcb89247f80e5476f68c6c`.
It is not yet the live image/tag. Do not deploy until APK validation is complete.

Direct optimized packaging has completed and its safety gate passed. Archived
artifact: `artifacts/releases/bobobooks-0.16.73-direct.apk`, SHA-256
`37c07d4c142ca4aac9cfff26b0163e148c9cb22b790432f557f6cc34f0b3980e`.
Manifest reports version 0.16.73 / code 234. Verified signing certificate SHA-256
`5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`
matches 0.16.72. Store packaging is still running in the same Gradle process;
neither the new archived Direct APK nor staged server image is live yet.

Physical TV and TalkBack acceptance remain unverified without a connected device.

## Device acceptance after publication

- Update over 0.16.72 without uninstalling; verify the server connection remains.
- In music, make rapid queue edits while a save is in progress. The final visible
  queue should remain the latest edit. With connectivity unavailable, a failed
  save should leave the edit visible and report failure (not claim it is synced).
- Rename a playlist, then navigate to another playlist while the request is
  pending. Completion must not return to the old selection. Rapid remove/move
  actions must target the originally selected tracks.
- Mirror an image-heavy Wiki vault. Budget feedback must remain honest when only
  some images fit; page reading and already cached images must remain usable.
- On TV, open Queue and select the on-screen Back button; it should return to
  the transport just like remote Back. Confirm D-pad focus and TalkBack seek
  announcements on physical hardware.

No connected ADB device was available during packaging. These checks remain
explicitly unverified rather than being inferred from unit or lint results.

## Publication verification

Both optimized packages completed in 7m 24s, including R8 and vital lint.
Both report code 234/version 0.16.73 and the same persistent signing certificate
as 0.16.72. Store archive SHA-256:
`dde288c491ad5701e89343ba80622b6fb39a8ddedaf6c0d1f0789cbb03a7ca19`.
The Store package remains a private verification artifact, not a Play Store
submission or a production store-signing decision.

The archived Direct package is mounted read-only in the live container. Its
checksum matches the verified Direct checksum above. The running server image
matches the staged image above, health is healthy with zero restarts, and
`/healthz` returns `ok`. All ten libraries loaded; the previously identified
malformed light-novel EPUB remains a non-blocking scan warning. Only opdshelf
was recreated; no user media was changed. The old APK archives remain available.

This release includes the previously pending TV, Wiki, queue, and playlist
changes documented in the September 7 audit notes. Device acceptance is still
pending; successful packaging is not proof of physical remote/TalkBack behavior.
