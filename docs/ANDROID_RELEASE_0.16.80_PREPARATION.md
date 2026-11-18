# Android 0.16.80 / code 241 preparation

Status: 0.16.80/code241 published on 2026-09-08 at 12:06 UTC. The chronological preparation notes below retain their original verification order; the published outcome is authoritative.

## Published outcome

- Combined optimized build completed successfully in 8m30s. Direct and Store package versions are 0.16.80/code241; both signatures match the established certificate.
- Explicit Direct/Store archive distribution gate exited zero. Store archive SHA-256: `0b6b5e34ed16afd3c807dc517b8918cdda581246bc943784a41bd1d3d22bb84c`.
- Authenticated update metadata and APK download, isolated server health, and real-server wiki browser tests passed before deployment.
- Personal Compose APK mount advanced to the immutable 0.16.80 Direct archive. Live image is `sha256:aa86c4955c76b5af2d4c01cb0890fc12496e0e8d6572d5ae3c84d0ff5b256d42`, started `2026-09-08T12:06:51.444154974Z`, healthy with zero restarts.
- All ten libraries loaded. The existing malformed EPUB missing `META-INF/container.xml` remains the sole light-novel scan warning; no media was modified.
- 0.16.79 image tag and both archived APKs remain available for rollback. No unrelated service was changed. Physical-device acceptance and broader roadmap work remain outstanding.

## Current-release-only notes

Fixes:
- Web audio/video progress saving recovers after temporary network failures; late responses from an old item cannot alter the current player's sync ownership.
- Rapid web playlist removals and reordering preserve sequential intent instead of restoring stale track lists.
- Android encrypted stores serialize first-use key creation across overlapping instances.
- Wiki attachment references preserve exact-case matches; unsafe missing-note paths do not become proposed local pages.

Performance:
- Android wiki caches path ordering for repeated searches instead of sorting matches on every query. Host-JVM measurements are documented separately; no phone/TV percentage claim belongs in release notes.
- PDF attachments stream to a bounded disk cache without buffering entire documents in memory.

Improvements:
- Android wiki PDF links open in the native PDF reader with local reading progress, loading feedback and cancellation when leaving the note.
- Relative Markdown note/PDF links and local heading links work alongside existing wiki-style links.

## Verification state

Full Go tests/vet, 540 frontend tests, module/distribution gates and focused browser fixtures passed. Both Android flavor full unit suites executed: 786 tests each, zero failures/errors/skips. Combined test/lint verification completed successfully in 4m36s; log `/tmp/bobobooks-next-batch-android.log`.

After current verification completes: apply version/changelog metadata, check its contract, build fresh optimized Direct and Store artifacts, verify versions/signatures, preserve immutable archives, inspect both artifacts with the distribution gate, build/health-check a candidate server and exercise authenticated update metadata/APK download. Only then update the personal Compose service, preserving 0.16.79 rollback. No unrelated services/configuration/private data are in scope.

Physical phone/TV testing is unavailable. PDF fragment page numbers, general attachment mirroring, file-backed wiki history/restore and other documented roadmap work remain incomplete. Existing UI is intentionally preserved except for contextual wiki loading feedback.

## Build handles

Version metadata contract check (`go test ./internal/web -run TestAndroidApp -count=1`) passed after the source version bump.

- Fresh Direct/Store optimized APK build: session 58909, `/tmp/bobobooks-01680-build.log`.
- Candidate image `opdshelf:android-0.16.80`: session 74374, `/tmp/bobobooks-01680-image.log`.

Both launched after the completed test/lint run. Poll these handles or inspect their processes before restarting anything. No new archive, signature certification or deployment has occurred yet.

Candidate server build completed successfully. Image ID: `sha256:aa86c4955c76b5af2d4c01cb0890fc12496e0e8d6572d5ae3c84d0ff5b256d42`. The isolated `bobobooks-01680-wiki-smoke` container passed health and the real-server wiki task persistence, properties, table and heading-navigation fixtures at 1440px and 390px (browser session 45385 exited zero). It mounted only synthetic fixture notes. APK build session 58909 is still running; artifact/update gates remain pending.

Direct artifact completed (`assembleDirectOptimized`), version 0.16.80/code241 verified by aapt2 and signature verified by apksigner. Archive: `artifacts/releases/bobobooks-0.16.80-direct.apk`, 7,062,376 bytes, SHA-256 `d63ab95108ed3c7c0d4dae4c2213632368a98c34f975fee668dc7a480f535aa1`. Certificate SHA-256 remains `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`.

Authenticated candidate login/update metadata/APK download test passed with exact archived byte count/hash (`scripts/android-update-smoke.mjs`). Both disposable smoke containers were stopped and automatically removed; synthetic host fixture retained. Store optimized build and the combined explicit-artifact distribution gate remain pending. Live service remains 0.16.79.
