# Android 0.16.77 / code 238

Status: published 2026-09-08 at 08:12 UTC. Live is 0.16.77/code238. The final record below supersedes earlier chronological pending notes.

## Batch

- Audio progress: exact committed seek acknowledgment, cancellation-aware replay/live sends, bounded latest-per-item delivery, admission checks after network-lock waits and remote reads, late-producer protection, durable acknowledgment transfer.
- Wiki: incremental offline byte budgeting, bounded/cancelable batches, selected-page retention, earlier encrypted snapshot size limits.
- Companion web music: favorite/playlist refresh ownership and failure retention, shared playlist detail reads, stable submitted track identities, explicit queue replacement ownership, dedicated favorites/queue modules.

Detailed evidence and limitations: `ANDROID_AUDIO_PROGRESS_REPLAY_2026-09-08.md`, `ANDROID_WIKI_OFFLINE_BUDGET_2026-09-08.md`, and `WEB_MUSIC_FAVORITE_REFRESH_2026-09-08.md`.

Current-release-only changelog and version metadata are prepared. Existing signing material is retained. No UI redesign or personal-library changes are included.

## Gates

Pending candidate unit/lint checks, optimized Direct/Store builds, Go/frontend/browser checks, artifact version/signature/privacy gates, immutable archives, isolated server/update smoke, then deployment. Existing debug tests do not prove optimized packaging. Physical mobile/TV testing remains unavailable.

Do not change the live APK mount or announce the update available before artifact and staging checks pass. Broader feature goals remain active.

## Candidate checks in progress

Full Go tests, Go vet, all 499 frontend tests, module checks, and isolated playlist/vault-mutation browser checks at 1440px and 390px passed. The combined Gradle invocation has completed both debug unit-test tasks and reached lint work; optimized packaging remains in the same running invocation. Its log is `/tmp/bobobooks-release-01677-gradle.log` (execution session 67610). Observe that process rather than starting another build while it remains active.

The live container was rechecked as healthy with zero restarts. No compose mount or live image changed.

## Server staging and race checks

Race-enabled web tests passed in 30.169s; vaultwriter passed from cache. Candidate image `opdshelf:android-0.16.77` built successfully with ID `sha256:faedef54deeb03d264f0546f388c5f91caa77c51245e565e3236f34c06263e53`. A disposable container on loopback port 18102, without personal mounts, returned health `ok`, returned the version-1 empty playlist summary, and ran PID 1 as UID/GID 65532. It was stopped and automatically removed after checking.

This preliminary startup smoke did not verify the new APK/update endpoint; that remains pending Android packaging. The original Gradle process (session 67610) is still active in lint work. The live image and APK mount remain unchanged.

## Android debug checks complete

Both candidate unit-result sets were counted directly: 701 tests each, zero failures/errors. Direct and Store debug lint reports each contain zero errors and 85 warnings (one more warning than the previous release's recorded total; not a claim of a clean warning audit). The original Gradle process has reached `minifyDirectOptimizedWithR8` and remains live, including a verified 30-second wait. Optimized artifacts are not yet certified; do not use leftover output files as evidence for this candidate.

## Direct artifact and authenticated update smoke

Direct optimized assembly completed. APK metadata reports 0.16.77/code238; signature verification passed with the existing certificate SHA-256 `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`. The immutable Direct archive is `artifacts/releases/bobobooks-0.16.77-direct.apk`, SHA-256 `6e892a0021c97a7b1ed492c9a4161c927d7c233161b48cf5f82c48480f7928da`. Explicit artifact readiness checks passed for this APK. The certificate remains the existing development-format certificate, not a production signing decision.

A separate disposable candidate server, using only synthetic credentials and the read-only Direct archive, passed device login (201), authenticated update metadata (238/0.16.77 with current notes), and authenticated APK download with exact hash/header verification. It was stopped and automatically removed. No personal data mounts were used.

Store minification has finished and final packaging checks remain in the original Gradle session 67610. Live publication still waits for the complete build and Store artifact verification. No live mount/image has changed.

## Final verification and publication

The original combined Gradle process completed successfully in 12m56s. Both optimized variants report 0.16.77/code238 and verify against the existing certificate. Explicit readiness checks passed for both immutable archives:

- Direct: `artifacts/releases/bobobooks-0.16.77-direct.apk`, SHA-256 `6e892a0021c97a7b1ed492c9a4161c927d7c233161b48cf5f82c48480f7928da`.
- Store: `artifacts/releases/bobobooks-0.16.77-store.apk`, SHA-256 `a7eb743926e7b4a9d959c06ee0b39d992ce47d725fb9bbbc2b2caa3d864aefe0`. This remains a private verification artifact, not a Play Store submission.

Published tested image `sha256:faedef54deeb03d264f0546f388c5f91caa77c51245e565e3236f34c06263e53` as `opdshelf:local`, changed only the Direct APK archive mount in the private compose override, and recreated only `opdshelf` without rebuilding. Container start: `2026-09-08T08:12:50.590502694Z`. Health passed, restart count zero, and the mounted APK hash matched the verified Direct archive.

All ten libraries reached ready: books324, anime39, tvseries36, movies3, wiki7, lightnovels5149, manga262, podcasts62, audiobooks409, music1388. The pre-existing malformed EPUB warning (missing META-INF/container.xml) remains; no source media was changed.

Rollback image `opdshelf:android-0.16.76` and prior immutable APK archives remain available. Physical mobile/TV acceptance and the broader feature roadmap remain outstanding; no claim of full goal completion.
