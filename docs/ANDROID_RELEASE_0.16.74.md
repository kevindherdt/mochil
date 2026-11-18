# Android 0.16.74 release preparation

Published September 7, 2026 at 17:44 UTC: 0.16.74 / code 235. The publication section below is authoritative; preparation/staging entries record the sequence of checks.

## Batch

- Playlist overviews load names, counts, and bounded artwork previews instead of all playlist tracks.
- Opening/playing a playlist fetches full tracks, with loading/retry feedback and stale-selection protection.
- Android item edits fetch only their target playlist on supported servers; older servers retain a fallback.
- Server playlist names containing literal percent signs are decoded once, not twice.
- Web single-track additions use atomic append instead of replacing a possibly stale local playlist.
- Single-playlist ZIP preparation no longer copies and sorts all other playlists.

Keep Android in-app release notes specific to Android-visible changes in this batch, not prior Wiki/TV/queue fixes. There is no new TV layout in this release.

## Completed pre-package checks

- Full Go suite, vet, complete audio/web race tests.
- Full Direct and Store Android unit suites: 622 tests each, zero failures/errors/skips.
- Frontend: 456 tests; isolated Chromium playlist flows at 1440px and 390px.
- Module-size and source store-readiness gates.
- Direct lint: zero errors, 84 warnings. Store lint remains running at this checkpoint.

Follow-up: both lint tasks completed successfully in 4m 49s, with zero errors and 84 warnings each. Worktree Android/server versions are now 0.16.74 / code 235; the live server and archived APK mount still remain 0.16.73. Optimized Direct/Store packaging has started with `verifyOptimizedSigning` using the existing persistent key. Packaging, signatures, archive checks, and publication are not yet complete.

## Remaining release gates

1. Collect final Store lint result; do not restart an active process.
2. Set Android and server release metadata to 0.16.74 / code 235 together in the worktree.
3. Build both optimized variants using the existing persistent key, not an ephemeral CI identity.
4. Verify APK versions, signatures, store-readiness checks, and SHA-256 values. Archive immutable APKs without overwriting prior releases.
5. Stage the matching server image. Deploy only after the archived Direct APK and server version metadata match.
6. Verify live health, restart count, and mounted APK hash. Record publication evidence here.

## Device acceptance still needed after publication

Update over the installed release without uninstalling. Verify playlist overview artwork/counts, opening and playing, moving/removing tracks, adding on two devices, and navigating away during a slow detail request. Try a playlist name containing `%`. Verify errors/retry without silently replacing playlists with empty arrays. Android TV remote/visual testing remains unverified without a connected device.

The synthetic server response measurement (20 tracks: 13,797 → 536 uncompressed bytes) is not a production latency or RAM claim. Cross-device compare-and-swap for replacement edits remains future work; atomic append does not solve every concurrent mutation.

## Staging checkpoint

Matching server image built successfully as `opdshelf:android-0.16.74`, image ID `sha256:249f01863118cee0f8ac951f6ccb64fa07627b0da5b0fec9b58d3782f335064d`. It has not been tagged as the live image or deployed. The live container still uses `sha256:33d6ad7e11c0afcd5032e1da14e90684c95f0b1c83dcb89247f80e5476f68c6c`, remains healthy with zero restarts, and mounts the archived 0.16.73 APK.

Optimized packaging remains active after Direct R8 completed. Do not start a second build solely because packaging takes several minutes. No ADB device is connected for physical acceptance testing.

Direct optimized packaging completed. APK version is 0.16.74 / code 235. `apksigner verify --print-certs` succeeded; certificate SHA-256 `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246` matches the installed release. The artifact readiness gate passed. Archived as `artifacts/releases/bobobooks-0.16.74-direct.apk`, SHA-256 `1c90ee59a4a5d71be109edcbe11f72d995333a5a4864f313861fe5a6086f9fc7`; archive and build output compare identically. Store optimized packaging is still running in the same process. The archive is not yet mounted live.

The staged image passed an isolated runtime smoke test on loopback port 18102, with no personal configuration/media mounted. It returned `ok` at `/healthz`, ran as UID 65532 with zero restarts, served the versioned empty summary, and created/read a synthetic `Release 100%25 / test` playlist through the actual HTTP router with its name intact. The `--rm` smoke container was stopped and automatically removed, including its anonymous test volumes. No live library/account data was changed. Store packaging has completed R8 and is still performing vital lint at this checkpoint.

## Publication verification

Both optimized packages completed in 8m 11s, including R8 and vital lint. Both report 0.16.74 / code 235 and pass signature verification with the existing certificate. Artifact readiness checks passed for both.

- Direct archive: `artifacts/releases/bobobooks-0.16.74-direct.apk`, SHA-256 `1c90ee59a4a5d71be109edcbe11f72d995333a5a4864f313861fe5a6086f9fc7`.
- Store archive: `artifacts/releases/bobobooks-0.16.74-store.apk`, SHA-256 `2cad2eec68e33f485b4e696314efa917885eb7fe70dba7865b577c7bdbd60ce6`.

The matching staged image was tagged `opdshelf:local` and Compose recreated only `opdshelf` with `--no-build --no-deps`. Live start time: `2026-09-07T17:44:18.753654451Z`. Image ID matches the staged image above. Health reports healthy, `/healthz` returns `ok`, restart count is zero, and the live mounted APK hash matches the archived Direct APK. Ten libraries loaded. The pre-existing malformed EPUB warning remains unchanged.

Older archived APKs are retained. Store output is a private verification artifact using the existing development-format persistent signing certificate, not a Play Store publication or production store-signing decision. In-place installation and physical-device acceptance remain unverified without an attached device.
