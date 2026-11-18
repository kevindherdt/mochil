# Android 0.16.78 / code 239

Status: published 2026-09-08 at 09:48 UTC. Live is 0.16.78/code239. The final publication record below supersedes chronological preparation notes.

## Batch

- Wiki: explicit committed-write/index-refresh outcomes, session-local duplicate-operation guards, cancellation recovery, request-setup cleanup, incremental markdown parsing and bounded table parsing.
- Playback: tighter single-progress response limits shared by audio/video; bounded latest-per-item video delivery, encrypted pending checkpoints, replay on player opening and matching local resume.
- Video identity: optional indexed media-revision preconditions prevent queued progress from updating a replaced file on an upgraded server.
- Companion web: safer markdown links and saved-but-refresh-failed wiki feedback without a layout redesign.

Detailed evidence and limitations live in `WIKI_PARITY_AUDIT_2026-09-08.md`, `ANDROID_VIDEO_PROGRESS_ORDERING_2026-09-08.md`, and `ANDROID_RESPONSE_BOUNDS_2026-09-08.md`.

## Pre-packaging verification

Before the version/changelog bump, full Go tests and vet, module checks, 507 frontend tests, and isolated wiki mutation browser checks at 1440px and 390px passed. Both Android debug test tasks were verified up to date by Gradle, not freshly executed. Version/changelog tests must be rerun after this bump.

## Outstanding release gates

Run candidate debug tests and lint, optimized Direct/Store assembly, artifact version/signature/privacy checks, immutable archive creation, and isolated authenticated update/download smoke before publication. Preserve the existing signing certificate. Do not announce an available update until deployment and mounted APK verification succeed.

Physical mobile/TV acceptance remains unavailable. Wiki recovery guards are session-local, not a durable wiki operation journal. Video retries occur on player opening, not through a continuous background service. Media-revision protection uses the indexed revision and requires an upgraded server. This release does not complete the broader feature goal.

## Candidate metadata verification

After the bump, uncached Android server endpoint tests and module checks passed. Both Android debug unit-test tasks executed successfully in a 20-second Gradle invocation (24 tasks executed, 28 up to date); log: `/tmp/bobobooks-01678-unit.log`. No packaging or deployment has started.

## Packaging in progress

The Direct and Store XML unit reports each contain 754 tests, zero failures, errors or skips. Race-enabled web tests passed in 31.287 seconds; vaultwriter passed from cache. Playlist interaction fixtures passed at both 1440px and 390px.

The combined lint/optimized build is now running in execution session `91614`, log `/tmp/bobobooks-01678-packaging.log`, last observed in Direct debug lint analysis. A separate candidate container build is running in session `40546`, log `/tmp/bobobooks-01678-image.log`, targeting `opdshelf:android-0.16.78`. Both processes were polled and confirmed live. Resume these handles rather than starting duplicate builds. No candidate artifact has been certified or deployed.

## Server candidate built

Container build session `40546` finished successfully. Candidate image: `opdshelf:android-0.16.78`, ID `sha256:ecf9fb3916295843635bd7211765a0c9ed4c8fca46ec169a26970bc60a21b3e8`.

A disposable container bound only to loopback port 18102, with no personal mounts, returned health `ok`, returned `[]` from `/api/audio-playlists`, and ran PID 1 with UID/GID 65532. The initial probe mistakenly requested `/api/playlists` and received 404; the route was inspected and the correct endpoint then passed. The disposable container was stopped and automatically removed.

Authenticated update/APK verification still awaits packaging. Android build session `91614` remains running and has advanced to Store debug lint. Live image and APK mount remain unchanged.

## Lint complete; optimized packaging remains live

Both freshly generated debug lint reports contain zero errors and 85 warnings (Direct report 09:37:57 UTC, Store 09:39:58 UTC). This is not a warning-free audit. Build session `91614` has advanced into Direct optimized packaging and was polled successfully as still running. No replacement build was started. Live Mochil remains healthy with restart count zero.

## Repeatable authenticated download check

Added `scripts/android-update-smoke.mjs` to verify fixture device login, version/code/current notes shape, download headers, byte count, and exact APK SHA-256. It accepts only an explicit loopback HTTP origin and uses synthetic credentials. Syntax verification and an end-to-end control run against a disposable 0.16.77 image/archive passed (7,062,376 bytes and the archived hash). The control container was stopped and automatically removed. This validates the harness, not the pending 0.16.78 APK.

The original Android session `91614` is still running after Direct R8; final optimized packaging and Store assembly remain pending. Do not certify old APK output files simply because they exist.

## Direct candidate verified

Direct optimized assembly completed. APK metadata reports 0.16.78/code239 and signature verification retains certificate SHA-256 `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`. The immutable archive `artifacts/releases/bobobooks-0.16.78-direct.apk` passed the explicit artifact readiness check. Its SHA-256 is `98bf2dd8c84f2075a2a556895fe58cbd72a9c6cde30d53270df2c56c77822958`.

The candidate image plus this read-only APK passed `scripts/android-update-smoke.mjs` on a disposable loopback server with synthetic credentials: login, version239/0.16.78, release-note categories, download headers, 7,062,376 bytes, exact hash. The fixture was stopped and automatically removed. No live configuration or user credentials were used.

Session `91614` remains running in Store R8. Publication still awaits successful completion and Store artifact checks; live is unchanged.

## Final verification and publication

The original Gradle session completed successfully in 13m8s. Both optimized APKs report 0.16.78/code239 and retain the existing signing certificate. Explicit artifact readiness checks passed for both archives:

- Direct: `artifacts/releases/bobobooks-0.16.78-direct.apk`, SHA-256 `98bf2dd8c84f2075a2a556895fe58cbd72a9c6cde30d53270df2c56c77822958`.
- Store: `artifacts/releases/bobobooks-0.16.78-store.apk`, SHA-256 `6083329b8d04ef8a86a7fd80888a3ec16d1bc7e320405869931cb6f7baca2534`. Private verification artifact only, not a Play Store submission or production signing decision.

Published the tested candidate image as `opdshelf:local`, changed only the Direct APK mount in the private compose override, and recreated only `opdshelf` without rebuilding. Start time: `2026-09-08T09:48:38.701721186Z`. Live image ID matches the candidate, health returned `ok`, restart count was zero, and the mounted APK hash matched the verified Direct archive.

All ten libraries reached ready: books324, anime39, tvseries36, movies3, wiki7, lightnovels5149, manga262, podcasts62, audiobooks409, music1388. The existing malformed EPUB warning (missing META-INF/container.xml) remains; source media was not modified. The prior 0.16.77 image and immutable APK archives remain available for rollback.

Physical mobile/TV acceptance and the broader roadmap remain outstanding. No claim of full goal completion.
