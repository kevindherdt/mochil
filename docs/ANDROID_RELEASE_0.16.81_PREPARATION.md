# 0.16.81 / code 242 preparation

Status: 0.16.81/code242 published on 2026-09-08 at 13:03 UTC. The published outcome below supersedes chronological preparation statuses.

## Published outcome

- Combined optimized build completed in 8m41s; both APK versions and established signing identity verified. Explicit Direct/Store distribution gate passed.
- Direct archive SHA-256: `61fcf55b1bda09b12e52ae5d80bacfcbc149e09539ecb4d3ce10339c10becb37`.
- Store archive SHA-256: `63fb6856f4b6608fa803554901b6dac455cbbd06526fbe9cbaf3ca51a5ae0250`.
- Live image: `sha256:6b34947699d825cb484235c3cd1824e31d125f995235a8ed124e965992e8999e`, started `2026-09-08T13:03:58.672320533Z`. Health check passed, restart count zero.
- All ten libraries loaded. The known malformed EPUB missing `META-INF/container.xml` remains the sole light-novel scan warning. No media files changed.
- Mounted download hash matches the certified Direct archive. Authenticated APK update/download and playlist revision tests plus real-server wiki fixtures passed before deployment.
- 0.16.80 image tag and archives retained for rollback. No unrelated services were changed. Physical-device acceptance, broader roadmap and documented conditional-edit limitations remain open.

## Current-release-only notes

Fixes:
- Web audio/video preserve intentional seeks across delayed saves, revision conflicts and temporary network failures.
- Duplicate playlist creation no longer empties an existing playlist.
- Web and Android playlist replacements use revision checks to detect another device's intervening changes.
- Android audio/video outboxes preserve unreadable stored progress rather than overwriting it, and overlapping audio journal instances serialize their writes.

Performance:
- Android audio recovery skips redundant encrypted writes when the saved outbox is unchanged.

Improvements:
- Playlist edit conflicts reload current details without automatically replaying the rejected replacement.
- Failed durable audio checkpoints record a content-free diagnostic event.

These notes must not imply all playlist operations are conditional: rename/delete and missing-media preservation remain separate gaps. Older clients/servers without revision headers retain legacy behavior. No broad visual/performance percentage claims; existing mobile UI remains unchanged.

## Running verification

- Full Go tests/vet, frontend tests and module/distribution checks: session 86576; `/tmp/bobobooks-01681-go.log`, `/tmp/bobobooks-01681-vet.log`, `/tmp/bobobooks-01681-web.log`.
- Both Android unit suites and lint: session 60973; `/tmp/bobobooks-01681-android-verification.log`.
- Browser playlist forms/drag, video progress bindings and wiki render/task fixtures: session 35023.

Poll existing handles rather than restarting based on elapsed time. After gates pass, apply source version/current-only notes, build fresh signed artifacts, archive without replacing earlier releases, run explicit artifact checks and isolated authenticated update/server tests, then deploy with 0.16.80 retained for rollback. Do not change unrelated containers or private media/configuration. Hardware mobile/TV acceptance and the broad roadmap remain unverified/incomplete.

Go tests/vet, all 550 frontend tests and module/distribution checks completed successfully (86576 exited zero). All listed browser fixtures passed (35023 exited zero), including 1440px/390px layouts where applicable. Android 60973 remains active in lint analysis. Live 0.16.80 was checked healthy with zero restarts.

Server release notes now contain only this batch's changes; the metadata contract test rejects prior release notes and requires the new seek, reduced-write and playlist-conflict notes. `go test ./internal/web -run TestAndroidApp -count=1` passed after this update. Android version bump/build must wait for the existing verification process to finish.

Server image build launched independently in session 64477, `/tmp/bobobooks-01681-image.log`, tag `opdshelf:android-0.16.81`. Current Android reports contain 795 tests per flavor with zero failures/errors. Store lint is still active in 60973. Added `scripts/android-playlist-revision-smoke.mjs` for the synthetic authenticated candidate: duplicate creation, intervening case-only rename, stale 412 and fresh-token PUT, followed by fixture playlist cleanup. Run it against the disposable update-smoke server, never the personal instance.

Android verification 60973 subsequently completed successfully in 4m23s. After completion, Android metadata advanced to 0.16.81/code242 and the combined fresh Direct/Store optimized build launched in session 32476, `/tmp/bobobooks-01681-build.log`. Poll existing build handles; no APK archive/signature checks or deployment yet.

## Candidate server verification

Image build 64477 completed successfully: `sha256:6b34947699d825cb484235c3cd1824e31d125f995235a8ed124e965992e8999e` (`opdshelf:android-0.16.81`).

- Disposable authenticated server passed `android-playlist-revision-smoke.mjs`: duplicate create 409, intervening state change, stale PUT 412 and fresh revision accepted. Synthetic playlist was removed by the test.
- Disposable synthetic vault server passed health and real-server wiki task persistence/properties/table/heading navigation at 1440/390px (33866 exited zero).
- Both smoke containers were stopped and automatically removed; only the synthetic host vault fixture remains. No personal data was mounted in either candidate.
- APK optimization is still running in 32476. Fresh artifact signatures, archive checks and authenticated APK download verification remain pending before deployment.

## Direct artifact

Direct assembly completed. aapt2 verifies 0.16.81/code242; apksigner verifies the established certificate SHA-256 `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`. Immutable archive: `artifacts/releases/bobobooks-0.16.81-direct.apk`, 7,062,376 bytes, SHA-256 `61fcf55b1bda09b12e52ae5d80bacfcbc149e09539ecb4d3ce10339c10becb37`.

Authenticated update metadata and exact APK byte/hash download test passed against the candidate image (`android-update-smoke.mjs`). Disposable update container stopped/removed afterwards. Store optimized build still runs in 32476; combined explicit artifact gate and deployment remain pending. 0.16.80 image/archive hashes were rechecked and retained for rollback.
