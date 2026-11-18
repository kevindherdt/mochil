# 0.16.82 / code 243 preparation

Status: published September 8, 2026 at 14:02 UTC. The outcome below supersedes chronological preparation notes.

## Published outcome

- Live image `sha256:bc649154b53742b4eb5ef44ea2a1a8466f244202e4cedff10c2770337c5a0507`, started `2026-09-08T14:02:40.124739739Z`, healthy with zero restarts.
- Combined optimized build passed in 8m30s. Both archives verify version 0.16.82/code243 and the established signing certificate; explicit combined artifact/distribution gate passed.
- Direct SHA-256: `6c304b4746096b868bd277593071a398f36d7f0562e9aa6ab09fad9df71e36e7`.
- Store SHA-256: `e875e2f41d326366ef72d51c7195f7f6d5b957b270fe9d7237fb0db4af4d42fe`.
- Authenticated candidate update/download matched exact Direct bytes/hash; live mounted APK hash matches the archived Direct package.
- All ten libraries loaded. The known malformed EPUB missing `META-INF/container.xml` remains the sole scan warning; no media files modified.
- 0.16.81 image and archives retained for rollback. Temporary smoke containers stopped. Only Mochil was redeployed; Pulse and other services untouched.
- Physical-device acceptance and broad roadmap work remain outstanding. These checks do not constitute Play Store approval.

## Scope

- Wiki filename-based and exact-case link resolution; repeated Markdown suffix and missing-note creation fixes.
- Background, cancellation-aware offline wiki image planning without unused outline construction.
- Revision-conditional playlist rename/delete in web and native clients.
- Visible-track remove/reorder preserves unseen references on supporting servers, separate from explicit whole-list replacement.

No mobile layout changes or new performance-percentage claims. Current-only notes are in `internal/web/android_distribution.go` with a regression rejecting prior-release wording.

## Active verification/build handles

- Session 86828: full Go tests/vet, frontend tests, module/distribution checks, then candidate Docker image build `opdshelf:android-0.16.82`. Logs `/tmp/bobobooks-01682-go.log`, `-vet.log`, `-web.log`, `-distribution.log`, `-image.log` (same prefix).
- Session 36307: both full native flavor unit suites and lint, then (only on success) Direct/Store optimized assemblies. Logs `/tmp/bobobooks-01682-android-verification.log` and `/tmp/bobobooks-01682-build.log`.
- Browser session 4368 completed successfully: playlist forms/drag, wiki render/tasks and video progress, at 1440/390 where applicable.

Go/frontend gates have passed, including all 555 frontend tests. Server image build and Android lint/build pipeline remain active. Revalidate these exact sessions; do not restart because a log has no recent output.

## Release gates still required

Confirm both pipelines finish, inspect fresh APK package/version and established certificate, archive without overwriting releases, and run explicit Direct/Store artifact checks. Exercise authenticated update/download hash, expanded playlist smoke checks and real-server wiki fixtures on disposable candidate containers. Only then update the live image and private APK bind mount together, recheck health/restarts/library readiness and exact downloaded bytes. Stop disposable containers afterward.

Rollback release 0.16.81 is retained as `opdshelf:android-0.16.81`, image `sha256:6b34947699d825cb484235c3cd1824e31d125f995235a8ed124e965992e8999e`; archive `artifacts/releases/bobobooks-0.16.81-direct.apk` SHA-256 `61fcf55b1bda09b12e52ae5d80bacfcbc149e09539ecb4d3ce10339c10becb37`. Established signing certificate SHA-256 is `5ed41e985ea64d59c664abde1c10cb43667f968a9618b4996a076f658e247246`. Never replace the signing key.

Physical-device acceptance and the broader project goal remain incomplete. Do not modify Pulse or personal media as part of this release.

## Candidate image and browser evidence

Session 86828 completed successfully. Candidate image: `sha256:ef6db1d65f114320d3596bd041e6837c4ced0a472f855823927e8db21d0dde46` (`opdshelf:android-0.16.82`).

Disposable candidate servers passed authenticated playlist creation/revision/rename/delete smoke checks and real-server wiki task persistence, properties, tables and heading navigation at 1440px and 390px (session 22657, successful). Both temporary containers were stopped; no personal data was mounted. The synthetic wiki host fixture remains for reuse.

Fresh Android unit reports contain 805 tests per flavor, zero failures/errors. Session 36307 remains in lint and will proceed to optimized assembly only on success. No new APK has been verified, archived or deployed yet. Continue polling this existing handle.

## Optimized build and repeated-track correction

Android verification completed successfully in 5m45s. Session 36307 has advanced to the combined optimized build (`/tmp/bobobooks-01682-build.log`), currently Direct R8; it is the same pipeline, not a replacement job.

A final server review found that stored playlists permit repeated tracks. Visible-edit merging now counts original occurrences rather than rejecting all duplicates, preserving unseen occurrences while allowing repeats in the requested ordering. Regressions cover repeated replacements, repeated originals and partially unseen duplicate occurrences. Android source is unchanged by this correction.

Session 68115 runs fresh full Go tests, ten focused race repetitions and a replacement candidate image build (`/tmp/bobobooks-01682-repeat-go.log`, `-repeat-race.log`, `-image-final.log`). The earlier candidate image hash above is superseded for release purposes once this build succeeds. Re-run candidate server checks against the final image before deployment.

Rollback 0.16.81 APK hash/image were rechecked unchanged. Nothing published yet.

## Final candidate server verified

Session 68115 completed successfully. Final candidate image is `sha256:bc649154b53742b4eb5ef44ea2a1a8466f244202e4cedff10c2770337c5a0507`, replacing the earlier candidate hash. Full Go tests and ten repeated merge race tests passed before that image build.

The final image passed the authenticated playlist smoke script and real-server wiki browser fixture again at both widths (session 40091, successful). Disposable final-playlist/final-wiki containers were stopped afterward.

Session 36307 remains active in optimized assembly, currently Direct release lint. The APK output observed at this point is still the old 0.16.81 artifact (12:58 timestamp); do not archive it as 0.16.82. Fresh version/certificate checks and both flavor artifact gates remain mandatory.

## Direct artifact verified

Direct assembly subsequently completed. `aapt2` verifies 0.16.82/code243; `apksigner` verifies the established certificate. Archive `artifacts/releases/bobobooks-0.16.82-direct.apk` is 7,062,376 bytes, SHA-256 `6c304b4746096b868bd277593071a398f36d7f0562e9aa6ab09fad9df71e36e7`.

The final candidate image passed authenticated update metadata and exact byte/hash download verification against this archived APK (`android-update-smoke.mjs`). The temporary update container was stopped afterward. Store R8 remains active in session 36307; Store signature/archive and combined distribution gate remain pending. No live deployment yet.
