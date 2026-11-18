# Android 0.16.76 candidate

**Published September 8, 2026 at 06:53 UTC.** The publication record below supersedes chronological preparation statuses.

Status: preparation, not published. Source manifests specify version 0.16.76/code 237. Live remains 0.16.75/code 236. Do not deploy candidate update metadata until the matching APK is verified and archived.

## Scope

- Android wiki draft-preserving mutations, post-write refresh reporting, backlink ownership, shared tree ancestry, and exact image-path matching.
- TV music queue persistence boundaries, identity-based actions, and live removal focus targets.
- Playback queue duplicate batching, removed-metadata cleanup, and preparation-before-mutation ordering.
- Paired server vault bounded reads and move/trash durability ordering.
- Web vault inline image rendering, page/attachment separation, empty-vault creation controls, and request/form ownership safeguards.

Current-release-only update notes describe this batch, not the prior release. Existing persistent optimized signing configuration is retained. No user data, compose mounts, or running containers were changed during preparation.

## Release gates

Pending: current Android unit tests/lint, full Go tests/vet, frontend/browser checks, optimized Direct/Store builds, signature/version verification, immutable artifact archives, isolated staged server smoke, then publication. Existing debug tests passed during implementation but do not substitute for optimized packaging. Physical Android/TV/TalkBack acceptance remains unavailable; do not claim it was performed. The broader roadmap remains open.

## Current verification

Full Go tests, Go vet, 476 frontend tests, module checks, and isolated browser mutation/playlist checks at 1440px and 390px passed. Both Android unit-test tasks completed within the combined candidate invocation. Android lint and optimized Direct/Store packaging are still active in that same invocation; completion and artifacts have not been verified. No release mount or live container was replaced.

## Staged image and Android progress

Both Android unit result sets contain 669 tests with zero failures. Direct lint completed; Store lint and optimized assembly remain active in the combined Gradle invocation.

Built the companion server under `opdshelf:android-0.16.76`, image ID `sha256:02c6aa20750000fd5d69f481933df0b940766b19dce8cfce338f9026504ddb27`. An isolated `--rm` smoke container on loopback port 18102 passed health and empty versioned playlist-summary checks; `/proc/1/status` confirms UID 65532. It had no personal config/media mounts and was stopped for automatic removal after checks. The staged image has not been tagged live. Matching APK download/authenticated update metadata checks remain pending verified APKs.

## Lint completed

Current Direct and Store debug lint XML reports each contain zero errors and 84 warnings (same count as the previous release, not a claim that all warnings are resolved). Optimized Direct packaging has reached R8 minification in the original Gradle process. Existing APK files are still from prior output until assembly completes; no archive/signature claim is made yet. Store optimized assembly remains in the same queued invocation. Live deployment is unchanged.

## Final verification and publication

The original combined Gradle invocation completed successfully in 12m42s, including both optimized variants. Both report version 0.16.76/code237 and pass signature verification with the prior certificate SHA-256 `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`. This remains the existing persistent development-format certificate, not a new production Play Store signing decision. Explicit artifact readiness checks passed. Final race-enabled web tests passed in 37.589s; vaultwriter passed as well.

Immutable archives created without replacing older releases:

- Direct: `artifacts/releases/bobobooks-0.16.76-direct.apk`, SHA-256 `ebf3298af027bc15606257cfb7715ca6198cf018a35e2905f4e2908f2b2164cd`.
- Store: `artifacts/releases/bobobooks-0.16.76-store.apk`, SHA-256 `610a7bbb35ab7de5399fade3d86cbd11887e5d0209cece52505dc7e8201a5681`.

An isolated staged container served the Direct APK with matching hash/version and passed test-device login plus authenticated update metadata checks. The initial login assertion expected 200; the endpoint correctly returned 201 for device creation, and the corrected check passed. No personal credentials were used. The temporary container was stopped and automatically removed.

Published matching server image `sha256:02c6aa20750000fd5d69f481933df0b940766b19dce8cfce338f9026504ddb27` and immutable Direct APK mount together. Container started at `2026-09-08T06:53:21.419609294Z`; healthz returned ok, restart count zero, mounted APK hash verified. All ten libraries became ready. The pre-existing malformed EPUB missing `META-INF/container.xml` remains a scan warning, not a startup failure. Previous server/APK release retained for rollback. Physical mobile/TV acceptance still remains unverified; broader goal remains active.
